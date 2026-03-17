# jara

A terminal user interface for monitoring and managing Juju environments.
Jara gives you a live, keyboard-driven overview of controllers, models,
applications, units, machines, and relations without leaving the terminal.

## Features

- Live status streaming with sub-second refresh
- vim-style navigation (j/k, g/G, Ctrl-u/d)
- Multi-level navigation stack: controllers > models > model overview > detail views
- Colour-coded Juju status indicators (active, blocked, waiting, error, ...)
- Debug log streaming with inline search and a two-pane filter modal
- Application scaling (+ / -)
- Configurable colour theme and key bindings via a YAML config file
- k9s-inspired header, crumb bar, and footer hint layout

## Installation

### Snap

    sudo snap install jara

After installing, grant jara access to your local Juju credentials:

    snap connect jara:juju-client-observe

### From source

    go install github.com/bschimke95/jara@latest

Or clone and build:

    git clone https://github.com/bschimke95/jara
    cd jara
    make build        # produces ./jara
    make install      # installs to $GOPATH/bin

Requires Go 1.25 or later and a reachable Juju controller.

## Usage

    jara [flags]

    Flags:
      --config string      path to config file (default: $XDG_CONFIG_HOME/jara/config.yaml)
      --refresh duration   status poll interval (default: 2s)
      --logLevel string    log level: debug, info, warn, error (default: warn)
      --logFile string     write logs to this file instead of stderr
      --readonly           disable write operations (scale, etc.)
      --command string     run a single command and exit

    Subcommands:
      jara version         print version information
      jara info            show config, skin, and log file paths

### Key bindings

| Key       | Action                        |
|-----------|-------------------------------|
| j / k     | move down / up                |
| g / G     | jump to top / bottom          |
| Ctrl-u/d  | page up / down                |
| enter     | select / drill down           |
| esc       | go back                       |
| :         | open command prompt           |
| U         | units view for selected app   |
| R         | relations view                |
| L         | debug log (filtered to app)   |
| l         | debug log (all)               |
| + / -     | scale application up / down   |
| F         | open debug log filter modal   |
| D         | clear debug log filter        |
| /         | search within debug log       |
| n / N     | next / previous search match  |
| q         | quit                          |

All key bindings can be remapped in the config file.

## Configuration

Jara follows the XDG Base Directory specification.  The default config
path is `$XDG_CONFIG_HOME/jara/config.yaml` (usually
`~/.config/jara/config.yaml`).  Override it with `--config` or the
`JARA_CONFIG_DIR` environment variable.

A fully annotated example is provided in `config.example.yaml`.

### Theme

    jara:
      ui:
        skin:
          primary:             "#00bfff"
          highlight:           "#1d4ed8"
          error:               "#ff0000"
          checkGreen:          "#00ff00"
          checkRed:            "#ff5555"
          searchHighlightFg:   "#000000"
          searchHighlightBg:   "#ffff00"

### Remapping keys

    jara:
      ui:
        keys:
          quit:       "q"
          filterOpen: "F"
          searchOpen: "/"

## Requirements

- A Juju 3.x controller accessible from the machine running jara
- `juju` CLI credentials already in place (`juju login` / `~/.local/share/juju`)

## Development

    make build        # compile
    make test         # unit tests
    make lint         # golangci-lint
    make fmt          # gofumpt
    make vet          # go vet

The codebase follows standard Go project layout under `internal/`:

    internal/
      api/        Juju client interface and real + mock implementations
      app/        Root Bubble Tea model and update dispatch
      color/      Global theme color variables and status-color helpers
      cmd/        Cobra CLI entry points (root, version, info)
      config/     YAML config loading, theme resolution, key-binding merge
      model/      Domain types (FullStatus, Application, Unit, ...)
      nav/        Navigation stack
      render/     Table row/column builders
      ui/         KeyMap, chrome (header, crumb bar, footer, border boxes)
      view/       Individual TUI views

## License

Apache 2.0 - see [LICENSE](LICENSE).
