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

package fulfillment

import (
	"context"
	"fmt"

	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	"github.com/NVIDIA/infra-controller/rest-api/provider"
	"github.com/NVIDIA/infra-controller/providers/catalog/internal"
	"github.com/NVIDIA/infra-controller/rest-api/providers/compute"
	"github.com/NVIDIA/infra-controller/rest-api/providers/networking"
)

// FulfillmentProvider implements the fulfillment feature provider.
type FulfillmentProvider struct {
	orderStore    OrderStoreInterface
	serviceStore  ServiceStoreInterface
	dbSession     *cdb.Session
	apiPathPrefix string
	networking    *networking.NetworkingProvider
	compute       *compute.ComputeProvider
	catalog       *catalog.CatalogProvider
}

// New creates a new FulfillmentProvider.
func New() *FulfillmentProvider {
	return &FulfillmentProvider{}
}

func (p *FulfillmentProvider) Name() string       { return "nico-fulfillment" }
func (p *FulfillmentProvider) Version() string    { return "0.1.0" }
func (p *FulfillmentProvider) Features() []string { return []string{"fulfillment"} }
func (p *FulfillmentProvider) Dependencies() []string {
	return []string{"nico-networking", "nico-compute", "nico-catalog"}
}

func (p *FulfillmentProvider) Init(ctx provider.ProviderContext) error {
	p.apiPathPrefix = ctx.APIPathPrefix

	// Use PostgreSQL if DB is available, else in-memory
	if ctx.DB != nil {
		p.dbSession = ctx.DB
		p.orderStore = NewOrderSQLStore(ctx.DB)
		p.serviceStore = NewServiceSQLStore(ctx.DB)
	} else {
		p.orderStore = NewOrderStore()
		p.serviceStore = NewServiceStore()
	}

	if ctx.Registry != nil {
		if np, ok := ctx.Registry.Get("nico-networking"); ok {
			netProvider, ok := np.(*networking.NetworkingProvider)
			if !ok {
				return fmt.Errorf("nico-networking provider has unexpected type")
			}
			p.networking = netProvider
		}

		if cp, ok := ctx.Registry.Get("nico-compute"); ok {
			compProvider, ok := cp.(*compute.ComputeProvider)
			if !ok {
				return fmt.Errorf("nico-compute provider has unexpected type")
			}
			p.compute = compProvider
		}

		if cat, ok := ctx.Registry.Get("nico-catalog"); ok {
			catProvider, ok := cat.(*catalog.CatalogProvider)
			if !ok {
				return fmt.Errorf("nico-catalog provider has unexpected type")
			}
			p.catalog = catProvider
		}
	}

	return nil
}

// InitStandalone initializes the provider for use outside the core process.
// It creates stores without a Registry (no cross-provider references in
// standalone mode; networking/compute/catalog remain nil).
func (p *FulfillmentProvider) InitStandalone(dbSession *cdb.Session) error {
	p.apiPathPrefix = "/api/v1"

	if dbSession != nil {
		p.dbSession = dbSession
		p.orderStore = NewOrderSQLStore(dbSession)
		p.serviceStore = NewServiceSQLStore(dbSession)
	} else {
		p.orderStore = NewOrderStore()
		p.serviceStore = NewServiceStore()
	}

	return nil
}

func (p *FulfillmentProvider) Shutdown(_ context.Context) error {
	return nil
}

// OrderStore returns the order store for cross-provider access.
func (p *FulfillmentProvider) OrderStore() OrderStoreInterface {
	return p.orderStore
}
