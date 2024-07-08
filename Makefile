
.PHONY: cli
cli:
	./scripts/tools.sh cli

.PHONY: setup
setup:
	./scripts/tools.sh setup

.PHONY: watch
watch:
	./scripts/tools.sh watch

.PHONY: test
test:
	./scripts/tools.sh test

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

.PHONY: tsc
tsc:
	./scripts/tools.sh tsc

.PHONY: prod
prod:
	./scripts/tools.sh prod

.PHONY: deploy
deploy:
	./scripts/tools.sh deploy

.PHONY: corpora
corpora:
	./scripts/corpora_export.sh
