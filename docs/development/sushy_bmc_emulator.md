# Testing Hardware Discovery with Sushy BMC Emulator

Sushy (`sushy-tools`) is an OpenStack Redfish emulator backed by libvirt.
It lets you test NICo's site explorer hardware discovery without physical
BMC hardware. The site explorer connects to Sushy via standard Redfish,
detects the `Sushy` hardware type, and produces exploration reports for
each managed VM.

Sushy also supports Redfish power control and boot-source override,
translating them into libvirt domain operations. This means NICo can
power on/off VMs and set PXE boot via Redfish — the full provisioning
flow works if DHCP/PXE network connectivity is available.

## Prerequisites

- libvirt with QEMU/KVM
- Podman (to run the Sushy container)
- An OpenShift cluster (CRC works) with NICo deployed

## Sushy Emulator Setup

### Create libvirt VMs

Create VMs with fixed UUIDs and MAC addresses. Sushy uses the VM UUID
as the Redfish System ID.

```bash
# Example: 3 VMs with predictable UUIDs
for i in 01 02 03; do
  virt-install --name edge-ipc-$i \
    --uuid aaaaaaaa-1001-4000-8000-aabbccddee$i \
    --memory 16384 --vcpus 8 \
    --os-variant generic --boot uefi \
    --disk size=50 \
    --network network=default,mac=aa:bb:cc:dd:ee:$i \
    --noautoconsole --noreboot
done
```

### Configure Sushy

Create `/etc/sushy-emulator/sushy-emulator.conf`:

```python
SUSHY_EMULATOR_LISTEN_IP = "0.0.0.0"
SUSHY_EMULATOR_LISTEN_PORT = 10443
SUSHY_EMULATOR_SSL_CERT = "/etc/sushy-emulator/ssl/sushy.crt"
SUSHY_EMULATOR_SSL_KEY = "/etc/sushy-emulator/ssl/sushy.key"
SUSHY_EMULATOR_LIBVIRT_URI = "qemu:///system"
SUSHY_EMULATOR_ALLOWED_INSTANCES = [
    "aaaaaaaa-1001-4000-8000-aabbccddee01",
    "aaaaaaaa-1001-4000-8000-aabbccddee02",
    "aaaaaaaa-1001-4000-8000-aabbccddee03"
]
```

Generate a self-signed certificate with the host IP as a SAN so
clients can verify it without disabling TLS checks:

```bash
HOST_IP="192.168.1.51"  # adjust to your host's IP
sudo mkdir -p /etc/sushy-emulator/ssl
sudo openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:secp384r1 \
  -nodes -days 3650 -subj '/CN=sushy-emulator' \
  -addext "subjectAltName=IP:${HOST_IP},IP:127.0.0.1,DNS:localhost" \
  -keyout /etc/sushy-emulator/ssl/sushy.key \
  -out /etc/sushy-emulator/ssl/sushy.crt
```

### Patch the ServiceRoot template

The nv-redfish library requires a `Links` object in the Redfish
ServiceRoot. The default Sushy template does not include it. Create
`/etc/sushy-emulator/root.json`.

This file is a Jinja2 template — the `{% if %}` directives are
rendered by Sushy at request time:

```jinja2
{
    "@odata.type": "#ServiceRoot.v1_5_0.ServiceRoot",
    "Id": "RedvirtService",
    "Name": "Redvirt Service",
    "RedfishVersion": "1.5.0",
    "Vendor": "Sushy",
    "UUID": "85775665-c110-4b85-8989-e6162170b3ec",
    {% if feature_set == "full" %}
    "Chassis": {
        "@odata.id": "/redfish/v1/Chassis"
    },
    {% endif %}
    "Systems": {
        "@odata.id": "/redfish/v1/Systems"
    },
    {% if feature_set != "minimum" %}
    "Managers": {
        "@odata.id": "/redfish/v1/Managers"
    },
    {% endif %}
    {% if feature_set == "full" %}
    "Registries": {
        "@odata.id": "/redfish/v1/Registries"
    },
    "CertificateService": {
        "@odata.id": "/redfish/v1/CertificateService"
    },
    "UpdateService": {
        "@odata.id": "/redfish/v1/UpdateService"
    },
    {% endif %}
    "Links": {
        "Sessions": {
            "@odata.id": "/redfish/v1/SessionService/Sessions"
        }
    },
    "@odata.id": "/redfish/v1/",
    "@Redfish.Copyright": "Copyright 2014-2016 Distributed Management Task Force, Inc. (DMTF). For the full DMTF copyright policy, see http://www.dmtf.org/about/policies/copyright."
}
```

### Run Sushy as a systemd service

Create `/etc/systemd/system/sushy-emulator.service`.

> **Note:** `--privileged` and host networking are required because
> Sushy needs access to the libvirt socket. Run this only on a
> development workstation, not on shared or production hosts.

The `root.json` bind-mount path depends on the Python version inside the
container image. Verify with `podman run --rm <image> python3 --version`
and adjust if needed.

