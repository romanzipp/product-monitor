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
      -o /out/portasplit-monitor ./cmd/portasplit-monitor

# -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.version=$VERSION

WORKDIR /app
COPY --from=builder /out/portasplit-monitor /app/portasplit-monitor

# SQLite database lives on a mounted volume (see Helm chart / -v /data).
ENV DB_PATH=/data/portasplit-monitor.db
VOLUME ["/data"]

USER nonroot:nonroot
ENTRYPOINT ["/app/portasplit-monitor"]
