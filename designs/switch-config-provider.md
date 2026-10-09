# switch-config Provider Design

## Software Design Document

## Revision History

| Version | Date | Modified By | Description |
| :---: | :---: | :---- | :---- |
| 0.1 | 2026-10-09 | Red Hat NCP Team | Initial draft |

---

# 1. Introduction

## 1.1 Purpose

This document describes the `switch-config` provider sidecar — an external
NICo provider (per NEP-0008) that handles the full switch configuration
management lifecycle for NVIDIA NV-OS switches. It replaces NVIDIA Config
Manager as the authoritative source for switch config rendering, versioning,
and push operations, using NICo's existing data model as the source of truth.

## 1.2 Scope

The provider is responsible for:

- Rendering switch configurations from NICo's intended state (via
  `NicoSwitchService`)
- Storing versioned config snapshots in a provider-owned PostgreSQL schema
- Validating cable topology by comparing LLDP-observed neighbors against
  NICo's topology model
- Serving operator UI pages via NICo's HTTP proxy mechanism
- Registering provider-contributed nav entries via `GetResourceTypes`

## 1.3 Provider identity

```
name:     nico-switch-config
features: ["switch-config"]
task_queue: nico-switch-config-task-queue
```

---

# 2. Architecture

```
NICo REST API (cloud)
  │
  ├── NicoSwitchService gRPC ──────────────────────────────┐
  │   (ListSwitches, GetSwitchRoutingConfig, etc.)         │
  │                                                         ▼
  ├── HTTP proxy (HandleRequest) ◄────── switch-config provider (Go)
  │   /provider/switch-config/*                │
  │                                            ├── Config templating (text/template + CUE)
  ├── GetResourceTypes cache                   ├── Config Store (PostgreSQL schema)
  │   → sidebar "Switches" nav entry           ├── Temporal worker (cable validation)
  │                                            └── HTML page serving (operator UI)
  │
  └── Hook dispatch
      post-switch-config-rendered ──► Config Store write (async)
```

---

# 3. Config Templating

## 3.1 Rendering context

The provider queries `NicoSwitchService.GetSwitchRoutingConfig` and
`NicoSwitchService.GetSwitch` to build a `RenderContext` struct passed to the
template engine:

```go
// RenderContext is the data model passed to all config templates.
type RenderContext struct {
    Switch          providerv1.NicoSwitch
    RoutingConfig   providerv1.SwitchRoutingConfig
    // SiteID is the NICo site UUID this switch belongs to.
    SiteID          string
    // Extra carries operator-supplied key-value overrides from the Config Store.
    Extra           map[string]string
}
```

## 3.2 NV-OS CLI templates (Go text/template + Sprig)

Free-form text templates for NV-OS CLI syntax (`.nvue` files). Template
directory is operator-configurable via `SWITCH_CONFIG_TEMPLATE_DIR` env var,
defaulting to `/etc/switch-config/templates/`. Templates are loaded at runtime
— no recompilation on edit.

```
templates/
  base.nvue.tmpl         # global BGP ASN, loopback, management VRF
  vpc-vrf.nvue.tmpl      # per-VRF config (one render per VRF in RoutingConfig.Vrfs)
  evpn.nvue.tmpl         # EVPN fabric parameters
  cable-validation.nvue.tmpl  # LLDP probe configuration
```

Example template fragment:
```
nv set router bgp autonomous-system {{ .RoutingConfig.BgpPeers | first | field "PeerAsn" }}
{{- range .RoutingConfig.Vrfs }}
nv set vrf {{ .Name }} router bgp autonomous-system {{ $.RoutingConfig.BgpPeers | ... }}
{{- end }}
```

## 3.3 NVUE JSON templates (CUE)

Structured NVUE JSON payloads use CUE for schema validation, following the
pattern established in `providers/spectrum-fabric/` (which uses map-based NVUE
calls). CUE schemas are colocated with the templates:

```
cue/
  switch_base.cue        # base NVUE schema (validated against NVUE OpenAPI)
  vpc_vrf.cue            # VRF configuration schema
```

The CUE layer validates the rendered JSON before it is pushed to the switch via
NVUE REST API, catching schema violations without a live switch.

## 3.4 Rust side (site-plane templates, future)

When NEP-0009 (site-native providers) is implemented, the template engine can
run at-site using Tera (already a workspace dep at `tera = "1.20"`). The Rust
`RenderContext` equivalent mirrors the Go struct above. Templates loaded from
`CARBIDE_SWITCH_TEMPLATE_DIRECTORY` (same pattern as
`CARBIDE_PXE_TEMPLATE_DIRECTORY`). This allows operators to edit templates on
the site host without recompiling NICo Core.

---

# 4. Config Store

## 4.1 Schema

Provider-owned PostgreSQL schema `switch_config` (via provider-sdk
`ConnectWithSchema`):

```sql
CREATE TABLE switch_config_versions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    switch_id    UUID NOT NULL,           -- NICo Switch UUID
    filename     TEXT NOT NULL,           -- e.g. "base.nvue", "vpc-vrf-prod.nvue"
    file_type    TEXT NOT NULL,           -- "nvue-cli" | "nvue-json" | "cue"
    content      BYTEA NOT NULL,          -- gzip-compressed rendered content
    content_hash TEXT NOT NULL,           -- sha256 hex; deduplication key
    author       TEXT NOT NULL,
    commit_msg   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Deduplication: skip insert if (switch_id, filename, content_hash) exists
    UNIQUE (switch_id, filename, content_hash)
);
CREATE INDEX ON switch_config_versions (switch_id, filename, created_at DESC);
```

