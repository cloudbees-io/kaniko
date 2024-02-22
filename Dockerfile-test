FROM alpine:3.19 AS build
RUN echo test > /test-file

FROM docker.io/rancher/cowsay AS cowsay
RUN cowsay hello

FROM quay.io/containers/podman:v4.9.0 as podman

FROM alpine:3.19
RUN echo test
COPY --from=build /test-file /test-file
COPY --from=cowsay /usr/local/bin/cowsay /usr/local/bin/cowsay
COPY --from=podman /usr/bin/crun /usr/local/bin/crun
