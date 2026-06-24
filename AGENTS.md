# lettr — Developer Instructions for Agents

## Quick setup

```
make setup        # copies .env.template → .env
```

`.env` must have `PORT` (required), `GITHUB_TOKEN` (required for dev), `IMPRINT_URL` (optional).

## Key commands

All dev/test/lint commands route through `scripts/tools.sh` which builds and runs them in Docker containers.

```
make help       # show all commands via tools.sh
make fmt        # gofmt across repo
make check      # run preflight (tailwind build + check)
make test       # build Docker tester image → runs bats + eslint + go test -v ./...
make lint       # golangci-lint (docker) + eslint (docker cli container)
make lint-fix   # same as lint but with --fix
make shellcheck # lint all .sh files via container
make shellcheck-fix  # auto-fix shellcheck issues via git apply
make build      # synonym for make img (builds devtools + prod images)
make img        # build devtools and prod container images
make watch      # dev server with air hot-reload (port 9026)
make prod       # build prod image + run it on port 9033
make cli        # exec into devtools container shell
make corpora    # export corpora data
make renovate   # run renovate in local dry-run mode
```

## Build order (critical)

Generated assets (`web/static/generated/main.js`, `web/static/generated/output.css`) are Go-embedded and **must exist before `go build`**. They are produced by:

```bash
cd web && npm install
./../scripts/tailwind_build.sh   # Tailwind CSS
./node_modules/.bin/esbuild app/main.ts --tsconfig=app/tsconfig.json --bundle --minify --outfile=static/generated/main.js
```

The CI pipeline (`.github/workflows/go.yml`) runs this exact order: Node setup → npm install → Tailwind → esbuild → Go build.

## Testing

- **Go tests**: `make test` — runs inside the `tester` Docker build target (bats + eslint + `go test -v ./...`). Tests cannot run directly outside this container because the tester target verifies embedded assets exist.
- **Benchmarks**: `make bench` — runs `go test -bench=. -run=^$ -cpu=1 -benchmem -count=10` inside the devtools container.
- Test files: `*_test.go` in root and `pkg/` subdirectories (10 test files total).
- **Bats tests**: live in `scripts/checks/bats/` (`embeds_check.bats`, `tailwind_sanity_check.bats`) — these enforce the "embedded assets exist" check inside the `tester` target.
- **Playwright (browser) tests**: live in `tests/playwright/` — run via `make playwright` / `make playwright-ui`.

## CI pipeline order (.github/workflows/go.yml)

1. Build TypeScript + Tailwind (in workflow, no container)
2. Shellcheck (container)
3. Go tests (via Docker `tester` target — **not** `go test` directly)
4. golangci-lint (native action, v2.2.1)
5. Prod image build (via Docker `prod` target)

## Architecture

- **Entrypoint**: `main.go` — creates HTTP server, loads word databases from `configs/*.txt` via `//go:embed`, sets up graceful shutdown.
- **Packages** (`pkg/`):
  - `puzzle/` — game logic: word database, puzzle generation, match checking
  - `router/` + `router/routes/` — HTTP routing
  - `session/` — in-memory session management with cleanup goroutine
  - `state/` — server state + Prometheus metrics
  - `language/`, `middleware/`, `notification/`, `github/`, `assert/`
- **Frontend** (`web/`): TypeScript + HTMX + Tailwind CSS 4. Built with esbuild → outputs to `web/static/generated/`.
- **Word lists**: `configs/*.txt` files (English and German corpora + word lists).

## Deployment

- Deploy via `fly deploy` using `fly.toml` (target: Fly.io, port 9026, region AMS).
- `make deploy` runs `fly deploy --build-arg "GIT_REVISION=..."`.
- Dockerfile (`container-images/app/Dockerfile`) stages: `node` (build deps) → `builder-and-dev` (full build + dev tools) → `tester` (tests) → `prod` (minimal Alpine image).

## Gotchas

- **Never run `go test` directly** outside the Docker tester target — it will fail because embedded assets may be missing.
- **`.env` loading**: `tools.sh` sources `.env` if present before running commands. Missing `.env` means no `GITHUB_TOKEN` → warning but not fatal.
- **`.devcontainer/devcontainer.json` is incomplete** — missing trailing comma after the `image` line. Do not edit unless asked.
- **Docker image pinning**: All container images in `tools.sh`, `shellcheck.sh`, and `container-images/app/Dockerfile` use digest-pinned tags (renovate updates them).
- **htmx response targets + oob-swap** are both used; no ADR yet on whether to consolidate.
- **Renovate**: Uses `custom.regex` manager to update `_IMAGE` variables in shell scripts, plus `github-actions` and `gomod` managers. Run `make renovate` to preview updates.