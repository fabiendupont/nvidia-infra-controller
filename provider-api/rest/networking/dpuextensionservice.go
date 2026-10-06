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

package networking

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	// DpuExtensionServiceTypeKubernetesPod is the service type for Kubernetes Pod
	DpuExtensionServiceTypeKubernetesPod = "KubernetesPod"
	// DpuExtensionServiceMaxObservabilityConfigs is the max number of observability configs allowed per service version
	DpuExtensionServiceMaxObservabilityConfigs = 20
	// DpuExtensionServiceMaxObservabilityConfigNameLength is the max length for an observability config name
	DpuExtensionServiceMaxObservabilityConfigNameLength = 64
	// DpuExtensionServiceMaxObservabilityPropertyLength is the max length for endpoint and path properties
	DpuExtensionServiceMaxObservabilityPropertyLength = 128
)

var (
	dpuExtensionServiceObservabilityPromEndpointBadRE = regexp.MustCompile(`[^a-zA-Z0-9:\-]+`)
	dpuExtensionServiceObservabilityLogPathBadRE      = regexp.MustCompile(`[^a-zA-Z0-9\-_\/\.\@]+`)
)

// APIDpuExtensionServiceCreateRequest is the data structure to capture user request to create a new DpuExtensionService
type APIDpuExtensionServiceCreateRequest struct {
	// Name is the name of the DpuExtensionService
	Name string `json:"name"`
	// Description is the description of the DpuExtensionService
	Description *string `json:"description"`
	// ServiceType is the type of service
	ServiceType string `json:"serviceType"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Data is the deployment spec for the DPU Extension Service
	Data string `json:"data"`
	// Credentials are the credentials to download resources
	Credentials *APIDpuExtensionServiceCredentials `json:"credentials"`
	// Observability is the observability configuration for the DPU Extension Service version
	Observability *APIDpuExtensionServiceObservability `json:"observability"`
}

// Validate ensures that the values passed in request are acceptable
func (descr APIDpuExtensionServiceCreateRequest) Validate() error {
	err := validation.ValidateStruct(&descr,
		validation.Field(&descr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&descr.ServiceType,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validation.In(DpuExtensionServiceTypeKubernetesPod).Error("must be 'KubernetesPod'")),
		validation.Field(&descr.SiteID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&descr.Data,
			validation.Required.Error(rest.ValidationErrorValueRequired)),
	)
	if err != nil {
		return err
	}

	// Validate credentials if provided
	if descr.Credentials != nil {
		err = descr.Credentials.Validate()
		if err != nil {
			return err
		}
	}

	if descr.Observability != nil {
		err = descr.Observability.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// APIDpuExtensionServiceUpdateRequest is the data structure to capture user request to update a DpuExtensionService
type APIDpuExtensionServiceUpdateRequest struct {
	// Name is the name of the DpuExtensionService
	Name *string `json:"name"`
	// Description is the description of the DpuExtensionService
	Description *string `json:"description"`
	// Data is the deployment spec for the DPU Extension Service
	Data *string `json:"data"`
	// Credentials are the credentials to download resources
	Credentials *APIDpuExtensionServiceCredentials `json:"credentials"`
	// Observability is the observability configuration for the DPU Extension Service version
	Observability *APIDpuExtensionServiceObservability `json:"observability"`
}

// Validate ensures that the values passed in request are acceptable
func (desur APIDpuExtensionServiceUpdateRequest) Validate() error {
	err := validation.ValidateStruct(&desur,
		validation.Field(&desur.Name,
			validation.When(desur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength)),
			validation.When(desur.Name != nil, validation.By(rest.ValidateNameCharacters)),
			validation.When(desur.Name != nil, validation.Length(2, 256).Error(rest.ValidationErrorStringLength))),
	)
	if err != nil {
		return err
	}

	// Validate credentials if provided
	if desur.Credentials != nil {
		err = desur.Credentials.Validate()
		if err != nil {
			return err
		}
	}

	if desur.Observability != nil {
		err = desur.Observability.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// APIDpuExtensionServiceCredentials is the data structure for registry credentials
type APIDpuExtensionServiceCredentials struct {
	// RegistryURL is the URL for the registry
	RegistryURL string `json:"registryUrl"`
	// Username for the registry
	Username *string `json:"username"`
	// Password for the registry
	Password *string `json:"password"`
}

// Validate ensures that the credentials are valid
func (desc APIDpuExtensionServiceCredentials) Validate() error {
	return validation.ValidateStruct(&desc,
		validation.Field(&desc.RegistryURL,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.URL.Error("must be a valid URL")),
		validation.Field(&desc.Username,
			validation.When(desc.RegistryURL != "", validation.Required.Error("`username` must be specified if `registryUrl` is specified"))),
		validation.Field(&desc.Password,
			validation.When(desc.RegistryURL != "", validation.Required.Error("`password` must be specified if `registryUrl` is specified"))),
	)
}

// APIDpuExtensionService is the data structure to capture API representation of a DpuExtensionService
type APIDpuExtensionService struct {
	// ID is the unique UUID v4 identifier for the DpuExtensionService
	ID string `json:"id"`
	// Name is the name of the DpuExtensionService
	Name string `json:"name"`
	// Description is the description of the DpuExtensionService
	Description *string `json:"description"`
	// ServiceType is the type of service
	ServiceType string `json:"serviceType"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Version is the latest version of the DPU Extension Service
	Version *string `json:"version"`
	// VersionInfo holds the details for the latest version
	VersionInfo *APIDpuExtensionServiceVersionInfo `json:"versionInfo"`
	// ActiveVersions is a list of active versions available for deployment
	ActiveVersions []string `json:"activeVersions"`
	// Status is the status of the DpuExtensionService
	Status string `json:"status"`
	// StatusHistory is the status detail records for the DpuExtensionService over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// Created indicates the ISO datetime string for when the DpuExtensionService was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the DpuExtensionService was last updated
	Updated time.Time `json:"updated"`
}

