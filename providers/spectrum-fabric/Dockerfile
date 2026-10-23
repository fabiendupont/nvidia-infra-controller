# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/spectrum-fabric

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/spectrum-fabric/go.mod providers/spectrum-fabric/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/spectrum-fabric/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /spectrum-fabric-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /spectrum-fabric-provider /spectrum-fabric-provider
EXPOSE 9443
ENTRYPOINT ["/spectrum-fabric-provider"]
