# Stage1: Build the pvc-autoresizer binary
FROM --platform=$BUILDPLATFORM golang:1.25.7-bookworm@sha256:564e366a28ad1d70f460a2b97d1d299a562f08707eb0ecb24b659e5bd6c108e1 AS builder

ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Copy the go source
COPY constants.go constants.go
COPY cmd/ cmd/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -ldflags="-w -s" -a -o pvc-autoresizer cmd/*.go

# Stage2: setup runtime container
FROM scratch
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workspace/pvc-autoresizer .
EXPOSE 8080
USER 10000:10000

ENTRYPOINT ["/pvc-autoresizer"]
