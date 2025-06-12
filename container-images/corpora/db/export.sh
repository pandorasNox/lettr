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
echo "start export script"
DATASETS_DIR=${1?missing first argument datasets dir (absolut path)}
echo "use datasets from dir: ${DATASETS_DIR}"

EXPORT_DIR=${2?missing second argument export dir (absolut path)}
echo "write corpora export to: ${EXPORT_DIR}"

func_ensureExportDirectoryIsWritable() {
  EXP_DIR=${1?missing first argument export dir (absolut path)}
  EXP_DIR="${EXP_DIR}/exports"
  echo "ensure export dir is writable: ${EXP_DIR}"

  if test ! -d "${EXP_DIR}"; then
    return
  fi

  # ensure host target folder doesn't exist
  (
    cd "${EXP_DIR}"
    existingExportFiles=$(find . -name "*-export.txt" -maxdepth 1 -mindepth 1 -type f -printf '%f\n')
    echo "delete existing files..."

    # shellcheck disable=SC2086
    set -- ${existingExportFiles}
    for file in "${@}"; do
      echo "...delete old file ${file}"
      rm "${file}"
    done
  )
  rmdir "${EXP_DIR}"
}

func_do() {
    dirs=$(find "${DATASETS_DIR}" -maxdepth 1 -mindepth 1 -type d -printf '%f\n')
    # echo "dirs: '${dirs}'"

    # shellcheck disable=SC2086
    set -- ${dirs}

    tmpDir=$(mktemp -d)

    mkdir -p /tmp/exports
    chown mysql:mysql /tmp/exports
    echo "...done writing export dir"

    for dir in "${@}"; do
        # echo "og dir: '${dir}'";
        dir=${dir%*/};      # remove the trailing "/"
        echo "process database '${dir##*/}'...";    # print everything after the final "/"

        queryFileName=query.txt
        tmpQueryFilePath="${tmpDir}/${queryFileName}"

        exportFilePath="/tmp/exports/corpora-${dir}-export.txt"

cat << EOF > "${tmpQueryFilePath}"
USE ${dir};
SELECT LOWER(word) FROM words
WHERE CHAR_LENGTH(word) = 5
  AND word RLIKE "^[A-Z]?[a-z]+$"
  AND freq > 1
ORDER BY freq DESC, word
INTO OUTFILE '${exportFilePath}'
  LINES TERMINATED BY '\n'
EOF

        # run export
        # shellcheck disable=SC2002
        cat "${tmpQueryFilePath}" | mariadb -uroot -p'example' "${dir}"
        echo "...done exporting ${dir}"

        # cleanup tmp files
        rm "${tmpQueryFilePath}"

    done

    # remove tmp query dir
    rmdir "${tmpDir}"

    func_ensureExportDirectoryIsWritable "${EXPORT_DIR}"

    mv /tmp/exports "${EXPORT_DIR}"
    echo "...done moving new corpora exports"
}

func_do
