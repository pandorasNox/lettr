FROM node:22.15.0-alpine3.20@sha256:686b8892b69879ef5bfd6047589666933508f9a5451c67320df3070ba0e9807b AS node

# =============================================================

FROM golang:1.24.3-alpine@sha256:ef18ee7117463ac1055f5a370ed18b8750f01589f13ea0b48642f5792b234044 AS builder-and-dev

ARG GIT_REVISION=${GIT_REVISION}
ENV GIT_REVISION=${GIT_REVISION}

ENV WORKDIR=/workdir
WORKDIR ${WORKDIR}

RUN apk add --no-cache \
    git \
    tmux \
    || apk update && apk upgrade

ENV AIR_VERSION=1.51.0
RUN go install github.com/cosmtrek/air@v${AIR_VERSION}

# in root
RUN mkdir -p /scripts
COPY ./container-images/app/tmux.sh /scripts/tmux.sh

# in workdir
RUN mkdir -p "${WORKDIR}/scripts"
COPY ./scripts/tailwind_build.sh "${WORKDIR}/scripts/tailwind_build.sh"

# Copy Nodejs with dependencies from node image
COPY --from=node /usr/lib /usr/lib
COPY --from=node /usr/local/share /usr/local/share
COPY --from=node /usr/local/lib /usr/local/lib
COPY --from=node /usr/local/include /usr/local/include
COPY --from=node /usr/local/bin /usr/local/bin

COPY ./web/package*.json ${WORKDIR}/web/
RUN set -eu; cd ${WORKDIR}/web/; npm install; cd ${WORKDIR}

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./pkg ./pkg
RUN go mod download

COPY ./main.go ./main.go
COPY ./configs ./configs
COPY ./web ./web

RUN set -eu; \
    cd ${WORKDIR}/web/; \
    ./../scripts/tailwind_build.sh; \
    npx tsc --project app/tsconfig.json; \
    cd ${WORKDIR}; \
    go build -buildvcs=false -o /tmp/lettr -ldflags="-X 'main.Revision=${GIT_REVISION}'";

ENTRYPOINT ["/usr/bin/env", "ash"]

# =============================================================

FROM builder-and-dev AS tester

RUN go test -v ./...

# =============================================================

FROM alpine:3.21@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS prod

COPY --from=builder-and-dev /tmp/lettr /usr/local/bin/

#CMD ["lettr"]
ENTRYPOINT ["lettr"]
