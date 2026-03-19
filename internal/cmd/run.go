package cmd

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"k8s.io/klog/v2"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/app"
	"github.com/bschimke95/jara/internal/config"
	"github.com/spf13/cobra"
)

func run(_ *cobra.Command, _ []string) error {
	// 1. Load configuration from file.
	cfg := config.NewDefault()
	if err := cfg.Load(*jaraFlags.ConfigFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// 2. Override with CLI flags.
	cfg.Override(jaraFlags)

	// 3. Set up logging.
	logPath := cfg.Jara.LogFile
	if logPath == "" {
		logPath = config.DefaultLogFile()
	}
	cleanup, err := setupLogging(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not set up logging to %s: %v\n", logPath, err)
		log.SetOutput(io.Discard)
	} else {
		defer cleanup()
	}

	// 4. Resolve theme and key bindings from config.
	theme := config.ResolveTheme(cfg.Jara.UI.Skin)
	keys := config.ResolveKeyMap(cfg.Jara.UI.Keys)

	// 5. Connect to Juju.
	client, err := api.NewJujuClient(api.WithCharmhubURL(cfg.Jara.CharmhubURL))
	if err != nil {
		return fmt.Errorf("creating Juju client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// 6. Build and run the TUI.
	m := app.New(client, app.WithTheme(theme), app.WithKeyMap(keys), app.WithConfig(cfg))
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

// setupLogging redirects all log output (Go's standard logger, klog, and
// stderr) into the given file so that noisy library warnings don't corrupt
// the Bubble Tea TUI. Returns a cleanup function that must be deferred.
func setupLogging(path string) (cleanup func(), err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	// Redirect Go's standard logger.
	log.SetOutput(f)

	// Silence klog (Kubernetes client-go) by sending its output to the file
	// and suppressing stderr.
	klogFlags := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(klogFlags)
	_ = klogFlags.Set("logtostderr", "false")
	_ = klogFlags.Set("stderrthreshold", "FATAL")
	klog.SetOutput(f)

	// Redirect the process stderr so any other C or Go library writing
	// directly to fd 2 goes to the log file instead of the terminal.
	origStderr := os.Stderr
	os.Stderr = f

	return func() {
		os.Stderr = origStderr
		klog.Flush()
		_ = f.Close()
	}, nil
}
