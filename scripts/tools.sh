#!/usr/bin/env bash

set -o errexit
set -o nounset
# set -o xtrace

if set +o | grep -F 'set +o pipefail' > /dev/null; then
  # shellcheck disable=SC3040
  set -o pipefail
fi

if set +o | grep -F 'set +o posix' > /dev/null; then
  # shellcheck disable=SC3040
  set -o posix
fi

# -----------------------------------------------------------------------------

#SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SCRIPT_DIR=$(dirname "$0"); SCRIPT_DIR=$(eval "cd \"${SCRIPT_DIR}\" && pwd")
echo "SCRIPT_DIR: ${SCRIPT_DIR}"

# -----------------------------------------------------------------------------

# load .env file, if exists

# if ! test -f ".env"; then
#   echo ".env file expected but none found" 1>&2
#   exit 1
# fi

if test -f ".env"; then
  echo "loading .env file"
  set -o allexport;
  # shellcheck disable=SC1091 # 'shellcheck source=.env' won't work herre for us as .env won't be available in ci
  . "${SCRIPT_DIR}/../.env"; #source file
  set +o allexport;
fi

# -----------------------------------------------------------------------------
# check expected env vars (likely expected from .env file)

func_check_expected_env_vars() {
  if test "${GITHUB_TOKEN:-}" = ""; then
    echo "WARNING: GITHUB_TOKEN expected, but is empty (not in .env file for local dev?)"
  fi
}

func_check_expected_env_vars

# -----------------------------------------------------------------------------

APP_PORT=9026;

# -----------------------------------------------------------------------------

DEVTOOLS_IMG_NAME=lettr_dev_tools
PROD_IMG_NAME=lettr_prod
CLI_CONTAINER_NAME=lettr_cli_con

func_cli() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -it "${CONTAINER_NAME}" ash
}

func_setup() {
  (
    # shellcheck disable=SC2235
    if (! test -f ".env") && test -f ".env.template" ; then
      cp .env.template .env
    fi
  )
}

func_gofmt() {(
  set -Eeuo pipefail;

  echo "🚗 start gofmt";

  find . -name "*.go" -print0 | xargs -0 gofmt -l -s -w;

  echo "🏁 done gofmt";
)}

func_build_img() {
  PROGRESS=

  # handle flags, see https://stackoverflow.com/a/22395652
  while test $# -gt 0 ; do
    # options with arguments
    case "$1" in
    --img-name=*) IMG_NAME="${1##--img-name=}" ; shift; continue; break ;;
    --target=*) TARGET="${1##--target=}" ; shift; continue; break ;;
    --progress=*) PROGRESS="${1##--progress=}" ; shift; continue; break ;;
    esac

    # unknown - up to you - positional argument or error?
    echo "Unknown option $1"
    shift
  done

  IMG_NAME=${IMG_NAME:?"--img-name flag is missing, usage: --img-name=a-name"}
  TARGET=${TARGET:?"--target flag is missing, usage: --target=a-build-target"}

  _progressArgs=""
  if test -n "${PROGRESS}"; then
    _progressArgs="--progress=${PROGRESS}"
  fi

  # shellcheck disable=SC2086
  docker build \
    -t "${IMG_NAME}" \
    -f container-images/app/Dockerfile \
    --target "${TARGET}" \
    --build-arg "GIT_REVISION=$(git rev-parse --verify --short HEAD)" \
    ${_progressArgs} \
    .

  printf '%s' "${IMG_NAME}"
}

func_start_prod() {
    func_build_img --img-name="${PROD_IMG_NAME}" --target=prod;

    docker run --rm \
      -e PORT="9033" \
      -p 9033:9033 \
      ${PROD_IMG_NAME}
}

func_start_idle_container() {
  IMG_NAME=${1:?"first param missing, which is expected to be a chosen image name"}
  CONTAINER_NAME=${2:?"second param missing, which is expected to be a chosen container name"}

  if ! (docker ps --format "{{.Names}}" | grep "${CONTAINER_NAME}"); then
    docker run -d --rm \
      --name "${CONTAINER_NAME}" \
      --user "$(id -u):$(id -g)" \
      -w "/workdir" \
      -v "${PWD}":"/workdir" \
      --entrypoint=ash \
      "${IMG_NAME}" -c "while true; do sleep 2000000; done"
      # -v "${PWD}/tmp/local_go_dev_dir":"/go" \
  fi
}

