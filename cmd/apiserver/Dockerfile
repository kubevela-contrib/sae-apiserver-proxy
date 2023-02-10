ARG BASE_IMAGE
# Build the manager binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.19-alpine as builder
ARG GOPROXY
ENV GOPROXY=${GOPROXY:-https://proxy.golang.org}
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/apiserver/main.go cmd/apiserver/main.go
COPY pkg/ pkg/

# Build
ARG TARGETARCH

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -a -ldflags "-s -w" \
    -o sae-apiserver-proxy-${TARGETARCH} cmd/apiserver/main.go

# Before copying the Go binary directly to the final image,
# add them to the intermdediate upx image
FROM gruebel/upx:latest as upx
ARG TARGETARCH
COPY --from=builder /workspace/sae-apiserver-proxy-${TARGETARCH} /workspace/sae-apiserver-proxy-${TARGETARCH}
# Compress the binary and copy it to final image
RUN upx --best --lzma -o /workspace/sae-apiserver-proxy-${TARGETARCH}-upx /workspace/sae-apiserver-proxy-${TARGETARCH}

# Overwrite `BASE_IMAGE` by passing `--build-arg=BASE_IMAGE=gcr.io/distroless/static:nonroot`

FROM ${BASE_IMAGE:-alpine:3.15}
# This is required by daemon connnecting with cri
RUN apk add --no-cache ca-certificates bash expat
RUN apk add curl

WORKDIR /

ARG TARGETARCH
COPY --from=upx /workspace/sae-apiserver-proxy-${TARGETARCH}-upx /usr/local/bin/sae-apiserver-proxy

CMD ["sae-apiserver-proxy"]
