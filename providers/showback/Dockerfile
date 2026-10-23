# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/showback

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/showback/go.mod providers/showback/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/showback/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /showback-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /showback-provider /showback-provider
EXPOSE 9443
ENTRYPOINT ["/showback-provider"]