func_watch() {
  func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

  docker run -it --rm \
    -w "/workdir" \
    -v "${PWD}":"/workdir" \
    -p "${APP_PORT}":"${APP_PORT}" \
    -e PORT="${APP_PORT}" \
    -e GITHUB_TOKEN="${GITHUB_TOKEN:?"No github token via GITHUB_TOKEN env variable provided"}" \
    -e IMPRINT_URL="${IMPRINT_URL:-}" \
    --entrypoint=ash \
    "${DEVTOOLS_IMG_NAME}" -c "cd ./web/; npm install; cd ..; air --build.cmd 'cd ./web/ && npx @tailwindcss/cli --input app/css/input.css --output static/generated/output.css && ./node_modules/.bin/esbuild app/main.ts --tsconfig=app/tsconfig.json --bundle --minify --sourcemap --outfile=static/generated/main.js && cd .. && go build -buildvcs=false -ldflags=\"-X 'main.Revision=$(git rev-parse --verify --short HEAD)' -X 'main.FaviconPath=/static/assets/favicon_dev'\" -o ./tmp/main' --build.bin './tmp/main' -build.include_ext 'go,tpl,tmpl,templ,html,js,ts,css,json,png,ico,webmanifest' -build.exclude_dir 'assets,tmp,vendor,web/node_modules,web/static/generated'"
}

func_exec_cli() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t "${CONTAINER_NAME}" ash -c "$@"
}

func_down() {
  docker stop -t1 "${CLI_CONTAINER_NAME}" || true # ' || true ' for "No such container: lettr_cli_con" error (ignore if not exists)
  docker compose down

  docker compose --file tests/playwright/playwright.docker-compose.yml down || true;
  docker compose --file tests/playwright/playwright.docker-compose.yml \
    --file tests/playwright/playwright-ui-patch.docker-compose.yml down \
    || true \
  ;
}

func_skopeo_cli() {
  # renovate: datasource=docker
  SKOPEO_CONTAINER_IMAGE=quay.io/skopeo/stable:v1.19.0@sha256:d08fe48978c027ff0f5eeaeb4b1c12c12612427a559f3e49f576f0109dfbfca4;
  docker run -it --rm --entrypoint=bash "${SKOPEO_CONTAINER_IMAGE}"
}

func_typescript_build() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t "${CONTAINER_NAME}" ash -ce "cd ./web/; npm install; ./node_modules/.bin/esbuild app/main.ts --tsconfig=app/tsconfig.json --bundle --minify --sourcemap --outfile=static/generated/main.js;"
}

func_tailwind_build() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t "${CONTAINER_NAME}" ash -ce "cd ./web/; npm install; npx @tailwindcss/cli --input app/css/input.css --output static/generated/output.css;"
}

func_deploy() {
  fly deploy --build-arg "GIT_REVISION=$(git rev-parse --verify --short HEAD)"
}

func_check() {
  echo "run func_check";
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t "${CONTAINER_NAME}" ash -ce \
    "cd ./web/; npm install > /dev/null; ./../scripts/tailwind_build.sh" \
  ;
}

func_golangci_lint() {(
  echo "run golangci-lint";

  # renovate: datasource=docker
  GOLANGCI_LINT_CONTAINER_IMAGE=golangci/golangci-lint:v2.4.0-alpine@sha256:a93d021e12afdb31b11a3d2dab39cfc45b2ec950977029ffed636e2098cb784c;

  _fix=false
  # handle flags, see https://stackoverflow.com/a/22395652
  while test $# -gt 0 ; do
    if test "$1" = "--fix" ; then _fix=true ; shift ; continue; fi
    echo "Unknown option $1" && exit 1
    shift
  done

  _golangciLintExtraArgs=""
  if test "${_fix}" = "true"; then
    _golangciLintExtraArgs="${_golangciLintExtraArgs} --fix"
  fi

  docker run -t --rm \
    --entrypoint=ash \
    -w /workdir \
    -v "$(pwd)":/workdir \
    -v golanglint-go-build-cache-vol:/root/.cache/go-build \
    -v golanglint-go-root-vol:/usr/local/go \
    -v golanglint-go-mod-cache-vol:/go/pkg/mod \
    -v golanglint-lint-cache-vol:/root/.cache/golangci-lint \
    "${GOLANGCI_LINT_CONTAINER_IMAGE}" \
    -ce "golangci-lint run ${_golangciLintExtraArgs} --config ./.golangci.yml -v" \
  ;
)}

func_shellcheck() {(
  "${SCRIPT_DIR}/checks/shellcheck.sh"
)}

func_shellcheck_fix() {(
  "${SCRIPT_DIR}/checks/shellcheck.sh" --fix
)}

func_renovate() {(

  # renovate: datasource=docker
  CONTAINER_IMAGE=docker.io/renovate/renovate:41.66.2@sha256:d9965278f5bb202c67e20507aa118665657287a4c1e46fd00d51891ae97a21af;

  # note:
  #   * in regards to --platform=local
  #     * see https://docs.renovatebot.com/modules/platform/local/
  #   * in regards to --dry-run
  #     * see: https://docs.renovatebot.com/self-hosted-configuration/#dryrun
  #     * as time of writing: `--platform=local` makes `--dry-run=lookup`` default
  #     * for better debugging to e.g. see what branches renovate would create see `--dry-run=full`
  #       * as time of writing: "'full': Performs a dry run by logging messages instead of creating/updating/deleting branches and PRs"
  docker run --env LOG_LEVEL=debug --rm --user "$(id -u):$(id -g)" --volume "${PWD:?}:${PWD:?}:ro" \
    --workdir "${PWD:?}" "${CONTAINER_IMAGE}" \
    --platform=local \
    --dry-run=lookup \
    --enabled-managers=custom.regex

  # example:
  #   --platform=local \
  #   --dry-run=full \

)}

