# providers/dpfhcp

Manages DPU cluster provisioning via DPFHCPProvisioner CRs on the management OpenShift cluster.

## Status

Partial implementation. K8s CRD lifecycle, Temporal workflows for async provisioning, and hook bindings for site lifecycle events are implemented. Provisioning records use an in-memory store (Phase 3 will add PostgreSQL persistence in the `dpfhcp` schema).

## Module

`github.com/NVIDIA/infra-controller/providers/dpfhcp`

## Routes

- `POST /api/v1/sites/:siteId/dpf-hcp` — Provision DPF HCP infrastructure for a site
- `GET /api/v1/sites/:siteId/dpf-hcp` — Get DPF HCP provisioning status
- `DELETE /api/v1/sites/:siteId/dpf-hcp` — Deprovision DPF HCP infrastructure

## Dependencies

- `nico-site` — site must exist before DPF HCP can be provisioned
