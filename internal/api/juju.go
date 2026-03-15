package api

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/client/client"
	"github.com/juju/juju/api/common"
	"github.com/juju/juju/api/connector"
	"github.com/juju/juju/api/jujuclient"
	corelogger "github.com/juju/juju/core/logger"
	"github.com/juju/juju/rpc/params"

	"github.com/bschimke95/jara/internal/model"
)

// nopLogger is a no-op implementation of core/logger.Logger.
type nopLogger struct{}

func (nopLogger) Criticalf(context.Context, string, ...any)                                 {}
func (nopLogger) Errorf(context.Context, string, ...any)                                    {}
func (nopLogger) Warningf(context.Context, string, ...any)                                  {}
func (nopLogger) Infof(context.Context, string, ...any)                                     {}
func (nopLogger) Debugf(context.Context, string, ...any)                                    {}
func (nopLogger) Tracef(context.Context, string, ...any)                                    {}
func (nopLogger) Logf(context.Context, corelogger.Level, corelogger.Labels, string, ...any) {}
func (nopLogger) IsLevelEnabled(corelogger.Level) bool                                      { return false }
func (n nopLogger) Child(string, ...string) corelogger.Logger                               { return n }
func (n nopLogger) GetChildByName(string) corelogger.Logger                                 { return n }
func (nopLogger) Helper()                                                                   {}

// JujuClient connects to a real Juju controller using the local client store.
type JujuClient struct {
	store          jujuclient.ClientStore
	controllerName string
	modelUUID      string
	conn           api.Connection
}

// JujuClientOption configures a JujuClient.
type JujuClientOption func(*JujuClient)

// WithController sets the controller name to connect to.
func WithController(name string) JujuClientOption {
	return func(c *JujuClient) {
		c.controllerName = name
	}
}

// WithModelUUID sets the model UUID to query status for.
func WithModelUUID(uuid string) JujuClientOption {
	return func(c *JujuClient) {
		c.modelUUID = uuid
	}
}

// NewJujuClient creates a new client backed by the real Juju API.
// It reads controller/account info from the local Juju client store
// (typically ~/.local/share/juju).
//
// If no controller name is provided via WithController, the current
// controller from the client store is used.
func NewJujuClient(opts ...JujuClientOption) (*JujuClient, error) {
	store := jujuclient.NewFileClientStore()

	c := &JujuClient{
		store: store,
	}
	for _, opt := range opts {
		opt(c)
	}

	// Default to the current controller if none specified.
	if c.controllerName == "" {
		name, err := store.CurrentController()
		if err != nil {
			return nil, fmt.Errorf("no current controller: %w", err)
		}
		c.controllerName = name
	}

	return c, nil
}

// Close closes any open API connections.
func (c *JujuClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SelectController switches the client to target a different controller
// and persists the selection to the local Juju client store so that
// subsequent juju CLI invocations also use the new controller.
func (c *JujuClient) SelectController(name string) error {
	if err := c.store.SetCurrentController(name); err != nil {
		return fmt.Errorf("persisting controller selection: %w", err)
	}
	c.controllerName = name
	c.modelUUID = "" // reset model so the new controller's current model is used
	return nil
}

// ControllerName returns the name of the currently targeted controller.
func (c *JujuClient) ControllerName() string {
	return c.controllerName
}

// SelectModel switches the client to target the given model (qualified name
// "owner/name") within the current controller and persists the selection.
func (c *JujuClient) SelectModel(qualifiedName string) error {
	if err := c.store.SetCurrentModel(c.controllerName, qualifiedName); err != nil {
		return fmt.Errorf("persisting model selection: %w", err)
	}
	c.modelUUID = "" // will be resolved lazily on next connect()
	return nil
}

// connect establishes an API connection to the configured controller and model.
// If no model UUID was explicitly provided, it resolves the current model
// from the client store so that the connection is model-scoped (required
// for the "Client" facade used by Status).
func (c *JujuClient) connect(ctx context.Context) (api.Connection, error) {
	modelUUID := c.modelUUID
	if modelUUID == "" {
		// Resolve the current model for this controller from the client store.
		modelName, err := c.store.CurrentModel(c.controllerName)
		if err != nil {
			return nil, fmt.Errorf("resolving current model for controller %q: %w", c.controllerName, err)
		}
		modelDetails, err := c.store.ModelByName(c.controllerName, modelName)
		if err != nil {
			return nil, fmt.Errorf("getting model details for %q: %w", modelName, err)
		}
		modelUUID = modelDetails.ModelUUID
	}

	cfg := connector.ClientStoreConfig{
		ControllerName: c.controllerName,
		ModelUUID:      modelUUID,
		ClientStore:    c.store,
	}

	cs, err := connector.NewClientStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connector: %w", err)
	}

	conn, err := cs.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to controller %q: %w", c.controllerName, err)
	}

	return conn, nil
}

