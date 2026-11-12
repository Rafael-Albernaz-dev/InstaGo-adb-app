SHELL := /bin/bash
export GOTOOLCHAIN := local
export GOFLAGS := -mod=readonly

.PHONY: hygiene-check check hooks

hygiene-check:
	bash scripts/ci/check-repo-hygiene.sh

check: hygiene-check

hooks:
	git config core.hooksPath .githooks
