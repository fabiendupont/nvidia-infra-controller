// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/temporal"

	cClient "github.com/NVIDIA/infra-controller/rest-api/site-workflow/pkg/grpc/client"

	mp "github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner"
	corev1 "github.com/NVIDIA/infra-controller/rest-api/proto/core/gen/v1"

	swe "github.com/NVIDIA/infra-controller/rest-api/site-workflow/pkg/error"
)

// provisionHookPayload is the payload emitted on pre/post-machine-provision hooks.
type provisionHookPayload struct {
	MachineID   string `json:"machine_id"`
	Provisioner string `json:"provisioner"`
	// Hostname is set from InstanceAllocationRequest.Config.Tenant.Hostname when present.
	Hostname string `json:"hostname,omitempty"`
	// OSImageID is set from InstanceAllocationRequest.Config.Os.OsImageId when present.
	// The Metal3 provider uses this to resolve the image URL from NICo's OS registry.
	OSImageID string `json:"os_image_id,omitempty"`
	// BMCAddress, BMCUsername, BMCPassword are NOT populated here — the Metal3
	// provider reads them from its own operator-supplied configuration (Kubernetes
	// Secrets / environment variables) since BMC credentials are managed via Vault,
	// not stored in NICo's DB.
	OSType string `json:"os_type,omitempty"` // "nico-managed" | "user-provisioned"
}

// deprovisionHookPayload is the payload emitted on pre/post-machine-deprovision hooks.
type deprovisionHookPayload struct {
	MachineID   string `json:"machine_id"`
	Provisioner string `json:"provisioner"`
}

// ManageInstance is an activity wrapper for Instance management tasks that allows injecting DB access
type ManageInstance struct {
	coreGrpcAtomicClient *cClient.CoreGrpcAtomicClient
	// provisioner is the pluggable bare-metal provisioning backend. When non-nil
	// and not CoreGRPCProvisioner, CreateInstanceOnSite and DeleteInstanceOnSite
	// delegate to it instead of calling Core gRPC. Nil means Core gRPC (legacy path).
	// Deprecated: prefer registry for multi-backend support.
	provisioner mp.MachineProvisioner
	// registry maps per-machine annotation values to MachineProvisioner backends.
	// When set, it takes precedence over provisioner for backend selection.
	// The annotation key is mp.AnnotationKey ("nico.nvidia.com/provisioner").
	registry *mp.Registry
	// hooks fires lifecycle hooks to registered providers. Nil when no hook
	// runner is available (e.g. in the legacy workflow binary path).
	hooks mp.HookFirer
}

// Function Update NICo Instance with the Site Controller
func (mm *ManageInstance) UpdateInstanceOnSite(ctx context.Context, request *corev1.InstanceConfigUpdateRequest) error {
	logger := log.With().Str("Activity", "UpdateInstanceOnSite").Logger()

	logger.Info().Msg("Starting activity")

	var err error

	// Validate request
	if request == nil {
		err = errors.New("received empty Instance config update request")
	} else if request.InstanceId == nil {
		err = errors.New("received Instance config update request without Instance ID")
	}

	if err != nil {
		return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
	}

	// Call Core gRPC API endpoint
	grpcClient := mm.coreGrpcAtomicClient.GetClient()
	if grpcClient == nil {
		return cClient.ErrCoreGrpcClientNotConnected
	}
	grpcServiceClient := grpcClient.GrpcServiceClient()

	start := time.Now()
	_, err = grpcServiceClient.UpdateInstanceConfig(ctx, request)
	duration := time.Since(start)
	if err != nil {
		logger.Warn().Err(err).Dur("grpc_duration", duration).Msg("Failed to update config for Instance using Core gRPC API")
		return swe.WrapErr(err)
	}
	logger.Info().Dur("grpc_duration", duration).Msg("Completed activity")

	return nil
}

