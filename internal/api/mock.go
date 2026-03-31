package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

// MockClient is a fully stateful in-memory client for testing and UI
// development. All state mutations (scaling, controller/model selection) are
// reflected in subsequent Status() calls.
type MockClient struct {
	mu sync.Mutex

	controllerName string
	currentModel   string

	controllers []model.Controller
	// models keyed by controller name.
	models map[string][]model.ModelSummary
	// status is the mutable model status snapshot.
	status *model.FullStatus
	// nextMachineID tracks the next machine ID to allocate.
	nextMachineID int
}

// NewMockClient creates a new mock client with synthetic data.
func NewMockClient() *MockClient {
	c := &MockClient{
		controllerName: "prod-aws",
		currentModel:   "admin/default",
		controllers: []model.Controller{
			{Name: "prod-aws", Cloud: "aws", Region: "us-east-1", Addr: "10.0.0.1:17070", Version: "3.6.1", Status: "available", Models: 4, Machines: 12, HA: "3", Access: "superuser"},
			{Name: "staging-gce", Cloud: "gce", Region: "us-central1", Addr: "10.1.0.1:17070", Version: "3.6.0", Status: "available", Models: 2, Machines: 5, HA: "none", Access: "admin"},
			{Name: "dev-local", Cloud: "localhost", Region: "localhost", Addr: "127.0.0.1:17070", Version: "3.5.4", Status: "available", Models: 1, Machines: 3, HA: "none", Access: "superuser"},
		},
		models: map[string][]model.ModelSummary{
			"prod-aws": {
				{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "uuid-0001", Current: true},
				{Name: "admin/staging", ShortName: "staging", Owner: "admin", Type: "iaas", UUID: "uuid-0002"},
			},
			"staging-gce": {
				{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "uuid-0003", Current: true},
			},
			"dev-local": {
				{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "uuid-0004", Current: true},
			},
		},
		nextMachineID: 8, // machines 0-7 already allocated
	}
	c.status = c.buildInitialStatus()
	return c
}

// Close is a no-op for the mock client.
func (c *MockClient) Close() error { return nil }

// ControllerName returns the name of the currently targeted controller.
func (c *MockClient) ControllerName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.controllerName
}

// SelectController switches the mock to target a different controller.
func (c *MockClient) SelectController(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ctrl := range c.controllers {
		if ctrl.Name == name {
			c.controllerName = name
			c.currentModel = ""
			return nil
		}
	}
	return fmt.Errorf("controller %q not found", name)
}

// SelectModel switches the mock to target the given model.
func (c *MockClient) SelectModel(qualifiedName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	models, ok := c.models[c.controllerName]
	if !ok {
		return fmt.Errorf("no models for controller %q", c.controllerName)
	}
	for _, m := range models {
		if m.Name == qualifiedName {
			c.currentModel = qualifiedName
			return nil
		}
	}
	return fmt.Errorf("model %q not found on controller %q", qualifiedName, c.controllerName)
}

// Controllers returns the synthetic controller list.
func (c *MockClient) Controllers(_ context.Context) ([]model.Controller, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Return a copy.
	out := make([]model.Controller, len(c.controllers))
	copy(out, c.controllers)
	return out, nil
}

// Models returns the synthetic model list for the given controller.
func (c *MockClient) Models(_ context.Context, controllerName string) ([]model.ModelSummary, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	models, ok := c.models[controllerName]
	if !ok {
		return nil, fmt.Errorf("controller %q not found", controllerName)
	}
	out := make([]model.ModelSummary, len(models))
	copy(out, models)
	// Mark the current model.
	for i := range out {
		out[i].Current = out[i].Name == c.currentModel
	}
	return out, nil
}

// Status returns a snapshot of the current mutable status.
func (c *MockClient) Status(_ context.Context) (*model.FullStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cloneStatus(), nil
}

// ListSecrets returns the secrets from the current status.
func (c *MockClient) ListSecrets(_ context.Context) ([]model.Secret, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.cloneStatus()
	return s.Secrets, nil
}

// RevealSecret returns synthetic decoded content for a mock secret.
func (c *MockClient) RevealSecret(_ context.Context, uri string, _ int) (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sec := range c.status.Secrets {
		if sec.URI == uri {
			switch sec.Label {
			case "db-password":
				return map[string]string{
					"username": "postgres",
					"password": "super-secret-pg-pass-42",
				}, nil
			case "api-key":
				return map[string]string{
					"api-key": "gf_sa_1a2b3c4d5e6f7g8h9i0j",
				}, nil
			case "tls-cert":
				return map[string]string{
					"certificate": "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJAL...",
					"private-key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBg...",
				}, nil
			default:
				return map[string]string{"value": "mock-secret-value"}, nil
			}
		}
	}
	return nil, fmt.Errorf("secret %q not found", uri)
}

