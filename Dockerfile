# Build the manager binary
FROM golang:1.19 as builder

WORKDIR /workspace

COPY . .

RUN go mod tidy

RUN make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/controller-manager /controller-manager
COPY --from=builder /workspace/bin/apiserver /apiserver
USER 65532:65532

#ENTRYPOINT ["/manager"]
