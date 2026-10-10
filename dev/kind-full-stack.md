# Kind Full Stack — NICo Core + REST in Kind

Runs the complete NICo stack (Core + REST API + site-agent + Vault + providers) fully
inside Kind. No host-side processes required.

## Quick start

```bash
cd rest-api
make kind-reset KIND_CORE=true
```

First run builds the `nico-rest-core` image (~15 min cold Rust build; subsequent builds
use Docker layer cache). After that, `make kind-reset KIND_CORE=true` takes the same
time as the default `make kind-reset`.

## What gets deployed

| Component | Image | Notes |
|-----------|-------|-------|
| NICo REST API | `localhost:5000/nico-rest-api:latest` | Go, built from `rest-api/` |
| NICo Core (nico-api) | `localhost:5000/nico-rest-core:latest` | Rust, built from `Dockerfile.nico-core` |
| Vault | `hashicorp/vault:1.20.2` | Dev mode, auto-unsealed |
| Site-agent | `localhost:5000/nico-rest-site-agent:latest` | Connects to in-cluster Core |
| PostgreSQL | Existing in-cluster | Shared with REST API, Core uses `nico` DB |
| Temporal | Existing in-cluster | Shared |
| Keycloak | Existing in-cluster | Shared |

## `KIND_CORE` variable

```makefile
KIND_CORE ?= true        # "true" = real Core in Kind (default)
KIND_CORE=mock           # old mock-core behaviour
KIND_CORE=local          # Core runs on host (LOCAL_CORE_HOST:LOCAL_CORE_PORT)
```

## Design decisions and gotchas

These were all discovered by running in Kind and are captured in the kustomize base
files under `rest-api/deploy/kustomize/base/core/`.

### TLS: ECDSA required

NICo Core uses Rustls, which requires ECDSA keys. cert-manager defaults to RSA 2048,
which causes the TLS acceptor to fail silently at startup. The `certificate.yaml`
explicitly requests `algorithm: ECDSA, size: 256`.

```yaml
spec:
  privateKey:
    algorithm: ECDSA
    size: 256
```

### Vault: service account token must be hidden

NICo Core auto-detects whether it is running in Kubernetes by checking for
`/var/run/secrets/kubernetes.io/serviceaccount/token`. If present, it uses Kubernetes
service account auth to Vault instead of `VAULT_TOKEN`. The deployment mounts an
`emptyDir` over that path and sets `automountServiceAccountToken: false` so the Core
uses the `VAULT_TOKEN` env var directly (set to `dev-root-token`).

### PostgreSQL: TLS disabled via env var

In-cluster PostgreSQL does not have TLS enabled. Core defaults to requiring TLS when
`[tls]` is configured. Set `DISABLE_TLS_ENFORCEMENT=1` (recognised by Core) to skip
TLS for the DB connection without changing the TOML config.

### TLS config field names

The `[tls]` section uses Rust struct field names, not short aliases:

```toml
[tls]
root_cafile_path       = "/etc/nico-core/ca.crt"
identity_pemfile_path  = "/etc/nico-core/tls.crt"
identity_keyfile_path  = "/etc/nico-core/tls.key"
admin_root_cafile_path = "/etc/nico-core/ca.crt"
```

### Auth section required even in bypass_rbac mode

`bypass_rbac = true` disables Casbin policy enforcement but the TLS listener still
parses the `[auth]` section to build the mTLS trust context. Omitting it causes a
panic. Minimal permissive config:

```toml
[auth]
permissive_mode = true

[auth.trust]
spiffe_trust_domain       = "nico.local"
spiffe_service_base_paths = ["nico"]
spiffe_machine_base_path  = "machine"
additional_issuer_cns     = []
```

### Resource pools: vpc-vni required

Core requires `vpc-vni` alongside the standard `vni`, `lo-ip`, and `vlan-id` pools.
Missing it causes a startup error after Vault auth succeeds.

### CA cert rotation: site-agent must trust Core's CA

The site-agent uses `core-grpc-client-site-agent-certs` to verify Core's TLS cert.
If cert-manager rotates the CA, the old CA cert in the site-agent's secret will no
longer verify Core's new cert. Symptom: `crypto/rsa: verification error` in
site-agent logs. Fix: copy `ca.crt` from `nico-rest-core-grpc-certs` into
`core-grpc-client-site-agent-certs`.

This is handled automatically on a fresh `make kind-reset KIND_CORE=true` since both
certificates are issued by the same `nico-rest-ca-issuer` with a fresh CA each time.

## Validation

After `make kind-reset KIND_CORE=true` completes:

```bash
# Token
TOKEN=$(curl -sf -X POST \
  http://localhost:8082/realms/nico-dev/protocol/openid-connect/token \
  -d "client_id=nico-api&client_secret=nico-local-secret&grant_type=password&username=admin@example.com&password=adminpassword" \
  | jq -r .access_token)

# Create a site (exercises REST → Core gRPC path)
INFRA_ID=$(curl -s http://localhost:8388/v2/org/test-org/nico/infrastructure-provider/current \
  -H "Authorization: Bearer $TOKEN" | jq -r .id)
curl -s -X POST http://localhost:8388/v2/org/test-org/nico/site \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"name\":\"test-site\",\"infrastructureProviderId\":\"$INFRA_ID\"}" | jq .id
```

A UUID response confirms the full REST → Core gRPC path is working.

## Known limitations

- **Flow gRPC unavailable**: the site-agent logs `Flow gRPC: name resolver error`
  because the Flow service is not deployed in the dev cluster. This is expected and
  does not affect Core functionality.
- **mock-core still running**: `make kind-reset` does not remove the mock-core
  deployment. It can be deleted manually: `kubectl delete deployment nico-rest-mock-core -n nico-rest`.
- **machine-a-tron**: not deployed. Machine state transitions that require machine-a-tron
  will not complete. Run integration tests against a cluster with the full Rust stack.
- **Rust build time**: first build of `Dockerfile.nico-core` takes ~15 min (full
  `cargo build --release` with no cache). The `COPY pxe/ pxe/` layer is required because
  `carbide-ipxe-renderer` embeds `pxe/ipxe/local/embed.ipxe` at build time.
