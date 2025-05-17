#!/usr/bin/env sh

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
SCRIPT_DIR=$(dirname "$0"); SCRIPT_DIR=$(eval "cd \"$SCRIPT_DIR\" && pwd")

# -----------------------------------------------------------------------------


func_prepend() {(
  _charAmount=${1:?'first argument should be amount of spaces'};
  _char=${2:?'second argument should be a single character to repeat+prepend text with'};
  _inputText=${3:?'second argument should be input text'};

  # check _charAmount is positve integer expression
  test "${_charAmount}" -eq "${_charAmount#-}" || (echo "error: first argument should be positve integer"; exit 1;);

  _fill=$( for i in $(seq 1 "${_charAmount}"); do printf "${_char}" ''; done );
  _res="$( printf '%s' "${_inputText}" | awk "{print \"${_fill}\" \$0}" )";

  printf '%s' "${_res}";
)}
export func_prepend;

_tw_stdout=$( \
  cd "${SCRIPT_DIR}"/../web/; \
  npx @tailwindcss/cli --input app/css/input.css --output static/generated/output.css 2>&1; \
);

if ( echo "${_tw_stdout}" | grep "warn" | grep "No utility classes were detected in your source files" > /dev/null ); then
  printf 'error: found unwamnted warning in tailwind output\n  tailwindoutput:\n%s\n' "$( func_prepend "6" " " "$(func_prepend "4"  ">" "${_tw_stdout}" )" ) ";
  echo "error: found unwamnted warning in tailwind output (see above)"
  exit 1;
fi

echo "√ tailwind build ok"
