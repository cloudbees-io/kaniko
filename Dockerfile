# Stage 1: Build custom Kaniko wrapper (optional)
FROM golang:1.24.0-alpine3.21 AS build
WORKDIR /work
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo \
    -ldflags='-w -extldflags "-static"' \
    -o kaniko-action main.go

# Stage 2: Use Chainguard's maintained Kaniko image
FROM cgr.dev/chainguard/kaniko:latest

# OR for FIPS version:
# FROM cgr.dev/chainguard/kaniko-fips:latest

# Copy your custom binary into the image
COPY --from=build /work/kaniko-action /kaniko/kaniko-action

# Set entrypoint to your wrapper (or fallback to kaniko-executor)
ENTRYPOINT ["/kaniko/kaniko-action"]
# Alternatively:
# ENTRYPOINT ["/kaniko/executor"]
