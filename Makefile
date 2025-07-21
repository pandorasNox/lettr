
.PHONY: help
help:
	./scripts/tools.sh --help

.PHONY: cli
cli:
	./scripts/tools.sh cli

.PHONY: setup
setup:
	./scripts/tools.sh setup

.PHONY: fmt
fmt:
	./scripts/tools.sh fmt

.PHONY: watch
watch:
	./scripts/tools.sh watch

.PHONY: test
test:
	./scripts/tools.sh test

.PHONY: check
check:
	./scripts/tools.sh check

.PHONY: bench
bench:
	./scripts/tools.sh bench

.PHONY: down
down:
	./scripts/tools.sh down

.PHONY: skocli
skocli:
	./scripts/tools.sh skocli

.PHONY: img
img:
	./scripts/tools.sh img

.PHONY: build
build:
	./scripts/tools.sh img

.PHONY: lint
lint:
	./scripts/tools.sh lint

.PHONY: lint-fix
lint-fix:
	./scripts/tools.sh lint-fix

.PHONY: shellcheck
shellcheck:
	./scripts/tools.sh shellcheck

.PHONY: shellcheck-fix
shellcheck-fix:
	./scripts/tools.sh shellcheck-fix

.PHONY: tsc
tsc:
	./scripts/tools.sh tsc

.PHONY: twind
twind:
	./scripts/tools.sh twind

.PHONY: tailwind
tailwind:
	./scripts/tools.sh tailwind

.PHONY: prod
prod:
	./scripts/tools.sh prod

.PHONY: deploy
deploy:
	./scripts/tools.sh deploy

.PHONY: corpora
corpora:
	./scripts/corpora_export.sh

.PHONY: renovate
renovate:
	./scripts/tools.sh renovate

.PHONY: playwright-ui
playwright-ui:
	./scripts/tools.sh playwright-ui
