package app

import (
	"context"
	"testing"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/config"
	"github.com/bschimke95/jara/internal/model"
)

func TestDeployApplicationReadOnly(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Jara.ReadOnly = true

	m := Model{
		client: api.NewMockClient(),
		cfg:    cfg,
	}

	msg := m.deployApplication("", model.DeployOptions{CharmName: "redis-k8s", ApplicationName: "redis"})()
	if msg == nil {
		t.Fatal("expected error message in read-only mode")
	}
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
}

func TestDeployApplicationTargetsModel(t *testing.T) {
	client := api.NewMockClient()
	m := Model{
		client: client,
		cfg:    config.NewDefault(),
	}

	msg := m.deployApplication("admin/default", model.DeployOptions{CharmName: "redis-k8s", ApplicationName: "redis"})()
	if msg != nil {
		t.Fatalf("deploy command returned unexpected message: %T", msg)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if _, exists := status.Applications["redis"]; !exists {
		t.Fatal("expected redis application after deploy")
	}
}
