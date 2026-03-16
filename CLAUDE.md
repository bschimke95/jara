# Jara Project Guide for Claude

## Project Overview

**Jara** is a Terminal User Interface (TUI) application written in Go that provides an interactive way to observe and interact with Juju clusters. It is inspired by the design and interaction patterns of `k9s`, a popular Kubernetes CLI tool.

### Key Features

- **Interactive TUI**: Built with Bubble Tea v2, providing a responsive and intuitive terminal interface
- **k9s-style UI**: Matches the visual aesthetics and navigation patterns of k9s
- **Multi-view Navigation**: Switch between Applications, Units, Machines, and Relations views
- **Real-time Status**: Polls the Juju controller every 3 seconds for status updates
- **Debug-Log Streaming**: Live `juju debug-log` stream with inline search (`/`, `n/N`) and a two-pane vim-navigable filter modal (`Shift+F`)
- **Log Filtering**: Filter by level, application, unit, machine, module, or label; active filter shown as chips in the border title; `Shift+D` clears the filter
- **Command Mode**: Supports command-line style interactions (`:` prefix) and filtering (`/` prefix)
- **Controller Selection**: Selecting a controller persists the choice to the local Juju client store (`~/.local/share/juju/`), keeping jara in sync with the `juju` CLI

### Module & Build Info

- **Go Module**: `github.com/bschimke95/jara`
- **Go Version**: 1.25+ (uses latest language features; `go.mod` declares `go 1.25.8`)
- **Key Dependencies**:
  - `charm.land/bubbletea/v2` - TUI framework
  - `charm.land/bubbles/v2` - UI components (table, text input, help)
  - `charm.land/lipgloss/v2` - Terminal styling and layout
  - `github.com/atotto/clipboard` - Clipboard integration (optional)

### Directory Structure

```
jara/
├── cmd/jara/
│   └── main.go                 # Entry point
├── internal/
│   ├── api/
│   │   └── client.go          # Juju API client interface + mock implementation
│   ├── color/
│   │   └── color.go           # k9s-inspired color theme (dark blue palette)
│   ├── model/
│   │   └── types.go           # Domain types (Controller, Model, Application, Unit, etc.)
│   ├── nav/
│   │   └── nav.go             # Navigation stack and view routing
│   ├── render/
│   │   └── render.go          # Transforms domain types into table rows/columns
│   ├── ui/
│   │   ├── chrome.go          # Header, border, footer, key hints rendering
│   │   ├── keys.go            # Vim-style key bindings
│   │   └── (other UI utilities)
│   ├── view/
│   │   ├── view.go            # View interface and common message types
│   │   ├── applications.go    # Applications table view
│   │   ├── units.go           # Units table view
│   │   ├── machines.go        # Machines table view
│   │   ├── relations.go       # Relations table view
│   │   ├── debuglog.go        # Debug-log streaming view (search, filter modal integration)
│   │   └── filtermodal.go     # Two-pane vim-navigable filter modal overlay
│   └── app/
│       └── app.go             # Root Bubble Tea model, layout composition
└── go.mod, go.sum
```

---

## Sub-Agents

Project-specific Claude Code sub-agents live in `.claude/agents/`. They provide focused expertise and can be invoked explicitly or picked up automatically based on the task context.

### Available Agents

| Agent | File | Invoke when… |
|---|---|---|
| `agent-organizer` | `.claude/agents/agent-organizer.md` | A task spans multiple concerns (new feature end-to-end, CLI migration, release). Decomposes work and routes to the right specialists. |
| `golang-pro` | `.claude/agents/golang-pro.md` | Writing new Go code, implementing features, adding `Client` interface methods. |
| `cli-developer` | `.claude/agents/cli-developer.md` | Designing or implementing the Cobra CLI layer in `cmd/jara/`. |
| `refactoring-specialist` | `.claude/agents/refactoring-specialist.md` | Removing type assertions, decomposing large view/render functions, improving structure without changing behaviour. |
| `qa-expert` | `.claude/agents/qa-expert.md` | Writing or improving tests, expanding `MockClient`, adding table-driven test cases. |
| `code-reviewer` | `.claude/agents/code-reviewer.md` | Reviewing a diff or PR for correctness, lint compliance, and test coverage. |
| `debugger` | `.claude/agents/debugger.md` | Diagnosing test failures, race conditions, nil panics, or broken TUI rendering. |
| `documentation-expert` | `.claude/agents/documentation-expert.md` | Writing godoc comments, package-level docs, or updating this file. |