// APIDpuExtensionServiceSummary is the data structure to capture API summary of a DpuExtensionService
type APIDpuExtensionServiceSummary struct {
	// ID is the unique UUID v4 identifier for the DpuExtensionService
	ID string `json:"id"`
	// Name is the name of the DpuExtensionService
	Name string `json:"name"`
	// ServiceType is the type of service
	ServiceType string `json:"serviceType"`
	// LatestVersion is the latest version of the DPU Extension Service
	LatestVersion *string `json:"latestVersion"`
	// Status is the status of the DpuExtensionService
	Status string `json:"status"`
}

// APIDpuExtensionServiceVersionInfo is the data structure for version information
type APIDpuExtensionServiceVersionInfo struct {
	// Version is the version identifier
	Version string `json:"version"`
	// Data is the deployment spec
	Data string `json:"data"`
	// HasCredentials indicates if this version has credentials
	HasCredentials bool `json:"hasCredentials"`
	// Created indicates when this version was created
	Created time.Time `json:"created"`
	// Observability is the observability configuration for this version
	Observability *APIDpuExtensionServiceObservability `json:"observability"`
}

// APIDpuExtensionServiceObservability is the data structure for DPU Extension Service observability
type APIDpuExtensionServiceObservability struct {
	// Configs are the observability configurations for the service version
	Configs []APIDpuExtensionServiceObservabilityConfig `json:"configs"`
}

// Validate ensures that the observability configuration is valid
func (deso APIDpuExtensionServiceObservability) Validate() error {
	err := validation.ValidateStruct(&deso,
		validation.Field(&deso.Configs,
			validation.By(func(value any) error {
				configs, ok := value.([]APIDpuExtensionServiceObservabilityConfig)
				if !ok {
					return fmt.Errorf("must be a valid list of observability configs")
				}
				if len(configs) > DpuExtensionServiceMaxObservabilityConfigs {
					return fmt.Errorf("must not contain more than %d observability configs", DpuExtensionServiceMaxObservabilityConfigs)
				}
				return nil
			})),
	)
	if err != nil {
		return err
	}

	for idx, cfg := range deso.Configs {
		if err = cfg.Validate(); err != nil {
			return fmt.Errorf("configs[%d]: %w", idx, err)
		}
	}

	return nil
}

