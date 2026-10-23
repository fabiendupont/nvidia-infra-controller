# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/dpfhcp

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/dpfhcp/go.mod providers/dpfhcp/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/dpfhcp/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /dpfhcp-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /dpfhcp-provider /dpfhcp-provider
EXPOSE 9443
ENTRYPOINT ["/dpfhcp-provider"]
