FROM node:22.17.0-alpine3.22@sha256:5340cbfc2df14331ab021555fdd9f83f072ce811488e705b0e736b11adeec4bb AS node

# =============================================================

FROM golang:1.24.4-alpine@sha256:68932fa6d4d4059845c8f40ad7e654e626f3ebd3706eef7846f319293ab5cb7a AS builder-and-dev

ARG GIT_REVISION=${GIT_REVISION}
ENV GIT_REVISION=${GIT_REVISION}

ENV WORKDIR=/workdir
WORKDIR ${WORKDIR}

RUN apk add --no-cache \
    bats \
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
RUN set -eu; \
    cd ${WORKDIR}/web/; \
    npm install;

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./pkg ./pkg
RUN go mod download

COPY ./main.go ./main.go
COPY ./main_test.go ./main_test.go
COPY ./configs ./configs
COPY ./web ./web

RUN set -eu; \
    cd ${WORKDIR}/web/; \
    ./../scripts/tailwind_build.sh;
RUN set -eu; \
    cd ${WORKDIR}/web/; \
    ./node_modules/.bin/esbuild app/main.ts --tsconfig=app/tsconfig.json --bundle --minify --outfile=static/generated/main.js;
RUN set -eu; \
    cd ${WORKDIR}; \
    go build -buildvcs=false -o /tmp/lettr -ldflags="-X 'main.Revision=${GIT_REVISION}'";

ENTRYPOINT ["/usr/bin/env", "ash"]

# =============================================================

FROM builder-and-dev AS tester

COPY ./scripts/checks "${WORKDIR}/scripts/checks"

RUN set -e; \
    echo "◌ start running bats tests"; \
    bats scripts/checks/bats; \
    echo "√ done running bats tests";

RUN set -eu; \
        cd ${WORKDIR}/web/; \
        npx eslint --config="./eslint.config.mjs" app/ ;

RUN go test -v ./...

# =============================================================

FROM alpine:3.22@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715 AS prod

COPY --from=builder-and-dev /tmp/lettr /usr/local/bin/

#CMD ["lettr"]
ENTRYPOINT ["lettr"]
