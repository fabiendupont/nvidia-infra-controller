# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.26 AS builder
WORKDIR /workspace/providers/health

# Copy only the modules this provider needs.
# GOWORK=off ensures the workspace file is ignored; replace directives in go.mod
# resolve provider-api and provider-sdk from the copied local paths.
COPY provider-api/ /workspace/provider-api/
COPY provider-sdk/ /workspace/provider-sdk/
COPY providers/health/go.mod providers/health/go.sum ./
ENV GOWORK=off
ENV GONOSUMDB=github.com/NVIDIA/infra-controller/*,github.com/fabiendupont/*
RUN go mod download

COPY providers/health/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /health-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /health-provider /health-provider
EXPOSE 9443
ENTRYPOINT ["/health-provider"]