## 4.2 HTTP routes (via GetRoutes / HandleRequest)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/provider/switch-config/switches` | List switches with config status |
| GET | `/provider/switch-config/switches/{id}` | Switch detail (tabs: history, neighbors, actions) |
| GET | `/provider/switch-config/switches/{id}/config` | Latest config version |
| GET | `/provider/switch-config/switches/{id}/config/history` | All versions paginated |
| POST | `/provider/switch-config/switches/{id}/actions/render` | Re-render + store |
| POST | `/provider/switch-config/switches/{id}/actions/push` | Push latest config to switch via NVUE |
| POST | `/provider/switch-config/switches/{id}/actions/validate-cables` | Trigger cable validation workflow |

## 4.3 Config persistence hook

The provider registers an async reaction on `switch-config/post-render`:

```go
// After render completes, store the result asynchronously.
func (p *Provider) GetHookRegistrations(...) {
    return &providerv1.HookRegistrationList{Registrations: []*providerv1.HookRegistration{
        {Type: ASYNC, Feature: "switch-config", Event: "post-render"},
    }}
}
```

---

# 5. Cable Validation

Implemented as a Temporal workflow on the provider's own task queue
(`nico-switch-config-task-queue`), using the Temporal client from
`provider-sdk.NewTemporalClient`.

## 5.1 Workflow activities

```
CableValidationWorkflow(switch_id)
  │
  ├── FetchExpectedNeighbors
  │     NicoSwitchService.ListExpectedNeighbors(switch_id)
  │
  ├── DiscoverLLDPNeighbors
  │     SSH/NETCONF to switch → parse LLDP neighbor table
  │     Credentials from credential_ref via NICo secrets API
  │
  ├── CompareNeighbors
  │     Diff expected vs observed per port
  │     Produce mismatch list
  │
  └── ReportResults
        NicoSwitchService.ReportObservedNeighbors(switch_id, observed, timestamp)
        Write validation result to Config Store
        Emit NATS event "switch-config/cable-validation-complete"
```

## 5.2 Trigger

- Operator action: `POST /provider/switch-config/switches/{id}/actions/validate-cables`
- Lifecycle event: when switch reaches `SWITCH_STATE_READY` for the first time
  (async hook on `switch/post-state-ready`)

---

# 6. Operator UI

The provider serves complete HTML pages via `HandleRequest`. Auth is handled by
NICo's middleware before the request reaches `HandleRequest`, so the provider
receives an authenticated request context.

## 6.1 Pages

| Page | Path | Description |
|------|------|-------------|
| Switch list | `/provider/switch-config/ui/` | All switches with state badges |
| Switch detail | `/provider/switch-config/ui/switches/{id}` | Tabs: Config History, Neighbors, Actions |
| Workflow status | `/provider/switch-config/ui/workflows/{run_id}` | Live Temporal run status |

## 6.2 Resource type registration

```go
func (p *Provider) GetResourceTypes(...) {
    return &providerv1.GetResourceTypesResponse{
        ResourceTypes: []*providerv1.ResourceTypeDescriptor{{
            Name:      "switch",
            Plural:    "switches",
            ApiPrefix: "/provider/switch-config",
            Actions: []*providerv1.ActionDescriptor{
                {Name: "push-config",       Label: "Push Config",       Method: "POST",
                 Path: "/switches/{id}/actions/push",           RequiresConfirmation: true},
                {Name: "render",            Label: "Re-render Config",  Method: "POST",
                 Path: "/switches/{id}/actions/render",         RequiresConfirmation: false},
                {Name: "validate-cables",   Label: "Validate Cables",   Method: "POST",
                 Path: "/switches/{id}/actions/validate-cables", RequiresConfirmation: false},
            },
        }},
    }
}
```

NICo renders a "Switches" entry in the sidebar pointing to
`/provider/switch-config/ui/` for operators.

---

# 7. Data Model Gaps

Fields defined in the proto that NICo does not yet persist and must be added
before those fields can be populated:

| Field | Gap description | Follow-up |
|-------|-----------------|-----------|
| `NicoSwitch.platform` | NOS type not stored; inferred via NVUE probe | Add to Switch DB model |
| `NicoSwitch.management_ip` | NVOS management IP not on Switch row | Add column, populate from DHCP |
| `SwitchPort` (repeated) | Port-level inventory not persisted | New `switch_ports` table |
| `SwitchRoutingConfig.bgp_peers` | BGP topology not in NICo | New `switch_bgp_peers` table |
| `SwitchRoutingConfig.isis_interfaces` | IS-IS config not stored | New `switch_isis_config` table |
| `SwitchRoutingConfig.evpn` | EVPN params not stored | New `switch_evpn_config` table |
| `ListExpectedNeighbors` | Cabling not modeled in NICo today | New `switch_cable_links` table |
| `ReportObservedNeighbors` | No storage for observed neighbors | Extend `switch_cable_links` |
