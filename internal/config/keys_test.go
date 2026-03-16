package config

import (
	"testing"

	"github.com/bschimke95/jara/internal/ui"
)

func TestResolveKeyMapDefaults(t *testing.T) {
	km := ResolveKeyMap(KeysConfig{})
	def := ui.DefaultKeyMap()

	// With no overrides, the result should match the default.
	if km.Quit.Keys() == nil {
		t.Error("Quit binding should have keys")
	}
	// Check that the default quit keys are preserved.
	defKeys := def.Quit.Keys()
	gotKeys := km.Quit.Keys()
	if len(defKeys) != len(gotKeys) {
		t.Errorf("Quit keys length = %d, want %d", len(gotKeys), len(defKeys))
	}
}

func TestResolveKeyMapOverrides(t *testing.T) {
	keys := KeysConfig{
		Quit: &KeyBindingConfig{
			Keys: []string{"ctrl+q"},
		},
		Help: &KeyBindingConfig{
			Keys: []string{"F1", "?"},
		},
	}

	km := ResolveKeyMap(keys)

	// Quit should be overridden.
	quitKeys := km.Quit.Keys()
	if len(quitKeys) != 1 || quitKeys[0] != "ctrl+q" {
		t.Errorf("Quit keys = %v, want [ctrl+q]", quitKeys)
	}

	// Help should be overridden.
	helpKeys := km.Help.Keys()
	if len(helpKeys) != 2 || helpKeys[0] != "F1" {
		t.Errorf("Help keys = %v, want [F1, ?]", helpKeys)
	}

	// Back should remain at default.
	def := ui.DefaultKeyMap()
	defBackKeys := def.Back.Keys()
	gotBackKeys := km.Back.Keys()
	if len(defBackKeys) != len(gotBackKeys) {
		t.Errorf("Back keys length = %d, want %d (default)", len(gotBackKeys), len(defBackKeys))
	}
}

func TestResolveKeyMapEmptyOverride(t *testing.T) {
	// Empty keys slice should not override.
	keys := KeysConfig{
		Quit: &KeyBindingConfig{
			Keys: []string{},
		},
	}

	km := ResolveKeyMap(keys)
	def := ui.DefaultKeyMap()

	// Should keep default since keys is empty.
	defKeys := def.Quit.Keys()
	gotKeys := km.Quit.Keys()
	if len(defKeys) != len(gotKeys) {
		t.Errorf("Quit keys with empty override = %v, want default %v", gotKeys, defKeys)
	}
}
