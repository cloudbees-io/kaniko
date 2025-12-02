FROM golang:1.25.4-alpine3.22 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:1.25.0-a25793dbe083d453dfcec2511c5d4085748dbdb7-27
COPY --from=build /work/kaniko-action /usr/local/bin/kaniko-action
ENTRYPOINT ["/usr/local/bin/kaniko-action"]