// Controllers returns the list of controllers from the local client store.
// This does not require an API connection — it reads from the local
// Juju configuration files.
func (c *JujuClient) Controllers(_ context.Context) ([]model.Controller, error) {
	allControllers, err := c.store.AllControllers()
	if err != nil {
		return nil, fmt.Errorf("listing controllers: %w", err)
	}

	controllers := make([]model.Controller, 0, len(allControllers))
	for name, details := range allControllers {
		ctrl := model.Controller{
			Name:    name,
			Cloud:   details.Cloud,
			Region:  details.CloudRegion,
			Version: details.AgentVersion,
			Status:  "available",
		}

		// Use the first API endpoint as the address.
		if len(details.APIEndpoints) > 0 {
			ctrl.Addr = details.APIEndpoints[0]
		}

		// Derive HA status from controller machine count.
		if details.ControllerMachineCount > 1 {
			ctrl.HA = fmt.Sprintf("%d", details.ControllerMachineCount)
		} else {
			ctrl.HA = "none"
		}

		// Machine count.
		if details.MachineCount != nil {
			ctrl.Machines = *details.MachineCount
		}

		// Count models from the client store.
		models, err := c.store.AllModels(name)
		if err == nil {
			ctrl.Models = len(models)
		}

		// Account access.
		account, err := c.store.AccountDetails(name)
		if err == nil && account != nil {
			ctrl.Access = account.LastKnownAccess
		}

		controllers = append(controllers, ctrl)
	}

	// Sort controllers by name for stable output.
	sort.Slice(controllers, func(i, j int) bool {
		return controllers[i].Name < controllers[j].Name
	})

	return controllers, nil
}

