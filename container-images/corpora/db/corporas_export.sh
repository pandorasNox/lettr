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

DATASETS_DIR=${1?missing first argument datasets dir}
echo ${DATASETS_DIR}

func_do() {
    dirs=$(find ${DATASETS_DIR} -maxdepth 1 -mindepth 1 -type d -printf '%f\n')
    # echo "dirs: '${dirs}'"

    set -- ${dirs}

    for dir in "${@}"; do
        # echo "og dir: '${dir}'";
        dir=${dir%*/};      # remove the trailing "/"
        # echo "${dir##*/}";    # print everything after the final "/"

        mariadb -u'root' -p'example' -e "DROP DATABASE IF EXISTS ${dir}"
        (
            cd ${DATASETS_DIR}/${dir};
            mariadb -uroot -p'example' < ${dir}-import.sql
        )
    done
}

func_do
