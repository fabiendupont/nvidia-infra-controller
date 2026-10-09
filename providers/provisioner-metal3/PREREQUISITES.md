# Prerequisites for nico-provisioner-metal3

The following must be deployed before installing the Metal3 provisioner provider.

## 1. Metal3 Bare Metal Operator

Metal3 BMO provides the `BareMetalHost` CRD and reconciles it against physical
hardware via Ironic. The provider creates and watches BareMetalHost CRs in the
namespace configured by `METAL3_NAMESPACE` (default: `metal3-system`).

**OpenShift 4.22:** BMO is included automatically when the cluster is installed
with `platform: baremetal` (IPI or UPI bare-metal). It runs in the
`openshift-machine-api` namespace.

**Standalone (any cluster):**
```bash
helm install baremetal-operator \
  oci://quay.io/metal3-io/baremetal-operator \
  --namespace metal3-system --create-namespace
```

## 2. Ironic

Ironic handles the actual BMC communication (IPMI/Redfish), PXE boot, and OS
deployment on behalf of BMO. In OCP 4.22, Ironic is deployed alongside BMO in
the baremetal platform namespace.

Verify Ironic is running:
```bash
kubectl get pods -n openshift-machine-api | grep ironic
```

## 3. Ironic Inspector

Ironic Inspector runs in the IPA ramdisk during machine inspection and stores
hardware data (NIC MACs, CPU, TPM EK cert, IB GUIDs, DPU fingerprint) in the
Ironic Inspector API.

Required for the `post-machine-inspect` hook to perform TPM EK and platform
serial correlation. Without it, the hook passes through silently.

Configure the provider to use Ironic Inspector:
```bash
helm upgrade nico-provisioner-metal3 helm/charts/nico-provisioner-metal3/ \
  --set ironic.inspectorURL=http://ironic-inspector.metal3-system:5050
```

In OCP 4.22, Ironic Inspector is included. Find its service:
```bash
kubectl get svc -n openshift-machine-api | grep inspector
```

## 4. Custom IPA image (optional — enables TPM/IB/GPU/DPU inspection)

The stock OCP IPA image does not include NVIDIA hardware managers. Without the
custom image, `extra.tpm_ekcert`, `extra.ib_port_guids`, `extra.gpu_uuids`, and
`extra.dpu_psid` fields are absent from Ironic Inspector's introspection data,
and the `post-machine-inspect` hook's attestation correlation is skipped.

Build and push the custom image:
```bash
git clone https://github.com/rh-ecosystem-edge/nico-ipa-extensions
cd nico-ipa-extensions
BASE=$(oc adm release info --image-for=ironic-agent)
podman build --build-arg BASE_IMAGE=$BASE -t quay.io/<org>/nico-ipa:latest .
podman push quay.io/<org>/nico-ipa:latest
```

Enable in the Helm chart:
```bash
helm upgrade nico-provisioner-metal3 helm/charts/nico-provisioner-metal3/ \
  --set customIPA.enabled=true \
  --set customIPA.image=quay.io/<org>/nico-ipa:latest
```

This modifies the cluster-scoped `Provisioning` CR (`provisioning-configuration`),
affecting ALL BareMetalHost resources in the cluster. Confirm with the cluster
administrator before enabling.

## 5. Machine annotation

Machines to be provisioned by Metal3 must have the annotation
`nico.nvidia.com/provisioner: metal3` on their NICo expectedMachine resource.
Without this annotation, the provider passes through silently and NICo falls
back to Core gRPC provisioning.

Set the annotation before the machine is discovered:
```bash
kubectl patch expectedmachine <name> --type=merge \
  -p '{"metadata":{"annotations":{"nico.nvidia.com/provisioner":"metal3"}}}'
```

## 6. KUBECONFIG or in-cluster service account

The Metal3 provider creates BareMetalHost CRs and Secrets in the Metal3
namespace. It needs credentials to the cluster where BMO runs.

**Same cluster (default):** Set `kubeconfig.inCluster: true` in values (already
the default). The provider uses its pod service account.

**Remote cluster:** Create a Secret containing a kubeconfig for the remote
cluster and set `kubeconfig.secretName: <secret-name>` in values.

## Verification

After all prerequisites are in place and the provider is installed:

```bash
# Provider registered with NICo
kubectl logs -n <nico-ns> deploy/nico-rest-api | grep -i "metal3\|provisioner"

# BareMetalHost CRD available
kubectl get crd baremetalhosts.metal3.io

# Ironic Inspector reachable
curl -s http://<ironic-inspector-url>/v1/introspection | jq .
```
