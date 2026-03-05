FROM golang:1.26.0-alpine3.22 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:1.25.11-1b6ad2870b96098d39e01d2a69564c6fee4c1396-43
COPY --from=build /work/kaniko-action /kaniko/cloudbees-kaniko-action
ENTRYPOINT ["/kaniko/cloudbees-kaniko-action"]
