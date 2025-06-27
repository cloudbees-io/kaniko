# Build the custom Go binary
FROM golang:1.24.0-alpine3.21 AS build

WORKDIR /work

# Install git (needed for `go mod download` with some dependencies)
RUN apk add --no-cache git

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

# Final Kaniko executor image (from the archived but working GCR source)
FROM gcr.io/kaniko-project/executor:v1.23.2

# Copy custom binary into Kaniko image
COPY --from=build /work/kaniko-action /kaniko/kaniko-action

# Set entrypoint to the custom binary
ENTRYPOINT ["/kaniko/kaniko-action"]
