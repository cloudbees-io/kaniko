FROM golang:1.20.5-alpine3.18 AS build

WORKDIR /work

COPY go.mod* go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /usr/local/bin/kaniko-action main.go

FROM gcr.io/kaniko-project/executor:v1.11.0 AS kaniko

FROM alpine:latest

COPY --from=build /usr/local/bin/kaniko-action /usr/local/bin/kaniko-action

COPY --from=kaniko /kaniko/executor /kaniko/executor

RUN mkdir -p /kaniko/.docker

ENV PATH="${PATH}:/kaniko/"

ENTRYPOINT ["kaniko-action"]