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
	"encoding/json"
	"fmt"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// JSONResponse builds an HTTPResponse with a JSON-encoded body.
func JSONResponse(statusCode int, body any) (*providerv1.HTTPResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return &providerv1.HTTPResponse{
		StatusCode: int32(statusCode),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: data,
	}, nil
}

// ErrorResponse builds an HTTPResponse with a JSON error message.
func ErrorResponse(statusCode int, message string) (*providerv1.HTTPResponse, error) {
	return JSONResponse(statusCode, map[string]string{
		"error": message,
	})
}
