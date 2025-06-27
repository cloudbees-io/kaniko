# ---- Stage 1: Build binaries ----
FROM golang:1.24 AS builder
WORKDIR /src

ARG TARGETARCH
ARG TARGETOS
ENV GOARCH=$TARGETARCH
ENV GOOS=$TARGETOS
ENV CGO_ENABLED=0
ENV GOBIN=/usr/local/bin

# Required for docker config
RUN mkdir -p /kaniko/.docker

# Copy all source files
COPY . .

# Install Docker credential helpers
RUN go install github.com/GoogleCloudPlatform/docker-credential-gcr@v2.0.0
RUN go install github.com/awslabs/amazon-ecr-credential-helper/...@v0.7.0
RUN go install github.com/chrismellard/docker-credential-acr-env@latest

# Build Kaniko binaries
RUN \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  make out/executor out/warmer

# ---- Stage 2: Get CA certificates ----
FROM debian:bookworm-slim AS certs
RUN apt update && apt install -y ca-certificates

# ---- Stage 3: Busybox for debugging ----
FROM busybox:musl AS busybox

# ---- Stage 4: Base scratch image ----
FROM scratch AS kaniko-base-slim

# Create required folder with permissions using busybox
RUN --mount=from=busybox,dst=/usr/ ["busybox", "sh", "-c", "mkdir -p /kaniko && chmod 777 /kaniko"]

# Copy CA certs
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /kaniko/ssl/certs/

# Required files
COPY files/nsswitch.conf /etc/nsswitch.conf

ENV HOME=/root
ENV USER=root
ENV PATH=/usr/local/bin:/kaniko
ENV SSL_CERT_DIR=/kaniko/ssl/certs

# ---- Stage 5: Base Kaniko image ----
FROM kaniko-base-slim AS kaniko-base

COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-gcr /kaniko/docker-credential-gcr
COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-ecr-login /kaniko/docker-credential-ecr-login
COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-acr-env /kaniko/docker-credential-acr-env

COPY --from=builder /kaniko/.docker /kaniko/.docker

ENV DOCKER_CONFIG=/kaniko/.docker/
ENV DOCKER_CREDENTIAL_GCR_CONFIG=/kaniko/.config/gcloud/docker_credential_gcr_config.json
WORKDIR /workspace

# ---- Stage 6: Warmer ----
FROM kaniko-base AS kaniko-warmer
COPY --from=builder /src/out/warmer /kaniko/warmer
ENTRYPOINT ["/kaniko/warmer"]

# ---- Stage 7: Executor ----
FROM kaniko-base AS kaniko-executor
COPY --from=builder /src/out/executor /kaniko/executor
ENTRYPOINT ["/kaniko/executor"]

# ---- Stage 8: Debug ----
FROM kaniko-executor AS kaniko-debug
ENV PATH=/usr/local/bin:/kaniko:/busybox

COPY --from=builder /src/out/warmer /kaniko/warmer
COPY --from=busybox /bin /busybox

# Add /busybox to PATH and link shell
VOLUME /busybox
RUN ["/busybox/mkdir", "-p", "/bin"]
RUN ["/busybox/ln", "-s", "/busybox/sh", "/bin/sh"]

# ---- Stage 9: Slim ----
FROM kaniko-base-slim AS kaniko-slim
COPY --from=builder /src/out/executor /kaniko/executor
ENTRYPOINT ["/kaniko/executor"]
