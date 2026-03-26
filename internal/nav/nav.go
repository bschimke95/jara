package nav

import (
	"sort"
	"strings"
)

// ViewID identifies a view type.
type ViewID int

const (
	ControllerView ViewID = iota
	ModelView
	ApplicationsView
	UnitsView
	MachinesView
	RelationsView
	DebugLogView
	ModelsView
	SecretsView
	SecretDetailView
)

// String returns the human-readable name of the view.
func (v ViewID) String() string {
	switch v {
	case ControllerView:
		return "Controllers"
	case ModelView:
		return "Model"
	case ApplicationsView:
		return "Applications"
	case UnitsView:
		return "Units"
	case MachinesView:
		return "Machines"
	case RelationsView:
		return "Relations"
	case DebugLogView:
		return "Debug Log"
	case ModelsView:
		return "Models"
	case SecretsView:
		return "Secrets"
	case SecretDetailView:
		return "Secret"
	default:
		return "Unknown"
	}
}

// CommandAliases maps command strings to view IDs.
var CommandAliases = map[string]ViewID{
	"controllers":  ControllerView,
	"controller":   ControllerView,
	"ctrl":         ControllerView,
	"model":        ModelView,
	"mod":          ModelView,
	"applications": ApplicationsView,
	"app":          ApplicationsView,
	"apps":         ApplicationsView,
	"units":        UnitsView,
	"unit":         UnitsView,
	"machines":     MachinesView,
	"machine":      MachinesView,
	"mach":         MachinesView,
	"relations":    RelationsView,
	"relation":     RelationsView,
	"rel":          RelationsView,
	"debug-log":    DebugLogView,
	"debuglog":     DebugLogView,
	"log":          DebugLogView,
	"logs":         DebugLogView,
	"models":       ModelsView,
	"model-list":   ModelsView,
	"secrets":      SecretsView,
	"secret":       SecretsView,
	"sec":          SecretsView,
}

// ResolveCommand looks up a command string and returns the matching ViewID.
func ResolveCommand(cmd string) (ViewID, bool) {
	v, ok := CommandAliases[cmd]
	return v, ok
}

// CommandMatch represents a command suggestion with its canonical name and target.
type CommandMatch struct {
	Command string
	Target  ViewID
}

// MatchCommands returns all commands that start with the given prefix,
// deduplicated by target view and sorted alphabetically. Built-in commands
// like "quit" are included.
func MatchCommands(prefix string) []CommandMatch {
	if prefix == "" {
		return nil
	}
	prefix = strings.ToLower(prefix)
	seen := make(map[ViewID]bool)
	var matches []CommandMatch
	// Collect unique matches by target.
	// Use a sorted list of keys for deterministic output.
	keys := make([]string, 0, len(CommandAliases))
	for k := range CommandAliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, cmd := range keys {
		target := CommandAliases[cmd]
		if strings.HasPrefix(cmd, prefix) && !seen[target] {
			seen[target] = true
			matches = append(matches, CommandMatch{Command: cmd, Target: target})
		}
	}
	// Built-in commands.
	if strings.HasPrefix("quit", prefix) || strings.HasPrefix("q", prefix) {
		matches = append(matches, CommandMatch{Command: "quit"})
	}
	return matches
}

// Stack implements a simple navigation stack (view history).
type Stack struct {
	entries []StackEntry
}

// StackEntry records a view and optional context.
type StackEntry struct {
	View    ViewID
	Context string
}

// NewStack creates a stack with the initial view.
func NewStack(initial ViewID) *Stack {
	return &Stack{
		entries: []StackEntry{{View: initial}},
	}
}

// Push adds a view to the stack.
func (s *Stack) Push(entry StackEntry) {
	s.entries = append(s.entries, entry)
}

// Pop removes and returns the top entry. Returns false if only one entry remains.
func (s *Stack) Pop() (StackEntry, bool) {
	if len(s.entries) <= 1 {
		return StackEntry{}, false
	}
	top := s.entries[len(s.entries)-1]
	s.entries = s.entries[:len(s.entries)-1]
	return top, true
}

// Current returns the current (top) entry.
func (s *Stack) Current() StackEntry {
	return s.entries[len(s.entries)-1]
}

// Breadcrumbs returns the display names of all entries in the stack.
func (s *Stack) Breadcrumbs() []string {
	crumbs := make([]string, len(s.entries))
	for i, e := range s.entries {
		name := e.View.String()
		if e.Context != "" {
			name += "(" + e.Context + ")"
		}
		crumbs[i] = name
	}
	return crumbs
}

// Reset replaces the stack with a single entry, discarding all history.
func (s *Stack) Reset(entry StackEntry) {
	s.entries = []StackEntry{entry}
}

// Depth returns the current stack depth.
func (s *Stack) Depth() int {
	return len(s.entries)
}
