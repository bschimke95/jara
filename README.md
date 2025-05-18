# Jara

[![Go Report Card](https://goreportcard.com/badge/github.com/bschimke95/jara)](https://goreportcard.com/report/github.com/bschimke95/jara)

A terminal-based Juju client for managing your Juju models and applications with a modern, user-friendly interface.

> ⚠️ **Note:** This project is currently in active development and not yet ready for ~production~ any use.

## Features

- View and manage Juju models
- Monitor application status
- Navigate through your Juju environment with ease
- Clean, intuitive terminal interface

## Installation

### Prerequisites

- Go 1.21 or later
- Juju 3.x installed and configured

### From Source

1. Clone the repository:

   ```bash
   git clone https://github.com/bschimke95/jara.git
   cd jara
   ```

2. Build and run:

   ```bash
   go run cmd/main.go
   ```

> **Note:** Snap package support is planned but not yet available.

## Development

### Building

To build the project:

```bash
go build -o jara ./cmd/main.go
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - A powerful little TUI framework
- [Juju](https://juju.is/) - The open source application modeling tool
