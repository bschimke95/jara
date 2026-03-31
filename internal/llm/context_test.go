package llm

import (
	"context"
	"testing"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

func TestFormatStatusContext_Nil(t *testing.T) {
	got := FormatStatusContext(nil)
	if got != "No cluster status available." {
		t.Fatalf("expected no-status message, got: %s", got)
	}
}

func TestFormatStatusContext_HealthyCluster(t *testing.T) {
	status := &model.FullStatus{
		Model: model.ModelInfo{
			Name: "test-model", Cloud: "aws", Region: "us-east-1", Version: "3.6.0",
		},
		Applications: map[string]model.Application{
			"postgresql": {
				Name: "postgresql", Status: "active", Charm: "postgresql", Scale: 2,
				Units: []model.Unit{
					{Name: "postgresql/0", WorkloadStatus: "active", AgentStatus: "idle"},
					{Name: "postgresql/1", WorkloadStatus: "active", AgentStatus: "idle"},
				},
			},
		},
		Machines: map[string]model.Machine{
			"0": {ID: "0", Status: "started", DNSName: "10.0.0.1", Base: "ubuntu@22.04"},
		},
		Relations: []model.Relation{
			{
				ID: 1, Interface: "pgsql", Status: "joined",
				Endpoints: []model.Endpoint{
					{ApplicationName: "postgresql", Name: "db", Role: "provider"},
				},
			},
		},
	}

	got := FormatStatusContext(status)

	// Should contain model name.
	if !containsStr(got, "test-model") {
		t.Error("expected model name in output")
	}
	// Should contain app name.
	if !containsStr(got, "postgresql") {
		t.Error("expected application name in output")
	}
	// Healthy units should NOT be listed individually (only non-nominal).
	if containsStr(got, "postgresql/0") {
		t.Error("nominal units should be omitted from output")
	}
	// Should contain summary.
	if !containsStr(got, "1 applications") {
		t.Error("expected summary in output")
	}
}

func TestFormatStatusContext_BlockedUnit(t *testing.T) {
	status := &model.FullStatus{
		Model: model.ModelInfo{Name: "prod", Cloud: "gce", Region: "us-central1", Version: "3.5.0"},
		Applications: map[string]model.Application{
			"grafana": {
				Name: "grafana", Status: "blocked", Charm: "grafana", Scale: 1,
				StatusMessage: "missing relation",
				Units: []model.Unit{
					{
						Name: "grafana/0", WorkloadStatus: "blocked", AgentStatus: "idle",
						WorkloadMessage: "missing required relation",
					},
				},
			},
		},
		Machines:  map[string]model.Machine{},
		Relations: []model.Relation{},
	}

	got := FormatStatusContext(status)

	// Blocked unit should be listed.
	if !containsStr(got, "grafana/0") {
		t.Error("blocked unit should appear in output")
	}
	if !containsStr(got, "missing required relation") {
		t.Error("unit message should appear in output")
	}
}

func TestMockClient_StreamsResponse(t *testing.T) {
	client := NewMockClient(0) // instant
	ch, err := client.ChatStream(context.Background(), []Message{{Role: RoleUser, Content: "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var content string
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("channel closed without done event")
			}
			if ev.Err != nil {
				t.Fatalf("unexpected stream error: %v", ev.Err)
			}
			if ev.Done {
				if content == "" {
					t.Error("expected non-empty response")
				}
				return
			}
			content += ev.Delta
		case <-timeout:
			t.Fatal("mock stream timed out")
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && // avoid trivial false positives
		timeIndependentContains(s, sub)
}

func timeIndependentContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
