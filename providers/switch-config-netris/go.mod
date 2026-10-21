module github.com/NVIDIA/infra-controller/providers/switch-config-netris

go 1.26.4

require (
	github.com/NVIDIA/infra-controller/provider-api v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/provider-sdk v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/providers/switch-config v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.34.0
	google.golang.org/grpc v1.73.0
)

replace github.com/NVIDIA/infra-controller/provider-api => ../../provider-api
replace github.com/NVIDIA/infra-controller/provider-sdk => ../../provider-sdk
replace github.com/NVIDIA/infra-controller/providers/switch-config => ../switch-config
replace google.golang.org/genproto v0.0.0-20200423170343-7949de9c1215 => google.golang.org/genproto v0.0.0-20240903143218-8af14fe29dc1
