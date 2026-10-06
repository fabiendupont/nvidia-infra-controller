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

package sdk

import (
	"crypto/tls"
	"crypto/x509"
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

func certDir() string {
	if dir := os.Getenv("CERTDIR"); dir != "" {
		return dir
	}
	return defaultCertDir
}

func loadCerts(dir string) (*x509.CertPool, tls.Certificate, error) {
	caCert, err := os.ReadFile(filepath.Join(dir, defaultCACert))
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, tls.Certificate{}, fmt.Errorf("parse CA cert from %s", dir)
	}

	cert, err := tls.LoadX509KeyPair(
		filepath.Join(dir, defaultCertFile),
		filepath.Join(dir, defaultKeyFile),
	)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("load cert/key: %w", err)
	}

	return pool, cert, nil
}

// LoadTLSCredentials returns client-side gRPC transport credentials.
// Falls back to insecure credentials when certs are not present.
func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	dir := certDir()

	pool, cert, err := loadCerts(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return insecure.NewCredentials(), nil
		}
		return nil, fmt.Errorf("loading client TLS from %s: %w", dir, err)
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
		RootCAs: pool,
	}

	return credentials.NewTLS(tlsCfg), nil
}

// LoadServerTLSCredentials returns server-side gRPC transport credentials
// with mutual TLS. Falls back to insecure credentials when certs are not present.
func LoadServerTLSCredentials() (credentials.TransportCredentials, error) {
	dir := certDir()

	pool, cert, err := loadCerts(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return insecure.NewCredentials(), nil
		}
		return nil, fmt.Errorf("loading server TLS from %s: %w", dir, err)
	}

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}

	return credentials.NewTLS(tlsCfg), nil
}
