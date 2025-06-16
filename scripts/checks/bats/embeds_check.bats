#!/usr/bin/env bats

SCRIPT_DIR=${BATS_TEST_DIRNAME}
PROJECT_ROOT_DIR=$(realpath "${SCRIPT_DIR}/../../..")

@test "generated directory contains only embeddings we expect" {
  _generatedFilesDirPath="${PROJECT_ROOT_DIR}/web/static/generated"

  run ls -1 "${_generatedFilesDirPath}"
  test "${status}" -eq 0

  expected=$'main.js\noutput.css'
  result="$(printf "%s\n" "${lines[@]}" | sort)"

  echo "result: ${result}"
  echo "expected: ${expected}"
  test "${result}" = "${expected}"
}
