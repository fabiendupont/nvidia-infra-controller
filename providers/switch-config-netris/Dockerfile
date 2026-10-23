# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/switch-config-netris

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/switch-config/ /workspace/providers/switch-config/
COPY providers/switch-config-netris/go.mod providers/switch-config-netris/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/switch-config-netris/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /switch-config-netris-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /switch-config-netris-provider /switch-config-netris-provider
EXPOSE 9443
ENTRYPOINT ["/switch-config-netris-provider"]
