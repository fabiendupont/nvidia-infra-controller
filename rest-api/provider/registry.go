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

package provider

import (
	"context"
	"fmt"
	"sync"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// Registry manages provider lifecycle: registration, dependency resolution,
// initialization, and shutdown.
type Registry struct {
	mu           sync.RWMutex
	providers    map[string]Provider
	features     map[string]string // feature name → provider name
	order        []string          // init order after dependency resolution
	hooks        *hookRegistry
	providerGroup *echo.Group     // org-scoped route group; set by InitAPIServer
}

// SetProviderGroup stores the org-scoped Echo group used to mount provider
// routes. Called once by InitAPIServer before starting KubernetesDiscovery.
func (r *Registry) SetProviderGroup(g *echo.Group) {
	r.mu.Lock()
	r.providerGroup = g
	r.mu.Unlock()
}

// ProviderGroup returns the org-scoped Echo route group, or nil if not set.
func (r *Registry) ProviderGroup() *echo.Group {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providerGroup
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		features:  make(map[string]string),
		hooks:     newHookRegistry(),
	}
}

// Register adds a provider to the registry and maps its features.
func (r *Registry) Register(p Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.register(p)
}

func (r *Registry) register(p Provider) error {
	name := p.Name()
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}
	r.providers[name] = p
	for _, f := range p.Features() {
		if existing, ok := r.features[f]; ok {
			return fmt.Errorf("feature %q already provided by %q, cannot register %q", f, existing, name)
		}
		r.features[f] = name
	}
	return nil
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// FeatureProvider returns the provider registered for a given feature.
func (r *Registry) FeatureProvider(feature string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.features[feature]
	if !ok {
		return nil, false
	}
	return r.providers[name], true
}

// ResolveDependencies performs a topological sort of providers based on their
// declared dependencies. Returns an error on circular or missing dependencies.
func (r *Registry) ResolveDependencies() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	visited := make(map[string]int) // 0=unvisited, 1=in-progress, 2=done
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		switch visited[name] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("circular dependency detected involving %q", name)
		}
		visited[name] = 1

		p, ok := r.providers[name]
		if !ok {
			return fmt.Errorf("missing dependency: provider %q not registered", name)
		}

		for _, dep := range p.Dependencies() {
			if _, ok := r.providers[dep]; !ok {
				return fmt.Errorf("provider %q depends on %q, which is not registered", name, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		visited[name] = 2
		order = append(order, name)
		return nil
	}

	for name := range r.providers {
		if err := visit(name); err != nil {
			return err
		}
	}

	r.order = order
	return nil
}

// InitAll initializes all providers in dependency order.
func (r *Registry) InitAll(ctx ProviderContext) error {
	r.mu.RLock()
	if r.order == nil {
		r.mu.RUnlock()
		if err := r.ResolveDependencies(); err != nil {
			return err
		}
		r.mu.RLock()
	}
	defer r.mu.RUnlock()
	for _, name := range r.order {
		if err := r.providers[name].Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", name, err)
		}
	}
	return nil
}

// ShutdownAll shuts down all providers in reverse dependency order.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		if err := r.providers[r.order[i]].Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown provider %q: %w", r.order[i], err)
		}
	}
	return nil
}

// APIProviders returns all registered providers that implement APIProvider.
func (r *Registry) APIProviders() []APIProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []APIProvider
	for _, name := range r.order {
		if ap, ok := r.providers[name].(APIProvider); ok {
			result = append(result, ap)
		}
	}
	return result
}

// WorkflowProviders returns all registered providers that implement WorkflowProvider.
func (r *Registry) WorkflowProviders() []WorkflowProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []WorkflowProvider
	for _, name := range r.order {
		if wp, ok := r.providers[name].(WorkflowProvider); ok {
			result = append(result, wp)
		}
	}
	return result
}

// Unregister removes a provider from the registry, cleaning up its features
// and hooks. The provider's Shutdown method is NOT called — the caller is
// responsible for that.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("provider %q not registered", name)
	}

	for _, f := range p.Features() {
		delete(r.features, f)
	}

	delete(r.providers, name)

	newOrder := make([]string, 0, len(r.order))
	for _, n := range r.order {
		if n != name {
			newOrder = append(newOrder, n)
		}
	}
	r.order = newOrder

	if ep, ok := p.(*ExternalProvider); ok {
		r.unregisterProviderHooks(ep)
	}

	log.Info().Str("provider", name).Msg("provider unregistered")
	return nil
}
