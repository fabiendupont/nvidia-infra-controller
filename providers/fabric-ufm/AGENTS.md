# providers/fabric-ufm

Manages InfiniBand fabric configuration via UFM Enterprise. Syncs NICo IB partition (PKEY) state to UFM.

## Status

Implemented. Communicates directly with the UFM Enterprise REST API (not via AAP/Ansible) to manage InfiniBand partition (PKEY) lifecycle.

Hook registrations:
- `post-create-ib-partition` (async) → creates PKEY on UFM
- `post-delete-ib-partition` (async) → removes PKEY from UFM
- `pre-create-ib-partition` (sync) → validates PKEY doesn't conflict with existing UFM partitions

## Module

`github.com/NVIDIA/infra-controller/providers/fabric-ufm`

## Configuration

All configuration via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `UFM_URL` | Yes | — | UFM Enterprise REST API base URL (e.g., `https://ufm.lab:443`) |
| `UFM_USERNAME` | Yes | — | HTTP basic auth username |
| `UFM_PASSWORD` | Yes | — | HTTP basic auth password |
| `UFM_TLS_SKIP_VERIFY` | No | `false` | Skip TLS certificate verification |
| `UFM_SYNC_IB_PARTITION` | No | `true` | Enable IB partition sync hooks |
| `UFM_JOB_POLL_INTERVAL` | No | `2s` | UFM async job poll interval |
| `UFM_JOB_TIMEOUT` | No | `2m` | UFM async job wait timeout |

## Dependencies

- `nico-networking` — IB partition events are sourced from the networking provider
