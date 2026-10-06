# providers/compute

The compute provider implements the `compute` feature of NICo's extensible architecture. It wraps all existing Instance, InstanceType, Machine, Allocation, OperatingSystem, SSHKey, SSHKeyGroup, MachineCapability, MachineValidation, SKU, Rack, and Tray API handlers and Temporal workflows. It depends on `nico-networking`. It exposes the `computesvc.Service` cross-domain interface so other providers can read machine and instance state. Runs compiled-in or as an external gRPC process via `cmd/compute-provider/`.
