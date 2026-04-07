package relations

import (
	"strings"
	"testing"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/view"
)

func TestRenderDatabagPane_NilData(t *testing.T) {
	result := renderDatabagPane(nil, nil, 60, 20, color.DefaultStyles())
	if !strings.Contains(result, "Select a relation") {
		t.Error("expected placeholder text for nil data")
	}
}

func TestRenderDatabagPane_WithData(t *testing.T) {
	rel := &model.Relation{
		ID:        1,
		Interface: "pgsql",
		Endpoints: []model.Endpoint{
			{ApplicationName: "pg", Name: "db", Role: "provider"},
			{ApplicationName: "app", Name: "db", Role: "requirer"},
		},
	}
	rd := &model.RelationData{
		ApplicationData: map[string]map[string]string{
			"pg":  {"version": "14", "host": "10.0.0.1"},
			"app": {"version": "1"},
		},
		UnitData: map[string]map[string]string{
			"pg/0":  {"ingress-address": "10.0.0.1", "leader": "true"},
			"pg/1":  {"ingress-address": "10.0.0.2"},
			"app/0": {"ingress-address": "10.0.1.1"},
		},
	}

	result := renderDatabagPane(rd, rel, 80, 40, color.DefaultStyles())
	if !strings.Contains(result, "Databags") {
		t.Error("expected 'Databags' outer title")
	}
	if !strings.Contains(result, "Application Data") {
		t.Error("expected 'Application Data' box title")
	}
	if !strings.Contains(result, "Unit Data") {
		t.Error("expected 'Unit Data' box title")
	}
	if !strings.Contains(result, "pg") {
		t.Error("expected 'pg' application name")
	}
	if !strings.Contains(result, "10.0.0.1") {
		t.Error("expected ingress address in output")
	}
}

func TestColoredBorderBox(t *testing.T) {
	box := coloredBorderBox("hello", "Title", 30, appColors[0])
	if !strings.Contains(box, "Title") {
		t.Error("expected title in colored border box")
	}
	if !strings.Contains(box, "hello") {
		t.Error("expected content in colored border box")
	}
}

func TestRenderDatabagPane_PerUnitBoxes(t *testing.T) {
	rel := &model.Relation{
		ID:        2,
		Interface: "http",
		Endpoints: []model.Endpoint{
			{ApplicationName: "web", Name: "http", Role: "provider"},
		},
	}
	rd := &model.RelationData{
		ApplicationData: map[string]map[string]string{
			"web": {"port": "80"},
		},
		UnitData: map[string]map[string]string{
			"web/0": {"ingress-address": "10.0.0.1", "leader": "true"},
			"web/1": {"ingress-address": "10.0.0.2"},
		},
	}

	result := renderDatabagPane(rd, rel, 80, 40, color.DefaultStyles())
	// Each unit should appear in its own box.
	if strings.Count(result, "web/0") < 1 {
		t.Error("expected web/0 in its own unit box")
	}
	if strings.Count(result, "web/1") < 1 {
		t.Error("expected web/1 in its own unit box")
	}
}

func TestSortedKV(t *testing.T) {
	data := map[string]string{
		"beta":  "2",
		"alpha": "1",
		"gamma": "3",
	}
	pairs := sortedKV(data, 40)
	if len(pairs) != 3 {
		t.Fatalf("sortedKV() returned %d pairs, want 3", len(pairs))
	}
	// Should be sorted by key.
	if !strings.HasPrefix(pairs[0].key, "alpha") {
		t.Errorf("first key = %q, want alpha", pairs[0].key)
	}
	if !strings.HasPrefix(pairs[1].key, "beta") {
		t.Errorf("second key = %q, want beta", pairs[1].key)
	}
}

func TestPadToHeight(t *testing.T) {
	content := "line1\nline2"
	result := view.PadToHeight(content, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("padToHeight() produced %d lines, want 5", len(lines))
	}
}
