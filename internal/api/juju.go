package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/client/action"
	"github.com/juju/juju/api/client/application"
	"github.com/juju/juju/api/client/applicationoffers"
	"github.com/juju/juju/api/client/client"
	jujuSecrets "github.com/juju/juju/api/client/secrets"
	jujuStorage "github.com/juju/juju/api/client/storage"
	"github.com/juju/juju/api/common"
	"github.com/juju/juju/api/connector"
	"github.com/juju/juju/api/jujuclient"
	"github.com/juju/juju/core/base"
	"github.com/juju/juju/core/constraints"
	corelogger "github.com/juju/juju/core/logger"
	"github.com/juju/juju/core/relation"
	coreSecrets "github.com/juju/juju/core/secrets"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/loggo/v2"
	"github.com/juju/names/v6"
	"gopkg.in/yaml.v3"

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
	mu             sync.RWMutex
	store          jujuclient.ClientStore
	controllerName string
	modelUUID      string
	conn           api.Connection
	charmhubURL    string
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

// WithCharmhubURL sets the Charmhub API base URL used for suggestion lookups.
func WithCharmhubURL(rawURL string) JujuClientOption {
	return func(c *JujuClient) {
		c.charmhubURL = strings.TrimSpace(rawURL)
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
		store:       store,
		charmhubURL: "https://api.charmhub.io",
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

type charmhubFindResponse struct {
	Results []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"results"`
}

// CharmhubSuggestions queries Charmhub for charm names used by deploy autocomplete.
func (c *JujuClient) CharmhubSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.charmhubURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("charmhub URL is empty")
	}

	endpoint, err := url.Parse(baseURL + "/v2/charms/find")
	if err != nil {
		return nil, fmt.Errorf("parsing charmhub URL: %w", err)
	}
	params := endpoint.Query()
	params.Set("fields", "name,type")
	if q := strings.TrimSpace(query); q != "" {
		params.Set("q", q)
	}
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building charmhub request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying charmhub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("charmhub returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload charmhubFindResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding charmhub response: %w", err)
	}

	seen := make(map[string]struct{}, len(payload.Results))
	out := make([]string, 0, len(payload.Results))
	for _, item := range payload.Results {
		if item.Type != "" && item.Type != "charm" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// charmhubInfoResponse mirrors the Charmhub /v2/charms/info response for
// the relations field.
type charmhubInfoResponse struct {
	Result struct {
		Relations struct {
			Provides map[string]charmhubRelation `json:"provides"`
			Requires map[string]charmhubRelation `json:"requires"`
			Peers    map[string]charmhubRelation `json:"peers"`
		} `json:"relations"`
	} `json:"result"`
}

type charmhubRelation struct {
	Interface   string `json:"interface"`
	Description string `json:"description"`
}

// CharmRelationInfo queries Charmhub for a charm's endpoint metadata.
func (c *JujuClient) CharmRelationInfo(ctx context.Context, charmName string) (map[string]model.CharmEndpoint, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.charmhubURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("charmhub URL is empty")
	}

	endpoint, err := url.Parse(baseURL + "/v2/charms/info/" + url.PathEscape(charmName))
	if err != nil {
		return nil, fmt.Errorf("parsing charmhub URL: %w", err)
	}
	params := endpoint.Query()
	params.Set("fields", "result.relations")
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building charmhub info request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying charmhub info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("charmhub info returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload charmhubInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding charmhub info: %w", err)
	}

	result := make(map[string]model.CharmEndpoint)
	for name, r := range payload.Result.Relations.Provides {
		result[name] = model.CharmEndpoint{Interface: r.Interface, Role: "provider", Description: r.Description}
	}
	for name, r := range payload.Result.Relations.Requires {
		result[name] = model.CharmEndpoint{Interface: r.Interface, Role: "requirer", Description: r.Description}
	}
	for name, r := range payload.Result.Relations.Peers {
		result[name] = model.CharmEndpoint{Interface: r.Interface, Role: "peer", Description: r.Description}
	}
	return result, nil
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.store.SetCurrentController(name); err != nil {
		return fmt.Errorf("persisting controller selection: %w", err)
	}
	c.controllerName = name
	c.modelUUID = "" // reset model so the new controller's current model is used
	return nil
}

// ControllerName returns the name of the currently targeted controller.
func (c *JujuClient) ControllerName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.controllerName
}

// SelectModel switches the client to target the given model (qualified name
// "owner/name") within the current controller and persists the selection.
func (c *JujuClient) SelectModel(qualifiedName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
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
	c.mu.RLock()
	modelUUID := c.modelUUID
	controllerName := c.controllerName
	c.mu.RUnlock()

	if modelUUID == "" {
		// Resolve the current model for this controller from the client store.
		modelName, err := c.store.CurrentModel(controllerName)
		if err != nil {
			return nil, fmt.Errorf("resolving current model for controller %q: %w: %w", controllerName, err, ErrNoSelectedModel)
		}
		modelDetails, err := c.store.ModelByName(controllerName, modelName)
		if err != nil {
			return nil, fmt.Errorf("getting model details for %q: %w", modelName, err)
		}
		modelUUID = modelDetails.ModelUUID
	}

	cfg := connector.ClientStoreConfig{
		ControllerName: controllerName,
		ModelUUID:      modelUUID,
		ClientStore:    c.store,
	}

	cs, err := connector.NewClientStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connector: %w", err)
	}

	conn, err := cs.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to controller %q: %w", controllerName, err)
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
	defer func() { _ = conn.Close() }()

	statusClient := client.NewClient(conn, nopLogger{})
	result, err := statusClient.Status(ctx, &client.StatusArgs{
		Patterns: []string{},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching status: %w", err)
	}

	return convertFullStatus(result), nil
}

// ScaleApplication adjusts the unit count for an application by delta
// (positive to scale up, negative to scale down).
//
// For CAAS (Kubernetes) models this uses the ScaleApplication API.
// For IAAS (machine) models it uses AddUnits / DestroyUnits instead,
// because the ScaleApplication API is only supported on container models.
func (c *JujuClient) ScaleApplication(ctx context.Context, appName string, delta int) error {
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	modelType, err := c.currentModelType()
	if err != nil {
		return fmt.Errorf("determining model type: %w", err)
	}

	appClient := application.NewClient(conn)

	if modelType == "caas" {
		_, err = appClient.ScaleApplication(ctx, application.ScaleApplicationParams{
			ApplicationName: appName,
			ScaleChange:     delta,
		})
		if err != nil {
			return fmt.Errorf("scaling %q by %+d: %w", appName, delta, err)
		}
		return nil
	}

	// IAAS model: use AddUnits / DestroyUnits.
	if delta > 0 {
		_, err = appClient.AddUnits(ctx, application.AddUnitsParams{
			ApplicationName: appName,
			NumUnits:        delta,
		})
		if err != nil {
			return fmt.Errorf("adding %d unit(s) to %q: %w", delta, appName, err)
		}
		return nil
	}

	// Scale down: fetch status to identify which units to remove.
	statusClient := client.NewClient(conn, nopLogger{})
	result, err := statusClient.Status(ctx, &client.StatusArgs{
		Patterns: []string{},
	})
	if err != nil {
		return fmt.Errorf("fetching status for scale-down: %w", err)
	}

	app, ok := result.Applications[appName]
	if !ok {
		return fmt.Errorf("application %q not found", appName)
	}

	// Collect unit names and sort so we remove the highest-numbered first.
	unitNames := make([]string, 0, len(app.Units))
	for name := range app.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)

	removeCount := -delta
	if removeCount > len(unitNames) {
		removeCount = len(unitNames)
	}
	toDestroy := unitNames[len(unitNames)-removeCount:]

	_, err = appClient.DestroyUnits(ctx, application.DestroyUnitsParams{
		Units: toDestroy,
	})
	if err != nil {
		return fmt.Errorf("removing %d unit(s) from %q: %w", -delta, appName, err)
	}
	return nil
}

// DeployApplication deploys a charm from the configured repository into the
// current model.
func (c *JujuClient) DeployApplication(ctx context.Context, opts model.DeployOptions) error {
	if strings.TrimSpace(opts.CharmName) == "" {
		return fmt.Errorf("charm name cannot be empty")
	}

	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	appClient := application.NewClient(conn)
	arg := application.DeployFromRepositoryArg{
		CharmName:       strings.TrimSpace(opts.CharmName),
		ApplicationName: strings.TrimSpace(opts.ApplicationName),
		ConfigYAML:      "",
		Trust:           opts.Trust,
	}
	if opts.Channel != "" {
		channel := strings.TrimSpace(opts.Channel)
		arg.Channel = &channel
	}
	if opts.Base != "" {
		parsedBase, parseErr := base.ParseBaseFromString(strings.TrimSpace(opts.Base))
		if parseErr != nil {
			return fmt.Errorf("parsing base %q: %w", opts.Base, parseErr)
		}
		arg.Base = &parsedBase
	}
	if opts.Constraints != "" {
		cons, parseErr := constraints.Parse(strings.TrimSpace(opts.Constraints))
		if parseErr != nil {
			return fmt.Errorf("parsing constraints %q: %w", opts.Constraints, parseErr)
		}
		arg.Cons = cons
	}
	if opts.NumUnits != nil {
		units := *opts.NumUnits
		arg.NumUnits = &units
	}
	if opts.Revision != nil {
		rev := *opts.Revision
		arg.Revision = &rev
	}
	if len(opts.Config) > 0 {
		configData, marshalErr := yaml.Marshal(opts.Config)
		if marshalErr != nil {
			return fmt.Errorf("marshalling config: %w", marshalErr)
		}
		arg.ConfigYAML = string(configData)
	}

	_, _, errs := appClient.DeployFromRepository(ctx, arg)
	if len(errs) > 0 {
		messages := make([]string, 0, len(errs))
		for _, deployErr := range errs {
			if deployErr == nil {
				continue
			}
			messages = append(messages, deployErr.Error())
		}
		if len(messages) > 0 {
			return fmt.Errorf("deploying charm %q: %s", opts.CharmName, strings.Join(messages, "; "))
		}
	}
	return nil
}

// RelateApplications adds a relation between two endpoints using the
// Application facade's AddRelation call.
func (c *JujuClient) RelateApplications(ctx context.Context, endpointA, endpointB string) error {
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	appClient := application.NewClient(conn)
	_, err = appClient.AddRelation(ctx, []string{endpointA, endpointB}, nil)
	if err != nil {
		return fmt.Errorf("adding relation %q <-> %q: %w", endpointA, endpointB, err)
	}
	return nil
}

// DestroyRelation removes a relation between two endpoints using the
// Application facade's DestroyRelation call.
func (c *JujuClient) DestroyRelation(ctx context.Context, endpointA, endpointB string) error {
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	appClient := application.NewClient(conn)
	err = appClient.DestroyRelation(ctx, nil, nil, endpointA, endpointB)
	if err != nil {
		return fmt.Errorf("removing relation %q <-> %q: %w", endpointA, endpointB, err)
	}
	return nil
}

// RelationData fetches application and unit databag contents for a relation.
// It works by finding units involved in the relation (from the current status)
// and calling UnitsInfo to retrieve their relation settings.
func (c *JujuClient) RelationData(ctx context.Context, relationID int) (*model.RelationData, error) {
	// First get status to discover which units are involved in this relation.
	status, err := c.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching status for relation data: %w", err)
	}

	// Find the relation and its endpoint applications.
	var rel *model.Relation
	for i := range status.Relations {
		if status.Relations[i].ID == relationID {
			rel = &status.Relations[i]
			break
		}
	}
	if rel == nil {
		return nil, fmt.Errorf("relation %d not found", relationID)
	}

	// Collect all unit tags for the endpoint applications.
	var unitTags []names.UnitTag
	for _, ep := range rel.Endpoints {
		if app, ok := status.Applications[ep.ApplicationName]; ok {
			for _, u := range app.Units {
				unitTags = append(unitTags, names.NewUnitTag(u.Name))
			}
		}
	}
	if len(unitTags) == 0 {
		return &model.RelationData{
			ApplicationData: make(map[string]map[string]string),
			UnitData:        make(map[string]map[string]string),
		}, nil
	}

	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	appClient := application.NewClient(conn)
	infos, err := appClient.UnitsInfo(ctx, unitTags)
	if err != nil {
		return nil, fmt.Errorf("fetching unit info for relation data: %w", err)
	}

	result := &model.RelationData{
		ApplicationData: make(map[string]map[string]string),
		UnitData:        make(map[string]map[string]string),
	}

	for _, info := range infos {
		if info.Error != nil {
			continue
		}
		for _, erd := range info.RelationData {
			if erd.RelationId != relationID {
				continue
			}
			// Application data — keyed by the endpoint name's application.
			appName := strings.SplitN(info.Tag, "/", 2)[0]
			if strings.HasPrefix(appName, "unit-") {
				// Tag is "unit-<app>-<num>"; extract app name.
				appName = strings.TrimPrefix(info.Tag, "unit-")
				if idx := strings.LastIndex(appName, "-"); idx >= 0 {
					appName = appName[:idx]
				}
			}

			if len(erd.ApplicationData) > 0 {
				if _, ok := result.ApplicationData[appName]; !ok {
					ad := make(map[string]string, len(erd.ApplicationData))
					for k, v := range erd.ApplicationData {
						ad[k] = fmt.Sprintf("%v", v)
					}
					result.ApplicationData[appName] = ad
				}
			}

			// Unit data — keyed by unit name from the UnitRelationData map.
			for uName, urd := range erd.UnitRelationData {
				if !urd.InScope {
					continue
				}
				if _, exists := result.UnitData[uName]; exists {
					continue
				}
				ud := make(map[string]string, len(urd.UnitData))
				for k, v := range urd.UnitData {
					ud[k] = fmt.Sprintf("%v", v)
				}
				result.UnitData[uName] = ud
			}
		}
	}

	return result, nil
}

// ListSecrets returns the secrets for the current model using the Secrets facade.
func (c *JujuClient) ListSecrets(ctx context.Context) ([]model.Secret, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	secClient := jujuSecrets.NewClient(conn)
	details, err := secClient.ListSecrets(ctx, false, coreSecrets.Filter{})
	if err != nil {
		return nil, fmt.Errorf("listing secrets: %w", err)
	}

	result := make([]model.Secret, 0, len(details))
	for _, d := range details {
		if d.Error != "" {
			continue
		}
		m := d.Metadata
		sec := model.Secret{
			URI:            m.URI.String(),
			Label:          m.Label,
			Description:    m.Description,
			Owner:          m.Owner.String(),
			RotatePolicy:   string(m.RotatePolicy),
			Revision:       m.LatestRevision,
			AutoPrune:      m.AutoPrune,
			CreateTime:     m.CreateTime,
			UpdateTime:     m.UpdateTime,
			ExpireTime:     m.LatestExpireTime,
			NextRotateTime: m.NextRotateTime,
		}
		for _, rev := range d.Revisions {
			backend := ""
			if rev.BackendName != nil {
				backend = *rev.BackendName
			}
			sec.Revisions = append(sec.Revisions, model.SecretRevision{
				Revision:  rev.Revision,
				CreatedAt: rev.CreateTime,
				ExpiredAt: rev.ExpireTime,
				Backend:   backend,
			})
		}
		for _, a := range d.Access {
			sec.Access = append(sec.Access, model.SecretAccessInfo{
				Target: a.Target,
				Scope:  a.Scope,
				Role:   string(a.Role),
			})
		}
		result = append(result, sec)
	}
	return result, nil
}

// RevealSecret returns the decoded key-value content of a single secret.
func (c *JujuClient) RevealSecret(ctx context.Context, uri string, revision int) (map[string]string, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	parsed, err := coreSecrets.ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing secret URI: %w", err)
	}

	filter := coreSecrets.Filter{URI: parsed}
	if revision > 0 {
		filter.Revision = &revision
	}

	secClient := jujuSecrets.NewClient(conn)
	details, err := secClient.ListSecrets(ctx, true, filter)
	if err != nil {
		return nil, fmt.Errorf("revealing secret: %w", err)
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("secret %q not found", uri)
	}
	if details[0].Error != "" {
		return nil, fmt.Errorf("revealing secret: %s", details[0].Error)
	}
	if details[0].Value == nil {
		return nil, fmt.Errorf("secret %q has no value", uri)
	}
	return details[0].Value.Values()
}

func (c *JujuClient) ListOffers(ctx context.Context) ([]model.Offer, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	offersClient := applicationoffers.NewClient(conn)
	details, err := offersClient.ListOffers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing offers: %w", err)
	}

	result := make([]model.Offer, 0, len(details))
	for _, d := range details {
		o := model.Offer{
			Name:            d.OfferName,
			ApplicationName: d.ApplicationName,
			OfferURL:        d.OfferURL,
			CharmURL:        d.CharmURL,
			TotalConnCount:  len(d.Connections),
		}
		for _, ep := range d.Endpoints {
			o.Endpoints = append(o.Endpoints, ep.Name)
		}
		for _, conn := range d.Connections {
			if conn.Status == relation.Joined {
				o.ActiveConnCount++
			}
		}
		result = append(result, o)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// AppConfig returns the configuration key-value pairs for an application.
func (c *JujuClient) AppConfig(ctx context.Context, appName string) ([]model.ConfigEntry, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	appClient := application.NewClient(conn)
	result, err := appClient.Get(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("getting app config: %w", err)
	}

	var entries []model.ConfigEntry
	for key, raw := range result.CharmConfig {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		e := model.ConfigEntry{Key: key}
		if v, ok := m["value"]; ok {
			e.Value = fmt.Sprintf("%v", v)
		}
		if v, ok := m["default"]; ok {
			e.Default = fmt.Sprintf("%v", v)
		}
		if v, ok := m["source"]; ok {
			e.Source = fmt.Sprintf("%v", v)
		}
		if v, ok := m["type"]; ok {
			e.Type = fmt.Sprintf("%v", v)
		}
		if v, ok := m["description"]; ok {
			e.Description = fmt.Sprintf("%v", v)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ApplicationActions returns the available charm actions for an application.
func (c *JujuClient) ApplicationActions(ctx context.Context, appName string) ([]model.ActionSpec, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	actionClient := action.NewClient(conn)
	specs, err := actionClient.ApplicationCharmActions(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("fetching charm actions: %w", err)
	}

	result := make([]model.ActionSpec, 0, len(specs))
	for name, spec := range specs {
		result = append(result, model.ActionSpec{
			Name:        name,
			Description: spec.Description,
			Params:      spec.Params,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// RunAction executes a named action on a unit and waits for the result.
func (c *JujuClient) RunAction(ctx context.Context, unitName, actionName string, params map[string]string) (*model.ActionResult, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	actionParams := make(map[string]interface{}, len(params))
	for k, v := range params {
		actionParams[k] = v
	}

	actionClient := action.NewClient(conn)
	enqueued, err := actionClient.EnqueueOperation(ctx, []action.Action{{
		Receiver:   names.NewUnitTag(unitName).String(),
		Name:       actionName,
		Parameters: actionParams,
	}})
	if err != nil {
		return nil, fmt.Errorf("enqueuing action: %w", err)
	}
	if len(enqueued.Actions) == 0 {
		return nil, fmt.Errorf("no actions enqueued")
	}
	if enqueued.Actions[0].Action == nil {
		return nil, fmt.Errorf("enqueued action has no action details")
	}

	actionID := enqueued.Actions[0].Action.ID

	// Poll for completion.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			results, err := actionClient.Actions(ctx, []string{actionID})
			if err != nil {
				return nil, fmt.Errorf("fetching action result: %w", err)
			}
			if len(results) == 0 {
				continue
			}
			r := results[0]
			if r.Status == "pending" || r.Status == "running" {
				continue
			}
			ar := &model.ActionResult{
				ID:        r.Action.ID,
				Status:    r.Status,
				Message:   r.Message,
				Output:    r.Output,
				Enqueued:  r.Enqueued,
				Started:   r.Started,
				Completed: r.Completed,
			}
			if r.Error != nil {
				ar.Message = r.Error.Error()
			}
			return ar, nil
		}
	}
}

// ListStorage returns all storage instances in the current model.
func (c *JujuClient) ListStorage(ctx context.Context) ([]model.StorageInstance, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	storageClient := jujuStorage.NewClient(conn)
	details, err := storageClient.ListStorageDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing storage: %w", err)
	}

	result := make([]model.StorageInstance, 0, len(details))
	for _, d := range details {
		kind := "unknown"
		switch d.Kind {
		case 1:
			kind = "block"
		case 2:
			kind = "filesystem"
		}
		si := model.StorageInstance{
			ID:         strings.TrimPrefix(d.StorageTag, "storage-"),
			Kind:       kind,
			Status:     d.Status.Status.String(),
			Persistent: d.Persistent,
			Life:       string(d.Life),
		}
		if d.OwnerTag != "" {
			si.Owner = strings.TrimPrefix(strings.TrimPrefix(d.OwnerTag, "unit-"), "application-")
		}
		// Derive pool from first attachment if available.
		for _, att := range d.Attachments {
			if att.Location != "" {
				si.Pool = att.Location
				break
			}
		}
		result = append(result, si)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// currentModelType returns the model type ("iaas" or "caas") for the
// currently targeted model by reading from the local client store.
func (c *JujuClient) currentModelType() (string, error) {
	c.mu.RLock()
	controllerName := c.controllerName
	c.mu.RUnlock()

	modelName, err := c.store.CurrentModel(controllerName)
	if err != nil {
		return "", fmt.Errorf("resolving current model: %w: %w", err, ErrNoSelectedModel)
	}
	details, err := c.store.ModelByName(controllerName, modelName)
	if err != nil {
		return "", fmt.Errorf("getting model details: %w", err)
	}
	return string(details.ModelType), nil
}

// DebugLog connects to the controller and streams debug log messages.
// The returned channel emits log entries until the context is cancelled
// or the connection is closed.
func (c *JujuClient) DebugLog(ctx context.Context, filter model.DebugLogFilter) (<-chan model.LogEntry, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	params := common.DebugLogParams{
		Backlog: 100,
	}
	if filter.Backlog > 0 {
		params.Backlog = uint(filter.Backlog)
	}
	if filter.Level != "" {
		if lvl, ok := loggo.ParseLevel(filter.Level); ok {
			params.Level = lvl
		}
	}
	params.IncludeEntity = filter.IncludeEntities
	// Expand application names to "unit-<app>-*" glob patterns so users can
	// filter by application without knowing individual unit numbers.
	for _, app := range filter.Applications {
		params.IncludeEntity = append(params.IncludeEntity, "unit-"+app+"-*")
	}
	params.ExcludeEntity = filter.ExcludeEntities
	params.IncludeModule = filter.IncludeModules
	params.ExcludeModule = filter.ExcludeModules
	params.IncludeLabels = filter.IncludeLabels
	params.ExcludeLabels = filter.ExcludeLabels

	jujuClient := client.NewClient(conn, nopLogger{})
	msgs, err := jujuClient.WatchDebugLog(ctx, params)
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
		Name:             name,
		Status:           a.Status.Status,
		StatusMessage:    a.Status.Info,
		Charm:            a.Charm,
		CharmChannel:     a.CharmChannel,
		CharmRev:         a.CharmRev,
		Scale:            a.Scale,
		Exposed:          a.Exposed,
		WorkloadVersion:  a.WorkloadVersion,
		Since:            a.Status.Since,
		EndpointBindings: a.EndpointBindings,
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
