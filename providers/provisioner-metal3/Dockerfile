# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/provisioner-metal3

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY rest-api/ /workspace/rest-api/
COPY providers/provisioner-metal3/go.mod providers/provisioner-metal3/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/provisioner-metal3/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /provisioner-metal3-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /provisioner-metal3-provider /provisioner-metal3-provider
EXPOSE 9443
ENTRYPOINT ["/provisioner-metal3-provider"]
