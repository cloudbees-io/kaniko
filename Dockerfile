FROM golang:1.26.2-alpine3.22 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:1.25.12-7976b4c476e47be2da6de19bf5c35fb35a1c5ddc-45
COPY --from=build /work/kaniko-action /kaniko/cloudbees-kaniko-action
ENTRYPOINT ["/kaniko/cloudbees-kaniko-action"]
