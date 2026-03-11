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
	Name            string
	Status          string
	StatusMessage   string
	Charm           string
	CharmChannel    string
	CharmRev        int
	Scale           int
	Exposed         bool
	WorkloadVersion string
	Base            string
	Since           *time.Time
	Units           []Unit
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

// FullStatus is the aggregate snapshot of a Juju model.
type FullStatus struct {
	Model        ModelInfo
	Applications map[string]Application
	Machines     map[string]Machine
	Relations    []Relation
	FetchedAt    time.Time
}
