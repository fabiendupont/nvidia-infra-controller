# AGENTS.md — providers/fulfillment

This is the `nico-fulfillment` provider sub-module.

## Module

`github.com/NVIDIA/infra-controller/providers/fulfillment`

## Overview

Standalone gRPC provider implementing the NicoProvider protocol defined in
`provider-api/`. Discovered by NICo REST via Kubernetes ConfigMap labels and
communicates over TCP with optional SPIFFE/mTLS.

## Source origin

Merged from:
- `ncx-infra-controller-provider-nvidia-fulfillment` (standalone repo, initial commit)
- `ncx-infra-controller-rest` branch `extensible-architecture` (SQL persistence layer)

## Storage

Schema: `fulfillment` within the shared PostgreSQL cluster.
Connection via `provider-sdk/db.ConnectWithSchema` (Phase 3 — see TODOs in provider.go / server.go).

## Building

```bash
go build ./cmd/provider/
```
