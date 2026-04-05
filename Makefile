.PHONY: build build-vhs lint test test-integration test-vhs test-vhs-update ensure-vhs fmt vet tidy all

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -w -s \
	-X github.com/bschimke95/jara/internal/cmd.version=$(VERSION) \
	-X github.com/bschimke95/jara/internal/cmd.commit=$(COMMIT) \
	-X github.com/bschimke95/jara/internal/cmd.date=$(DATE)

all: lint test build

build:
	go build -ldflags "$(LDFLAGS)" -o jara ./cmd/jara

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...

fmt:
	gofumpt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

# VHS builds use fixed version strings so golden files are stable across commits.
VHS_LDFLAGS = -w -s \
	-X github.com/bschimke95/jara/internal/cmd.version=test \
	-X github.com/bschimke95/jara/internal/cmd.commit=test \
	-X github.com/bschimke95/jara/internal/cmd.date=2000-01-01T00:00:00Z

build-vhs:
	go build -ldflags "$(VHS_LDFLAGS)" -o jara ./cmd/jara

# VHS dependency installer — ensures vhs, ttyd, and ffmpeg are available.
ensure-vhs:
	@command -v ffmpeg >/dev/null 2>&1 || { \
		echo "Installing ffmpeg…"; \
		if command -v apt-get >/dev/null 2>&1; then sudo apt-get update -qq && sudo apt-get install -y -qq ffmpeg; \
		elif command -v brew >/dev/null 2>&1; then brew install ffmpeg; \
		else echo "Error: install ffmpeg manually" && exit 1; fi; \
	}
	@command -v ttyd >/dev/null 2>&1 || { \
		echo "Installing ttyd…"; \
		if command -v apt-get >/dev/null 2>&1; then sudo apt-get update -qq && sudo apt-get install -y -qq ttyd; \
		elif command -v brew >/dev/null 2>&1; then brew install ttyd; \
		else echo "Error: install ttyd manually" && exit 1; fi; \
	}
	@command -v vhs >/dev/null 2>&1 || { \
		echo "Installing vhs…"; \
		go install github.com/charmbracelet/vhs@latest; \
	}

# VHS integration tests — compare last captured frame of each golden against accepted state.
# Uses the disk golden files as the accepted reference (not git HEAD), so 'make test-vhs-update'
# accepts new output locally without requiring a commit first.
test-vhs: build-vhs ensure-vhs
	@fail=0; tmpdir=$$(mktemp -d); mkdir -p "$$tmpdir/ref" "$$tmpdir/bak"; \
	for f in tests/vhs/golden/*.ascii; do \
		name=$$(basename "$$f"); \
		cp "$$f" "$$tmpdir/bak/$$name"; \
		awk 'BEGIN{prev=""} /^─/{prev=buf; buf=""} {buf=buf$$0"\n"} END{printf "%s", prev}' "$$f" > "$$tmpdir/ref/$$name"; \
	done; \
	for tape in tests/vhs/*.tape; do \
		[ "$$(basename "$$tape")" = "_setup.tape" ] && continue; \
		echo "▶ $$tape"; \
		JARA_ROOT="$$(pwd)" vhs "$$tape" || { cp "$$tmpdir/bak/"*.ascii tests/vhs/golden/; rm -rf "$$tmpdir"; exit 1; }; \
	done; \
	for golden in tests/vhs/golden/*.ascii; do \
		name=$$(basename "$$golden"); \
		awk 'BEGIN{prev=""} /^─/{prev=buf; buf=""} {buf=buf$$0"\n"} END{printf "%s", prev}' "$$golden" > "$$tmpdir/new"; \
		if ! diff -q "$$tmpdir/ref/$$name" "$$tmpdir/new" > /dev/null 2>&1; then \
			echo "✗ $$name: last frame differs"; \
			diff "$$tmpdir/ref/$$name" "$$tmpdir/new" || true; \
			fail=1; \
		fi; \
	done; \
	cp "$$tmpdir/bak/"*.ascii tests/vhs/golden/; \
	rm -rf "$$tmpdir"; \
	if [ $$fail -ne 0 ]; then echo "\n✗ Golden files differ. Run 'make test-vhs-update' to accept changes."; exit 1; fi
	@echo "\n✓ All VHS tests passed."

# Regenerate golden files from current behavior.
test-vhs-update: build-vhs ensure-vhs
	@for tape in tests/vhs/*.tape; do \
		[ "$$(basename "$$tape")" = "_setup.tape" ] && continue; \
		echo "▶ $$tape"; \
		JARA_ROOT="$$(pwd)" vhs "$$tape" || exit 1; \
	done
	@echo "\n✓ Golden files regenerated. Review and commit the changes."
