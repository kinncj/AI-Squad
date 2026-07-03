# Makefile — maple repo root
# Targets for building, testing, and maintaining the MAPLE platform itself.
# For targets in your project template, see template/Makefile.
.PHONY: build-tui build-tui-all build-app test test-app lint lint-app sdlc-report sdlc-rotate-logs clean help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -s -w -X main.version=$(VERSION)

## Build the maple TUI binary
build-tui:
	@echo "Building maple TUI..."
	@rm -f tui/template && cp -rL template tui/template
	@go build -ldflags="$(LDFLAGS)" -o maple ./tui
	@rm -rf tui/template && ln -s ../template tui/template
	@echo "Built: ./maple"

## Build the rebuilt maple binary (feature/better-ui-ux; parallel to ./maple until parity)
build-app:
	@echo "Building maple (rebuild)..."
	@go build -ldflags="$(LDFLAGS)" -o bin/maple ./app/cmd/maple
	@echo "Built: ./bin/maple"

## Run the rebuilt binary's unit tests
test-app:
	@go test ./app/...

## gofmt-check the rebuilt binary sources
lint-app:
	@gofmt -e ./app >/dev/null && echo "gofmt(app): clean" || (echo "gofmt(app): issues found" && exit 1)

## Cross-compile maple for all platforms
build-tui-all:
	@mkdir -p dist
	@rm -f tui/template && cp -rL template tui/template
	GOOS=darwin  GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-darwin-amd64  ./tui
	GOOS=darwin  GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/maple-darwin-arm64  ./tui
	GOOS=linux   GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-linux-amd64   ./tui
	GOOS=linux   GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/maple-linux-arm64   ./tui
	GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/maple-windows-amd64.exe ./tui
	@rm -rf tui/template && ln -s ../template tui/template
	@echo "Binaries in dist/"

## Run the test suite for this repo
test:
	@bash tests/cli/test_ai_squad.sh

## Lint Go code
lint:
	@gofmt -e ./tui >/dev/null && echo "gofmt: clean" || (echo "gofmt: issues found" && exit 1)

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
	@echo "  make build-tui          Build maple TUI binary"
	@echo "  make build-tui-all      Cross-compile for darwin/linux/windows"
	@echo "  make test               Run test suite (218 tests)"
	@echo "  make lint               Lint Go code"
	@echo "  make sdlc-report        Print cost + invocation report"
	@echo "  make sdlc-rotate-logs   Rotate .claude/logs/ (keep last 5)"
	@echo "  make clean              Remove built binaries"
	@echo ""
