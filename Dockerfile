FROM golang:1.22.1-alpine3.19 AS build

WORKDIR /work

COPY go.mod* go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /usr/local/bin/kaniko-action main.go

FROM gcr.io/kaniko-project/executor:v1.22.0

COPY --from=build /usr/local/bin/kaniko-action /usr/local/bin/kaniko-action

ENTRYPOINT ["kaniko-action"]
