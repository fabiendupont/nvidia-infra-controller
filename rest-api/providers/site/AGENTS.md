# providers/site

The site provider implements the `site` feature of NICo's extensible architecture. It wraps all existing Site, ExpectedMachine, ExpectedPowerShelf, and ExpectedSwitch API handlers and Temporal workflows. It depends on `nico-compute`. Runs compiled-in or as an external gRPC process via `cmd/site-provider/`.