### When to Use `agent-organizer`

For large tasks that cross multiple concerns, invoke `agent-organizer` first. It will decompose the work into steps and assign each to the appropriate specialist. Example triggers:

- "Implement a new `ScaleApplication` command end-to-end"
- "Migrate `cmd/jara/main.go` to Cobra"
- "Add a new view for Offers"

For small, focused tasks (e.g. "add a unit test for `render.FormatSince`"), invoke the specialist agent directly.

---

## Go Coding Standards for This Project

### 1. Package Organization

- **Cohesive Packages**: Each `internal/` subdirectory is a single package with a clear responsibility:
  - `api` - external service communication
  - `color` - styling constants and utilities
  - `model` - domain types
  - `nav` - navigation/routing
  - `render` - data transformation for display
  - `ui` - UI components and styling
  - `view` - view implementations (MVC pattern)
  - `app` - root application state and coordination

### 2. Naming Conventions

#### Functions & Methods
- **CamelCase** for exported functions: `New()`, `Update()`, `View()`, `Init()`
- **camelCase** for unexported helpers: `pollStatus()`, `handleNavigate()`, `contentHeight()`
- **Descriptive names**: `HeaderContent()`, `BorderBox()`, `HintsForView()`

#### Types & Interfaces
- **CamelCase** for exported types: `Model`, `Client`, `View`, `KeyHint`, `FullStatus`
- **Suffix conventions**:
  - `*Model` - Bubble Tea models
  - `*View` - View implementations implementing the `View` interface
  - `*Msg` - Message types (used as events)
- Example: `type StatusUpdatedMsg struct { Status *model.FullStatus }`

#### Constants & Variables
- **SCREAMING_SNAKE_CASE** for exported constants: `const pollInterval = 3 * time.Second`
- **camelCase** for unexported: `const logo = "..."`
- **Implicit types** where clear: `const modeNormal inputMode = iota`

#### Interfaces
- **Minimal, focused interfaces**: `View` interface has 4 methods (Init, Update, View, SetSize)
- **Reader/Writer suffix pattern** (where applicable)
- **Embed interfaces** to compose behavior

### 3. Code Structure & Patterns

#### Model-View-Controller (MVC)
- **Model**: Holds state (`Model`, domain types in `model/`)
- **View**: Renders UI (`view/` implementations, `ui/chrome.go`)
- **Controller**: Routes messages/updates (`Update()` methods, `app.go`)

