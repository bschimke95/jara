package deploymodal

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/ui"
)

func TestParseConfigMap(t *testing.T) {
	cfg, err := parseConfigMap("foo=bar, baz = qux")
	if err != nil {
		t.Fatalf("parseConfigMap() error: %v", err)
	}
	if cfg["foo"] != "bar" {
		t.Errorf("foo = %q, want %q", cfg["foo"], "bar")
	}
	if cfg["baz"] != "qux" {
		t.Errorf("baz = %q, want %q", cfg["baz"], "qux")
	}
}

func TestParseConfigMapInvalid(t *testing.T) {
	_, err := parseConfigMap("invalid")
	if err == nil {
		t.Fatal("expected error for invalid config entry")
	}
}

func TestOptionsParsesNumericFields(t *testing.T) {
	m := New("", ui.DefaultKeyMap(), nil, nil)
	m.charm = "postgresql"
	m.numUnits = "3"
	m.revision = "12"
	m.constraints = "cores=2 mem=4G"
	m.config = "profile=production"
	m.trust = true

	opts, err := m.options()
	if err != nil {
		t.Fatalf("options() error: %v", err)
	}
	if opts.CharmName != "postgresql" {
		t.Errorf("charm = %q, want %q", opts.CharmName, "postgresql")
	}
	if opts.NumUnits == nil || *opts.NumUnits != 3 {
		t.Fatalf("num units = %v, want 3", opts.NumUnits)
	}
	if opts.Revision == nil || *opts.Revision != 12 {
		t.Fatalf("revision = %v, want 12", opts.Revision)
	}
	if opts.Constraints != "cores=2 mem=4G" {
		t.Errorf("constraints = %q", opts.Constraints)
	}
	if opts.Config["profile"] != "production" {
		t.Errorf("config profile = %q, want %q", opts.Config["profile"], "production")
	}
	if !opts.Trust {
		t.Error("expected trust=true")
	}
}

func TestRefreshAutocompleteForCharmField(t *testing.T) {
	m := New("", ui.DefaultKeyMap(), []string{"postgresql", "prometheus", "mysql"}, nil)
	m.leftCursor = fieldCharm
	m.input.SetValue("p")
	m.refreshAutocomplete()

	if len(m.autocomplete) != 2 {
		t.Fatalf("autocomplete len = %d, want 2", len(m.autocomplete))
	}
	if m.autocomplete[0] != "postgresql" || m.autocomplete[1] != "prometheus" {
		t.Fatalf("unexpected autocomplete: %v", m.autocomplete)
	}
}

func TestEditingTabAcceptsSuggestion(t *testing.T) {
	m := New("", ui.DefaultKeyMap(), []string{"postgresql", "prometheus"}, nil)
	m.leftCursor = fieldCharm
	m.focus = focusRight
	m.editing = true
	m.input.SetValue("prom")
	m.refreshAutocomplete()

	updated, _ := m.Update(tea.KeyPressMsg{Text: "tab", Code: tea.KeyTab})
	modal, ok := updated.(*Modal)
	if !ok {
		t.Fatalf("updated type = %T, want *Modal", updated)
	}
	if modal.input.Value() != "prometheus" {
		t.Fatalf("input value = %q, want %q", modal.input.Value(), "prometheus")
	}
}

func TestFilteredSuggestionsPrefixFirst(t *testing.T) {
	m := New("", ui.DefaultKeyMap(), []string{"x-postgresql", "postgresql", "my-postgresql"}, nil)
	m.leftCursor = fieldCharm
	m.input.SetValue("post")

	got := m.filteredSuggestions()
	if len(got) != 3 {
		t.Fatalf("filteredSuggestions len = %d, want 3", len(got))
	}
	if got[0] != "postgresql" {
		t.Fatalf("first suggestion = %q, want prefix match %q", got[0], "postgresql")
	}
}
