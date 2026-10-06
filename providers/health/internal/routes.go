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

package health

import (
	"net/http"

	echo "github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registerRoutes registers all HTTP routes on the given Echo group.
func (s *Server) registerRoutes(group *echo.Group) {
	s.registerFaultRoutes(group)
	s.registerServiceEventRoutes(group)
	s.registerClassificationRoutes(group)
	s.registerWebhookRoutes(group)
	s.registerMetricsRoute(group)
}

// registerFaultRoutes adds operator-facing fault management endpoints.
func (s *Server) registerFaultRoutes(group *echo.Group) {
	if s.faultHandler == nil {
		return
	}

	prefix := s.apiPathPrefix + "/health/events"

	group.Add(http.MethodPost, prefix+"/ingest", s.faultHandler.handleIngestFault)
	group.Add(http.MethodGet, prefix+"/summary", s.faultHandler.handleGetFaultSummary)
	group.Add(http.MethodGet, prefix, s.faultHandler.handleListFaultEvents)
	group.Add(http.MethodGet, prefix+"/:id", s.faultHandler.handleGetFaultEvent)
	group.Add(http.MethodPatch, prefix+"/:id", s.faultHandler.handleUpdateFaultEvent)
	group.Add(http.MethodPost, prefix+"/:id/remediate", s.faultHandler.handleTriggerRemediation)
}

// registerServiceEventRoutes adds tenant-facing service event endpoints.
func (s *Server) registerServiceEventRoutes(group *echo.Group) {
	if s.serviceEventHandler == nil {
		return
	}

	prefix := s.apiPathPrefix + "/tenant/:tenantId/service-events"

	group.Add(http.MethodGet, prefix, s.serviceEventHandler.handleListServiceEvents)
	group.Add(http.MethodGet, prefix+"/active", s.serviceEventHandler.handleGetActiveServiceEvents)
	group.Add(http.MethodGet, prefix+"/:id", s.serviceEventHandler.handleGetServiceEvent)
}

// registerClassificationRoutes adds operator-facing classification management endpoints.
func (s *Server) registerClassificationRoutes(group *echo.Group) {
	if s.classificationHandler == nil {
		return
	}

	prefix := s.apiPathPrefix + "/health/classifications"

	group.Add(http.MethodGet, prefix, s.classificationHandler.handleListClassifications)
	group.Add(http.MethodPut, prefix+"/:classification", s.classificationHandler.handleUpdateClassification)
}

// registerWebhookRoutes adds webhook ingestion endpoints for external alert sources.
func (s *Server) registerWebhookRoutes(group *echo.Group) {
	if s.webhookHandler == nil {
		return
	}

	prefix := s.apiPathPrefix + "/health/webhooks"

	group.Add(http.MethodPost, prefix+"/alertmanager", s.webhookHandler.handleAlertManagerWebhook)
}

// registerMetricsRoute exposes fault event Prometheus metrics.
// The metrics endpoint is separate from NICo's global /metrics so that
// fault-specific metrics can be scraped independently.
func (s *Server) registerMetricsRoute(group *echo.Group) {
	if s.metrics == nil {
		return
	}

	// Refresh gauges before each scrape
	handler := promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{})

	group.Add(http.MethodGet, s.apiPathPrefix+"/health/metrics", func(c echo.Context) error {
		s.metrics.RefreshOpenFaults()
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
