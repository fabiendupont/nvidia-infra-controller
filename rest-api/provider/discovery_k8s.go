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
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	providerLabel        = "nico.nvidia.com/provider"
	providerAddrAnn      = "nico.nvidia.com/provider-address"
	informerResyncPeriod = 5 * time.Minute
)

// KubernetesDiscovery watches ConfigMaps in a namespace for provider
// advertisements and dynamically registers/deregisters external providers.
type KubernetesDiscovery struct {
	mu        sync.Mutex
	clientset kubernetes.Interface
	registry  *Registry
	namespace string
	providers map[string]*ExternalProvider
	cancel    context.CancelFunc
	initCtx   ProviderContext
}

// NewKubernetesDiscovery creates a discovery instance using in-cluster
// credentials.
func NewKubernetesDiscovery(registry *Registry, namespace string, initCtx ProviderContext) (*KubernetesDiscovery, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("building in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating k8s clientset: %w", err)
	}
	return &KubernetesDiscovery{
		clientset: clientset,
		registry:  registry,
		namespace: namespace,
		providers: make(map[string]*ExternalProvider),
		initCtx:   initCtx,
	}, nil
}

// NewKubernetesDiscoveryFromKubeconfig creates a discovery instance using an
// explicit kubeconfig file path.
func NewKubernetesDiscoveryFromKubeconfig(registry *Registry, namespace, kubeconfigPath string, initCtx ProviderContext) (*KubernetesDiscovery, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig from %s: %w", kubeconfigPath, err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating k8s clientset: %w", err)
	}
	return &KubernetesDiscovery{
		clientset: clientset,
		registry:  registry,
		namespace: namespace,
		providers: make(map[string]*ExternalProvider),
		initCtx:   initCtx,
	}, nil
}

// Start begins watching the namespace for provider ConfigMaps.
func (kd *KubernetesDiscovery) Start(ctx context.Context) {
	ctx, kd.cancel = context.WithCancel(ctx)

	factory := informers.NewSharedInformerFactoryWithOptions(
		kd.clientset,
		informerResyncPeriod,
		informers.WithNamespace(kd.namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = providerLabel
		}),
	)

	informer := factory.Core().V1().ConfigMaps().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    kd.onAdd,
		UpdateFunc: kd.onUpdate,
		DeleteFunc: kd.onDelete,
	})

	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())
	log.Info().Str("namespace", kd.namespace).Msg("kubernetes provider discovery started")
}

// Stop cancels the informer watch loop.
func (kd *KubernetesDiscovery) Stop() {
	if kd.cancel != nil {
		kd.cancel()
	}
}

func (kd *KubernetesDiscovery) onAdd(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return
	}
	kd.connectProvider(cm)
}

func (kd *KubernetesDiscovery) onUpdate(oldObj, newObj interface{}) {
	cm, ok := newObj.(*corev1.ConfigMap)
	if !ok {
		return
	}
	kd.mu.Lock()
	_, existed := kd.providers[cm.Name]
	kd.mu.Unlock()
	if existed {
		kd.disconnectProvider(cm.Name)
	}
	kd.connectProvider(cm)
}

func (kd *KubernetesDiscovery) onDelete(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		cm, ok = tombstone.Obj.(*corev1.ConfigMap)
		if !ok {
			return
		}
	}
	kd.disconnectProvider(cm.Name)
}

func (kd *KubernetesDiscovery) connectProvider(cm *corev1.ConfigMap) {
	address := cm.Annotations[providerAddrAnn]
	if address == "" {
		log.Warn().Str("configmap", cm.Name).Msg("provider configmap missing address annotation")
		return
	}

	ep, err := ConnectExternalProviderTCP(address)
	if err != nil {
		log.Error().Err(err).Str("configmap", cm.Name).Str("address", address).Msg("failed to connect to provider")
		return
	}

	if err := kd.registry.Register(ep); err != nil {
		log.Error().Err(err).Str("provider", ep.Name()).Msg("failed to register discovered provider")
		ep.conn.Close()
		return
	}

	if err := ep.Init(kd.initCtx); err != nil {
		log.Error().Err(err).Str("provider", ep.Name()).Msg("failed to initialize discovered provider")
		kd.registry.Unregister(ep.Name())
		ep.conn.Close()
		return
	}

	kd.mu.Lock()
	kd.providers[cm.Name] = ep
	kd.mu.Unlock()

	log.Info().Str("provider", ep.Name()).Str("address", address).Msg("discovered and registered provider")
}

func (kd *KubernetesDiscovery) disconnectProvider(cmName string) {
	kd.mu.Lock()
	ep, ok := kd.providers[cmName]
	if !ok {
		kd.mu.Unlock()
		return
	}
	delete(kd.providers, cmName)
	kd.mu.Unlock()

	kd.registry.Unregister(ep.Name())
	ep.Shutdown(context.Background())
	log.Info().Str("provider", ep.Name()).Msg("provider deregistered (configmap deleted)")
}
