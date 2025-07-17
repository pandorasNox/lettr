#!/usr/bin/env bash

set -o errexit
set -o nounset
# set -o xtrace

if set +o | grep -F 'set +o pipefail' > /dev/null; then
  set -o pipefail
fi

if set +o | grep -F 'set +o posix' > /dev/null; then
  set -o posix
fi

# -----------------------------------------------------------------------------

# start app
docker compose --file tests/playwright/playwright.docker-compose.yml up -d lettr-app
# run tests
docker compose --file tests/playwright/playwright.docker-compose.yml up playwright
# shutdown app
docker compose --file tests/playwright/playwright.docker-compose.yml stop lettr-app
