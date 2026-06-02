SHELL := /bin/sh

NOTE ?= baseline
PKGS := ./...
RESULTS := logs/results.jsonl
IF := go run ./cmd/iterforge
TEMPLATES := text-normalization extraction prompt-optimization ranking

.PHONY: help test run baseline summarize compare new loop fmt vet clean reset-results check validate templates-smoke

help:
	@echo "Autoresearch Go Starter"
	@echo ""
	@echo "Targets:"
	@echo "  make test                 Run all Go tests"
	@echo "  make run NOTE='message'   Run one experiment and append to $(RESULTS)"
	@echo "  make baseline             Run one baseline experiment"
	@echo "  make summarize            Summarize experiment results"
	@echo "  make compare              Compare best vs latest run (or BASELINE/CANDIDATE ids)"
	@echo "  make new NAME=myproj      Scaffold a new IterForge project"
	@echo "  make validate             Validate policy.yaml against the schema"
	@echo "  make check                Validate policy, run fmt check, vet, and tests"
	@echo "  make templates-smoke      Scaffold each template and run its check+baseline"
	@echo "  make fmt                  Format Go code"
	@echo "  make vet                  Run go vet"
	@echo "  make loop N=10            Run N experiments using the current candidate"
	@echo "  make clean                Remove Go build/test cache"
	@echo "  make reset-results        Reset logs/results.jsonl to empty"
	@echo ""
	@echo "Variables:"
	@echo "  NOTE='hypothesis text'    Experiment note for make run"
	@echo "  N=10                      Number of iterations for make loop"

fmt:
	@gofmt -w candidates evals cmd internal

test:
	@go test $(PKGS)

vet:
	@go vet $(PKGS)

validate:
	@$(IF) validate-policy

check:
	@test -z "$$(gofmt -l candidates evals cmd internal)" || \
		(echo "Go files need formatting:"; gofmt -l candidates evals cmd internal; exit 1)
	@$(IF) validate-policy
	@go vet $(PKGS)
	@go test $(PKGS)

run:
	@$(IF) run -note "$(NOTE)"

baseline:
	@$(MAKE) run NOTE="baseline"

summarize:
	@$(IF) summarize

compare:
	@$(IF) compare $(if $(BASELINE),-baseline "$(BASELINE)") $(if $(CANDIDATE),-candidate "$(CANDIDATE)")

new:
	@test -n "$(NAME)" || (echo "usage: make new NAME=<project-name>"; exit 1)
	@$(IF) init "$(NAME)"

loop:
	@N=$${N:-10}; \
	i=1; \
	while [ $$i -le $$N ]; do \
		echo "== experiment $$i/$$N =="; \
		$(IF) run -note "loop iteration $$i" || exit $$?; \
		i=$$((i + 1)); \
	done

templates-smoke:
	@tmp=$$(mktemp -d); \
	for t in $(TEMPLATES); do \
		echo "== smoke: $$t =="; \
		$(IF) init -dir $$tmp -template $$t "smoke_$$t" || { rm -rf $$tmp; exit 1; }; \
		( cd "$$tmp/smoke_$$t" && $(MAKE) check && $(MAKE) baseline ) || { rm -rf $$tmp; exit 1; }; \
	done; \
	rm -rf $$tmp; \
	echo "all templates OK"

clean:
	@go clean -cache -testcache

reset-results:
	@: > $(RESULTS)
