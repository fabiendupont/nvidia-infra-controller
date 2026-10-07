// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fulfillment

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/NVIDIA/infra-controller/rest-api/provider"
)

// FulfillmentProvider implements the fulfillment feature provider.
type FulfillmentProvider struct {
	orderStore    OrderStoreInterface
	serviceStore  ServiceStoreInterface
	bunDB         *bun.DB
	apiPathPrefix string
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

// Init initializes the compiled-in fulfillment provider. DB is accessed via
// ProviderContext.DB (the API server's bun session).
func (p *FulfillmentProvider) Init(ctx provider.ProviderContext) error {
	p.apiPathPrefix = ctx.APIPathPrefix

	if ctx.DB != nil {
		p.bunDB = ctx.DB.DB
		p.orderStore = NewOrderSQLStore(p.bunDB)
		p.serviceStore = NewServiceSQLStore(p.bunDB)
	} else {
		p.orderStore = NewOrderStore()
		p.serviceStore = NewServiceStore()
	}

	return nil
}

// InitWithBun initializes the provider with a pre-connected bun.DB.
// Used by the standalone gRPC server after ConnectWithSchema.
func (p *FulfillmentProvider) InitWithBun(db *bun.DB) error {
	p.apiPathPrefix = "/api/v1"
	p.bunDB = db
	p.orderStore = NewOrderSQLStore(db)
	p.serviceStore = NewServiceSQLStore(db)
	return nil
}

// InitInMemory initializes the provider with in-memory stores.
func (p *FulfillmentProvider) InitInMemory() error {
	p.apiPathPrefix = "/api/v1"
	p.orderStore = NewOrderStore()
	p.serviceStore = NewServiceStore()
	return nil
}

func (p *FulfillmentProvider) Shutdown(_ context.Context) error {
	return nil
}

// OrderStore returns the order store for cross-provider access.
func (p *FulfillmentProvider) OrderStore() OrderStoreInterface {
	return p.orderStore
}
