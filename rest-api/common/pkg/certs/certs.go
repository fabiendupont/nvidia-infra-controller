/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package certs

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultCertDir  = "/var/run/secrets/spiffe.io"
	defaultCACert   = "ca.crt"
	defaultCertFile = "tls.crt"
	defaultKeyFile  = "tls.key"
)

var ErrNotPresent = errors.New("certificates are not present")

// Config holds explicit file paths for the CA cert, TLS cert, and TLS key.
type Config struct {
	CACert  string
	TLSCert string
	TLSKey  string
}

func (c Config) IsSet() bool {
	return c.CACert != "" && c.TLSCert != "" && c.TLSKey != ""
}

func (c Config) Validate() error {
	set := 0
	if c.CACert != "" {
		set++
	}
	if c.TLSCert != "" {
		set++
	}
	if c.TLSKey != "" {
		set++
	}

	if set != 0 && set != 3 {
		return errors.New("ca-cert, tls-cert, and tls-key must all be provided together")
	}

	return nil
}

func (c Config) loadCerts() (*x509.CertPool, tls.Certificate, error) {
	caCert, err := os.ReadFile(c.CACert)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("failed to read CA cert %q: %w", c.CACert, err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, tls.Certificate{}, fmt.Errorf("failed to parse CA cert %q", c.CACert)
	}

	cert, err := tls.LoadX509KeyPair(c.TLSCert, c.TLSKey)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("failed to load cert/key (%q, %q): %w", c.TLSCert, c.TLSKey, err)
	}

	return certPool, cert, nil
}

// TLSConfig builds a client-side tls.Config from the explicit file paths in c.
// GetClientCertificate is used instead of Certificates to unconditionally
// present the client certificate during the TLS handshake, bypassing Go's
// issuer-matching logic that can silently send an empty certificate list.
func (c Config) TLSConfig(serverName string) (*tls.Config, error) {
	certPool, cert, err := c.loadCerts()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
		RootCAs:    certPool,
		ServerName: serverName,
	}, nil
}

func (c Config) ServerTLSConfig() (*tls.Config, error) {
	certPool, cert, err := c.loadCerts()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}, nil
}

// --- Environment-based resolution (SPIFFE / CERTDIR) ---

// IsTLSAvailable reports whether TLS certificates can be resolved from
// explicit paths in c, the CERTDIR env var, or the default SPIFFE directory.
func IsTLSAvailable(c Config) bool {
	if c.IsSet() {
		for _, path := range []string{c.CACert, c.TLSCert, c.TLSKey} {
			if _, err := os.Stat(path); err != nil {
				return false
			}
		}
		return true
	}

	certDir := os.Getenv("CERTDIR")
	if certDir == "" {
		certDir = defaultCertDir
	}

	for _, name := range []string{defaultCACert, defaultCertFile, defaultKeyFile} {
		if _, err := os.Stat(filepath.Join(certDir, name)); err != nil {
			return false
		}
	}

	return true
}

// ResolveServer returns a server-side TLS config and source description.
// Uses explicit paths from c if set, otherwise falls back to CERTDIR / SPIFFE.
func ResolveServer(c Config) (*tls.Config, string, error) {
	if err := c.Validate(); err != nil {
		return nil, "", err
	}

	if c.IsSet() {
		tlsConfig, err := c.ServerTLSConfig()
		return tlsConfig, c.CACert, err
	}

	return EnvServerTLSConfig()
}

// EnvTLSConfig resolves cert paths from the CERTDIR environment variable,
// falling back to the default SPIFFE directory, and returns a client-side
// tls.Config. Returns ErrNotPresent if no cert files are found.
func EnvTLSConfig() (*tls.Config, string, error) {
	return tlsConfigFromDir(
		func(c Config) (*tls.Config, error) {
			return c.TLSConfig("")
		},
	)
}

// EnvServerTLSConfig resolves cert paths from the CERTDIR environment variable,
// falling back to the default SPIFFE directory, and returns a server-side
// tls.Config. Returns ErrNotPresent if no cert files are found.
func EnvServerTLSConfig() (*tls.Config, string, error) {
	return tlsConfigFromDir(
		func(c Config) (*tls.Config, error) {
			return c.ServerTLSConfig()
		},
	)
}

func tlsConfigFromDir(
	build func(Config) (*tls.Config, error),
) (*tls.Config, string, error) {
	certDir := os.Getenv("CERTDIR")
	if certDir == "" {
		certDir = defaultCertDir
	}

	tlsConfig, err := build(
		Config{
			CACert:  filepath.Join(certDir, defaultCACert),
			TLSCert: filepath.Join(certDir, defaultCertFile),
			TLSKey:  filepath.Join(certDir, defaultKeyFile),
		},
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, certDir, ErrNotPresent
		}
		return nil, certDir, fmt.Errorf("loading certs from %q: %w", certDir, err)
	}

	return tlsConfig, certDir, nil
}

// --- gRPC credential helpers ---

// GRPCClientCredentials returns transport credentials for a gRPC client.
// If certificates are not present (ErrNotPresent), it returns insecure
// credentials so the client can still connect without mTLS.
func GRPCClientCredentials() (credentials.TransportCredentials, error) {
	tlsConfig, _, err := EnvTLSConfig()
	if err != nil {
		if errors.Is(err, ErrNotPresent) {
			return insecure.NewCredentials(), nil
		}
		return nil, fmt.Errorf("resolving client TLS: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// GRPCServerCredentials returns transport credentials for a gRPC server.
// Unlike the client side, the server must have valid certificates; if they
// are not present an error is returned.
func GRPCServerCredentials() (credentials.TransportCredentials, error) {
	tlsConfig, _, err := EnvServerTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("resolving server TLS: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}