// Function to Create (allocate) NICo Instance with the Site Controller
func (mm *ManageInstance) CreateInstanceOnSite(ctx context.Context, request *corev1.InstanceAllocationRequest) error {
	logger := log.With().Str("Activity", "CreateInstanceOnSite").Logger()

	logger.Info().Msg("Starting activity")

	var err error

	// Validate request
	if request == nil {
		err = errors.New("received empty create Instance request")
	} else if request.MachineId == nil {
		err = errors.New("received create Instance request without Machine ID")
	}

	if err != nil {
		return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
	}

	// Select the provisioner backend for this machine. When a registry is set,
	// use the per-machine annotation to choose the backend; fall back to the
	// single-provisioner field for backward compatibility with callers that use
	// NewManageInstanceWithProvisioner directly.
	var selectedProvisioner mp.MachineProvisioner
	if mm.registry != nil {
		// Annotation-based selection: read nico.nvidia.com/provisioner from the
		// allocation request's Metadata.Labels. The API handler copies the
		// machine's nico.nvidia.com/provisioner label into the request metadata
		// when dispatching; if absent, CoreGRPCProvisioner is used unchanged.
		annotationValue := ""
		for _, label := range request.GetMetadata().GetLabels() {
			if label.GetKey() == mp.AnnotationKey {
				annotationValue = label.GetValue()
				break
			}
		}
		selectedProvisioner = mm.registry.Select(annotationValue)
	} else {
		selectedProvisioner = mm.provisioner
	}

	// Delegate to the pluggable provisioner when one is registered and it is
	// not the Core gRPC sentinel (which means "use the existing path below").
	if selectedProvisioner != nil && selectedProvisioner.Name() != mp.BackendCoreGRPC {
		provisionerName := selectedProvisioner.Name()
		machineID := request.GetMachineId().GetId()

		// Build the provision request with everything available from the allocation request.
		// BMC URL and credentials are intentionally omitted: they are managed via Vault and
		// supplied to the external provisioner through operator-configured Kubernetes Secrets
		// or environment variables, not through NICo's DB or proto.
		provReq := mp.ProvisionRequest{
			MachineID: machineID,
			Hostname:  request.GetConfig().GetTenant().GetHostname(),
			Extra:     map[string]string{},
		}
		if osID := request.GetConfig().GetOs().GetOsImageId(); osID != nil {
			provReq.Extra["os_image_id"] = osID.GetValue()
		}

		// Fire pre-machine-provision sync hook. If a registered provider (e.g. Metal3)
		// returns an error, provisioning is aborted — the hook acts as a gate.
		if mm.hooks != nil {
			hookPayload := provisionHookPayload{
				MachineID:   machineID,
				Provisioner: provisionerName,
				Hostname:    provReq.Hostname,
				OSImageID:   provReq.Extra["os_image_id"],
				OSType:      "nico-managed",
			}
			if err := mm.hooks.FireSync(ctx, "compute", "pre-machine-provision", hookPayload); err != nil {
				logger.Warn().Err(err).Str("provisioner", provisionerName).Msg("pre-machine-provision hook rejected provisioning")
				return swe.WrapErr(err)
			}
		}

		logger.Info().Str("provisioner", provisionerName).Msg("delegating to external provisioner")
		if err := selectedProvisioner.Provision(ctx, provReq); err != nil {
			logger.Warn().Err(err).Str("provisioner", provisionerName).Msg("external provisioner failed")
			return swe.WrapErr(err)
		}

		// Fire post-machine-provision async hook (non-blocking).
		if mm.hooks != nil {
			mm.hooks.FireAsync(ctx, "compute", "post-machine-provision", provisionHookPayload{
				MachineID:   machineID,
				Provisioner: provisionerName,
				OSType:      "nico-managed",
			})
		}
		return nil
	}

	// Call Core gRPC API endpoint (default path)
	grpcClient := mm.coreGrpcAtomicClient.GetClient()
	if grpcClient == nil {
		return cClient.ErrCoreGrpcClientNotConnected
	}
	grpcServiceClient := grpcClient.GrpcServiceClient()

	start := time.Now()
	_, err = grpcServiceClient.AllocateInstance(ctx, request)
	duration := time.Since(start)
	if err != nil {
		logger.Warn().Err(err).Dur("grpc_duration", duration).Msg("Failed to create Instance using Core gRPC API")
		return swe.WrapErr(err)
	}
	logger.Info().Dur("grpc_duration", duration).Msg("Completed activity")

	return nil
}

