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
  docker compose --file tests/playwright/playwright.docker-compose.yml down -t 3;
}

trap -- 'code=$?; echo "🧹 running cleanup (via trap)"; func_cleanup; echo "✔ done cleanup"; if test "$code" -eq "0"; then echo "🟢 Script exited with code $code"; else echo "🔴 Script exited with code $code"; fi ; exit $code;' EXIT

# -----------------------------------------------------------------------------

# start app
docker compose --file tests/playwright/playwright.docker-compose.yml up -d lettr-app caddy
# run tests
docker compose --file tests/playwright/playwright.docker-compose.yml up --exit-code-from playwright playwright

# shutdown app (via trap)
