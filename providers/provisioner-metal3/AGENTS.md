# providers/provisioner-metal3

Provisions bare-metal machines via Metal3 Bare Metal Operator (BMO) and
its `BareMetalHost` CRD. BMO reconciles `BareMetalHost` objects against
physical hardware using Ironic (IPMI/Redfish/PXE). NICo creates and
watches the CRs; BMO handles all low-level provisioning.

This provider is **hook-driven** — it registers no HTTP routes. It intercepts
the `pre-machine-provision` and `post-machine-deprovision` sync hooks. Only
machines annotated with `nico.nvidia.com/provisioner: metal3` are handled;
all others pass through to NICo's built-in Core gRPC path.

## Per-machine routing

Add this annotation to a NICo Machine object to opt into Metal3 provisioning:

```yaml
annotations:
  nico.nvidia.com/provisioner: metal3
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `KUBECONFIG` | (in-cluster) | Path to kubeconfig for the management cluster |
| `METAL3_NAMESPACE` | `metal3-system` | Namespace for BareMetalHost CRs and BMC Secrets |
| `LISTEN_ADDR` | `:9443` | gRPC listen address |
| `METRICS_ADDR` | `:9090` | Prometheus metrics address |
