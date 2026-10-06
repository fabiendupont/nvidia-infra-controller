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
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// NewMetricsServer creates a Prometheus registry and starts an HTTP server
// exposing /metrics in a background goroutine. When addr is empty, the value
// is read from METRICS_ADDR, falling back to ":9090".
func NewMetricsServer(addr string) (*prometheus.Registry, error) {
	if addr == "" {
		addr = os.Getenv("METRICS_ADDR")
	}
	if addr == "" {
		addr = ":9090"
	}

	reg := prometheus.NewRegistry()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		log.Info().Str("addr", addr).Msg("metrics server listening")
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("metrics server exited with error")
		}
	}()

	return reg, nil
}
