# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/switch-config

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
# nvue-client-go is vendored in-repo; go.mod replace points to ../../vendor/nvidia-nvue-client-go
COPY vendor/nvidia-nvue-client-go/ /workspace/vendor/nvidia-nvue-client-go/
COPY providers/switch-config/go.mod providers/switch-config/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*,github.com/fabiendupont/*
RUN go mod download

COPY providers/switch-config/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /switch-config-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /switch-config-provider /switch-config-provider
EXPOSE 9443
ENTRYPOINT ["/switch-config-provider"]
