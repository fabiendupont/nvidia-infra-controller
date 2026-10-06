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
	"fmt"
	"net/http"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// RouteHandler handles an HTTP request forwarded by the NICo core.
type RouteHandler func(req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error)

// Router dispatches incoming HTTP requests to registered handlers based on
// method and path.
type Router struct {
	routes map[string]RouteHandler
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]RouteHandler),
	}
}

// Handle registers a handler for a method+path combination.
func (r *Router) Handle(method, path string, handler RouteHandler) {
	r.routes[method+" "+path] = handler
}

// Dispatch finds and invokes the handler for the given request.
func (r *Router) Dispatch(req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	key := req.GetMethod() + " " + req.GetPath()
	handler, ok := r.routes[key]
	if !ok {
		return ErrorResponse(http.StatusNotFound, fmt.Sprintf("no handler for %s", key))
	}
	return handler(req)
}
