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

func_gofmt() {(
  set -Eeuo pipefail;

  echo "🚗 start gofmt";

  find . -name "*.go" | xargs gofmt -l -s -w;

  echo "🏁 done gofmt";
)}

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
    -e IMPRINT_URL="${IMPRINT_URL:-}" \
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

func_tailwind_build() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t ${CONTAINER_NAME} ash -ce "cd ./web/; npm install; npx tailwindcss --config app/tailwind.config.js --input app/css/input.css --output static/generated/output.css;"
}

func_deploy() {
  fly deploy --build-arg "GIT_REVISION=$(git rev-parse --verify --short HEAD)"
}

func_prepend() {(
  # echo "debug: func_prepend start" >&2;
  _charAmount=${1:?'first argument should be amount of spaces'};
  _char=${2:?'second argument should be a single character to repeat+prepend text with'};
  _inputText=${3:?'second argument should be input text'};

  # check _charAmount is positve integer expression
  test "${_charAmount}" -eq "${_charAmount#-}" || (echo "error: first argument should be positve integer"; exit 1;);

  # echo "debug: func_prepend => _inputText (before fill): ${_inputText}";
  # _fill=$(printf ' %.0s' {1..${_charAmount}});
  _fill=$( for ((i = 0; i < _charAmount; i++)); do printf "${_char}" ''; done );
  # _fill="        ";
  # printf '%s' "$(printf '%s' "${_inputText}" | awk "{print '${_fill}' $0}" )";
  _res="$( printf '%s' "${_inputText}" | awk "{print \"${_fill}\" \$0}" )";
  # echo "debug: func_prepend => _res (after fill): ${_res}" >&2;

  printf '%s' "${_res}";

  # echo "debug: func_prepend end" >&2
)}
export func_prepend;

func_check() {
  echo "run func_check";
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;

    func_start_idle_container "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  _tw_stdout=$(docker exec -t ${CONTAINER_NAME} ash -ce \
    "cd ./web/; npm install; npx tailwindcss --config app/tailwind.config.js --input app/css/input.css --output static/generated/output.css;" \
  );
    # | od -xc \


  echo "before debug";
  # echo "${_tw_stdout}" | grep "warn - No utility classes were detected in your source files";
  # echo "${_tw_stdout}" | grep "w   a   r   n"
  # echo "exit code: $?";
  
  # echo "DEBUG tailwind_out: \n${_tw_stdout}";
  
  # if !(echo "${_tw_stdout}" | grep "warn - No utility classes were detected in your source files"); then
  # if (echo "${_tw_stdout}" | grep "w   a   r   n" > /dev/null); then
  # if (echo "${_tw_stdout}" | grep "warn - No utility classes were detected in your source files" > /dev/null); then
  if ( echo "${_tw_stdout}" | grep "warn" | grep "No utility classes were detected in your source files" > /dev/null ); then
    echo "debug inside if";
    printf 'error: found unwamnted warning in tailwind output\n  tailwindoutput:\n%s\n' "$( func_prepend "6" " " "$(func_prepend "4"  ">" "${_tw_stdout}" )" ) ";
    echo "error: found unwamnted warning in tailwind output (see above)"
    exit 1;
  fi
}

# -----------------------------------------------------------------------------

#   up                ...
#   down              ...
__usage="
Usage: $(basename $0) [OPTIONS]

Options:
  --help|-h         show help
  bench             start go docker container + run go bench tests
  check             run preflight checks
  cli               start container + exec into
  deploy            deploy app via fly cli-tool (flyctl) to fly.io (expects fl/flyctl clt-tool to be installed)
  down              stop + delete all started local docker container
  fmt               run gofmt across repo files
  img               build all container images
  prod              build prod container image + start container running on extra port (should be printed after start)
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
  echo "$__usage"
else
    if [ $1 == "--help" ] || [ $1 == "-h" ]
    then
      echo "$__usage"
      exit 0;
    fi

    if [ $1 == "check" ]
    then
      func_check
      exit 0;
    fi

    if [ $1 == "cli" ]
    then
      func_cli
      exit 0;
    fi

    if [ $1 == "setup" ]
    then
      func_setup
      exit 0;
    fi

    if [ $1 == "fmt" ]
    then
      func_gofmt
      exit 0;
    fi

    if [ $1 == "watch" ]
    then
      func_watch
      exit 0;
    fi

    if [ $1 == "test" ]
    then
      # func_exec_cli "go test -v ."
      func_exec_cli "go test -v ./..."
      # go test -run Test_HandleSession ./pkg/session
      exit 0;
    fi

    if [ $1 == "bench" ]
    then
      func_exec_cli "go test -bench=. -run=^$ -cpu=1 -benchmem -count=10"
      exit 0;
    fi

    if [ $1 == "down" ]
    then
      func_down
      exit 0;
    fi

    if [ $1 == "skocli" ]
    then
      func_skopeo_cli
      exit 0;
    fi

    if [ $1 == "img" ]
    then
      func_build_img --img-name="${DEVTOOLS_IMG_NAME}" --target=builder-and-dev;
      func_build_img --img-name="${PROD_IMG_NAME}" --target=prod;
      exit 0;
    fi

    if [ $1 == "tsc" ]
    then
      func_typescript_build
      exit 0;
    fi

    if [ $1 == "twind" ] || [ $1 == "tailwind" ]
    then
      func_tailwind_build
      exit 0;
    fi

    if [ $1 == "prod" ]
    then
      func_start_prod
      exit 0;
    fi

    if [ $1 == "deploy" ]
    then
      func_deploy
      exit 0;
    fi

    if [ $1 != "" ]
    then
      echo "error: nor argument provided"

      echo "$__usage"
    
      exit 1;
    fi
fi
