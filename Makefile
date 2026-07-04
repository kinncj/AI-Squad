# Makefile — maple repo root
# Targets for building, testing, and maintaining the MAPLE platform itself.
# For targets in your project template, see template/Makefile.
.PHONY: build-tui build-tui-all build-app test test-app lint lint-app sdlc-report sdlc-rotate-logs clean help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -s -w -X main.version=$(VERSION)

## Build the maple binary (canonical name kept as build-tui for CI/docs)
## app/cmd/maple/template is a symlink → ../../../template; go:embed can't follow it,
## so swap it for a real copy for the build, then restore the symlink.
build-tui:
	@echo "Building maple..."
	@rm -f app/cmd/maple/template && cp -rL template app/cmd/maple/template
	@go build -ldflags="$(LDFLAGS)" -o maple ./app/cmd/maple; \
		status=$$?; rm -rf app/cmd/maple/template && ln -s ../../../template app/cmd/maple/template; exit $$status
	@echo "Built: ./maple"

## Alias — build into ./bin/maple for local dev
build-app:
	@echo "Building maple (bin/)..."
	@rm -f app/cmd/maple/template && cp -rL template app/cmd/maple/template
	@go build -ldflags="$(LDFLAGS)" -o bin/maple ./app/cmd/maple; \
		status=$$?; rm -rf app/cmd/maple/template && ln -s ../../../template app/cmd/maple/template; exit $$status
	@echo "Built: ./bin/maple"

## Run the rebuilt binary's unit tests (with the same template dance for the embed)
test-app:
	@rm -f app/cmd/maple/template && cp -rL template app/cmd/maple/template
	@go test ./app/... && go vet ./app/...; \
		status=$$?; rm -rf app/cmd/maple/template && ln -s ../../../template app/cmd/maple/template; exit $$status

## gofmt-check the rebuilt binary sources
lint-app:
	@gofmt -e ./app >/dev/null && echo "gofmt(app): clean" || (echo "gofmt(app): issues found" && exit 1)

## Cross-compile maple for all platforms
build-tui-all:
	@mkdir -p dist
	@rm -f app/cmd/maple/template && cp -rL template app/cmd/maple/template
	GOOS=darwin  GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-darwin-amd64  ./app/cmd/maple
	GOOS=darwin  GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/maple-darwin-arm64  ./app/cmd/maple
	GOOS=linux   GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-linux-amd64   ./app/cmd/maple
	GOOS=linux   GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/maple-linux-arm64   ./app/cmd/maple
	GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-windows-amd64.exe ./app/cmd/maple
	@rm -rf app/cmd/maple/template && ln -s ../../../template app/cmd/maple/template
	@echo "Binaries in dist/"

## Run the test suite for this repo
test:
	@bash tests/cli/test_ai_squad.sh

## Lint Go code
lint:
	@gofmt -e ./app >/dev/null && echo "gofmt: clean" || (echo "gofmt: issues found" && exit 1)

## Print per-story agent invocation counts and estimated costs
## Reads .claude/logs/skills.jsonl; safe to run offline (shows cached data)
sdlc-report:
	@if [ ! -f .claude/logs/skills.jsonl ]; then \
		echo "No skills.jsonl found at .claude/logs/skills.jsonl"; \
		echo "Run some agent workflows first."; \
		exit 0; \
	fi
	@echo "=== MAPLE SDLC Cost Report ==="
	@echo ""
	@python3 scripts/sdlc-report.py .claude/logs/skills.jsonl 2>/dev/null || \
		python3 -c " \
import json, sys, collections; \
lines = [json.loads(l) for l in open('.claude/logs/skills.jsonl') if l.strip()]; \
by_story = collections.defaultdict(list); \
[by_story[l.get('story','unknown')].append(l) for l in lines]; \
print(f'Stories: {len(by_story)}  Total invocations: {len(lines)}'); \
[print(f'  {s}: {len(v)} invocations') for s,v in sorted(by_story.items())] \
"

## Rotate .claude/logs/ — keep last 5 compressed, delete older
## Safe to run any time; also triggered by post-merge hook
sdlc-rotate-logs:
	@bash scripts/sdlc/rotate-logs.sh

## Remove built binaries
clean:
	@rm -f maple dist/maple-*
	@echo "Cleaned."

## Show available targets
help:
	@echo ""
	@echo "  make build-tui          Build the maple binary (./maple, from app/cmd/maple)"
	@echo "  make build-app          Build into ./bin/maple (local dev)"
	@echo "  make build-tui-all      Cross-compile for darwin/linux/windows"
	@echo "  make test-app           Run app unit tests (with the embed dance)"
	@echo "  make test               Run the shell test suite"
	@echo "  make lint               Lint Go code (gofmt ./app)"
	@echo "  make sdlc-report        Print cost + invocation report"
	@echo "  make sdlc-rotate-logs   Rotate .claude/logs/ (keep last 5)"
	@echo "  make clean              Remove built binaries"
	@echo ""