// ScaleApplication adjusts the unit count for an application by delta.
// Positive delta adds new units; negative delta removes from the tail.
func (c *MockClient) ScaleApplication(_ context.Context, appName string, delta int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	app, ok := c.status.Applications[appName]
	if !ok {
		return fmt.Errorf("application %q not found", appName)
	}

	newScale := app.Scale + delta
	if newScale < 0 {
		return fmt.Errorf("cannot scale %q below 0 (current: %d, delta: %d)", appName, app.Scale, delta)
	}

	if delta > 0 {
		// Add units.
		for i := range delta {
			unitIdx := len(app.Units) + i
			machineID := fmt.Sprintf("%d", c.nextMachineID)
			c.nextMachineID++

			now := time.Now()
			unit := model.Unit{
				Name:            fmt.Sprintf("%s/%d", appName, unitIdx),
				WorkloadStatus:  "waiting",
				WorkloadMessage: "installing agent",
				AgentStatus:     "allocating",
				Machine:         machineID,
				PublicAddress:   fmt.Sprintf("10.0.2.%d", unitIdx+100),
				Since:           &now,
			}
			app.Units = append(app.Units, unit)

			// Add a corresponding machine.
			c.status.Machines[machineID] = model.Machine{
				ID:          machineID,
				Status:      "started",
				DNSName:     fmt.Sprintf("ip-10-0-2-%d.ec2.internal", unitIdx+100),
				IPAddresses: []string{fmt.Sprintf("10.0.2.%d", unitIdx+100)},
				InstanceID:  fmt.Sprintf("i-mock%s", machineID),
				Base:        app.Base,
				Hardware:    "arch=amd64 cores=2 mem=4096M",
				Since:       &now,
			}
		}
	} else if delta < 0 {
		// Remove units from the tail.
		removeCount := -delta
		if removeCount > len(app.Units) {
			removeCount = len(app.Units)
		}
		// Remove associated machines.
		for _, u := range app.Units[len(app.Units)-removeCount:] {
			delete(c.status.Machines, u.Machine)
		}
		app.Units = app.Units[:len(app.Units)-removeCount]
	}

	app.Scale = newScale
	c.status.Applications[appName] = app
	return nil
}

// DeployApplication adds a new synthetic application to the current status.
func (c *MockClient) DeployApplication(_ context.Context, opts model.DeployOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if opts.CharmName == "" {
		return fmt.Errorf("charm name cannot be empty")
	}
	charmName := opts.CharmName
	appName := opts.ApplicationName
	if appName == "" {
		appName = charmName
	}
	if _, exists := c.status.Applications[appName]; exists {
		return fmt.Errorf("application %q already exists", appName)
	}

	now := time.Now()
	machineID := fmt.Sprintf("%d", c.nextMachineID)
	addressSuffix := c.nextMachineID + 100
	c.nextMachineID++

	newApp := model.Application{
		Name:          appName,
		Status:        "waiting",
		StatusMessage: "deploying charm",
		Charm:         charmName,
		CharmChannel:  "stable",
		Scale:         1,
		Exposed:       false,
		Base:          "ubuntu@22.04",
		Since:         &now,
		Units: []model.Unit{
			{
				Name:            fmt.Sprintf("%s/0", appName),
				WorkloadStatus:  "waiting",
				WorkloadMessage: "deploying charm",
				AgentStatus:     "allocating",
				Machine:         machineID,
				PublicAddress:   fmt.Sprintf("10.0.3.%d", addressSuffix),
				Since:           &now,
			},
		},
	}
	c.status.Applications[appName] = newApp

	c.status.Machines[machineID] = model.Machine{
		ID:          machineID,
		Status:      "started",
		DNSName:     fmt.Sprintf("ip-10-0-3-%s.ec2.internal", machineID),
		IPAddresses: []string{fmt.Sprintf("10.0.3.%d", addressSuffix)},
		InstanceID:  fmt.Sprintf("i-mock%s", machineID),
		Base:        newApp.Base,
		Hardware:    "arch=amd64 cores=2 mem=4096M",
		Since:       &now,
	}

	return nil
}

