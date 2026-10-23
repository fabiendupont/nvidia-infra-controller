# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/fulfillment

COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/catalog/ /workspace/providers/catalog/
COPY rest-api/ /workspace/rest-api/
COPY providers/fulfillment/go.mod providers/fulfillment/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*
RUN go mod download

COPY providers/fulfillment/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fulfillment-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /fulfillment-provider /fulfillment-provider
EXPOSE 9443
ENTRYPOINT ["/fulfillment-provider"]
