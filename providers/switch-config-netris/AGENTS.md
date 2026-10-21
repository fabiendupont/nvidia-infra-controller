NICo switch-config provider for Netris-managed switches.

Handles switches annotated with `nico.nvidia.com/switch-config-provider: netris`.
Implements `cfg.SwitchConfigurator` from `providers/switch-config/configurator/`
(the shared switch-config feature contract).

Pushes VRF, BGP, and VxLAN config via the Netris SDN controller REST API.
See `internal/netris.go` for the Netris API endpoints to implement.
