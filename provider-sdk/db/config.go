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

package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// SecretString wraps sensitive string data and prevents accidental exposure
// in logs/JSON.
type SecretString struct {
	Value string `json:"-"` // Never serialize the actual value
}

// NewSecretString creates a new SecretString with the given string.
func NewSecretString(v string) SecretString {
	return SecretString{Value: v}
}

// String implements fmt.Stringer to hide the actual value in string
// representations.
func (s SecretString) String() string {
	return "******"
}

// MarshalJSON implements json.Marshaler to hide the value during JSON
// serialization.
func (s SecretString) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// IsEmpty returns true if the secret string has no value.
func (s SecretString) IsEmpty() bool {
	return strings.TrimSpace(s.Value) == ""
}

// IsEqual returns true if the given secret string is the same as this one.
func (s SecretString) IsEqual(n SecretString) bool {
	return s.Value == n.Value
}

// Credential holds authentication information with password protection.
type Credential struct {
	User     string       `json:"user"`     // User name
	Password SecretString `json:"password"` // Password (masked in JSON/logs)
}

// NewCredential creates a Credential with the given user and password.
func NewCredential(user string, password string) Credential {
	return Credential{
		User:     user,
		Password: NewSecretString(password),
	}
}

// NewCredentialFromEnv creates a Credential from environment variables.
func NewCredentialFromEnv(userEnv string, passwordEnv string) Credential {
	return Credential{
		User:     os.Getenv(userEnv),
		Password: NewSecretString(os.Getenv(passwordEnv)),
	}
}

// IsValid returns true if the credential has a non-empty username.
func (cred *Credential) IsValid() bool {
	return strings.TrimSpace(cred.User) != ""
}

// Config represents the configuration needed to connect to a database.
type Config struct {
	Host              string
	Port              int
	DBName            string
	Credential        Credential
	CACertificatePath string
}

// Validate checks if the Config fields are set correctly.
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port must be between (0, 65535]")
	}

	if c.DBName == "" {
		return errors.New("database name is required")
	}

	if !c.Credential.IsValid() {
		return errors.New("valid credential is required")
	}

	return nil
}

// ConfigFromEnv builds a Config from environment variables.
// Reads: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME,
// DB_CERT_PATH (optional CA certificate).
func ConfigFromEnv() (Config, error) {
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return Config{}, ErrInvalidPort
	}

	cred := NewCredentialFromEnv("DB_USER", "DB_PASSWORD")
	if !cred.IsValid() {
		return Config{}, ErrInvalidCredential
	}

	return Config{
		Host:              os.Getenv("DB_HOST"),
		Port:              port,
		Credential:        cred,
		DBName:            os.Getenv("DB_NAME"),
		CACertificatePath: os.Getenv("DB_CERT_PATH"),
	}, nil
}

// BuildDSN builds the Data Source Name (DSN) string for connecting to
// the database.
func (c *Config) BuildDSN() string {
	dsn := fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v?sslmode=",
		url.PathEscape(c.Credential.User),
		url.PathEscape(c.Credential.Password.Value),
		c.Host,
		c.Port,
		c.DBName,
	)

	if len(c.CACertificatePath) > 0 {
		dsn += fmt.Sprintf("prefer&sslrootcert=%v", c.CACertificatePath)
	} else {
		dsn += "prefer"
	}

	return dsn
}
