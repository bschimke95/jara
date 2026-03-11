# Jara Project Guide for Claude

## Project Overview

**Jara** is a Terminal User Interface (TUI) application written in Go that provides an interactive way to observe and interact with Juju clusters. It is inspired by the design and interaction patterns of `k9s`, a popular Kubernetes CLI tool.

### Key Features

- **Interactive TUI**: Built with Bubble Tea v2, providing a responsive and intuitive terminal interface
- **k9s-style UI**: Matches the visual aesthetics and navigation patterns of k9s
- **Multi-view Navigation**: Switch between Applications, Units, Machines, and Relations views
- **Real-time Status**: Polls the Juju controller every 3 seconds for status updates
- **Command Mode**: Supports command-line style interactions (`:` prefix) and filtering (`/` prefix)
- **Controller Selection**: Selecting a controller persists the choice to the local Juju client store (`~/.local/share/juju/`), keeping jara in sync with the `juju` CLI

### Module & Build Info

- **Go Module**: `github.com/bschimke95/jara`
- **Go Version**: 1.23+ (uses latest language features)
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
│   │   └── relations.go       # Relations table view
│   └── app/
│       └── app.go             # Root Bubble Tea model, layout composition
└── go.mod, go.sum
```

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

- While the current codebase doesn't have extensive tests, follow these patterns:
  - Test files in the same package: `file.go` + `file_test.go`
  - Table-driven tests for multiple cases
  - Use `testing.T` standard library
  - Mock interfaces (e.g., `MockClient` in `api/`)

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

### Building

```bash
go build ./cmd/jara        # Build main binary
go build ./...             # Build all packages
```

### Testing

```bash
go test ./...              # Run all tests
go test -v ./internal/...  # Verbose output
go vet ./...               # Check for vet issues
```

### Code Quality

```bash
gofmt -w ./internal ./cmd  # Auto-format
go mod tidy                # Clean up go.mod
```

---

## Future Enhancements

- Streaming updates instead of polling
- Write operations (scale applications, restart units)
- Configuration file support
- Custom color themes
- Filtering and search across views
- Detailed resource inspection panel
