FROM golang:1.26 AS builder
WORKDIR /workspace
COPY provider-api/ provider-api/
COPY provider-sdk/ provider-sdk/
COPY providers/switch-config/ providers/switch-config/
WORKDIR /workspace/providers/switch-config
RUN CGO_ENABLED=0 GOOS=linux GOWORK=off GONOSUMDB="github.com/NVIDIA/infra-controller/*" \
    go mod tidy && \
    go build -o /switch-config-provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /switch-config-provider /switch-config-provider
EXPOSE 9443
ENTRYPOINT ["/switch-config-provider"]