// CreateInstancesOnSite is an activity to create (allocate) multiple NICo Instances with the Site Controller
// in a single transaction. This is the batch version of CreateInstanceOnSite.
func (mm *ManageInstance) CreateInstancesOnSite(ctx context.Context, request *corev1.BatchInstanceAllocationRequest) error {
	logger := log.With().Str("Activity", "CreateInstancesOnSite").Logger()

	var err error
	if request == nil {
		err = errors.New("received empty batch create Instance request")
	} else if len(request.InstanceRequests) == 0 {
		err = errors.New("received batch create Instance request with no instances")
	}
	if err != nil {
		return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
	}

	logger = log.With().Str("Activity", "CreateInstancesOnSite").Int("Count", len(request.InstanceRequests)).Logger()
	logger.Info().Msg("Starting batch instance allocation activity")

	for i, req := range request.InstanceRequests {
		if req.MachineId == nil {
			err = errors.New("received create Instance request without Machine ID at index " + string(rune(i)))
			return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
		}
	}

	grpcClient := mm.coreGrpcAtomicClient.GetClient()
	if grpcClient == nil {
		return cClient.ErrCoreGrpcClientNotConnected
	}
	grpcServiceClient := grpcClient.GrpcServiceClient()

	start := time.Now()
	_, err = grpcServiceClient.AllocateInstances(ctx, request)
	duration := time.Since(start)
	if err != nil {
		logger.Warn().Err(err).Dur("grpc_duration", duration).Msg("Failed to batch create Instances using Core gRPC API")
		return swe.WrapErr(err)
	}

	logger.Info().Int("Count", len(request.InstanceRequests)).Dur("grpc_duration", duration).Msg("Completed batch instance allocation activity")
	return nil
}

// Function to Create (allocate) NICo Instance with the Site Controller
func (mm *ManageInstance) RebootInstanceOnSite(ctx context.Context, request *corev1.InstancePowerRequest) error {
	logger := log.With().Str("Activity", "RebootInstanceOnSite").Logger()

	logger.Info().Msg("Starting activity")

	var err error

	// Validate request
	if request == nil {
		err = errors.New("received empty reboot Instance request")
	} else if request.InstanceId == nil {
		err = errors.New("received reboot Instance request without Instance ID")
	}

	if err != nil {
		return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
	}

	// Call Core gRPC API endpoint
	grpcClient := mm.coreGrpcAtomicClient.GetClient()
	if grpcClient == nil {
		return cClient.ErrCoreGrpcClientNotConnected
	}
	grpcServiceClient := grpcClient.GrpcServiceClient()

	start := time.Now()
	_, err = grpcServiceClient.InvokeInstancePower(ctx, request)
	duration := time.Since(start)
	if err != nil {
		logger.Warn().Err(err).Dur("grpc_duration", duration).Msg("Failed to reboot Instance using Core gRPC API")
		return swe.WrapErr(err)
	}
	logger.Info().Dur("grpc_duration", duration).Msg("Completed activity")

	return nil
}

