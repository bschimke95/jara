package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestMockClient_InitialState(t *testing.T) {
	client := NewMockClient()

	// Check initial controllers
	controllers, err := client.Controllers(context.Background())
	if err != nil {
		t.Fatalf("Controllers() failed: %v", err)
	}
	if len(controllers) != 3 {
		t.Errorf("got %d controllers, want 3", len(controllers))
	}

	// Check initial models for prod-aws
	models, err := client.Models(context.Background(), "prod-aws")
	if err != nil {
		t.Fatalf("Models() failed: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("got %d models, want 2", len(models))
	}

	// Check default controller is selected
	controllerName := client.ControllerName()
	if controllerName != "prod-aws" {
		t.Errorf("controller name = %q, want %q", controllerName, "prod-aws")
	}
}

func TestMockClient_ControllerSelection(t *testing.T) {
	client := NewMockClient()

	// Switch to staging-gce
	err := client.SelectController("staging-gce")
	if err != nil {
		t.Fatalf("SelectController() failed: %v", err)
	}

	if client.ControllerName() != "staging-gce" {
		t.Errorf("controller name = %q, want %q", client.ControllerName(), "staging-gce")
	}

	// Check models are filtered for new controller
	models, err := client.Models(context.Background(), "staging-gce")
	if err != nil {
		t.Fatalf("Models() failed: %v", err)
	}

	if len(models) != 1 {
		t.Errorf("got %d models for staging-gce, want 1", len(models))
	}

	// Try invalid controller
	err = client.SelectController("nonexistent")
	if err == nil {
		t.Error("SelectController() should fail for nonexistent controller")
	}
}

func TestMockClient_ModelSelection(t *testing.T) {
	client := NewMockClient()

	// Select staging model (available on prod-aws)
	err := client.SelectModel("admin/staging")
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	// Check model status updates
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status.Model.Name != "production" {
		t.Errorf("model name = %q, want %q", status.Model.Name, "production")
	}

	// Try invalid model
	err = client.SelectModel("admin/nonexistent")
	if err == nil {
		t.Error("SelectModel() should fail for nonexistent model")
	}
}

func TestMockClient_ScaleApplication(t *testing.T) {
	client := NewMockClient()

	// Select model with applications
	err := client.SelectModel("admin/default")
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	// Check initial scale
	app, exists := status.Applications["postgresql"]
	if !exists {
		t.Fatal("postgresql app not found")
	}
	initialScale := app.Scale

	// Scale up by 2
	err = client.ScaleApplication(context.Background(), "postgresql", 2)
	if err != nil {
		t.Fatalf("ScaleApplication() failed: %v", err)
	}

	// Check new scale
	status, err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	app = status.Applications["postgresql"]
	if app.Scale != initialScale+2 {
		t.Errorf("scale = %d, want %d", app.Scale, initialScale+2)
	}

	// Check units were added
	if len(app.Units) != app.Scale {
		t.Errorf("got %d units, want %d", len(app.Units), app.Scale)
	}
}

func TestMockClient_ConcurrentAccess(t *testing.T) {
	client := NewMockClient()

	err := client.SelectModel("admin/default")
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Concurrent scaling operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.ScaleApplication(context.Background(), "postgresql", 1)
			if err != nil {
				errors <- fmt.Errorf("concurrent scale error: %v", err)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Final state should be consistent
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	app := status.Applications["postgresql"]
	if len(app.Units) != app.Scale {
		t.Errorf("units count = %d, want %d", len(app.Units), app.Scale)
	}
}

func TestMockClient_ErrorHandling(t *testing.T) {
	client := NewMockClient()

	// Test various error conditions
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "select invalid controller",
			fn:   func() error { return client.SelectController("invalid") },
		},
		{
			name: "select invalid model",
			fn:   func() error { return client.SelectModel("invalid") },
		},
		{
			name: "scale invalid app",
			fn: func() error {
				_ = client.SelectModel("admin/default")
				return client.ScaleApplication(context.Background(), "invalid", 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestMockClient_DeployApplication(t *testing.T) {
	client := NewMockClient()

	err := client.SelectModel("admin/default")
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	err = client.DeployApplication(context.Background(), model.DeployOptions{
		CharmName:       "redis-k8s",
		ApplicationName: "redis",
	})
	if err != nil {
		t.Fatalf("DeployApplication() failed: %v", err)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	app, exists := status.Applications["redis"]
	if !exists {
		t.Fatal("redis app not found after deploy")
	}
	if app.Charm != "redis-k8s" {
		t.Errorf("app charm = %q, want %q", app.Charm, "redis-k8s")
	}
	if app.Scale != 1 {
		t.Errorf("app scale = %d, want 1", app.Scale)
	}
	if len(app.Units) != 1 {
		t.Errorf("unit count = %d, want 1", len(app.Units))
	}
}

func TestMockClient_DeployApplicationErrors(t *testing.T) {
	client := NewMockClient()

	err := client.SelectModel("admin/default")
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	tests := []struct {
		name      string
		charmName string
		appName   string
	}{
		{name: "empty charm", charmName: "", appName: "foo"},
		{name: "duplicate app", charmName: "ubuntu", appName: "ubuntu-app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DeployApplication(context.Background(), model.DeployOptions{
				CharmName:       tt.charmName,
				ApplicationName: tt.appName,
			})
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestMockClient_CharmhubSuggestions(t *testing.T) {
	client := NewMockClient()

	all, err := client.CharmhubSuggestions(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("CharmhubSuggestions() failed: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("expected non-empty suggestions")
	}

	filtered, err := client.CharmhubSuggestions(context.Background(), "graf", 10)
	if err != nil {
		t.Fatalf("CharmhubSuggestions(filtered) failed: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatal("expected filtered suggestions")
	}
	for _, name := range filtered {
		if !containsIgnoreCase(name, "graf") {
			t.Fatalf("filtered suggestion %q does not match query", name)
		}
	}

	limited, err := client.CharmhubSuggestions(context.Background(), "", 2)
	if err != nil {
		t.Fatalf("CharmhubSuggestions(limited) failed: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("limited suggestions len = %d, want 2", len(limited))
	}
}

func containsIgnoreCase(s, q string) bool {
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(q))
}
