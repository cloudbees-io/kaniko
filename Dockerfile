FROM golang:1.26.5-alpine3.24 AS build
WORKDIR /work
COPY go.mod* go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o kaniko-action main.go

FROM 020229604682.dkr.ecr.us-east-1.amazonaws.com/actions/chainguard-dev-kaniko-base:1.25.16-7872282270c360d3e60875c8b0795173a3207dfb-49
COPY --from=build /work/kaniko-action /kaniko/cloudbees-kaniko-action
ENTRYPOINT ["/kaniko/cloudbees-kaniko-action"]
