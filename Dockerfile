FROM golang:1.26.0-alpine3.22 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:1.25.11-15dd46da0d98501c53ac10c0a6d849fd2ab0b375-42
COPY --from=build /work/kaniko-action /kaniko/cloudbees-kaniko-action
ENTRYPOINT ["/kaniko/cloudbees-kaniko-action"]