// CharmhubSuggestions returns synthetic charm names for autocomplete.
func (c *MockClient) CharmhubSuggestions(_ context.Context, query string, limit int) ([]string, error) {
	base := []string{
		"postgresql",
		"postgresql-k8s",
		"mysql",
		"redis-k8s",
		"prometheus-k8s",
		"grafana-k8s",
		"traefik-k8s",
		"nginx-ingress-integrator",
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]string, 0, len(base))
	for _, name := range base {
		if q == "" || strings.Contains(strings.ToLower(name), q) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CharmRelationInfo returns synthetic charm endpoint metadata.
func (c *MockClient) CharmRelationInfo(_ context.Context, charmName string) (map[string]model.CharmEndpoint, error) {
	data := map[string]map[string]model.CharmEndpoint{
		"postgresql": {
			"db":           {Interface: "pgsql", Role: "provider", Description: "Provides the pgsql database interface"},
			"db-admin":     {Interface: "pgsql", Role: "provider", Description: "Provides administrative database access"},
			"replication":  {Interface: "pgdata", Role: "peer", Description: "Internal replication between units"},
			"certificates": {Interface: "tls-certificates", Role: "requirer", Description: "TLS certificates for encryption"},
			"monitoring":   {Interface: "prometheus-scrape", Role: "provider", Description: "Exposes Prometheus metrics endpoint"},
		},
		"ubuntu": {
			"db":      {Interface: "pgsql", Role: "requirer", Description: "Requires a database connection"},
			"ingress": {Interface: "ingress", Role: "requirer", Description: "Requires an ingress endpoint"},
			"logging": {Interface: "loki-push", Role: "requirer", Description: "Push logs to a Loki aggregator"},
		},
		"grafana-k8s": {
			"grafana-source": {Interface: "grafana-datasource", Role: "requirer", Description: "Receives data sources for dashboards"},
			"database":       {Interface: "pgsql", Role: "requirer", Description: "Requires database for persistence"},
			"ingress":        {Interface: "ingress", Role: "requirer", Description: "Requires an ingress endpoint"},
		},
		"prometheus-k8s": {
			"grafana-source":   {Interface: "grafana-datasource", Role: "provider", Description: "Provides Prometheus as a Grafana datasource"},
			"metrics-endpoint": {Interface: "prometheus-scrape", Role: "requirer", Description: "Scrapes metrics from workloads"},
			"ingress":          {Interface: "ingress", Role: "requirer", Description: "Requires an ingress endpoint"},
		},
		"nginx-ingress-integrator": {
			"ingress":       {Interface: "ingress", Role: "provider", Description: "Provides ingress routing for applications"},
			"ingress-proxy": {Interface: "ingress", Role: "provider", Description: "Provides proxy-mode ingress routing"},
		},
	}
	if eps, ok := data[charmName]; ok {
		return eps, nil
	}
	return nil, nil
}

// RelateApplications adds a synthetic relation between two endpoints.
func (c *MockClient) RelateApplications(_ context.Context, endpointA, endpointB string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	parseEndpoint := func(ep string) (string, string) {
		if i := strings.IndexByte(ep, ':'); i >= 0 {
			return ep[:i], ep[i+1:]
		}
		return ep, ep
	}

	appA, nameA := parseEndpoint(endpointA)
	appB, nameB := parseEndpoint(endpointB)

	if _, ok := c.status.Applications[appA]; !ok {
		return fmt.Errorf("application %q not found", appA)
	}
	if _, ok := c.status.Applications[appB]; !ok {
		return fmt.Errorf("application %q not found", appB)
	}

	nextID := len(c.status.Relations) + 1
	c.status.Relations = append(c.status.Relations, model.Relation{
		ID:        nextID,
		Key:       endpointA + " " + endpointB,
		Interface: nameA,
		Status:    "joined",
		Scope:     "global",
		Endpoints: []model.Endpoint{
			{ApplicationName: appA, Name: nameA, Role: "provider"},
			{ApplicationName: appB, Name: nameB, Role: "requirer"},
		},
	})
	return nil
}

// DestroyRelation removes a relation matching the given endpoints.
func (c *MockClient) DestroyRelation(_ context.Context, endpointA, endpointB string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, r := range c.status.Relations {
		if len(r.Endpoints) >= 2 {
			e0 := r.Endpoints[0].ApplicationName + ":" + r.Endpoints[0].Name
			e1 := r.Endpoints[1].ApplicationName + ":" + r.Endpoints[1].Name
			if (e0 == endpointA && e1 == endpointB) || (e0 == endpointB && e1 == endpointA) {
				c.status.Relations = append(c.status.Relations[:i], c.status.Relations[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("relation %q <-> %q not found", endpointA, endpointB)
}

// RelationData returns synthetic databag contents for a relation.
func (c *MockClient) RelationData(_ context.Context, relationID int) (*model.RelationData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var rel *model.Relation
	for i := range c.status.Relations {
		if c.status.Relations[i].ID == relationID {
			rel = &c.status.Relations[i]
			break
		}
	}
	if rel == nil {
		return nil, fmt.Errorf("relation %d not found", relationID)
	}

	result := &model.RelationData{
		ApplicationData: make(map[string]map[string]string),
		UnitData:        make(map[string]map[string]string),
	}

	for _, ep := range rel.Endpoints {
		// Synthetic application databag.
		appData := map[string]string{
			"interface": rel.Interface,
			"endpoint":  ep.Name,
		}
		if ep.Role == "provider" {
			appData["version"] = "1"
			appData["host"] = "10.0.1.10"
		}
		result.ApplicationData[ep.ApplicationName] = appData

		// Synthetic unit databags.
		if app, ok := c.status.Applications[ep.ApplicationName]; ok {
			for _, u := range app.Units {
				ud := map[string]string{
					"ingress-address": u.PublicAddress,
					"egress-subnets":  u.PublicAddress + "/32",
				}
				if u.Leader {
					ud["leader"] = "true"
				}
				result.UnitData[u.Name] = ud
			}
		}
	}

	return result, nil
}

// DebugLog returns a channel of synthetic log entries.
func (c *MockClient) DebugLog(ctx context.Context, _ model.DebugLogFilter) (<-chan model.LogEntry, error) {
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

	// Fixed base time so VHS golden files remain stable across runs.
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

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
					Timestamp: baseTime.Add(time.Duration(i) * 500 * time.Millisecond),
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

// WatchStatus returns a channel of status snapshots at the given interval.
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

// mockEpoch is the fixed reference time used for all mock status timestamps so
// that VHS golden files are deterministic across runs.
var mockEpoch = time.Date(2026, 3, 26, 20, 0, 0, 0, time.UTC)

// buildInitialStatus creates the initial synthetic status data.
func (c *MockClient) buildInitialStatus() *model.FullStatus {
	now := mockEpoch
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
				EndpointBindings: map[string]string{"db": "", "db-admin": "", "replication": "", "certificates": "", "monitoring": ""},
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
				EndpointBindings: map[string]string{"db": "", "ingress": "", "logging": ""},
				Units: []model.Unit{
					{Name: "ubuntu-app/0", WorkloadStatus: "active", WorkloadMessage: "Ready", AgentStatus: "idle", Machine: "3", PublicAddress: "10.0.1.20", Ports: []string{"80/tcp", "443/tcp"}, Leader: true, Since: &tenMinAgo},
					{Name: "ubuntu-app/1", WorkloadStatus: "active", WorkloadMessage: "Ready", AgentStatus: "idle", Machine: "4", PublicAddress: "10.0.1.21", Ports: []string{"80/tcp", "443/tcp"}, Since: &tenMinAgo},
				},
			},
			"grafana": {
				Name: "grafana", Status: "blocked", StatusMessage: "Missing relation: database",
				Charm: "grafana-k8s", CharmChannel: "latest/stable", CharmRev: 106,
				Scale: 1, Exposed: false, Base: "ubuntu@22.04", Since: &fiveMinAgo,
				EndpointBindings: map[string]string{"grafana-source": "", "database": "", "ingress": ""},
				Units: []model.Unit{
					{Name: "grafana/0", WorkloadStatus: "blocked", WorkloadMessage: "Missing relation: database", AgentStatus: "idle", Machine: "5", PublicAddress: "10.0.1.30", Leader: true, Since: &fiveMinAgo},
				},
			},
			"prometheus": {
				Name: "prometheus", Status: "waiting", StatusMessage: "Waiting for relations",
				Charm: "prometheus-k8s", CharmChannel: "latest/stable", CharmRev: 171,
				Scale: 1, Exposed: false, WorkloadVersion: "2.47.0",
				Base: "ubuntu@22.04", Since: &fiveMinAgo,
				EndpointBindings: map[string]string{"grafana-source": "", "metrics-endpoint": "", "ingress": ""},
				Units: []model.Unit{
					{Name: "prometheus/0", WorkloadStatus: "waiting", WorkloadMessage: "Waiting for relations", AgentStatus: "idle", Machine: "6", PublicAddress: "10.0.1.40", Ports: []string{"9090/tcp"}, Leader: true, Since: &fiveMinAgo},
				},
			},
			"nginx-ingress": {
				Name: "nginx-ingress", Status: "active", StatusMessage: "Ingress ready",
				Charm: "nginx-ingress-integrator", CharmChannel: "latest/stable", CharmRev: 95,
				Scale: 1, Exposed: true, WorkloadVersion: "1.9.0",
				Base: "ubuntu@22.04", Since: &oneHourAgo,
				EndpointBindings: map[string]string{"ingress": "", "ingress-proxy": ""},
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
		Secrets: []model.Secret{
			{
				URI: "secret:9m4e2mr0ui3e8a215n4g", Label: "db-password", Description: "PostgreSQL superuser password",
				Owner: "application-postgresql", RotatePolicy: "monthly", Revision: 3,
				Backend: "internal", AutoPrune: true, CreateTime: oneHourAgo, UpdateTime: fiveMinAgo,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: oneHourAgo, ExpiredAt: &tenMinAgo, Backend: "internal"},
					{Revision: 2, CreatedAt: tenMinAgo, ExpiredAt: &fiveMinAgo, Backend: "internal"},
					{Revision: 3, CreatedAt: fiveMinAgo, Backend: "internal"},
				},
				Access: []model.SecretAccessInfo{
					{Target: "application-ubuntu-app", Scope: "relation-1", Role: "consume"},
				},
			},
			{
				URI: "secret:7h3k5p9q2w1x8z6y4v0t", Label: "api-key", Description: "Grafana admin API key",
				Owner: "application-grafana", RotatePolicy: "never", Revision: 1,
				Backend: "internal", AutoPrune: false, CreateTime: oneHourAgo, UpdateTime: oneHourAgo,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: oneHourAgo, Backend: "internal"},
				},
				Access: []model.SecretAccessInfo{
					{Target: "application-prometheus", Scope: "relation-2", Role: "consume"},
				},
			},
			{
				URI: "secret:2b4n6m8k0j3h5f7d9s1a", Label: "tls-cert", Description: "TLS certificate for ingress",
				Owner: "application-nginx-ingress", RotatePolicy: "quarterly", Revision: 2,
				Backend: "internal", AutoPrune: true, CreateTime: oneHourAgo, UpdateTime: tenMinAgo,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: oneHourAgo, ExpiredAt: &tenMinAgo, Backend: "internal"},
					{Revision: 2, CreatedAt: tenMinAgo, Backend: "internal"},
				},
				Access: []model.SecretAccessInfo{
					{Target: "application-ubuntu-app", Scope: "relation-3", Role: "consume"},
					{Target: "application-grafana", Scope: "relation-2", Role: "consume"},
				},
			},
		},
		FetchedAt: now,
	}
}