```ini
[Unit]
Description=Sushy Redfish Emulator for Libvirt
After=network-online.target libvirtd.service
Wants=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=10
ExecStartPre=-/usr/bin/podman kill sushy-emulator
ExecStartPre=-/usr/bin/podman rm sushy-emulator
ExecStart=/usr/bin/podman run --rm --name sushy-emulator \
  --net=host \
  --privileged \
  -v /var/run/libvirt:/var/run/libvirt \
  -v /etc/sushy-emulator:/etc/sushy-emulator:ro \
  -v /etc/sushy-emulator/root.json:/usr/local/lib/python3.12/site-packages/sushy_tools/emulator/templates/root.json:ro \
  quay.io/metal3-io/sushy-tools:latest \
  sushy-emulator --config /etc/sushy-emulator/sushy-emulator.conf
ExecStop=/usr/bin/podman stop sushy-emulator

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now sushy-emulator
```

### Verify

```bash
curl -sk https://localhost:10443/redfish/v1/Systems | python3 -m json.tool
```

You should see your VMs listed as Redfish Systems.

To verify from inside an OpenShift pod:

```bash
oc run test-curl --rm -i --restart=Never \
  --image=registry.access.redhat.com/ubi9/ubi-minimal \
  -- curl -sk https://<host-ip>:10443/redfish/v1
```

## NICo Site Explorer Configuration

Add the following to the site explorer config in `nicoApiSiteConfig`:

```toml
[site_explorer]
explore_mode = "nv-redfish"
override_target_ip = "<host-ip-reachable-from-cluster>"
override_target_port = 10443
dpu_policy = "no_dpu"
run_interval = "30s"
create_machines = true
```

- `override_target_ip`: The host IP that OpenShift pods can reach
  (not `localhost`). On CRC, the host's LAN IP works.
- `dpu_policy = "no_dpu"`: Sushy VMs have no DPUs.

## Register Expected Machines

Register each VM as an expected machine. The MAC address must match the
VM's NIC, and `bmc_ip_address` provides the static BMC IP for the
site explorer to use with `override_target_ip`.

> **Note:** The credentials below are test-only placeholders for a local
> emulator. Do not reuse them in shared environments. Prefer injecting
> credentials via Vault rather than passing them on the command line,
> which exposes them in shell history and process listings.

```bash
nico-admin-cli em add \
  --bmc-mac-address "aa:bb:cc:dd:ee:01" \
  --bmc-username "test-user" \
  --bmc-password "test-only-not-a-real-password" \
  --chassis-serial-number "QPX12345-01" \
  --bmc-ip-address "192.168.50.101" \
  --bmc-retain-credentials true \
  --dpu-mode no-dpu \
  --meta-name "edge-ipc-01"
```

The site explorer also needs:
- A **network segment** of type `underlay` with a prefix covering the
  `bmc_ip_address` range.
- **DHCP timestamps** on the preallocated machine interfaces (set
  `last_dhcp` in the `machine_interfaces` table, or call the
  `DiscoverDhcp` gRPC method).

## Vault Credentials

The site explorer requires these Vault KV entries. Use `vault kv put`
with `@file` syntax to store JSON objects (not strings):

> **Note:** Replace the placeholder credentials below with values
> appropriate for your test environment.

```bash
printf '{"UsernamePassword":{"username":"test-user","password":"test-only"}}' > /tmp/cred.json
vault kv put secrets/machines/bmc/site/root @/tmp/cred.json
vault kv put secrets/machines/all_hosts/site_default/uefi-metadata-items/auth @/tmp/cred.json
vault kv put secrets/machines/all_hosts/site_default/bmc-metadata-items/root @/tmp/cred.json
vault kv put secrets/machines/all_dpus/site_default/uefi-metadata-items/auth @/tmp/cred.json
vault kv put secrets/machines/all_dpus/site_default/bmc-metadata-items/root @/tmp/cred.json
rm /tmp/cred.json
```

A Vault Kubernetes auth role for `nico-api` is also required:

```bash
vault write auth/kubernetes/role/nico-api \
  bound_service_account_names=nico-api \
  bound_service_account_namespaces=nico-system \
  policies=nico-vault-policy \
  ttl=24h
```

## What to Expect

Once everything is configured, the site explorer logs should show:

```text
endpoint_explorations=3 endpoint_explorations_success=3
endpoint_explorations_failures=0
```

Each explored endpoint produces a report with:
- `EndpointType: Bmc`
- `MachineSetupStatus.IsDone: true` (Sushy has no real BIOS to configure)
- One system and one chassis per VM

## Limitations

- **PXE provisioning requires DHCP relay**: Sushy supports Redfish
  power control and boot-source override (translated to libvirt
  operations), so NICo can set PXE boot and power on VMs. However,
  the VMs must be able to reach NICo's DHCP/PXE services over the
  network for the full provisioning flow to work.
- **No DPU support**: Sushy VMs have no DPUs. Use `dpu_policy = "no_dpu"`.
- **No BIOS/lockdown**: All BIOS setup and lockdown checks are stubbed
  as no-ops for the Sushy hardware type.
- **No AccountService**: Sushy does not implement the Redfish
  `AccountService` endpoint. All credential operations (create user,
  change password) are accepted silently by the vendor implementation.
- **Single endpoint**: Sushy serves all VMs through one IP:port.
  `override_target_ip/port` redirects all BMC calls to that endpoint.
  In production, each server has its own BMC IP.
