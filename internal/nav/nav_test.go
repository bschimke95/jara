package nav

import "testing"

func TestStack_Push(t *testing.T) {
	s := NewStack(ControllerView)
	if s.Depth() != 1 {
		t.Errorf("initial depth = %d, want 1", s.Depth())
	}

	s.Push(StackEntry{View: ApplicationsView})
	if s.Depth() != 2 {
		t.Errorf("depth after push = %d, want 2", s.Depth())
	}
}

func TestStack_Pop(t *testing.T) {
	s := NewStack(ControllerView)
	s.Push(StackEntry{View: ApplicationsView})

	entry, ok := s.Pop()
	if !ok {
		t.Error("Pop() should return true when stack has multiple entries")
	}
	if entry.View != ApplicationsView {
		t.Errorf("popped view = %v, want %v", entry.View, ApplicationsView)
	}
	if s.Depth() != 1 {
		t.Errorf("depth after pop = %d, want 1", s.Depth())
	}
}

func TestStack_PopSingleEntry(t *testing.T) {
	s := NewStack(ControllerView)
	_, ok := s.Pop()
	if ok {
		t.Error("Pop() should return false when only one entry remains")
	}
	if s.Depth() != 1 {
		t.Errorf("depth should remain 1, got %d", s.Depth())
	}
}

func TestStack_Current(t *testing.T) {
	s := NewStack(ControllerView)
	s.Push(StackEntry{View: ApplicationsView})

	current := s.Current()
	if current.View != ApplicationsView {
		t.Errorf("current view = %v, want %v", current.View, ApplicationsView)
	}
}

func TestStack_Breadcrumbs(t *testing.T) {
	s := NewStack(ControllerView)
	s.Push(StackEntry{View: ApplicationsView, Context: "postgresql"})
	s.Push(StackEntry{View: UnitsView})

	crumbs := s.Breadcrumbs()
	expected := []string{"Controllers", "Applications(postgresql)", "Units"}

	if len(crumbs) != len(expected) {
		t.Errorf("breadcrumbs length = %d, want %d", len(crumbs), len(expected))
	}

	for i, expected := range expected {
		if crumbs[i] != expected {
			t.Errorf("breadcrumb[%d] = %q, want %q", i, crumbs[i], expected)
		}
	}
}

func TestStack_WithContext(t *testing.T) {
	s := NewStack(ControllerView)
	s.Push(StackEntry{
		View:    ApplicationsView,
		Context: "postgresql",
	})

	current := s.Current()
	if current.Context != "postgresql" {
		t.Errorf("context = %q, want postgresql", current.Context)
	}
}

func TestStack_Navigation(t *testing.T) {
	s := NewStack(ControllerView)
	s.Push(StackEntry{View: ApplicationsView})
	s.Push(StackEntry{View: UnitsView})

	if s.Depth() != 3 {
		t.Errorf("depth = %d, want 3", s.Depth())
	}

	entry, ok := s.Pop()
	if !ok || entry.View != UnitsView {
		t.Errorf("Pop() = %v, %t; want %v, true", entry.View, ok, UnitsView)
	}

	if s.Depth() != 2 {
		t.Errorf("depth after pop = %d, want 2", s.Depth())
	}

	current := s.Current()
	if current.View != ApplicationsView {
		t.Errorf("current after pop = %v, want %v", current.View, ApplicationsView)
	}
}

func TestViewID_String(t *testing.T) {
	tests := []struct {
		view ViewID
		want string
	}{
		{ControllerView, "Controllers"},
		{ApplicationsView, "Applications"},
		{UnitsView, "Units"},
		{MachinesView, "Machines"},
		{RelationsView, "Relations"},
		{DebugLogView, "Debug Log"},
		{ModelsView, "Models"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.view.String()
			if got != tt.want {
				t.Errorf("ViewID.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want ViewID
		ok   bool
	}{
		{"controllers", ControllerView, true},
		{"ctrl", ControllerView, true},
		{"apps", ApplicationsView, true},
		{"units", UnitsView, true},
		{"machines", MachinesView, true},
		{"relations", RelationsView, true},
		{"log", DebugLogView, true},
		{"models", ModelsView, true},
		{"invalid", ViewID(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got, ok := ResolveCommand(tt.cmd)
			if ok != tt.ok {
				t.Errorf("ResolveCommand(%q) ok = %t, want %t", tt.cmd, ok, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Errorf("ResolveCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
