// Package model defines the core domain types used throughout jara to represent
// Juju entities: controllers, models, applications, units, machines, relations,
// and log entries.
package model

import "time"

// Controller represents a Juju controller.
type Controller struct {
	Name     string
	Cloud    string
	Region   string
	Addr     string
	Version  string
	Status   string
	Models   int
	Machines int
	HA       string // e.g. "3" or "none"
	Access   string
}

// ModelSummary represents a Juju model listed under a controller.
type ModelSummary struct {
	Name      string
	ShortName string // unqualified name without owner prefix
	Owner     string
	Type      string
	UUID      string
	Current   bool // true if this is the currently selected model
}

// ModelInfo represents a Juju model within a controller.
type ModelInfo struct {
	Name    string
	Cloud   string
	Region  string
	Status  string
	Type    string
	Version string
}

// Application represents a deployed Juju application.
type Application struct {
	Name             string
	Status           string
	StatusMessage    string
	Charm            string
	CharmChannel     string
	CharmRev         int
	Scale            int
	Exposed          bool
	WorkloadVersion  string
	Base             string
	Since            *time.Time
	Units            []Unit
	EndpointBindings map[string]string // endpoint name → space binding
}

// Unit represents a single unit of a Juju application.
type Unit struct {
	Name            string
	WorkloadStatus  string
	WorkloadMessage string
	AgentStatus     string
	AgentMessage    string
	Machine         string
	PublicAddress   string
	Ports           []string
	Leader          bool
	Since           *time.Time
	Subordinates    []Unit
}

// Machine represents a machine provisioned in a Juju model.
type Machine struct {
	ID            string
	Status        string
	StatusMessage string
	DNSName       string
	IPAddresses   []string
	InstanceID    string
	Base          string
	Hardware      string
	Since         *time.Time
	Containers    []Machine
}

// Relation represents a relation (integration) between applications.
type Relation struct {
	ID        int
	Key       string
	Interface string
	Status    string
	Scope     string
	Endpoints []Endpoint
}

// Endpoint represents one side of a relation.
type Endpoint struct {
	ApplicationName string
	Name            string
	Role            string
}

// LogEntry represents a single structured log message from juju debug-log.
type LogEntry struct {
	ModelUUID string
	Entity    string
	Timestamp time.Time
	Severity  string
	Module    string
	Location  string
	Message   string
}

// DebugLogFilter holds the filtering parameters for a debug-log stream.
// Zero values mean "no filter" for each field.
type DebugLogFilter struct {
	// Level is the minimum log severity to include (e.g. "WARNING").
	// Empty string means all levels.
	Level string
	// Applications limits output to all units of these application names
	// (e.g. "postgresql"). Converted to "unit-appname-*" glob patterns for
	// the Juju API.
	Applications []string
	// IncludeEntities limits output to log lines from these entities
	// (e.g. "unit-postgresql-0", "machine-0").
	IncludeEntities []string
	// ExcludeEntities suppresses log lines from these entities.
	ExcludeEntities []string
	// IncludeModules limits output to these logger modules
	// (e.g. "juju.worker.uniter").
	IncludeModules []string
	// ExcludeModules suppresses log lines from these logger modules.
	ExcludeModules []string
	// IncludeLabels limits output to log lines carrying all of these labels (key=value).
	IncludeLabels map[string]string
	// ExcludeLabels suppresses log lines carrying any of these labels (key=value).
	ExcludeLabels map[string]string
	// Backlog is the number of historical lines to replay on connect.
	// Zero uses the implementation default (100).
	Backlog int
	// NoTail instructs the server to return only existing log entries
	// (the backlog) and then close the stream, rather than continuing
	// to tail new log lines. Use this when you want a bounded fetch.
	NoTail bool
}

// Secret represents a Juju secret and its metadata.
type Secret struct {
	URI            string
	Label          string
	Description    string
	Owner          string // e.g. "application-postgresql", "unit-mysql-0", "model"
	RotatePolicy   string // "never", "hourly", "daily", "weekly", "monthly", "quarterly", "yearly"
	Revision       int
	Backend        string
	AutoPrune      bool
	CreateTime     time.Time
	UpdateTime     time.Time
	ExpireTime     *time.Time
	NextRotateTime *time.Time
	Revisions      []SecretRevision
	Access         []SecretAccessInfo
}

// SecretRevision represents a single revision of a secret.
type SecretRevision struct {
	Revision  int
	CreatedAt time.Time
	ExpiredAt *time.Time
	Backend   string
}

// SecretAccessInfo describes who has access to a secret.
type SecretAccessInfo struct {
	Target string // e.g. "application-grafana"
	Scope  string // e.g. "relation-42"
	Role   string // e.g. "consume"
}

// FullStatus is the aggregate snapshot of a Juju model.
type FullStatus struct {
	Model               ModelInfo
	Applications        map[string]Application
	Machines            map[string]Machine
	Relations           []Relation
	Secrets             []Secret
	ControllerTimestamp *time.Time
	FetchedAt           time.Time
}

// DeployOptions captures supported Juju deploy settings collected from the UI.
type DeployOptions struct {
	CharmName       string
	ApplicationName string
	Channel         string
	Base            string
	Constraints     string
	Config          map[string]string
	NumUnits        *int
	Revision        *int
	Trust           bool
}

// RelationData holds the application and unit databag contents for a single
// relation. Each key in the maps is an application or unit name; values are
// key-value settings from the respective databag.
type RelationData struct {
	// ApplicationData maps application name → databag key/value pairs.
	ApplicationData map[string]map[string]string
	// UnitData maps unit name → databag key/value pairs.
	UnitData map[string]map[string]string
}

// Offer represents a Juju application offer (cross-model relation endpoint).
type Offer struct {
	Name            string
	ApplicationName string
	OfferURL        string
	CharmURL        string
	Endpoints       []string // endpoint names
	ActiveConnCount int
	TotalConnCount  int
}

// ConfigEntry represents a single application configuration entry.
type ConfigEntry struct {
	Key         string
	Value       string
	Default     string
	Source      string // "default", "user", "model"
	Type        string // "string", "int", "bool", "float"
	Description string
}

// ActionSpec describes an available action for an application's charm.
type ActionSpec struct {
	Name        string
	Description string
	Params      map[string]interface{} // JSON-Schema style parameter definitions
}

// ActionResult holds the outcome of a single action execution.
type ActionResult struct {
	ID        string
	Status    string // pending, running, completed, failed, cancelled
	Message   string
	Output    map[string]interface{}
	Enqueued  time.Time
	Started   time.Time
	Completed time.Time
}

// StorageInstance represents a Juju storage instance.
type StorageInstance struct {
	ID         string // e.g. "data/0"
	Kind       string // "block" or "filesystem"
	Owner      string // unit or app name owning this storage
	Status     string // "attached", "detaching", etc.
	Persistent bool
	Life       string // "alive", "dying", "dead"
	Pool       string // storage pool name (if known)
}

// CharmEndpoint describes a single endpoint from a charm's metadata.
type CharmEndpoint struct {
	Interface   string
	Role        string
	Description string
}
