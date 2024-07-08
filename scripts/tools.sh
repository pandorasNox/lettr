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

# load .env file, if exists

# if ! test -f ".env"; then
#   echo ".env file expected but none found" 1>&2
#   exit 1
# fi

if test -f ".env"; then
  echo "loading .env file"
  set -o allexport;
  . ./.env; #source file
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

#   up                ...
#   down              ...
__usage="
Usage: $(basename $0) [OPTIONS]

Options:
  --help|-h         show help
  watch             start go server & reload server upon file chnages
  test              start go docker container + run go tests
  down              stop + delete all started local docker container
"

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

  docker exec -it ${CONTAINER_NAME} ash
}

func_setup() {
  (
    if (! test -f ".env") && test -f ".env.template" ; then
      cp .env.template .env
    fi
  )
}

func_build_img() {
  # handle flags, see https://stackoverflow.com/a/22395652
  while test $# -gt 0 ; do
    # options with arguments
    case "$1" in
    --img-name=*) IMG_NAME="${1##--img-name=}" ; shift; continue; break ;;
    --target=*) TARGET="${1##--target=}" ; shift; continue; break ;;
    esac

    # unknown - up to you - positional argument or error?
    echo "Unknown option $1"
    shift
  done

  IMG_NAME=${IMG_NAME:?"--img-name flag is missing, usage: --img-name=a-name"}
  TARGET=${TARGET:?"--target flag is missing, usage: --target=a-build-target"}

  docker build \
    -t "${IMG_NAME}" \
    -f container-images/app/Dockerfile \
    --target "${TARGET}" \
    --build-arg "GIT_REVISION=$(git rev-parse --verify --short HEAD)" \
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
      --name ${CONTAINER_NAME} \
      -w "/workdir" \
      -v "${PWD}":"/workdir" \
      --entrypoint=ash \
      ${IMG_NAME} -c "while true; do sleep 2000000; done"
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
    --entrypoint=ash \
    "${DEVTOOLS_IMG_NAME}" -c "cd ./web/; npm install; cd ..; air --build.cmd 'cd ./web/ && npx tailwindcss --config app/tailwind.config.js --input app/css/input.css --output static/generated/output.css && npx tsc --project app/tsconfig.json && cd .. && go build -buildvcs=false -ldflags=\"-X 'main.Revision=$(git rev-parse --verify --short HEAD)' -X 'main.FaviconPath=/static/assets/favicon_dev'\" -o ./tmp/main' --build.bin './tmp/main' -build.include_ext 'go,tpl,tmpl,templ,html,js,ts,json,png,ico,webmanifest' -build.exclude_dir 'assets,tmp,vendor,web/node_modules,web/static/generated'"
}

func_exec_cli() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t ${CONTAINER_NAME} ash -c "$@"
}

func_down() {
  docker stop -t1 "${CLI_CONTAINER_NAME}" || true # ' || true ' for "No such container: lettr_cli_con" error (ignore if not exists)
  docker compose down
}

func_skopeo_cli() {
  docker run -it --rm --entrypoint=bash quay.io/skopeo/stable:v1.14.2
}

func_typescript_build() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t ${CONTAINER_NAME} ash -ce "cd ./web/; npm install; npx tsc --project app/tsconfig.json;"
}

func_deploy() {
  fly deploy --build-arg "GIT_REVISION=$(git rev-parse --verify --short HEAD)"
}

# -----------------------------------------------------------------------------

if [ -z "$*" ]
then
  echo "$__usage"
else
    if [ $1 == "--help" ] || [ $1 == "-h" ]
    then
        echo "$__usage"
    fi

    if [ $1 == "cli" ]
    then
      func_cli
    fi

    if [ $1 == "setup" ]
    then
      func_setup
    fi

    if [ $1 == "watch" ]
    then
      func_watch
    fi

    if [ $1 == "test" ]
    then
      func_exec_cli "go test ."
    fi

    if [ $1 == "bench" ]
    then
      func_exec_cli "go test -bench=. -run=^$ -cpu=1 -benchmem -count=10"
    fi

    if [ $1 == "down" ]
    then
      func_down
    fi

    if [ $1 == "skocli" ]
    then
      func_skopeo_cli
    fi

    if [ $1 == "img" ]
    then
      func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;
      func_build_img --img-name="${PROD_IMG_NAME}" --target=prod;
    fi

    if [ $1 == "tsc" ]
    then
      func_typescript_build
    fi

    if [ $1 == "prod" ]
    then
      func_start_prod
    fi

    if [ $1 == "deploy" ]
    then
      func_deploy
    fi
fi