// Function to Delete NICo Instance with the Site Controller
func (mm *ManageInstance) DeleteInstanceOnSite(ctx context.Context, request *corev1.InstanceReleaseRequest) error {
	logger := log.With().Str("Activity", "DeleteInstanceOnSite").Logger()

	logger.Info().Msg("Starting activity")

	var err error

	// Validate request
	if request == nil {
		err = errors.New("received empty delete Instance request")
	} else if request.Id == nil || request.Id.Value == "" {
		err = errors.New("received delete Instance request without Instance ID")
	}

	if err != nil {
		return temporal.NewNonRetryableApplicationError(err.Error(), swe.ErrTypeInvalidRequest, err)
	}

	// Delegate to the pluggable provisioner when one is registered.
	if mm.provisioner != nil && mm.provisioner.Name() != "core-grpc" {
		provisionerName := mm.provisioner.Name()
		machineID := request.GetId().GetValue()

		// Fire pre-machine-deprovision sync hook (allows providers to prepare cleanup).
		if mm.hooks != nil {
			if err := mm.hooks.FireSync(ctx, "compute", "pre-machine-deprovision", deprovisionHookPayload{
				MachineID:   machineID,
				Provisioner: provisionerName,
			}); err != nil {
				logger.Warn().Err(err).Str("provisioner", provisionerName).Msg("pre-machine-deprovision hook failed")
				// Non-blocking: log but proceed — deprovisioning must not be blocked by hook failure.
			}
		}

		logger.Info().Str("provisioner", provisionerName).Msg("delegating deprovision to external provisioner")
		if err := mm.provisioner.Deprovision(ctx, machineID); err != nil {
			logger.Warn().Err(err).Str("provisioner", provisionerName).Msg("external provisioner deprovision failed")
			return swe.WrapErr(err)
		}

		// Fire post-machine-deprovision async hook (non-blocking cleanup signal).
		if mm.hooks != nil {
			mm.hooks.FireAsync(ctx, "compute", "post-machine-deprovision", deprovisionHookPayload{
				MachineID:   machineID,
				Provisioner: provisionerName,
			})
		}
		return nil
	}

	// Call Core gRPC API endpoint (default path)
	grpcClient := mm.coreGrpcAtomicClient.GetClient()
	if grpcClient == nil {
		return cClient.ErrCoreGrpcClientNotConnected
	}
	grpcServiceClient := grpcClient.GrpcServiceClient()

	start := time.Now()
	_, err = grpcServiceClient.ReleaseInstance(ctx, request)
	duration := time.Since(start)
	if err != nil {
		logger.Warn().Err(err).Dur("grpc_duration", duration).Msg("Failed to delete Instance using Core gRPC API")
		return swe.WrapErr(err)
	}
	logger.Info().Dur("grpc_duration", duration).Msg("Completed activity")

	return nil
}

// NewManageInstance returns a new ManageInstance activity using the Core gRPC path.
func NewManageInstance(coreGrpcAtomicClient *cClient.CoreGrpcAtomicClient) ManageInstance {
	return ManageInstance{
		coreGrpcAtomicClient: coreGrpcAtomicClient,
	}
}

// NewManageInstanceWithProvisioner returns a ManageInstance activity that
// delegates Provision/Deprovision to the given backend instead of Core gRPC
// and fires lifecycle hooks via the given HookFirer.
// Use this when deploying a single external provisioner (Metal3, Ironic, etc.).
// For multi-backend deployments, prefer NewManageInstanceWithRegistry.
func NewManageInstanceWithProvisioner(coreGrpcAtomicClient *cClient.CoreGrpcAtomicClient, prov mp.MachineProvisioner, hooks mp.HookFirer) ManageInstance {
	r := mp.NewRegistry()
	if prov != nil && prov.Name() != mp.BackendCoreGRPC {
		r.Register(prov.Name(), prov)
	}
	return ManageInstance{
		coreGrpcAtomicClient: coreGrpcAtomicClient,
		provisioner:          prov,
		registry:             r,
		hooks:                hooks,
	}
}

// NewManageInstanceWithRegistry returns a ManageInstance activity that selects
// the provisioner backend per-machine using the nico.nvidia.com/provisioner
// annotation from the machine's labels. CoreGRPCProvisioner is always the
// fallback when no annotation is present or the named backend is unknown.
func NewManageInstanceWithRegistry(coreGrpcAtomicClient *cClient.CoreGrpcAtomicClient, registry *mp.Registry, hooks mp.HookFirer) ManageInstance {
	return ManageInstance{
		coreGrpcAtomicClient: coreGrpcAtomicClient,
		registry:             registry,
		hooks:                hooks,
	}
}