// cloneStatus returns a deep copy of the current status so callers can't
// mutate the mock's internal state.
func (c *MockClient) cloneStatus() *model.FullStatus {
	s := c.status

	apps := make(map[string]model.Application, len(s.Applications))
	for k, app := range s.Applications {
		units := make([]model.Unit, len(app.Units))
		copy(units, app.Units)
		app.Units = units
		apps[k] = app
	}

	machines := make(map[string]model.Machine, len(s.Machines))
	for k, m := range s.Machines {
		ips := make([]string, len(m.IPAddresses))
		copy(ips, m.IPAddresses)
		m.IPAddresses = ips
		machines[k] = m
	}

	relations := make([]model.Relation, len(s.Relations))
	for i, r := range s.Relations {
		eps := make([]model.Endpoint, len(r.Endpoints))
		copy(eps, r.Endpoints)
		r.Endpoints = eps
		relations[i] = r
	}

	secrets := make([]model.Secret, len(s.Secrets))
	for i, sec := range s.Secrets {
		revs := make([]model.SecretRevision, len(sec.Revisions))
		copy(revs, sec.Revisions)
		sec.Revisions = revs
		access := make([]model.SecretAccessInfo, len(sec.Access))
		copy(access, sec.Access)
		sec.Access = access
		secrets[i] = sec
	}

	return &model.FullStatus{
		Model:        s.Model,
		Applications: apps,
		Machines:     machines,
		Relations:    relations,
		Secrets:      secrets,
		FetchedAt:    time.Now(),
	}
}