#### Bubble Tea Patterns
- **Init() tea.Cmd**: Return tea.Batch(), tea.Tick(), or nil
- **Update(tea.Msg) (tea.Model, tea.Cmd)**: Type-assert messages, return updated model + cmd
- **View() tea.View**: Return structured UI using lipgloss

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        // handle keys
    case view.NavigateMsg:
        // handle navigation
    default:
        // pass to active view
    }
    return m, cmd
}
```

#### Error Handling
- **Explicit returns** for error cases: `if err != nil { return m, errMsg{err} }`
- **Error type assertions** in Update: `case errMsg: m.err = msg.err`
- **No panic** in production code; errors flow through the message system

### 4. Interface Implementation

All view types implement the `View` interface:

```go
type View interface {
    tea.Model                                    // Init, Update, View
    SetSize(width, height int)                   // Set visible area
    SetStatus(status *model.FullStatus)          // Push new status data
}
```

Never use pointer receivers for small value types; use pointer receivers for types that need mutation or are > 128 bytes.

### 5. Formatting & Style

- **gofmt compliant**: All code passes `go fmt`
- **Imports**: Grouped in standard order (stdlib, blank line, external packages)
- **Line length**: No hard limit, but prefer readability (keep functions < 30 lines when possible)
- **Comments**: Exported functions/types have doc comments starting with the name:

```go
// BorderBox wraps content in a rounded border with an optional title.
func BorderBox(content, title string, width int) string { ... }
```

### 6. Concurrency & Goroutines

- **No goroutines in Update()**: Bubble Tea handles concurrency via channels
- **Cmd-based async**: Use `tea.Tick()` for polling, `tea.Batch()` to combine
- **Context usage**: Always use `context.WithTimeout()` for external calls (see `pollStatus()`)

### 7. Testing

**Definition of Done**: A feature is not complete until it has unit tests. Integration tests are also required when the feature involves stateful interactions between components (e.g., API client behaviour, multi-step workflows).

#### Unit Tests
- Test files live in the same package: `foo.go` → `foo_test.go`
- Use **table-driven tests** for multiple input/output cases:
  ```go
  tests := []struct {
      input string
      want  string
  }{
      {"active", "#00ff00"},
      {"blocked", "#ff5555"},
  }
  for _, tt := range tests {
      t.Run(tt.input, func(t *testing.T) { ... })
  }
  ```
- Use only `testing.T` from the standard library — no third-party assertion libraries
- Test unexported helpers directly within the same package (white-box testing)
- Keep tests fast: no sleeps, no real network calls

#### Integration Tests
- Use `MockClient` (in `internal/api/mock.go`) to exercise multi-step workflows without a real Juju controller
- `MockClient` is fully stateful: `ScaleApplication`, `SelectController`, `SelectModel` all mutate shared state and are reflected in subsequent `Status()` calls
- Test concurrent access when the code under test may be called from multiple goroutines
- Integration tests live alongside unit tests in the same `_test.go` files; no special build tag is required unless the test is genuinely slow (use `-tags integration` then)

#### What to Test
| Layer | What to cover |
|---|---|
| `nav` | Stack push/pop, breadcrumbs, command resolution |
| `color` | Status → color mapping, style consistency |
| `render` | Row generation, sorting, column scaling |
| `api` (mock) | State mutations, error paths, concurrency safety |
| `view/*` | Row counts, selection behaviour, message handling |

### 8. Common Patterns

#### Message Types (Events)
```go
type NavigateMsg struct {
    Target  nav.ViewID
    Context string
}
```

#### Conditional Commands
```go
if ready {
    return m, tea.Tick(time.Now(), func(_ time.Time) tea.Msg { return tickMsg{} })
}
return m, nil
```

#### Styling with lipgloss
```go
style := lipgloss.NewStyle().
    Foreground(color.Primary).
    Bold(true).
    Padding(1, 2)

rendered := style.Render(content)
```

#### String Building
```go
var body strings.Builder
body.WriteString("line 1\n")
body.WriteString("line 2\n")
result := body.String()
```

### 9. Exported vs Unexported

- **Unexported helpers** for internal logic: `handleNavigate()`, `updateActiveView()`
- **Exported interfaces** for contracts: `View`, `Client`, `KeyMap`
- **Exported types** for public data: `Model`, `FullStatus`, `Application`

### 10. Comments & Documentation

- **Package-level comments**: At the top of each file, start with `package <name>`
- **Function-level comments**: Document exported functions and complex logic
- **Inline comments**: Sparingly; prefer self-documenting code
- **TODO/FIXME**: Use `// TODO:` for future work, with context

---

## Development Workflow

A `Makefile` provides all common tasks. Prefer `make` targets over raw `go` commands.

### Building

```bash
make build          # Build ./cmd/jara
```

### Testing

```bash
make test           # go test -race ./...
make test-integration  # go test -race -tags integration ./...
```

Always run with `-race`; the `make` targets do this automatically.

### Linting

```bash
make lint           # golangci-lint run ./...
make vet            # go vet ./...
```

The linter is configured in `.golangci.yml`. Enabled linters: `govet`, `staticcheck`, `errcheck`, `gosimple`, `unused`, `ineffassign`, `revive`, `gofumpt`.

All lint warnings must be resolved before a change is considered ready. Do not suppress linter warnings with `//nolint` unless there is a documented reason.

### Formatting

```bash
make fmt            # gofumpt -w .
```

Code must be formatted with `gofumpt` (a stricter superset of `gofmt`). The CI pipeline enforces this via the `gofumpt` linter.

### Full pre-commit check

```bash
make all            # lint + test + build
```

### Module hygiene

```bash
make tidy           # go mod tidy
```

---

## Future Enhancements

- Streaming updates instead of polling
- Write operations (scale applications, restart units)
- Configuration file support
- Custom color themes
- Detailed resource inspection panel
- Expand `MockClient.DebugLog` to apply filter fields (level, entity, module) for integration testing