// ManageInstanceInventory is an activity wrapper for Instance inventory collection and publishing
type ManageInstanceInventory struct {
	config ManageInventoryConfig
}

// DiscoverInstanceInventory is an activity to collect Instance inventory and publish to Temporal queue
func (mmi *ManageInstanceInventory) DiscoverInstanceInventory(ctx context.Context) error {
	logger := log.With().Str("Activity", "DiscoverInstanceInventory").Logger()
	logger.Info().Msg("Starting activity")
	inventoryImpl := manageInventoryImpl[*corev1.InstanceId, *corev1.Instance, *corev1.InstanceInventory]{
		itemType:                          "Instance",
		config:                            mmi.config,
		internalFindIDs:                   instanceFindIDs,
		internalFindByIDs:                 instanceFindByIDs,
		internalPagedInventory:            instancePagedInventory,
		internalPagedInventoryPostProcess: instancePagedInventoryPostProcess,
	}
	return inventoryImpl.CollectAndPublishInventory(ctx, &logger)
}

// NewManageInstanceInventory returns a ManageInventory implementation for Instance activity
func NewManageInstanceInventory(config ManageInventoryConfig) ManageInstanceInventory {
	return ManageInstanceInventory{
		config: config,
	}
}

func instanceFindIDs(ctx context.Context, grpcClient *cClient.CoreGrpcClient) ([]*corev1.InstanceId, error) {
	grpcServiceClient := grpcClient.GrpcServiceClient()
	instanceIdList, err := grpcServiceClient.FindInstanceIds(ctx, &corev1.InstanceSearchFilter{})
	if err != nil {
		return nil, err
	}
	return instanceIdList.GetInstanceIds(), nil
}

func instanceFindByIDs(ctx context.Context, grpcClient *cClient.CoreGrpcClient, ids []*corev1.InstanceId) ([]*corev1.Instance, error) {
	grpcServiceClient := grpcClient.GrpcServiceClient()
	instanceList, err := grpcServiceClient.FindInstancesByIds(ctx, &corev1.InstancesByIdsRequest{
		InstanceIds: ids,
	})
	if err != nil {
		return nil, err
	}

	return instanceList.GetInstances(), nil
}

// instancePagedInventoryPostProcess will attach NSG propagation information for the inventory page of instances.
// This will only be called for pages with inventory.
func instancePagedInventoryPostProcess(ctx context.Context, grpcClient *cClient.CoreGrpcClient, inventory *corev1.InstanceInventory) (*corev1.InstanceInventory, error) {
	instanceIds := make([]string, len(inventory.GetInstances()))

	for i, instance := range inventory.GetInstances() {
		instanceIds[i] = instance.GetId().GetValue()
	}

	grpcServiceClient := grpcClient.GrpcServiceClient()
	propList, err := grpcServiceClient.GetNetworkSecurityGroupPropagationStatus(ctx, &corev1.GetNetworkSecurityGroupPropagationStatusRequest{
		InstanceIds: instanceIds,
	})

	if err != nil {
		return nil, err
	}

	inventory.NetworkSecurityGroupPropagations = propList.GetInstances()

	return inventory, nil
}

func instancePagedInventory(allItemIDs []*corev1.InstanceId, pagedItems []*corev1.Instance, input *pagedInventoryInput) *corev1.InstanceInventory {
	itemIDs := []string{}
	for _, id := range allItemIDs {
		itemIDs = append(itemIDs, id.GetValue())
	}

	// Create an inventory page with the subset of Machines
	instanceInventory := &corev1.InstanceInventory{
		Instances: pagedItems,
		Timestamp: &timestamppb.Timestamp{
			Seconds: time.Now().Unix(),
		},
		InventoryStatus: input.status,
		StatusMsg:       input.statusMessage,
		InventoryPage:   input.buildPage(),
	}
	if instanceInventory.InventoryPage != nil {
		instanceInventory.InventoryPage.ItemIds = itemIDs
	}
	return instanceInventory
}
