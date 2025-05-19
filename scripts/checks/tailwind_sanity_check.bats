#!/usr/bin/env bats

SCRIPT_DIR=${BATS_TEST_DIRNAME}

func_check_output_file_size(){(
  set -e
  set -u

  _outputFilePath="${SCRIPT_DIR}/../../web/static/generated/output.css"

  _outputFileBytes=$(du -b "${_outputFilePath}" | awk '{print $1}')
  _minExpectedBytes=7186
  if test "${_outputFileBytes}" -lt "${_minExpectedBytes}"; then
    echo "🔴 error: File is smaller than '${_minExpectedBytes}' bytes, file is '${_outputFileBytes}' bytes" >&2
    exit 1;
  fi

  # without @source / minify
  # 0.103 🔴 error: File is smaller than '102400' bytes, file is '5219' bytes
  # without @source / non-minify
  # 0.096 🔴 error: File is smaller than '102400' bytes, file is '6097' bytes
  #
  # vs
  #
  # with @source / minify
  # 0.102 🔴 error: File is smaller than '102400' bytes, file is '24622' bytes
  # with @source / non-minify
  # 0.094 🔴 error: File is smaller than '102400' bytes, file is '32218' bytes
)}

# Function to check if a file contains all strings in the 'needles' variable
func_check_template_contains_test_styles() {(
  set -e
  set -u

  _templateFilePath="${SCRIPT_DIR}/../../pkg/router/routes/templates/pages/test.html.tmpl"
  _needles="flex shadow-lg border bg-gray-300 invert"

  # Check if the file exists and is readable
  if [ ! -f "${_templateFilePath}" ]; then
    echo "🔴 error: File '${_templateFilePath}' does not exist."
    return 1
  fi

  if [ ! -r "${_templateFilePath}" ]; then
    echo "🔴 error: File '${_templateFilePath}' is not readable."
    return 1
  fi

  # Initialize a variable to track missing strings
  missing=""

  # Loop through each string in the needles variable
  for str in ${_needles}; do
    # Use grep to check if the string exists in the file
    if ! grep -F -q -- "$str" "${_templateFilePath}"; then
      missing="${missing} ${str}"
    fi
  done

  # Report the result
  if [ -z "${missing}" ]; then
    echo "...all strings are present in the template file."
    return 0
  else
    echo "🔴 error: The following strings are missing from the template file:${missing}"
    return 1
  fi
)}

# Function to check if a file contains all strings in the 'needles' variable
func_check_output_contains_test_styles() {(
  set -e
  set -u

  _outputFilePath="${SCRIPT_DIR}/../../web/static/generated/output.css"
  _needles="flex shadow-lg border bg-gray-300 invert"

  # Check if the file exists and is readable
  if [ ! -f "${_outputFilePath}" ]; then
    echo "🔴 error: File '${_outputFilePath}' does not exist."
    return 1
  fi

  if [ ! -r "${_outputFilePath}" ]; then
    echo "🔴 error: File '${_outputFilePath}' is not readable."
    return 1
  fi

  # Initialize a variable to track missing strings
  missing=""

  # Loop through each string in the needles variable
  for str in ${_needles}; do
    # Use grep to check if the string exists in the file
    if ! grep -F -q -- ".${str}{" "${_outputFilePath}"; then
      missing="${missing} ${str}"
    fi
  done

  # Report the result
  if [ -z "${missing}" ]; then
    echo "...all strings are present in the output file."
    return 0
  else
    echo "🔴 error: The following strings are missing from the output file:${missing}"
    return 1
  fi
)}

@test "func_check_output_file_size" {
    run func_check_output_file_size
    echo $output
    test "$status" -eq 0
}

@test "func_check_template_contains_test_styles" {
    run func_check_template_contains_test_styles
    echo $output
    test "$status" -eq 0
}

@test "func_check_output_contains_test_styles" {
    run func_check_output_contains_test_styles
    echo $output
    test "$status" -eq 0
}