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

func_cleanup() {
  # Add any cleanup logic here
  docker compose --file tests/playwright/playwright.docker-compose.yml stop lettr-app caddy
}

trap 'code=$?; echo "🏁 Script exited with code $code"; echo "🧹 running cleanup"; func_cleanup; exit $code;' EXIT

# -----------------------------------------------------------------------------

# start app
docker compose --file tests/playwright/playwright.docker-compose.yml up -d lettr-app caddy
# run tests
docker compose --file tests/playwright/playwright.docker-compose.yml up --exit-code-from playwright playwright

# shutdown app (via trap)
