# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.26

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
ARG TARGETOS TARGETARCH
# CI passes the resolved git tag (e.g. 1.0.0) via build-arg; a plain
# `docker build .` defaults to "dev".
ARG VERSION=dev
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
  go build -trimpath -ldflags="-s -w" \
  -o /out/product-monitor ./cmd/product-monitor

# Staged empty data dir, copied below with nonroot ownership so a fresh mounted
# volume inherits it and the nonroot process can create the SQLite DB.
RUN mkdir -p /data

# -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.version=$VERSION

WORKDIR /app
COPY --from=builder /out/product-monitor /app/product-monitor

# SQLite database lives on a mounted volume (see Helm chart / -v /data). The dir
# is nonroot-owned so a freshly created volume is writable by the app user.
COPY --from=builder --chown=nonroot:nonroot /data /data
VOLUME ["/data"]

WORKDIR /data

USER nonroot:nonroot
ENTRYPOINT ["/app/product-monitor"]