// APIDpuExtensionServiceObservabilityConfig is the data structure for a single DPU Extension Service observability config
type APIDpuExtensionServiceObservabilityConfig struct {
	// Name is the name of the service or service component being monitored
	Name *string `json:"name"`
	// Prometheus holds prometheus scrape configuration
	Prometheus *APIDpuExtensionServiceObservabilityConfigPrometheus `json:"prometheus,omitempty"`
	// Logging holds logging configuration
	Logging *APIDpuExtensionServiceObservabilityConfigLogging `json:"logging,omitempty"`
}

// Validate ensures that the observability config is valid
func (desoc APIDpuExtensionServiceObservabilityConfig) Validate() error {
	if desoc.Name != nil {
		if strings.TrimSpace(*desoc.Name) == "" {
			return fmt.Errorf("name must be non-empty")
		}
		if len(*desoc.Name) > DpuExtensionServiceMaxObservabilityConfigNameLength {
			return fmt.Errorf("name length must not exceed %d", DpuExtensionServiceMaxObservabilityConfigNameLength)
		}
	}

	configCount := 0
	if desoc.Prometheus != nil {
		configCount++
	}
	if desoc.Logging != nil {
		configCount++
	}
	if configCount != 1 {
		return fmt.Errorf("exactly one of `prometheus` or `logging` must be specified")
	}

	switch {
	case desoc.Prometheus != nil:
		if err := desoc.Prometheus.Validate(); err != nil {
			return err
		}
	case desoc.Logging != nil:
		if err := desoc.Logging.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// APIDpuExtensionServiceObservabilityConfigPrometheus is the data structure for prometheus observability config
type APIDpuExtensionServiceObservabilityConfigPrometheus struct {
	// ScrapeIntervalSeconds is how often prometheus should scrape the endpoint
	ScrapeIntervalSeconds uint32 `json:"scrapeIntervalSeconds"`
	// Endpoint is the prometheus scrape endpoint
	Endpoint string `json:"endpoint"`
}

// Validate ensures that the prometheus observability config is valid
func (desop APIDpuExtensionServiceObservabilityConfigPrometheus) Validate() error {
	err := validation.ValidateStruct(&desop,
		validation.Field(&desop.ScrapeIntervalSeconds, validation.Min(uint32(1)).Error("must be greater than 0")),
		validation.Field(&desop.Endpoint, validation.Required.Error(rest.ValidationErrorValueRequired)),
		validation.Field(&desop.Endpoint, validation.Length(0, DpuExtensionServiceMaxObservabilityPropertyLength).Error(fmt.Sprintf("length must not exceed %d", DpuExtensionServiceMaxObservabilityPropertyLength))),
	)
	if err != nil {
		return err
	}

	if dpuExtensionServiceObservabilityPromEndpointBadRE.MatchString(desop.Endpoint) {
		return fmt.Errorf("endpoint contains invalid characters")
	}

	return nil
}

// APIDpuExtensionServiceObservabilityConfigLogging is the data structure for logging observability config
type APIDpuExtensionServiceObservabilityConfigLogging struct {
	// Path is the log path to collect
	Path string `json:"path"`
}

// Validate ensures that the logging observability config is valid
func (desol APIDpuExtensionServiceObservabilityConfigLogging) Validate() error {
	err := validation.ValidateStruct(&desol,
		validation.Field(&desol.Path, validation.Required.Error(rest.ValidationErrorValueRequired)),
		validation.Field(&desol.Path, validation.Length(0, DpuExtensionServiceMaxObservabilityPropertyLength).Error(fmt.Sprintf("length must not exceed %d", DpuExtensionServiceMaxObservabilityPropertyLength))),
	)
	if err != nil {
		return err
	}

	if dpuExtensionServiceObservabilityLogPathBadRE.MatchString(desol.Path) {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}
