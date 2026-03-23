# Jara Copilot Instructions

Use this file as the primary coding guide for AI-assisted changes in this repository.

## Project Snapshot

- Project: jara
- Type: Terminal User Interface (TUI) for Juju environments
- Language: Go
- Module: github.com/bschimke95/jara
- Go version: 1.25.8/latest

Jara provides a keyboard-driven terminal experience for observing and operating Juju clusters.

## Core Stack

- charm.land/bubbletea/v2 for the event loop and model updates
- charm.land/bubbles/v2 for widgets and inputs
- charm.land/lipgloss/v2 for styles and layout
- github.com/spf13/cobra for CLI structure in internal/cmd
- gopkg.in/yaml.v3 for config parsing

## Current Repository Structure

Top-level:

- cmd/jara/main.go: executable entry point
- internal/: application packages
- config.example.yaml: reference config
- Makefile: standard build/test/lint/fmt targets

Internal packages:

- internal/api: Juju client interface and real/mock implementations
- internal/app: root Bubble Tea model, streams, input handling, navigation dispatch
- internal/cmd: Cobra command definitions (root, run, info, version)
- internal/color: status and theme color helpers
- internal/config: config loading, paths, flags, theme and key mapping
- internal/model: domain types
- internal/nav: navigation stack and routing state
- internal/ui: chrome, shared widgets, key maps
- internal/view: feature views split by domain
  - internal/view/controllers
  - internal/view/models
  - internal/view/modelview
  - internal/view/overview
  - internal/view/applications
  - internal/view/units
  - internal/view/machines
  - internal/view/relations
  - internal/view/debuglog

## Architectural Guidelines

- Preserve Bubble Tea update semantics:
  - keep state mutation deterministic inside Update handlers
  - return commands for async work, do not block Update
- Keep package boundaries clear:
  - API concerns stay in internal/api
  - rendering and view behavior stay in internal/view and internal/ui
  - navigation logic stays in internal/nav
  - root orchestration stays in internal/app
- Prefer small focused functions over large mixed-responsibility handlers.

## Go Coding Standards

- Follow idiomatic Go naming:
  - Exported: CamelCase
  - Unexported: lowerCamelCase
- Keep interfaces minimal and behavior-oriented.
- Favor explicit error returns over panics.
- Avoid hidden side effects and global mutable state.
- Use context for external calls with timeouts/cancellation when appropriate.
- Keep imports tidy and code gofumpt-formatted.

## Bubble Tea and TUI Conventions

- Handle key events via shared key maps and view-specific mappings.
- Keep navigation predictable (drill-down, back, breadcrumbs/crumb bar).
- Preserve vim-style movement behavior where already implemented.
- Keep debug-log interactions consistent:
  - inline search navigation
  - filter modal behavior
  - clear filter behavior

## CLI Conventions

- Add/modify command behavior in internal/cmd first, then wire through cmd/jara/main.go if required.
- Keep flags discoverable and consistent with README/config docs.
- Ensure readonly mode constraints are respected by write operations.

## Testing Requirements

Definition of done for behavior changes:

- Add or update tests in the same package as the changed code.
- Prefer table-driven tests for multiple scenarios.
- Use only the standard testing package.
- Keep tests deterministic and fast.

Important test areas:

- internal/nav: stack transitions, route resolution
- internal/config: parsing, defaults, key/theme merges, path behavior
- internal/color: status to color mapping
- internal/view/*: row builders, selection, state transitions, message handling
- internal/api/mock: stateful mutations and concurrent safety

Run before considering work complete:

- make fmt
- make lint
- make test
- make build

Use make test-integration when a change impacts integration-tagged paths.

## VHS Integration Tests

Every notable feature or view must have a corresponding VHS tape file in tests/vhs/.

- Tape files drive the TUI with the mock client (`--demo` flag) and capture ASCII golden files to tests/vhs/golden/.
- The shared setup is in tests/vhs/_setup.tape; all tapes source it via `Source tests/vhs/_setup.tape`.
- Use `Wait+Screen /pattern/` to block until expected content renders. Prefer short, truncation-safe patterns (e.g., a unique word like `grafana`) over fragile multi-column header regexes.
- Adding a new view or changing an existing view's rendering requires adding or updating the corresponding tape and regenerating the golden file.
- Run `make test-vhs` to validate golden files match. Run `make test-vhs-update` to regenerate them after intentional changes.
- VHS requires `vhs`, `ttyd`, and `ffmpeg`; `make test-vhs` installs missing dependencies automatically via the `ensure-vhs` target.

## Lint and Formatting

- Formatting is enforced with gofumpt.
- Linting is enforced with golangci-lint.
- Do not suppress lint warnings unless there is a clear, documented reason.

## Change Safety Checklist

- Avoid broad refactors unless requested.
- Do not change public behavior silently; update docs/tests for behavior changes.
- Keep configuration and keybinding compatibility stable.
- Preserve existing UX patterns unless the task explicitly asks for a redesign.

## Documentation Expectations

- Keep README and config example aligned with new flags or behavior.
- Add concise comments for non-obvious logic only.
- Keep this file updated when architecture, workflow, or package structure changes.
