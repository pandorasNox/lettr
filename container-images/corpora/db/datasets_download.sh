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

DOWNLOAD_DIR=${1?missing first argument download dir}
echo "${DOWNLOAD_DIR}"

func_download () {
  set -- \
    eng_news_2023_10K \
    deu_news_2023_10K \
  ;
    # eng_news_2023_1M \
    # deu_news_2023_1M \

  for item in "${@}"; do
    printf 'download %s...' "${item}";
    curl -s \
      --output "${DOWNLOAD_DIR}/${item}.tar.gz" \
      https://downloads.wortschatz-leipzig.de/corpora/"${item}".tar.gz \
    ;
    tar -xvzf "${DOWNLOAD_DIR}"/"${item}".tar.gz -C "${DOWNLOAD_DIR}"/ > /dev/null;
    rm "${DOWNLOAD_DIR}"/"${item}".tar.gz;
    printf ' done\n';
  done
}

func_download