func_playwright() {(
  ./tests/playwright/playwright.sh;
)}

func_playwright_ui() {(
  docker compose \
    --file tests/playwright/playwright.docker-compose.yml \
    --file tests/playwright/playwright-ui-patch.docker-compose.yml \
    up --build --force-recreate \
  ;
)}

# -----------------------------------------------------------------------------

#   up                ...
#   down              ...
__usage="
Usage: $(basename "$0") [OPTIONS]

Options:
  --help|-h         show help
  bench             start go docker container + run go bench tests
  check             run preflight checks
  cli               start container + exec into
  deploy            deploy app via fly cli-tool (flyctl) to fly.io (expects fl/flyctl clt-tool to be installed)
  down              stop + delete all started local docker container
  fmt               run gofmt across repo files
  img               build all container images
  lint              run golangci-lint via docker container
  prod              build prod container image + start container running on extra port (should be printed after start)
  renovate          run renovate in local mode via container
  shellcheck        run shellcheck via container
  shellcheck-fix    run shellcheck with format via container + apply found changes/diff
  setup             setup .env file
  skocli            via container provide skopeo tooling + exec into
  tailwind|twind    via container with npm/npx executing tailwind cli-tool to build css assets
  test              start go docker container + run go tests
  tsc               via container with npm/npx executing typescript cli-tool to build javascript assets
  watch             via container start go server & reload server upon file chnages
"

# -----------------------------------------------------------------------------

if [ -z "$*" ]
then
  echo "${__usage}"
else
    if [ "$1" == "--help" ] || [ "$1" == "-h" ]
    then
      echo "${__usage}"
      exit 0;
    fi

    if [ "$1" == "check" ]
    then
      func_check
      exit 0;
    fi

    if [ "$1" == "cli" ]
    then
      func_cli
      exit 0;
    fi

    if [ "$1" == "setup" ]
    then
      func_setup
      exit 0;
    fi

    if [ "$1" == "fmt" ]
    then
      func_gofmt
      exit 0;
    fi

    if [ "$1" == "watch" ]
    then
      func_watch
      exit 0;
    fi

    if [ "$1" == "test" ]
    then
      # func_exec_cli "go test -v ."
      # func_exec_cli "go test -v ./..."
      # go test -run Test_HandleSession ./pkg/session
      func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=tester --progress=plain;
      exit 0;
    fi

    if [ "$1" == "bench" ]
    then
      func_exec_cli "go test -bench=. -run=^$ -cpu=1 -benchmem -count=10"
      exit 0;
    fi

    if [ "$1" == "down" ]
    then
      func_down
      exit 0;
    fi

    if [ "$1" == "skocli" ]
    then
      func_skopeo_cli
      exit 0;
    fi

    if [ "$1" == "img" ]
    then
      func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;
      func_build_img --img-name="${PROD_IMG_NAME}" --target=prod;
      exit 0;
    fi

    if [ "$1" == "lint" ]
    then
      func_golangci_lint;
      func_exec_cli "cd web; npx eslint --config='./eslint.config.mjs' app/;"
      exit 0;
    fi

    if [ "$1" == "lint-fix" ]
    then
      func_golangci_lint --fix
      func_exec_cli "cd web; npx eslint --config='./eslint.config.mjs' --fix app/;"
      exit 0;
    fi

    if [ "$1" == "shellcheck" ]
    then
      func_shellcheck;
      exit 0;
    fi

    if [ "$1" == "shellcheck-fix" ]
    then
      func_shellcheck_fix;
      exit 0;
    fi

    if [ "$1" == "tsc" ]
    then
      func_typescript_build
      exit 0;
    fi

    if [ "$1" == "twind" ] || [ "$1" == "tailwind" ]
    then
      func_tailwind_build
      exit 0;
    fi

    if [ "$1" == "prod" ]
    then
      func_start_prod
      exit 0;
    fi

    if [ "$1" == "deploy" ]
    then
      func_deploy
      exit 0;
    fi

    if [ "$1" == "renovate" ]
    then
      func_renovate
      exit 0;
    fi

    if [ "$1" == "playwright" ]
    then
      func_playwright;
      exit 0;
    fi

    if [ "$1" == "playwright-ui" ]
    then
      func_playwright_ui;
      exit 0;
    fi

    if [ "$1" != "" ]
    then
      echo "error: nor argument provided"

      echo "${__usage}"
    
      exit 1;
    fi
fi
