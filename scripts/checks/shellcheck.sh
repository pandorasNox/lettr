#!/usr/bin/env sh

set -o errexit
set -o nounset

# -----------------------------------------------------------------------------

#SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SCRIPT_DIR=$(dirname "$0"); SCRIPT_DIR=$(eval "cd \"${SCRIPT_DIR}\" && pwd")
PROJECT_ROOT_DIR=$(realpath "${SCRIPT_DIR}/../..")

# -----------------------------------------------------------------------------

_fix=false

# handle flags, see https://stackoverflow.com/a/22395652
while test $# -gt 0 ; do

  # options with arguments
  if test "$1" = "--fix" ; then _fix=true ; shift ; continue; fi

  # unknown - up to you - positional argument or error?
  echo "Unknown option $1" && exit 1
  shift
done

# -----------------------------------------------------------------------------

# renovate: datasource=docker
SHELLCHECK_CONTAINER_IMAGE=docker.io/koalaman/shellcheck-alpine:v0.11.0@sha256:9955be09ea7f0dbf7ae942ac1f2094355bb30d96fffba0ec09f5432207544002;

# -----------------------------------------------------------------------------

func_shellcheck() {(
  set -o errexit;
  set -o nounset;

  set +e
  docker run -i --rm --entrypoint=ash -w /mnt/workdir -v "${PROJECT_ROOT_DIR}:/mnt/workdir" "${SHELLCHECK_CONTAINER_IMAGE}" -s <<EOF
    find . -name '*.sh' -print0 | xargs -0 shellcheck --rcfile /mnt/workdir/configs/.shellcheckrc;
EOF
  _shellcheckExitCode=${?}
  set -e
  if test "${_shellcheckExitCode}" != "0"; then
    echo "🔴 failed shellcheck (with exit_code='${_shellcheckExitCode}')"
    exit ${_shellcheckExitCode};
  fi
)}

# -----------------------------------------------------------------------------

func_fix_shellcheck_diff_paths() {(
  set -o errexit;
  set -o nounset;

  _diffFilePath=$1
  if ! test -f "${_diffFilePath}"; then
    echo "error: file with path='${_diffFilePath}' does not exist" >&2
    return 1
  fi

  # note: for shellechecks diff file, we need to replace the path './' with '/' in the git diff/patch here,
  #       bec otherwise `git apply` yields "error: invalid path './some/path' "
  cat "${_diffFilePath}" | sed 's|--- a/\./|--- a/|g' | sed 's|+++ b/\./|+++ b/|g'
)}

func_shellcheck_fix() {(
  set -o errexit;
  set -o nounset;

  _tmpDiffFile=$(mktemp)

  set +e
  docker run -i --rm --entrypoint=ash -w /mnt/workdir -v "${PROJECT_ROOT_DIR}":/mnt/workdir "${SHELLCHECK_CONTAINER_IMAGE}" -s <<EOF > "${_tmpDiffFile}"
    find . -name '*.sh' -print0 | xargs -0 shellcheck --format=diff --rcfile /mnt/workdir/configs/.shellcheckrc;
EOF
  _shellcheckExitCode=${?}
  set -e

  echo "   saved shellcheck diff to '${_tmpDiffFile}'"

  if test "${_shellcheckExitCode}" -eq 0; then
    echo "   note: found nothing to fix"
    exit 0;
  fi

  if ! test -s "${_tmpDiffFile}"; then
    echo "🔴 failed shellcheck (with exit_code='${_shellcheckExitCode}')"
    echo "🔴 error: shellcheck failed with finding issues, but shellcheck can not edit/format those itself (needs manual intervention, please run shellcheck without format to investigate).";
    exit 1;
  fi

  (
    cd "${PROJECT_ROOT_DIR}";
    echo '... run "git apply --check" with shellcheck diff';
    func_fix_shellcheck_diff_paths "${_tmpDiffFile}" | git apply --check;
    echo '... run "git apply" with shellcheck diff';
    func_fix_shellcheck_diff_paths "${_tmpDiffFile}" | git apply;
  )

  echo "   running shellcheck again, in case there were issues shellcheck format could not fix"
  func_shellcheck
)}

# -----------------------------------------------------------------------------

func_main() {(
  set -o errexit;
  set -o nounset;

  echo '🔍 start shellcheck';

  if test "${_fix}" = "true"; then
    func_shellcheck_fix;
    echo '🟢 done shellcheck fix';
    exit 0;
  fi

  func_shellcheck

  echo '🟢 done shellcheck';
)}

# -----------------------------------------------------------------------------

func_main
