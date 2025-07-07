FROM golang:1.24.0-alpine3.21 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:latest
COPY --from=build /work/kaniko-action /kaniko/kaniko-action
ENTRYPOINT ["/kaniko/kaniko-action"]
