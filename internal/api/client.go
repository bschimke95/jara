package api

import (
	"context"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

// Client defines the interface for fetching Juju status.
type Client interface {
	Status(ctx context.Context) (*model.FullStatus, error)
	Controllers(ctx context.Context) ([]model.Controller, error)
	Models(ctx context.Context, controllerName string) ([]model.ModelSummary, error)
	DebugLog(ctx context.Context) (<-chan model.LogEntry, error)
	// WatchStatus starts a background loop that pushes status snapshots onto
	// the returned channel at the given interval. The stream runs until the
	// context is cancelled. On transient errors the implementation should
	// reconnect with backoff rather than closing the channel.
	WatchStatus(ctx context.Context, interval time.Duration) (<-chan StatusUpdate, error)
	// ScaleApplication adjusts the unit count for an application by delta
	// (positive to scale up, negative to scale down).
	ScaleApplication(ctx context.Context, appName string, delta int) error
	Close() error
}

// StatusUpdate carries either a successful status snapshot or an error from
// the watch loop. Consumers should check Err first.
type StatusUpdate struct {
	Status *model.FullStatus
	Err    error
}

// MockClient returns synthetic data for UI development.
type MockClient struct{}

// NewMockClient creates a new mock client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Close is a no-op for the mock client.
func (c *MockClient) Close() error { return nil }

// Models returns synthetic model data for the mock client.
func (c *MockClient) Models(_ context.Context, _ string) ([]model.ModelSummary, error) {
	return []model.ModelSummary{
		{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "uuid-0001", Current: true},
		{Name: "admin/staging", ShortName: "staging", Owner: "admin", Type: "iaas", UUID: "uuid-0002"},
	}, nil
}

// Controllers returns synthetic controller data.
func (c *MockClient) Controllers(_ context.Context) ([]model.Controller, error) {
	return []model.Controller{
		{Name: "prod-aws", Cloud: "aws", Region: "us-east-1", Addr: "10.0.0.1:17070", Version: "3.6.1", Status: "available", Models: 4, Machines: 12, HA: "3", Access: "superuser"},
		{Name: "staging-gce", Cloud: "gce", Region: "us-central1", Addr: "10.1.0.1:17070", Version: "3.6.0", Status: "available", Models: 2, Machines: 5, HA: "none", Access: "admin"},
		{Name: "dev-local", Cloud: "localhost", Region: "localhost", Addr: "127.0.0.1:17070", Version: "3.5.4", Status: "available", Models: 1, Machines: 3, HA: "none", Access: "superuser"},
	}, nil
}

// Status returns synthetic status data.
func (c *MockClient) Status(_ context.Context) (*model.FullStatus, error) {
	now := time.Now()
	fiveMinAgo := now.Add(-5 * time.Minute)
	tenMinAgo := now.Add(-10 * time.Minute)
	oneHourAgo := now.Add(-1 * time.Hour)

	return &model.FullStatus{
		Model: model.ModelInfo{
			Name:    "production",
			Cloud:   "aws",
			Region:  "us-east-1",
			Status:  "available",
			Type:    "iaas",
			Version: "3.6.1",
		},
		Applications: map[string]model.Application{
			"postgresql": {
				Name: "postgresql", Status: "active", StatusMessage: "Live master (14.12)",
				Charm: "postgresql", CharmChannel: "14/stable", CharmRev: 468,
				Scale: 3, Exposed: false, WorkloadVersion: "14.12",
				Base: "ubuntu@22.04", Since: &oneHourAgo,
				Units: []model.Unit{
					{Name: "postgresql/0", WorkloadStatus: "active", WorkloadMessage: "Live master (14.12)", AgentStatus: "idle", Machine: "0", PublicAddress: "10.0.1.10", Ports: []string{"5432/tcp"}, Leader: true, Since: &oneHourAgo},
					{Name: "postgresql/1", WorkloadStatus: "active", WorkloadMessage: "Live secondary (14.12)", AgentStatus: "idle", Machine: "1", PublicAddress: "10.0.1.11", Ports: []string{"5432/tcp"}, Since: &oneHourAgo},
					{Name: "postgresql/2", WorkloadStatus: "active", WorkloadMessage: "Live secondary (14.12)", AgentStatus: "idle", Machine: "2", PublicAddress: "10.0.1.12", Ports: []string{"5432/tcp"}, Since: &oneHourAgo},
				},
			},
			"ubuntu-app": {
				Name: "ubuntu-app", Status: "active", StatusMessage: "Ready",
				Charm: "ubuntu", CharmChannel: "stable", CharmRev: 24,
				Scale: 2, Exposed: true, WorkloadVersion: "1.0",
				Base: "ubuntu@22.04", Since: &tenMinAgo,
				Units: []model.Unit{
					{Name: "ubuntu-app/0", WorkloadStatus: "active", WorkloadMessage: "Ready", AgentStatus: "idle", Machine: "3", PublicAddress: "10.0.1.20", Ports: []string{"80/tcp", "443/tcp"}, Leader: true, Since: &tenMinAgo},
					{Name: "ubuntu-app/1", WorkloadStatus: "active", WorkloadMessage: "Ready", AgentStatus: "idle", Machine: "4", PublicAddress: "10.0.1.21", Ports: []string{"80/tcp", "443/tcp"}, Since: &tenMinAgo},
				},
			},
			"grafana": {
				Name: "grafana", Status: "blocked", StatusMessage: "Missing relation: database",
				Charm: "grafana-k8s", CharmChannel: "latest/stable", CharmRev: 106,
				Scale: 1, Exposed: false, Base: "ubuntu@22.04", Since: &fiveMinAgo,
				Units: []model.Unit{
					{Name: "grafana/0", WorkloadStatus: "blocked", WorkloadMessage: "Missing relation: database", AgentStatus: "idle", Machine: "5", PublicAddress: "10.0.1.30", Leader: true, Since: &fiveMinAgo},
				},
			},
			"prometheus": {
				Name: "prometheus", Status: "waiting", StatusMessage: "Waiting for relations",
				Charm: "prometheus-k8s", CharmChannel: "latest/stable", CharmRev: 171,
				Scale: 1, Exposed: false, WorkloadVersion: "2.47.0",
				Base: "ubuntu@22.04", Since: &fiveMinAgo,
				Units: []model.Unit{
					{Name: "prometheus/0", WorkloadStatus: "waiting", WorkloadMessage: "Waiting for relations", AgentStatus: "idle", Machine: "6", PublicAddress: "10.0.1.40", Ports: []string{"9090/tcp"}, Leader: true, Since: &fiveMinAgo},
				},
			},
			"nginx-ingress": {
				Name: "nginx-ingress", Status: "active", StatusMessage: "Ingress ready",
				Charm: "nginx-ingress-integrator", CharmChannel: "latest/stable", CharmRev: 95,
				Scale: 1, Exposed: true, WorkloadVersion: "1.9.0",
				Base: "ubuntu@22.04", Since: &oneHourAgo,
				Units: []model.Unit{
					{Name: "nginx-ingress/0", WorkloadStatus: "active", WorkloadMessage: "Ingress ready", AgentStatus: "idle", Machine: "7", PublicAddress: "10.0.1.50", Ports: []string{"80/tcp", "443/tcp"}, Leader: true, Since: &oneHourAgo},
				},
			},
		},
		Machines: map[string]model.Machine{
			"0": {ID: "0", Status: "started", DNSName: "ip-10-0-1-10.ec2.internal", IPAddresses: []string{"10.0.1.10"}, InstanceID: "i-0abc001", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=4 mem=16384M", Since: &oneHourAgo},
			"1": {ID: "1", Status: "started", DNSName: "ip-10-0-1-11.ec2.internal", IPAddresses: []string{"10.0.1.11"}, InstanceID: "i-0abc002", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=4 mem=16384M", Since: &oneHourAgo},
			"2": {ID: "2", Status: "started", DNSName: "ip-10-0-1-12.ec2.internal", IPAddresses: []string{"10.0.1.12"}, InstanceID: "i-0abc003", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=4 mem=16384M", Since: &oneHourAgo},
			"3": {ID: "3", Status: "started", DNSName: "ip-10-0-1-20.ec2.internal", IPAddresses: []string{"10.0.1.20"}, InstanceID: "i-0abc004", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=2 mem=8192M", Since: &tenMinAgo},
			"4": {ID: "4", Status: "started", DNSName: "ip-10-0-1-21.ec2.internal", IPAddresses: []string{"10.0.1.21"}, InstanceID: "i-0abc005", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=2 mem=8192M", Since: &tenMinAgo},
			"5": {ID: "5", Status: "started", DNSName: "ip-10-0-1-30.ec2.internal", IPAddresses: []string{"10.0.1.30"}, InstanceID: "i-0abc006", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=2 mem=4096M", Since: &fiveMinAgo},
			"6": {ID: "6", Status: "started", DNSName: "ip-10-0-1-40.ec2.internal", IPAddresses: []string{"10.0.1.40"}, InstanceID: "i-0abc007", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=2 mem=4096M", Since: &fiveMinAgo},
			"7": {ID: "7", Status: "started", DNSName: "ip-10-0-1-50.ec2.internal", IPAddresses: []string{"10.0.1.50"}, InstanceID: "i-0abc008", Base: "ubuntu@22.04", Hardware: "arch=amd64 cores=2 mem=4096M", Since: &oneHourAgo},
		},
		Relations: []model.Relation{
			{ID: 1, Key: "postgresql:db ubuntu-app:db", Interface: "pgsql", Status: "joined", Scope: "global", Endpoints: []model.Endpoint{{ApplicationName: "postgresql", Name: "db", Role: "provider"}, {ApplicationName: "ubuntu-app", Name: "db", Role: "requirer"}}},
			{ID: 2, Key: "prometheus:grafana-source grafana:grafana-source", Interface: "grafana-datasource", Status: "joined", Scope: "global", Endpoints: []model.Endpoint{{ApplicationName: "prometheus", Name: "grafana-source", Role: "provider"}, {ApplicationName: "grafana", Name: "grafana-source", Role: "requirer"}}},
			{ID: 3, Key: "nginx-ingress:ingress ubuntu-app:ingress", Interface: "ingress", Status: "joined", Scope: "global", Endpoints: []model.Endpoint{{ApplicationName: "nginx-ingress", Name: "ingress", Role: "provider"}, {ApplicationName: "ubuntu-app", Name: "ingress", Role: "requirer"}}},
		},
		FetchedAt: now,
	}, nil
}

// DebugLog returns a channel of synthetic log entries for UI development.
func (c *MockClient) DebugLog(ctx context.Context) (<-chan model.LogEntry, error) {
	ch := make(chan model.LogEntry)

	entities := []string{"unit-postgresql-0", "unit-ubuntu-app-0", "machine-0", "unit-grafana-0", "unit-prometheus-0"}
	severities := []string{"INFO", "DEBUG", "WARNING", "ERROR", "INFO", "INFO", "DEBUG", "INFO"}
	modules := []string{"juju.worker.uniter", "juju.state", "juju.apiserver", "juju.worker.provisioner", "juju.network"}
	messages := []string{
		"running hook: config-changed",
		"agent connected to controller",
		"hook failed: install",
		"relation joined: db",
		"instance started successfully",
		"updating agent config",
		"status changed: active",
		"collecting metrics",
		"sending heartbeat",
		"unit is ready",
		"container started",
		"storage attached",
	}

	go func() {
		defer close(ch)
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				entry := model.LogEntry{
					Entity:    entities[i%len(entities)],
					Timestamp: time.Now(),
					Severity:  severities[i%len(severities)],
					Module:    modules[i%len(modules)],
					Location:  "agent.go:42",
					Message:   messages[i%len(messages)],
				}
				select {
				case ch <- entry:
				case <-ctx.Done():
					return
				}
				i++
			}
		}
	}()

	return ch, nil
}

// WatchStatus returns a channel of synthetic status snapshots for UI development.
func (c *MockClient) WatchStatus(ctx context.Context, interval time.Duration) (<-chan StatusUpdate, error) {
	ch := make(chan StatusUpdate)

	go func() {
		defer close(ch)
		for {
			status, err := c.Status(ctx)
			update := StatusUpdate{Status: status, Err: err}
			select {
			case ch <- update:
			case <-ctx.Done():
				return
			}
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// ScaleApplication is a no-op for the mock client.
func (c *MockClient) ScaleApplication(_ context.Context, _ string, _ int) error { return nil }
