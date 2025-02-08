FROM node:20.12.1-alpine3.19 AS node


FROM golang:1.22.6-alpine AS builder-and-dev

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

RUN mkdir -p /scripts
COPY ./container-images/app/tmux.sh /scripts/tmux.sh

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
    npx tailwindcss --config app/tailwind.config.js --input app/css/input.css --output static/generated/output.css; \
    npx tsc --project app/tsconfig.json; \
    cd ${WORKDIR}; \
    go build -buildvcs=false -o /tmp/lettr -ldflags="-X 'main.Revision=${GIT_REVISION}'";

ENTRYPOINT ["/usr/bin/env", "ash"]

# =============================================================

FROM alpine:3.19 AS prod

COPY --from=builder-and-dev /tmp/lettr /usr/local/bin/

#CMD ["lettr"]
ENTRYPOINT ["lettr"]