// Models returns the list of models for the given controller from the local
// Juju client store. No API connection is required.
func (c *JujuClient) Models(_ context.Context, controllerName string) ([]model.ModelSummary, error) {
	allModels, err := c.store.AllModels(controllerName)
	if err != nil {
		return nil, fmt.Errorf("listing models for controller %q: %w", controllerName, err)
	}

	currentModel, _ := c.store.CurrentModel(controllerName)

	summaries := make([]model.ModelSummary, 0, len(allModels))
	for qualifiedName, details := range allModels {
		// qualifiedName is "owner/name".
		owner, shortName, _ := strings.Cut(qualifiedName, "/")
		summaries = append(summaries, model.ModelSummary{
			Name:      qualifiedName,
			ShortName: shortName,
			Owner:     owner,
			Type:      string(details.ModelType),
			UUID:      details.ModelUUID,
			Current:   qualifiedName == currentModel,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries, nil
}

// Status connects to the controller and fetches the full model status.
func (c *JujuClient) Status(ctx context.Context) (*model.FullStatus, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	statusClient := client.NewClient(conn, nopLogger{})
	result, err := statusClient.Status(ctx, &client.StatusArgs{
		Patterns: []string{},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching status: %w", err)
	}

	return convertFullStatus(result), nil
}

// DebugLog connects to the controller and streams debug log messages.
// The returned channel emits log entries until the context is cancelled
// or the connection is closed.
func (c *JujuClient) DebugLog(ctx context.Context) (<-chan model.LogEntry, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	jujuClient := client.NewClient(conn, nopLogger{})
	msgs, err := jujuClient.WatchDebugLog(ctx, common.DebugLogParams{
		Backlog: 100,
	})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("starting debug-log stream: %w", err)
	}

	// Convert the Juju LogMessage channel to our domain LogEntry channel.
	entries := make(chan model.LogEntry)
	go func() {
		defer close(entries)
		defer func() { _ = conn.Close() }()

		for msg := range msgs {
			entries <- model.LogEntry{
				ModelUUID: msg.ModelUUID,
				Entity:    msg.Entity,
				Timestamp: msg.Timestamp,
				Severity:  msg.Severity,
				Module:    msg.Module,
				Location:  msg.Location,
				Message:   msg.Message,
			}
		}
	}()

	return entries, nil
}

// WatchStatus opens a persistent connection and polls Status at the given
// interval, sending snapshots on the returned channel. On transient
// connection errors it reconnects with exponential backoff (1s → 30s cap).
// The stream runs until ctx is cancelled.
func (c *JujuClient) WatchStatus(ctx context.Context, interval time.Duration) (<-chan StatusUpdate, error) {
	// Establish the initial connection so callers get an immediate error if
	// the controller is unreachable.
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan StatusUpdate)

	go func() {
		defer close(ch)
		defer func() {
			if conn != nil {
				_ = conn.Close()
			}
		}()

		const (
			initialBackoff = 1 * time.Second
			maxBackoff     = 30 * time.Second
		)
		backoff := initialBackoff

		for {
			// Ensure we have a live connection.
			if conn == nil {
				conn, err = c.connect(ctx)
				if err != nil {
					log.Printf("WatchStatus: reconnect failed: %v (retrying in %s)", err, backoff)
					select {
					case ch <- StatusUpdate{Err: fmt.Errorf("reconnecting: %w", err)}:
					case <-ctx.Done():
						return
					}
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return
					}
					backoff = min(backoff*2, maxBackoff)
					continue
				}
				backoff = initialBackoff // reset on successful reconnect
			}

			// Fetch status on the persistent connection.
			statusClient := client.NewClient(conn, nopLogger{})
			fetchCtx, fetchCancel := context.WithTimeout(ctx, 10*time.Second)
			result, err := statusClient.Status(fetchCtx, &client.StatusArgs{
				Patterns: []string{},
			})
			fetchCancel()

			if err != nil {
				log.Printf("WatchStatus: status fetch failed: %v", err)
				// Connection likely broken — tear it down so we reconnect.
				_ = conn.Close()
				conn = nil
				select {
				case ch <- StatusUpdate{Err: fmt.Errorf("fetching status: %w", err)}:
				case <-ctx.Done():
					return
				}
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return
				}
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			backoff = initialBackoff // reset on success

			fs := convertFullStatus(result)
			fs.FetchedAt = time.Now()

			select {
			case ch <- StatusUpdate{Status: fs}:
			case <-ctx.Done():
				return
			}

			// Wait for the next tick.
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// convertFullStatus maps the Juju API params.FullStatus to our domain model.
func convertFullStatus(s *params.FullStatus) *model.FullStatus {
	fs := &model.FullStatus{}

	// Model info.
	fs.Model = convertModelInfo(s.Model)

	// Applications.
	fs.Applications = make(map[string]model.Application, len(s.Applications))
	for name, app := range s.Applications {
		fs.Applications[name] = convertApplication(name, app)
	}

	// Machines.
	fs.Machines = make(map[string]model.Machine, len(s.Machines))
	for id, m := range s.Machines {
		fs.Machines[id] = convertMachine(id, m)
	}

	// Relations.
	fs.Relations = make([]model.Relation, 0, len(s.Relations))
	for _, r := range s.Relations {
		fs.Relations = append(fs.Relations, convertRelation(r))
	}

	return fs
}

// convertModelInfo maps params.ModelStatusInfo to model.ModelInfo.
func convertModelInfo(m params.ModelStatusInfo) model.ModelInfo {
	cloud := m.CloudTag
	// Strip the "cloud-" prefix from the cloud tag.
	if after, ok := strings.CutPrefix(cloud, "cloud-"); ok {
		cloud = after
	}

	return model.ModelInfo{
		Name:    m.Name,
		Cloud:   cloud,
		Region:  m.CloudRegion,
		Status:  m.ModelStatus.Status,
		Type:    m.Type,
		Version: m.Version,
	}
}

// convertApplication maps params.ApplicationStatus to model.Application.
func convertApplication(name string, a params.ApplicationStatus) model.Application {
	app := model.Application{
		Name:            name,
		Status:          a.Status.Status,
		StatusMessage:   a.Status.Info,
		Charm:           a.Charm,
		CharmChannel:    a.CharmChannel,
		CharmRev:        a.CharmRev,
		Scale:           a.Scale,
		Exposed:         a.Exposed,
		WorkloadVersion: a.WorkloadVersion,
		Since:           a.Status.Since,
	}

	// Base: format as "name@channel" if available.
	if a.Base.Name != "" {
		app.Base = a.Base.Name + "@" + a.Base.Channel
	}

	// Convert units.
	app.Units = convertUnits(a.Units)

	// If scale is not set but we have units, use their count.
	if app.Scale == 0 && len(app.Units) > 0 {
		app.Scale = len(app.Units)
	}

	return app
}

// convertUnits maps a params unit map to a sorted slice of model.Unit.
func convertUnits(units map[string]params.UnitStatus) []model.Unit {
	result := make([]model.Unit, 0, len(units))
	for name, u := range units {
		result = append(result, convertUnit(name, u))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// convertUnit maps params.UnitStatus to model.Unit.
func convertUnit(name string, u params.UnitStatus) model.Unit {
	// For IAAS (machine) models the address is in PublicAddress;
	// for CAAS (k8s) models it is in Address. Use whichever is set.
	addr := u.PublicAddress
	if addr == "" {
		addr = u.Address
	}

	unit := model.Unit{
		Name:            name,
		WorkloadStatus:  u.WorkloadStatus.Status,
		WorkloadMessage: u.WorkloadStatus.Info,
		AgentStatus:     u.AgentStatus.Status,
		AgentMessage:    u.AgentStatus.Info,
		Machine:         u.Machine,
		PublicAddress:   addr,
		Ports:           u.OpenedPorts,
		Leader:          u.Leader,
		Since:           u.WorkloadStatus.Since,
	}

	// Convert subordinates recursively.
	if len(u.Subordinates) > 0 {
		unit.Subordinates = convertUnits(u.Subordinates)
	}

	return unit
}

// convertMachine maps params.MachineStatus to model.Machine.
func convertMachine(id string, m params.MachineStatus) model.Machine {
	machine := model.Machine{
		ID:            id,
		Status:        m.AgentStatus.Status,
		StatusMessage: m.AgentStatus.Info,
		DNSName:       m.DNSName,
		IPAddresses:   m.IPAddresses,
		InstanceID:    string(m.InstanceId),
		Hardware:      m.Hardware,
		Since:         m.AgentStatus.Since,
	}

	// Base: format as "name@channel" if available.
	if m.Base.Name != "" {
		machine.Base = m.Base.Name + "@" + m.Base.Channel
	}

	// Convert containers.
	if len(m.Containers) > 0 {
		machine.Containers = make([]model.Machine, 0, len(m.Containers))
		for cID, container := range m.Containers {
			machine.Containers = append(machine.Containers, convertMachine(cID, container))
		}
		sort.Slice(machine.Containers, func(i, j int) bool {
			return machine.Containers[i].ID < machine.Containers[j].ID
		})
	}

	return machine
}

// convertRelation maps params.RelationStatus to model.Relation.
func convertRelation(r params.RelationStatus) model.Relation {
	endpoints := make([]model.Endpoint, 0, len(r.Endpoints))
	for _, ep := range r.Endpoints {
		endpoints = append(endpoints, model.Endpoint{
			ApplicationName: ep.ApplicationName,
			Name:            ep.Name,
			Role:            string(ep.Role),
		})
	}

	return model.Relation{
		ID:        r.Id,
		Key:       r.Key,
		Interface: r.Interface,
		Status:    r.Status.Status,
		Scope:     r.Scope,
		Endpoints: endpoints,
	}
}
