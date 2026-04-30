package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"frp-ui-backend/internal/control"
	"frp-ui-backend/internal/frp"
	"frp-ui-backend/internal/httpx"
	"frp-ui-backend/internal/security"
	"frp-ui-backend/internal/storage"
	"frp-ui-backend/internal/validate"
)

type Service struct {
	store    *storage.JSONFile[Data]
	pm       *ProcessManager
	client   *http.Client
	now      func() time.Time
	verify   func(context.Context, string, string) error
	apply    func(context.Context, string, string) error
	applyAll func(context.Context, string, []runtimeFrpcConfig) error
}

type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	defaultNode NodeConfig
}

var (
	nodeIDPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$`)
	subdomainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
)

func WithDefaultNode(node NodeConfig) ServiceOption {
	return func(opts *serviceOptions) {
		opts.defaultNode = node
	}
}

func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		ID:            DefaultNodeID,
		Name:          "Default frps",
		ServerAddr:    "149.118.158.112",
		FrpsPort:      7000,
		AuthMethod:    "token",
		WebBaseDomain: "ma1.gameuniverse.top",
		WebScheme:     "https",
		VhostHTTPPort: 8080,
	}
}

func OpenService(dataPath, frpcPath, workDir string, options ...ServiceOption) (*Service, error) {
	opts := serviceOptions{defaultNode: DefaultNodeConfig()}
	for _, option := range options {
		option(&opts)
	}

	initial := Data{}
	initial.Frpc.Path = frpcPath
	initial.Frpc.WorkDir = workDir
	NormalizeData(&initial)
	store, err := storage.OpenJSONFile(dataPath, initial, NormalizeData)
	if err != nil {
		return nil, err
	}
	svc := &Service{
		store:  store,
		pm:     NewProcessManager(filepath.Join(workDir, "frpc.log")),
		client: &http.Client{Timeout: 10 * time.Second},
		now:    time.Now,
	}
	svc.verify = svc.pm.Verify
	svc.apply = svc.pm.Reload
	svc.applyAll = svc.pm.Apply
	err = store.Update(func(data *Data) error {
		if frpcPath != "" {
			data.Frpc.Path = frpcPath
		}
		if workDir != "" {
			data.Frpc.WorkDir = workDir
		}
		if data.Frpc.AdminPassword == "" {
			token, err := security.NewToken()
			if err != nil {
				return err
			}
			data.Frpc.AdminPassword = token
		}
		svc.ensureDefaultNode(data, opts.defaultNode)
		return nil
	})
	return svc, err
}

func (s *Service) ensureDefaultNode(data *Data, configured NodeConfig) {
	NormalizeData(data)
	if configured.ID == "" {
		configured.ID = DefaultNodeID
	}
	if configured.Name == "" {
		configured.Name = "Default frps"
	}
	if configured.ServerAddr == "" {
		configured.ServerAddr = "149.118.158.112"
	}
	if configured.FrpsPort == 0 {
		configured.FrpsPort = 7000
	}
	if configured.AuthMethod == "" {
		configured.AuthMethod = "token"
	}
	if configured.WebBaseDomain == "" {
		configured.WebBaseDomain = "ma1.gameuniverse.top"
	}
	if configured.WebScheme == "" {
		configured.WebScheme = "https"
	}
	if configured.VhostHTTPPort == 0 {
		configured.VhostHTTPPort = 8080
	}

	now := s.now().UTC()
	existing, ok := data.Nodes[configured.ID]
	if !ok {
		configured.CreatedAt = now
		configured.UpdatedAt = now
		data.Nodes[configured.ID] = configured
		return
	}

	changed := false
	if configured.ServerAddr != "" && existing.ServerAddr != configured.ServerAddr {
		existing.ServerAddr = configured.ServerAddr
		changed = true
	}
	if configured.FrpsPort != 0 && existing.FrpsPort != configured.FrpsPort {
		existing.FrpsPort = configured.FrpsPort
		changed = true
	}
	if configured.AuthMethod != "" && existing.AuthMethod != configured.AuthMethod {
		existing.AuthMethod = configured.AuthMethod
		changed = true
	}
	if configured.AuthToken != "" && existing.AuthToken != configured.AuthToken {
		existing.AuthToken = configured.AuthToken
		changed = true
	}
	if configured.WebBaseDomain != "" && existing.WebBaseDomain != configured.WebBaseDomain {
		existing.WebBaseDomain = configured.WebBaseDomain
		changed = true
	}
	if configured.WebScheme != "" && existing.WebScheme != configured.WebScheme {
		existing.WebScheme = configured.WebScheme
		changed = true
	}
	if configured.VhostHTTPPort != 0 && existing.VhostHTTPPort != configured.VhostHTTPPort {
		existing.VhostHTTPPort = configured.VhostHTTPPort
		changed = true
	}
	if existing.Name == "" {
		existing.Name = configured.Name
		changed = true
	}
	if existing.CreatedAt.IsZero() {
		existing.CreatedAt = now
		changed = true
	}
	if changed {
		existing.UpdatedAt = now
		data.Nodes[existing.ID] = existing
	}
}

func (s *Service) ConfigureServer(req ConfigureServerRequest) (ClientState, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return ClientState{}, httpx.BadRequest("baseURL must be an absolute URL")
	}
	if strings.TrimSpace(req.FrpsHost) == "" {
		return ClientState{}, httpx.BadRequest("frpsHost is required")
	}
	if err := validate.Port(req.FrpsPort); err != nil {
		return ClientState{}, httpx.BadRequest(err.Error())
	}
	err := s.store.Update(func(data *Data) error {
		data.Server = ServerConfig{
			BaseURL:   baseURL,
			FrpsHost:  strings.TrimSpace(req.FrpsHost),
			FrpsPort:  req.FrpsPort,
			AuthToken: req.AuthToken,
		}
		return nil
	})
	if err != nil {
		return ClientState{}, err
	}
	return s.State()
}

func (s *Service) ListNodes() ([]NodeView, error) {
	data, err := s.store.Snapshot()
	if err != nil {
		return nil, err
	}
	return mustListNodeViews(data), nil
}

func (s *Service) CreateNode(req CreateNodeRequest) (NodeView, error) {
	now := s.now().UTC()
	node, err := buildNodeConfig(req, NodeConfig{}, now)
	if err != nil {
		return NodeView{}, err
	}
	err = s.store.Update(func(data *Data) error {
		NormalizeData(data)
		if _, ok := data.Nodes[node.ID]; ok {
			return httpx.Conflict("node already exists")
		}
		data.Nodes[node.ID] = node
		return nil
	})
	if err != nil {
		return NodeView{}, err
	}
	return toNodeView(node), nil
}

func (s *Service) UpdateNode(ctx context.Context, id string, req UpdateNodeRequest) (NodeView, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return NodeView{}, httpx.BadRequest("nodeId is required")
	}
	oldData, err := s.store.Snapshot()
	if err != nil {
		return NodeView{}, err
	}
	oldNode, ok := oldData.Nodes[id]
	if !ok {
		return NodeView{}, httpx.NotFound("node not found")
	}
	newData, err := cloneData(oldData)
	if err != nil {
		return NodeView{}, err
	}
	node, err := updateNodeConfig(id, req, oldNode, s.now().UTC())
	if err != nil {
		return NodeView{}, err
	}
	newData.Nodes[id] = node
	if enabledRuleUsesNode(oldData, id) || enabledRuleUsesNode(newData, id) {
		if err := s.applyData(ctx, oldData, newData); err != nil {
			return NodeView{}, err
		}
	} else {
		if err := s.store.Update(func(data *Data) error {
			NormalizeData(data)
			if _, ok := data.Nodes[id]; !ok {
				return httpx.NotFound("node not found")
			}
			data.Nodes[id] = node
			return nil
		}); err != nil {
			return NodeView{}, err
		}
	}
	return toNodeView(node), nil
}

func (s *Service) DeleteNode(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return httpx.BadRequest("nodeId is required")
	}
	if id == DefaultNodeID {
		return httpx.BadRequest("default node cannot be deleted")
	}
	return s.store.Update(func(data *Data) error {
		NormalizeData(data)
		if _, ok := data.Nodes[id]; !ok {
			return httpx.NotFound("node not found")
		}
		for _, rule := range data.PortRules {
			if rule.NodeID == id {
				return httpx.Conflict("node is used by port rules")
			}
		}
		delete(data.Nodes, id)
		return nil
	})
}

func (s *Service) DoctorNode(ctx context.Context, id string) (NodeDoctorResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return NodeDoctorResult{}, httpx.BadRequest("nodeId is required")
	}
	data, err := s.store.Snapshot()
	if err != nil {
		return NodeDoctorResult{}, err
	}
	node, ok := data.Nodes[id]
	if !ok {
		return NodeDoctorResult{}, httpx.NotFound("node not found")
	}

	result := NodeDoctorResult{Node: toNodeView(node)}
	addDoctorCheck := func(check NodeDoctorCheck) {
		result.Checks = append(result.Checks, check)
	}

	start := time.Now()
	if err := validateNodeConfig(node); err != nil {
		addDoctorCheck(NodeDoctorCheck{ID: "node-config", Name: "节点配置", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)})
		result.Overall = doctorOverall(result.Checks)
		return result, nil
	}
	addDoctorCheck(NodeDoctorCheck{
		ID:         "node-config",
		Name:       "节点配置",
		Status:     "pass",
		Message:    "节点基础配置有效",
		DurationMS: elapsedMS(start),
	})

	start = time.Now()
	if strings.TrimSpace(node.AuthToken) == "" {
		addDoctorCheck(NodeDoctorCheck{ID: "auth-token", Name: "Auth Token", Status: "fail", Message: "节点未配置 auth token", DurationMS: elapsedMS(start)})
	} else {
		addDoctorCheck(NodeDoctorCheck{ID: "auth-token", Name: "Auth Token", Status: "pass", Message: "节点已配置 auth token", DurationMS: elapsedMS(start)})
	}

	start = time.Now()
	address := net.JoinHostPort(node.ServerAddr, fmt.Sprint(node.FrpsPort))
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(dialCtx, "tcp", address)
	cancel()
	tcpOK := false
	if err != nil {
		addDoctorCheck(NodeDoctorCheck{
			ID:         "frps-tcp",
			Name:       "frps TCP 连通性",
			Status:     "fail",
			Message:    err.Error(),
			DurationMS: elapsedMS(start),
			Details:    map[string]string{"address": address},
		})
	} else {
		_ = conn.Close()
		tcpOK = true
		addDoctorCheck(NodeDoctorCheck{
			ID:         "frps-tcp",
			Name:       "frps TCP 连通性",
			Status:     "pass",
			Message:    "可以连接 frps",
			DurationMS: elapsedMS(start),
			Details:    map[string]string{"address": address},
		})
	}

	if tcpOK {
		addDoctorCheck(s.doctorFrpcLogin(ctx, data, node))
	} else {
		addDoctorCheck(NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "skipped", Message: "frps TCP 不可达，跳过登录验证"})
	}
	for _, check := range s.doctorWeb(ctx, node) {
		if result.TestedDomain == "" && check.Details != nil {
			result.TestedDomain = check.Details["domain"]
		}
		addDoctorCheck(check)
	}

	result.Overall = doctorOverall(result.Checks)
	return result, nil
}

func (s *Service) ListPortRules() ([]PortRule, error) {
	data, err := s.store.Snapshot()
	if err != nil {
		return nil, err
	}
	return sortedPortRules(data.PortRules), nil
}

func (s *Service) CreatePortRule(ctx context.Context, req CreatePortRuleRequest) (PortRule, error) {
	oldData, err := s.store.Snapshot()
	if err != nil {
		return PortRule{}, err
	}
	newData, err := cloneData(oldData)
	if err != nil {
		return PortRule{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule, err := s.buildPortRule(newData, "", portRuleInput{
		NodeID:     req.NodeID,
		Name:       req.Name,
		Protocol:   req.Protocol,
		LocalIP:    req.LocalIP,
		LocalPort:  req.LocalPort,
		RemotePort: req.RemotePort,
		Subdomain:  req.Subdomain,
		Domain:     req.Domain,
		Enabled:    enabled,
	})
	if err != nil {
		return PortRule{}, err
	}
	newData.PortRules[rule.ID] = rule
	if err := s.applyData(ctx, oldData, newData); err != nil {
		return PortRule{}, err
	}
	return rule, nil
}

func (s *Service) UpdatePortRule(ctx context.Context, id string, req UpdatePortRuleRequest) (PortRule, error) {
	oldData, err := s.store.Snapshot()
	if err != nil {
		return PortRule{}, err
	}
	if _, ok := oldData.PortRules[id]; !ok {
		return PortRule{}, httpx.NotFound("port rule not found")
	}
	newData, err := cloneData(oldData)
	if err != nil {
		return PortRule{}, err
	}
	rule, err := s.buildPortRule(newData, id, portRuleInput{
		NodeID:     req.NodeID,
		Name:       req.Name,
		Protocol:   req.Protocol,
		LocalIP:    req.LocalIP,
		LocalPort:  req.LocalPort,
		RemotePort: req.RemotePort,
		Subdomain:  req.Subdomain,
		Domain:     req.Domain,
		Enabled:    req.Enabled,
	})
	if err != nil {
		return PortRule{}, err
	}
	newData.PortRules[id] = rule
	if err := s.applyData(ctx, oldData, newData); err != nil {
		return PortRule{}, err
	}
	return rule, nil
}

func (s *Service) PatchPortRule(ctx context.Context, id string, req PatchPortRuleRequest) (PortRule, error) {
	if req.Enabled == nil {
		return PortRule{}, httpx.BadRequest("enabled is required")
	}
	oldData, err := s.store.Snapshot()
	if err != nil {
		return PortRule{}, err
	}
	rule, ok := oldData.PortRules[id]
	if !ok {
		return PortRule{}, httpx.NotFound("port rule not found")
	}
	newData, err := cloneData(oldData)
	if err != nil {
		return PortRule{}, err
	}
	rule.Enabled = *req.Enabled
	rule.UpdatedAt = s.now().UTC()
	newData.PortRules[id] = rule
	if _, err := s.validatePortRule(newData, rule); err != nil {
		return PortRule{}, err
	}
	if err := s.applyData(ctx, oldData, newData); err != nil {
		return PortRule{}, err
	}
	return rule, nil
}

func (s *Service) DeletePortRule(ctx context.Context, id string) error {
	oldData, err := s.store.Snapshot()
	if err != nil {
		return err
	}
	if _, ok := oldData.PortRules[id]; !ok {
		return httpx.NotFound("port rule not found")
	}
	newData, err := cloneData(oldData)
	if err != nil {
		return err
	}
	delete(newData.PortRules, id)
	return s.applyData(ctx, oldData, newData)
}

func (s *Service) JoinGroup(ctx context.Context, req ClientJoinGroupRequest) (ClientState, error) {
	if err := validate.GroupID(req.GroupID); err != nil {
		return ClientState{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Password(req.Password); err != nil {
		return ClientState{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Name(req.DeviceName); err != nil {
		return ClientState{}, httpx.BadRequest(err.Error())
	}
	data, err := s.store.Snapshot()
	if err != nil {
		return ClientState{}, err
	}
	if data.Server.BaseURL == "" {
		return ClientState{}, httpx.BadRequest("server is not configured")
	}

	var joined control.JoinGroupResponse
	err = s.postJSON(ctx, data.Server.BaseURL, "/v1/groups/"+url.PathEscape(req.GroupID)+"/join",
		control.JoinGroupRequest{Password: req.Password, DeviceName: req.DeviceName}, &joined)
	if err != nil {
		return ClientState{}, err
	}
	err = s.store.Update(func(data *Data) error {
		data.Group = GroupConfig{
			GroupID:     joined.GroupID,
			DeviceID:    joined.DeviceID,
			DeviceToken: joined.DeviceToken,
			SecretKey:   security.DeriveSecretKey(joined.GroupID, req.Password),
			JoinedAt:    s.now().UTC(),
		}
		return nil
	})
	if err != nil {
		return ClientState{}, err
	}
	if err := s.WriteConfig(); err != nil {
		return ClientState{}, err
	}
	return s.State()
}

func (s *Service) CreateExposure(ctx context.Context, req ClientCreateExposureRequest) (control.Exposure, error) {
	if req.Mode != "private" && req.Mode != "public" {
		return control.Exposure{}, httpx.BadRequest("mode must be private or public")
	}
	data, err := s.requireJoined()
	if err != nil {
		return control.Exposure{}, err
	}
	var exposure control.Exposure
	if req.Mode == "private" {
		payload := control.CreatePrivateExposureRequest{
			DeviceAuth: control.DeviceAuth{
				GroupID: data.Group.GroupID, DeviceID: data.Group.DeviceID, DeviceToken: data.Group.DeviceToken,
			},
			Name:      req.Name,
			LocalIP:   req.LocalIP,
			LocalPort: req.LocalPort,
		}
		err = s.postJSON(ctx, data.Server.BaseURL, "/v1/exposures/private", payload, &exposure)
	} else {
		payload := control.CreatePublicExposureRequest{
			DeviceAuth: control.DeviceAuth{
				GroupID: data.Group.GroupID, DeviceID: data.Group.DeviceID, DeviceToken: data.Group.DeviceToken,
			},
			NodeID:     req.NodeID,
			Name:       req.Name,
			Protocol:   req.Protocol,
			LocalIP:    req.LocalIP,
			LocalPort:  req.LocalPort,
			RemotePort: req.RemotePort,
			Domain:     req.Domain,
		}
		err = s.postJSON(ctx, data.Server.BaseURL, "/v1/exposures/public", payload, &exposure)
	}
	if err != nil {
		return control.Exposure{}, err
	}
	err = s.store.Update(func(data *Data) error {
		data.Exposures[exposure.ID] = exposure
		return nil
	})
	if err != nil {
		return control.Exposure{}, err
	}
	return exposure, s.WriteConfig()
}

func (s *Service) DeleteExposure(ctx context.Context, exposureID string) error {
	data, err := s.store.Snapshot()
	if err != nil {
		return err
	}
	if data.Server.BaseURL != "" {
		if err := s.deleteExposure(ctx, data, exposureID); err != nil {
			return err
		}
	}
	err = s.store.Update(func(data *Data) error {
		delete(data.Exposures, exposureID)
		for routeID, route := range data.AccessRoutes {
			if route.ExposureID == exposureID {
				delete(data.AccessRoutes, routeID)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.WriteConfig()
}

func (s *Service) CreateAccessRoute(ctx context.Context, req ClientCreateAccessRouteRequest) (control.AccessRoute, error) {
	data, err := s.requireJoined()
	if err != nil {
		return control.AccessRoute{}, err
	}
	payload := control.CreateAccessRouteRequest{
		DeviceAuth: control.DeviceAuth{
			GroupID: data.Group.GroupID, DeviceID: data.Group.DeviceID, DeviceToken: data.Group.DeviceToken,
		},
		ExposureID:       req.ExposureID,
		BindAddr:         req.BindAddr,
		BindPort:         req.BindPort,
		FallbackBindPort: req.FallbackBindPort,
	}
	var route control.AccessRoute
	if err := s.postJSON(ctx, data.Server.BaseURL, "/v1/access-routes", payload, &route); err != nil {
		return control.AccessRoute{}, err
	}
	err = s.store.Update(func(data *Data) error {
		data.AccessRoutes[route.ID] = route
		return nil
	})
	if err != nil {
		return control.AccessRoute{}, err
	}
	return route, s.WriteConfig()
}

func (s *Service) Reload(ctx context.Context) (FrpcStatus, error) {
	if err := s.WriteConfig(); err != nil {
		return FrpcStatus{}, err
	}
	data, err := s.store.Snapshot()
	if err != nil {
		return FrpcStatus{}, err
	}
	configs, err := s.renderClientConfigs(data)
	if err != nil {
		return FrpcStatus{}, err
	}
	err = s.applyConfigs(ctx, data, configs)
	return s.pm.StatusAll(configs), err
}

func (s *Service) Status() FrpcStatus {
	data, err := s.store.Snapshot()
	if err != nil {
		return FrpcStatus{ConfigPath: s.ConfigPath(), LastError: err.Error()}
	}
	configs, err := s.renderClientConfigs(data)
	if err != nil {
		return FrpcStatus{ConfigPath: s.ConfigPath(), LastError: err.Error()}
	}
	return s.pm.StatusAll(configs)
}

func (s *Service) Logs() string {
	data, err := s.store.Snapshot()
	if err != nil {
		return s.pm.Logs()
	}
	configs, err := s.renderClientConfigs(data)
	if err != nil {
		return s.pm.Logs()
	}
	return s.pm.LogsFor(configs)
}

func (s *Service) State() (ClientState, error) {
	data, err := s.store.Snapshot()
	if err != nil {
		return ClientState{}, err
	}
	return ClientState{
		Server:       data.Server,
		Group:        data.Group,
		Frpc:         data.Frpc,
		Exposures:    data.Exposures,
		AccessRoutes: data.AccessRoutes,
		Nodes:        mustListNodeViews(data),
		PortRules:    sortedPortRules(data.PortRules),
	}, nil
}

func (s *Service) WriteConfig() error {
	data, err := s.store.Snapshot()
	if err != nil {
		return err
	}
	configs, err := s.renderClientConfigs(data)
	if err != nil {
		return err
	}
	return s.writeRenderedConfigs(configs)
}

func (s *Service) ConfigPath() string {
	data, err := s.store.Snapshot()
	if err != nil || data.Frpc.WorkDir == "" {
		return filepath.Join("data", "frpc", "frpc.toml")
	}
	return primaryConfigPathFor(data)
}

func (s *Service) renderClientConfig(data Data) (string, error) {
	configs, err := s.renderClientConfigs(data)
	if err != nil {
		return "", err
	}
	if len(configs) == 0 {
		return "", nil
	}
	return configs[0].Raw, nil
}

func (s *Service) renderClientConfigs(data Data) ([]runtimeFrpcConfig, error) {
	if usesDirectMode(data) {
		return s.buildDirectFrpcConfigs(data)
	}
	cfg := s.buildFrpcConfig(data)
	return []runtimeFrpcConfig{{
		NodeID:     defaultProcessKey,
		ConfigPath: configPathFor(data),
		LogPath:    logPathForNode(data, defaultProcessKey),
		RestartKey: restartKeyForClientConfig(cfg),
		Raw:        frp.RenderClientConfig(cfg),
	}}, nil
}

func usesDirectMode(data Data) bool {
	return len(data.PortRules) > 0 || (data.Group.GroupID == "" && data.Server.FrpsHost == "" && data.Server.BaseURL == "")
}

func (s *Service) buildFrpcConfig(data Data) frp.ClientConfig {
	cfg := frp.ClientConfig{
		ServerAddr:    data.Server.FrpsHost,
		ServerPort:    data.Server.FrpsPort,
		AuthToken:     data.Server.AuthToken,
		AdminAddr:     data.Frpc.AdminAddr,
		AdminPort:     data.Frpc.AdminPort,
		AdminUser:     data.Frpc.AdminUser,
		AdminPassword: data.Frpc.AdminPassword,
		GroupID:       data.Group.GroupID,
		DeviceID:      data.Group.DeviceID,
		DeviceToken:   data.Group.DeviceToken,
	}
	for _, exposure := range data.Exposures {
		if !exposure.Enabled {
			continue
		}
		metas := map[string]string{
			"group_id":     exposure.GroupID,
			"device_id":    exposure.DeviceID,
			"device_token": data.Group.DeviceToken,
			"exposure_id":  exposure.ID,
		}
		if exposure.Mode == "private" {
			xtcpName, stcpName := frp.PrivateProxyNames(exposure.GroupID, exposure.ID)
			cfg.Proxies = append(cfg.Proxies,
				frp.Proxy{Name: xtcpName, Type: "xtcp", LocalIP: exposure.LocalIP, LocalPort: exposure.LocalPort, SecretKey: data.Group.SecretKey, Metadatas: metas},
				frp.Proxy{Name: stcpName, Type: "stcp", LocalIP: exposure.LocalIP, LocalPort: exposure.LocalPort, SecretKey: data.Group.SecretKey, Metadatas: metas},
			)
			continue
		}
		proxy := frp.Proxy{
			Name:       "public." + exposure.ID,
			Type:       exposure.Protocol,
			LocalIP:    exposure.LocalIP,
			LocalPort:  exposure.LocalPort,
			RemotePort: exposure.RemotePort,
			Metadatas:  metas,
		}
		if exposure.Domain != "" {
			proxy.CustomDomains = []string{exposure.Domain}
		}
		cfg.Proxies = append(cfg.Proxies, proxy)
	}
	for _, route := range data.AccessRoutes {
		exposure, ok := data.Exposures[route.ExposureID]
		if !ok || exposure.Mode != "private" || !exposure.Enabled {
			continue
		}
		xtcpName, stcpName := frp.PrivateProxyNames(exposure.GroupID, exposure.ID)
		cfg.Visitors = append(cfg.Visitors, frp.Visitor{
			Name:       "visitor." + route.ID + ".xtcp",
			Type:       "xtcp",
			ServerName: xtcpName,
			SecretKey:  data.Group.SecretKey,
			BindAddr:   route.BindAddr,
			BindPort:   route.BindPort,
		})
		if route.FallbackBindPort != 0 {
			cfg.Visitors = append(cfg.Visitors, frp.Visitor{
				Name:       "visitor." + route.ID + ".stcp",
				Type:       "stcp",
				ServerName: stcpName,
				SecretKey:  data.Group.SecretKey,
				BindAddr:   route.BindAddr,
				BindPort:   route.FallbackBindPort,
			})
		}
	}
	return cfg
}

func (s *Service) buildDirectFrpcConfig(data Data) (frp.ClientConfig, error) {
	configs, err := s.buildDirectFrpcConfigs(data)
	if err != nil {
		return frp.ClientConfig{}, err
	}
	if len(configs) == 0 {
		return frp.ClientConfig{}, nil
	}
	return s.buildDirectFrpcConfigForNode(data, configs[0].NodeID, nil)
}

func (s *Service) buildDirectFrpcConfigs(data Data) ([]runtimeFrpcConfig, error) {
	NormalizeData(&data)
	rulesByNode := map[string][]PortRule{}
	for _, rule := range sortedPortRules(data.PortRules) {
		if !rule.Enabled {
			continue
		}
		if _, err := s.validatePortRule(data, rule); err != nil {
			return nil, err
		}
		rulesByNode[rule.NodeID] = append(rulesByNode[rule.NodeID], rule)
	}
	nodeIDs := make([]string, 0, len(rulesByNode))
	for nodeID := range rulesByNode {
		nodeIDs = append(nodeIDs, nodeID)
	}
	if len(nodeIDs) == 0 {
		node, err := directNodeFor(data)
		if err != nil {
			return nil, err
		}
		nodeIDs = append(nodeIDs, node.ID)
	}
	sort.Slice(nodeIDs, func(i, j int) bool {
		if nodeIDs[i] == DefaultNodeID {
			return true
		}
		if nodeIDs[j] == DefaultNodeID {
			return false
		}
		return nodeIDs[i] < nodeIDs[j]
	})
	adminPorts, err := assignAdminPorts(data, nodeIDs)
	if err != nil {
		return nil, err
	}
	configs := make([]runtimeFrpcConfig, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		cfg, err := s.buildDirectFrpcConfigForNode(data, nodeID, rulesByNode[nodeID])
		if err != nil {
			return nil, err
		}
		cfg.AdminPort = adminPorts[nodeID]
		configs = append(configs, runtimeFrpcConfig{
			NodeID:     nodeID,
			ConfigPath: configPathForNode(data, nodeID),
			LogPath:    logPathForNode(data, nodeID),
			RestartKey: restartKeyForClientConfig(cfg),
			Raw:        frp.RenderClientConfig(cfg),
		})
	}
	return configs, nil
}

func (s *Service) buildDirectFrpcConfigForNode(data Data, nodeID string, rules []PortRule) (frp.ClientConfig, error) {
	node, ok := data.Nodes[nodeID]
	if !ok {
		return frp.ClientConfig{}, httpx.NotFound("node not found")
	}
	cfg := frp.ClientConfig{
		ServerAddr:    node.ServerAddr,
		ServerPort:    node.FrpsPort,
		AuthMethod:    node.AuthMethod,
		AuthToken:     node.AuthToken,
		AdminAddr:     data.Frpc.AdminAddr,
		AdminPort:     data.Frpc.AdminPort,
		AdminUser:     data.Frpc.AdminUser,
		AdminPassword: data.Frpc.AdminPassword,
	}
	for _, rule := range rules {
		cfg.Proxies = append(cfg.Proxies, frp.Proxy{
			Name:              proxyNameForRule(rule),
			Type:              rule.Protocol,
			LocalIP:           rule.LocalIP,
			LocalPort:         rule.LocalPort,
			RemotePort:        rule.RemotePort,
			CustomDomains:     customDomainsForRule(rule),
			HostHeaderRewrite: hostHeaderRewriteForRule(rule),
		})
	}
	return cfg, nil
}

func (s *Service) requireJoined() (Data, error) {
	data, err := s.store.Snapshot()
	if err != nil {
		return Data{}, err
	}
	if data.Server.BaseURL == "" {
		return Data{}, httpx.BadRequest("server is not configured")
	}
	if data.Group.GroupID == "" || data.Group.DeviceID == "" || data.Group.DeviceToken == "" {
		return Data{}, httpx.Unauthorized("client has not joined a group")
	}
	return data, nil
}

func (s *Service) postJSON(ctx context.Context, baseURL, endpoint string, payload any, dst any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(baseURL, endpoint), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("remote API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func (s *Service) deleteExposure(ctx context.Context, data Data, exposureID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, joinURL(data.Server.BaseURL, "/v1/exposures/"+url.PathEscape(exposureID)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Group-ID", data.Group.GroupID)
	req.Header.Set("X-Device-ID", data.Group.DeviceID)
	req.Header.Set("Authorization", "Bearer "+data.Group.DeviceToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("remote API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func joinURL(baseURL, endpoint string) string {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return strings.TrimRight(baseURL, "/") + endpoint
	}
	u.Path = path.Join(u.Path, endpoint)
	return u.String()
}

type portRuleInput struct {
	NodeID     string
	Name       string
	Protocol   string
	LocalIP    string
	LocalPort  int
	RemotePort int
	Subdomain  string
	Domain     string
	Enabled    bool
}

func (s *Service) buildPortRule(data Data, existingID string, input portRuleInput) (PortRule, error) {
	NormalizeData(&data)
	id := existingID
	now := s.now().UTC()
	createdAt := now
	if id == "" {
		id = newDirectID("port")
	} else if existing, ok := data.PortRules[id]; ok {
		createdAt = existing.CreatedAt
	}
	nodeID := strings.TrimSpace(input.NodeID)
	if nodeID == "" {
		nodeID = DefaultNodeID
	}
	protocol := strings.ToLower(strings.TrimSpace(input.Protocol))
	localIP := strings.TrimSpace(input.LocalIP)
	if localIP == "" {
		localIP = "127.0.0.1"
	}
	name := strings.TrimSpace(input.Name)
	remotePort := input.RemotePort
	subdomain := strings.TrimSpace(input.Subdomain)
	domain := strings.TrimSpace(input.Domain)
	if protocol == "http" {
		remotePort = 0
		if node, ok := data.Nodes[nodeID]; ok {
			var err error
			subdomain, domain, err = resolveWebDomain(node, id, name, subdomain, domain)
			if err != nil {
				return PortRule{}, err
			}
		}
		if name == "" {
			name = subdomain
			if name == "" {
				name = domain
			}
		}
	} else {
		subdomain = ""
		domain = ""
		if name == "" {
			name = fmt.Sprintf("%s-%d", protocol, remotePort)
		}
	}
	rule := PortRule{
		ID:         id,
		NodeID:     nodeID,
		Name:       name,
		Protocol:   protocol,
		LocalIP:    localIP,
		LocalPort:  input.LocalPort,
		RemotePort: remotePort,
		Subdomain:  subdomain,
		Domain:     domain,
		Enabled:    input.Enabled,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	}
	_, err := s.validatePortRule(data, rule)
	return rule, err
}

func (s *Service) validatePortRule(data Data, rule PortRule) (NodeConfig, error) {
	if rule.ID == "" {
		return NodeConfig{}, httpx.BadRequest("port rule id is required")
	}
	if rule.NodeID == "" {
		return NodeConfig{}, httpx.BadRequest("nodeId is required")
	}
	node, ok := data.Nodes[rule.NodeID]
	if !ok {
		return NodeConfig{}, httpx.NotFound("node not found")
	}
	if strings.TrimSpace(node.ServerAddr) == "" {
		return NodeConfig{}, httpx.BadRequest("node serverAddr is required")
	}
	if err := validate.Port(node.FrpsPort); err != nil {
		return NodeConfig{}, httpx.BadRequest("node " + err.Error())
	}
	authMethod := strings.ToLower(strings.TrimSpace(node.AuthMethod))
	if authMethod == "" {
		authMethod = "token"
	}
	if authMethod != "token" {
		return NodeConfig{}, httpx.BadRequest("node authMethod must be token")
	}
	if strings.TrimSpace(node.AuthToken) == "" {
		return NodeConfig{}, httpx.BadRequest("node auth token is not configured")
	}
	if err := validate.Name(rule.Name); err != nil {
		return NodeConfig{}, httpx.BadRequest(err.Error())
	}
	switch rule.Protocol {
	case "tcp", "udp", "http":
	default:
		return NodeConfig{}, httpx.BadRequest("protocol must be tcp, udp or http")
	}
	if err := validate.LocalIP(rule.LocalIP); err != nil {
		return NodeConfig{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Port(rule.LocalPort); err != nil {
		return NodeConfig{}, httpx.BadRequest(err.Error())
	}
	if rule.Protocol == "http" {
		if err := validateWebRule(node, rule); err != nil {
			return NodeConfig{}, err
		}
	} else {
		if err := validate.UserRemotePort(rule.RemotePort); err != nil {
			return NodeConfig{}, httpx.BadRequest(err.Error())
		}
		if !remotePortAllowed(rule.RemotePort, node.AllowPorts) {
			return NodeConfig{}, httpx.BadRequest("remote port is not allowed on this node")
		}
	}
	for _, existing := range data.PortRules {
		if existing.ID == rule.ID {
			continue
		}
		if rule.Protocol == "http" && existing.NodeID == rule.NodeID && existing.Protocol == "http" && strings.EqualFold(existing.Domain, rule.Domain) {
			return NodeConfig{}, httpx.Conflict("domain already in use")
		}
		if rule.Protocol != "http" && existing.NodeID == rule.NodeID && existing.Protocol == rule.Protocol && existing.RemotePort == rule.RemotePort {
			return NodeConfig{}, httpx.Conflict("remote port already in use")
		}
	}
	return node, nil
}

func resolveWebDomain(node NodeConfig, id, name, rawSubdomain, rawDomain string) (string, string, error) {
	base := normalizeDomain(node.WebBaseDomain)
	if base == "" {
		return "", "", httpx.BadRequest("node webBaseDomain is required for http rules")
	}
	subdomain := strings.ToLower(strings.TrimSpace(rawSubdomain))
	if subdomain != "" {
		if !subdomainPattern.MatchString(subdomain) {
			return "", "", httpx.BadRequest("subdomain must be 1-63 chars and contain only lowercase letters, numbers or -")
		}
	}
	domain := normalizeDomain(rawDomain)
	if domain == "" {
		if subdomain == "" {
			subdomain = sanitizeSubdomain(name)
		}
		if subdomain == "" {
			subdomain = sanitizeSubdomain(strings.ReplaceAll(id, "_", "-"))
		}
		domain = subdomain + "." + base
		return subdomain, domain, nil
	}
	if !domainUnderBase(domain, base) {
		return "", "", httpx.BadRequest("domain must be under " + base)
	}
	if subdomain != "" && domain != subdomain+"."+base {
		return "", "", httpx.BadRequest("subdomain and domain do not match")
	}
	if subdomain == "" && strings.HasSuffix(domain, "."+base) {
		prefix := strings.TrimSuffix(domain, "."+base)
		if !strings.Contains(prefix, ".") && subdomainPattern.MatchString(prefix) {
			subdomain = prefix
		}
	}
	return subdomain, domain, nil
}

func validateWebRule(node NodeConfig, rule PortRule) error {
	base := normalizeDomain(node.WebBaseDomain)
	if base == "" {
		return httpx.BadRequest("node webBaseDomain is required for http rules")
	}
	if rule.Domain == "" {
		return httpx.BadRequest("domain is required for http rules")
	}
	if !domainUnderBase(rule.Domain, base) {
		return httpx.BadRequest("domain must be under " + base)
	}
	if rule.Subdomain != "" && !subdomainPattern.MatchString(rule.Subdomain) {
		return httpx.BadRequest("subdomain must be 1-63 chars and contain only lowercase letters, numbers or -")
	}
	return nil
}

func (s *Service) doctorFrpcLogin(ctx context.Context, data Data, node NodeConfig) NodeDoctorCheck {
	start := time.Now()
	if strings.TrimSpace(node.AuthToken) == "" {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "skipped", Message: "缺少 auth token，跳过登录验证", DurationMS: elapsedMS(start)}
	}
	if strings.TrimSpace(data.Frpc.Path) == "" {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: "frpc path is empty", DurationMS: elapsedMS(start)}
	}
	workDir := data.Frpc.WorkDir
	if workDir == "" {
		workDir = filepath.Join("data", "frpc")
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	adminPort, err := freeTCPPort()
	if err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	adminPassword, err := security.NewToken()
	if err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	configPath := filepath.Join(workDir, "frpc-doctor."+safeFilePart(node.ID)+".toml")
	logPath := filepath.Join(workDir, "frpc-doctor."+safeFilePart(node.ID)+".log")
	raw := frp.RenderClientConfig(frp.ClientConfig{
		ServerAddr:    node.ServerAddr,
		ServerPort:    node.FrpsPort,
		AuthMethod:    defaultString(node.AuthMethod, "token"),
		AuthToken:     node.AuthToken,
		AdminAddr:     "127.0.0.1",
		AdminPort:     adminPort,
		AdminUser:     "admin",
		AdminPassword: adminPassword,
	})
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	defer os.Remove(configPath)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	defer logFile.Close()

	cmd := exec.Command(data.Frpc.Path, "-c", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start)}
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	finished := false
	defer func() {
		if finished {
			return
		}
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}()

	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: ctx.Err().Error(), DurationMS: elapsedMS(start)}
		case err := <-done:
			finished = true
			rawLog := doctorLogTail(logPath)
			msg := rawLog
			if msg == "" && err != nil {
				msg = err.Error()
			}
			if msg == "" {
				msg = "临时 frpc 进程提前退出"
			}
			return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: msg, DurationMS: elapsedMS(start), Details: map[string]string{"logPath": logPath}}
		case <-deadline.C:
			return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: "等待 frpc 登录超时", DurationMS: elapsedMS(start), Details: map[string]string{"logPath": logPath}}
		case <-ticker.C:
			rawLog := doctorLogTail(logPath)
			lower := strings.ToLower(rawLog)
			if strings.Contains(lower, "login to server success") {
				return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "pass", Message: "临时 frpc 登录 frps 成功", DurationMS: elapsedMS(start), Details: map[string]string{"adminPort": fmt.Sprint(adminPort), "logPath": logPath}}
			}
			if strings.Contains(lower, "login to server failed") || strings.Contains(lower, "authorization failed") || strings.Contains(lower, "auth failed") {
				return NodeDoctorCheck{ID: "frpc-login", Name: "frpc 登录验证", Status: "fail", Message: rawLog, DurationMS: elapsedMS(start), Details: map[string]string{"logPath": logPath}}
			}
		}
	}
}

func (s *Service) doctorWeb(ctx context.Context, node NodeConfig) []NodeDoctorCheck {
	base := normalizeDomain(node.WebBaseDomain)
	if base == "" {
		return []NodeDoctorCheck{
			{ID: "web-dns", Name: "Web 通配 DNS", Status: "skipped", Message: "节点未配置 webBaseDomain"},
			{ID: "web-http", Name: "Web 入口探测", Status: "skipped", Message: "节点未配置 webBaseDomain"},
		}
	}
	domain := "doctor-" + randomHex(4) + "." + base
	return []NodeDoctorCheck{
		s.doctorDNS(ctx, node, domain),
		s.doctorHTTP(ctx, node, domain),
	}
}

func (s *Service) doctorDNS(ctx context.Context, node NodeConfig, domain string) NodeDoctorCheck {
	start := time.Now()
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupHost(lookupCtx, domain)
	if err != nil {
		return NodeDoctorCheck{ID: "web-dns", Name: "Web 通配 DNS", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain}}
	}
	targets, targetErr := resolveHostIPs(lookupCtx, node.ServerAddr)
	if targetErr != nil {
		return NodeDoctorCheck{ID: "web-dns", Name: "Web 通配 DNS", Status: "warn", Message: "通配域名可解析，但无法解析 frps 地址用于比对", DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "resolved": strings.Join(ips, ", "), "serverAddr": node.ServerAddr}}
	}
	if !ipSetsIntersect(ips, targets) {
		return NodeDoctorCheck{ID: "web-dns", Name: "Web 通配 DNS", Status: "warn", Message: "通配域名已解析，但结果与 frps 地址不一致；如果使用 CDN/反代可以忽略", DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "resolved": strings.Join(ips, ", "), "target": strings.Join(targets, ", ")}}
	}
	return NodeDoctorCheck{ID: "web-dns", Name: "Web 通配 DNS", Status: "pass", Message: "通配域名解析正常", DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "resolved": strings.Join(ips, ", ")}}
}

func (s *Service) doctorHTTP(ctx context.Context, node NodeConfig, domain string) NodeDoctorCheck {
	start := time.Now()
	scheme := defaultString(node.WebScheme, "https")
	probeURL := scheme + "://" + domain + "/"
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, probeURL, nil)
	if err != nil {
		return NodeDoctorCheck{ID: "web-http", Name: "Web 入口探测", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "url": probeURL}}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return NodeDoctorCheck{ID: "web-http", Name: "Web 入口探测", Status: "fail", Message: err.Error(), DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "url": probeURL}}
	}
	defer resp.Body.Close()
	return NodeDoctorCheck{ID: "web-http", Name: "Web 入口探测", Status: "pass", Message: "Web 入口可访问，返回 " + resp.Status, DurationMS: elapsedMS(start), Details: map[string]string{"domain": domain, "url": probeURL, "status": resp.Status}}
}

func doctorOverall(checks []NodeDoctorCheck) string {
	overall := "pass"
	for _, check := range checks {
		switch check.Status {
		case "fail":
			return "fail"
		case "warn":
			overall = "warn"
		}
	}
	return overall
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

func randomHex(size int) string {
	if size <= 0 {
		size = 4
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func freeTCPPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address %s", ln.Addr())
	}
	return addr.Port, nil
}

func doctorLogTail(logPath string) string {
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}
	const max = 4096
	if len(raw) > max {
		raw = raw[len(raw)-max:]
	}
	return strings.TrimSpace(string(raw))
}

func resolveHostIPs(ctx context.Context, host string) ([]string, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []string{ip.String()}, nil
	}
	return net.DefaultResolver.LookupHost(ctx, host)
}

func ipSetsIntersect(left, right []string) bool {
	values := map[string]struct{}{}
	for _, value := range left {
		if ip := net.ParseIP(value); ip != nil {
			values[ip.String()] = struct{}{}
			continue
		}
		values[value] = struct{}{}
	}
	for _, value := range right {
		if ip := net.ParseIP(value); ip != nil {
			value = ip.String()
		}
		if _, ok := values[value]; ok {
			return true
		}
	}
	return false
}

func normalizeDomain(v string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(v), "."))
}

func domainUnderBase(domain, base string) bool {
	return domain == base || strings.HasSuffix(domain, "."+base)
}

func sanitizeSubdomain(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	lastDash := false
	for _, r := range v {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if (r == '-' || r == '_' || r == ' ' || r == '.') && b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 63 {
		out = strings.TrimRight(out[:63], "-")
	}
	if !subdomainPattern.MatchString(out) {
		return ""
	}
	return out
}

func proxyNameForRule(rule PortRule) string {
	if rule.Protocol == "http" {
		return "web." + rule.ID
	}
	return "port." + rule.ID
}

func customDomainsForRule(rule PortRule) []string {
	if rule.Protocol == "http" && rule.Domain != "" {
		return []string{rule.Domain}
	}
	return nil
}

func hostHeaderRewriteForRule(rule PortRule) string {
	if rule.Protocol != "http" {
		return ""
	}
	if rule.LocalIP == "" || rule.LocalIP == "0.0.0.0" || rule.LocalIP == "::" || rule.LocalIP == "[::]" {
		return "127.0.0.1"
	}
	return rule.LocalIP
}

func (s *Service) applyData(ctx context.Context, oldData, newData Data) error {
	NormalizeData(&newData)
	nextConfigs, err := s.renderClientConfigs(newData)
	if err != nil {
		return err
	}
	if err := s.verifyRenderedConfigs(ctx, newData, nextConfigs); err != nil {
		return err
	}

	oldConfigs, oldRenderErr := s.renderClientConfigs(oldData)
	if err := s.writeRenderedConfigs(nextConfigs); err != nil {
		return err
	}
	if err := s.store.Update(func(data *Data) error {
		*data = newData
		return nil
	}); err != nil {
		if oldRenderErr == nil {
			_ = s.writeRenderedConfigs(oldConfigs)
		}
		return err
	}
	if s.apply == nil && s.applyAll == nil {
		return nil
	}
	if err := s.applyConfigs(ctx, newData, nextConfigs); err != nil {
		_ = s.store.Update(func(data *Data) error {
			*data = oldData
			return nil
		})
		if oldRenderErr == nil {
			_ = s.writeRenderedConfigs(oldConfigs)
			_ = s.applyConfigs(ctx, oldData, oldConfigs)
		}
		return err
	}
	return nil
}

func (s *Service) verifyRenderedConfigs(ctx context.Context, data Data, configs []runtimeFrpcConfig) error {
	if s.verify == nil {
		return nil
	}
	if strings.TrimSpace(data.Frpc.Path) == "" {
		return httpx.BadRequest("frpc path is empty")
	}
	if err := os.MkdirAll(data.Frpc.WorkDir, 0o755); err != nil {
		return err
	}
	for _, cfg := range configs {
		tmp, err := os.CreateTemp(data.Frpc.WorkDir, "frpc-verify-*.toml")
		if err != nil {
			return err
		}
		tmpPath := tmp.Name()
		if _, err := tmp.WriteString(cfg.Raw); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
		verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = s.verify(verifyCtx, data.Frpc.Path, tmpPath)
		cancel()
		_ = os.Remove(tmpPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) writeConfigRaw(data Data, raw string) error {
	if err := os.MkdirAll(data.Frpc.WorkDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPathFor(data), []byte(raw), 0o600)
}

func (s *Service) writeRenderedConfigs(configs []runtimeFrpcConfig) error {
	for _, cfg := range configs {
		if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(cfg.ConfigPath, []byte(cfg.Raw), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyConfigs(ctx context.Context, data Data, configs []runtimeFrpcConfig) error {
	if s.applyAll != nil {
		return s.applyAll(ctx, data.Frpc.Path, configs)
	}
	if s.apply == nil {
		return nil
	}
	for _, cfg := range configs {
		if err := s.apply(ctx, data.Frpc.Path, cfg.ConfigPath); err != nil {
			return err
		}
	}
	return nil
}

func configPathFor(data Data) string {
	if data.Frpc.WorkDir == "" {
		return filepath.Join("data", "frpc", "frpc.toml")
	}
	return filepath.Join(data.Frpc.WorkDir, "frpc.toml")
}

func primaryConfigPathFor(data Data) string {
	if !usesDirectMode(data) {
		return configPathFor(data)
	}
	node, err := directNodeFor(data)
	if err != nil {
		return configPathFor(data)
	}
	return configPathForNode(data, node.ID)
}

func configPathForNode(data Data, nodeID string) string {
	if nodeID == "" || nodeID == DefaultNodeID {
		return configPathFor(data)
	}
	return filepath.Join(data.Frpc.WorkDir, "frpc."+safeFilePart(nodeID)+".toml")
}

func logPathForNode(data Data, nodeID string) string {
	if data.Frpc.WorkDir == "" {
		data.Frpc.WorkDir = filepath.Join("data", "frpc")
	}
	if nodeID == "" || nodeID == DefaultNodeID {
		return filepath.Join(data.Frpc.WorkDir, "frpc.log")
	}
	return filepath.Join(data.Frpc.WorkDir, "frpc."+safeFilePart(nodeID)+".log")
}

func assignAdminPorts(data Data, nodeIDs []string) (map[string]int, error) {
	base := data.Frpc.AdminPort
	if base == 0 {
		base = 7400
	}
	used := map[int]string{}
	out := make(map[string]int, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		port := base
		if nodeID != DefaultNodeID {
			port = candidateAdminPort(base, nodeID)
		}
		for {
			if owner, ok := used[port]; !ok || owner == nodeID {
				break
			}
			port++
			if port > 65535 {
				port = base + 1
			}
			if port == base {
				return nil, httpx.BadRequest("not enough available frpc admin ports")
			}
		}
		if err := validate.Port(port); err != nil {
			return nil, httpx.BadRequest("frpc admin port " + err.Error())
		}
		used[port] = nodeID
		out[nodeID] = port
	}
	return out, nil
}

func candidateAdminPort(base int, nodeID string) int {
	const span = 10000
	offset := stableSmallHash(nodeID)%span + 1
	port := base + offset
	if port > 65535 {
		port = 1024 + (port-1024)%64512
		if port == base {
			port++
		}
	}
	return port
}

func stableSmallHash(v string) int {
	h := 2166136261
	for _, r := range v {
		h ^= int(r)
		h *= 16777619
		if h < 0 {
			h = -h
		}
	}
	return h
}

func restartKeyForClientConfig(cfg frp.ClientConfig) string {
	return strings.Join([]string{
		cfg.ServerAddr,
		fmt.Sprint(cfg.ServerPort),
		defaultString(cfg.AuthMethod, "token"),
		cfg.AuthToken,
		defaultString(cfg.AdminAddr, "127.0.0.1"),
		fmt.Sprint(defaultInt(cfg.AdminPort, 7400)),
		defaultString(cfg.AdminUser, "admin"),
		cfg.AdminPassword,
	}, "\x00")
}

func defaultInt(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}

func safeFilePart(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := b.String()
	if out == "" {
		return "node"
	}
	return out
}

func directNodeFor(data Data) (NodeConfig, error) {
	for _, rule := range sortedPortRules(data.PortRules) {
		if !rule.Enabled {
			continue
		}
		node, ok := data.Nodes[rule.NodeID]
		if !ok {
			return NodeConfig{}, httpx.NotFound("node not found")
		}
		return node, nil
	}
	if node, ok := data.Nodes[DefaultNodeID]; ok {
		return node, nil
	}
	for _, node := range data.Nodes {
		return node, nil
	}
	return DefaultNodeConfig(), nil
}

func sortedPortRules(values map[string]PortRule) []PortRule {
	out := make([]PortRule, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func mustListNodeViews(data Data) []NodeView {
	nodes := make([]NodeView, 0, len(data.Nodes))
	for _, node := range data.Nodes {
		nodes = append(nodes, toNodeView(node))
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].ID == DefaultNodeID {
			return true
		}
		if nodes[j].ID == DefaultNodeID {
			return false
		}
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}

func toNodeView(node NodeConfig) NodeView {
	return NodeView{
		ID:            node.ID,
		Name:          node.Name,
		ServerAddr:    node.ServerAddr,
		FrpsPort:      node.FrpsPort,
		AuthMethod:    defaultString(node.AuthMethod, "token"),
		AuthTokenSet:  node.AuthToken != "",
		WebBaseDomain: node.WebBaseDomain,
		WebScheme:     defaultString(node.WebScheme, "https"),
		VhostHTTPPort: node.VhostHTTPPort,
		AllowPorts:    node.AllowPorts,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}
}

func buildNodeConfig(req CreateNodeRequest, existing NodeConfig, now time.Time) (NodeConfig, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = sanitizeNodeID(req.Name)
	}
	if id == "" {
		id = newDirectID("node")
	}
	node := NodeConfig{
		ID:            id,
		Name:          strings.TrimSpace(req.Name),
		ServerAddr:    strings.TrimSpace(req.ServerAddr),
		FrpsPort:      req.FrpsPort,
		AuthMethod:    strings.ToLower(strings.TrimSpace(req.AuthMethod)),
		AuthToken:     strings.TrimSpace(req.AuthToken),
		WebBaseDomain: normalizeDomain(req.WebBaseDomain),
		WebScheme:     strings.ToLower(strings.TrimSpace(req.WebScheme)),
		VhostHTTPPort: req.VhostHTTPPort,
		AllowPorts:    normalizePortRanges(req.AllowPorts),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if !existing.CreatedAt.IsZero() {
		node.CreatedAt = existing.CreatedAt
	}
	defaultNodeConfig(&node)
	if err := validateNodeConfig(node); err != nil {
		return NodeConfig{}, err
	}
	return node, nil
}

func updateNodeConfig(id string, req UpdateNodeRequest, existing NodeConfig, now time.Time) (NodeConfig, error) {
	node := NodeConfig{
		ID:            id,
		Name:          strings.TrimSpace(req.Name),
		ServerAddr:    strings.TrimSpace(req.ServerAddr),
		FrpsPort:      req.FrpsPort,
		AuthMethod:    strings.ToLower(strings.TrimSpace(req.AuthMethod)),
		AuthToken:     existing.AuthToken,
		WebBaseDomain: normalizeDomain(req.WebBaseDomain),
		WebScheme:     strings.ToLower(strings.TrimSpace(req.WebScheme)),
		VhostHTTPPort: req.VhostHTTPPort,
		AllowPorts:    normalizePortRanges(req.AllowPorts),
		CreatedAt:     existing.CreatedAt,
		UpdatedAt:     now,
	}
	if req.AuthToken != nil {
		node.AuthToken = strings.TrimSpace(*req.AuthToken)
	}
	if req.ClearAuthToken {
		node.AuthToken = ""
	}
	defaultNodeConfig(&node)
	if err := validateNodeConfig(node); err != nil {
		return NodeConfig{}, err
	}
	return node, nil
}

func defaultNodeConfig(node *NodeConfig) {
	if node.Name == "" {
		node.Name = node.ID
	}
	if node.FrpsPort == 0 {
		node.FrpsPort = 7000
	}
	if node.AuthMethod == "" {
		node.AuthMethod = "token"
	}
	if node.WebScheme == "" {
		node.WebScheme = "https"
	}
	if node.VhostHTTPPort == 0 && node.WebBaseDomain != "" {
		node.VhostHTTPPort = 8080
	}
}

func validateNodeConfig(node NodeConfig) error {
	if !nodeIDPattern.MatchString(node.ID) {
		return httpx.BadRequest("node id must be 2-64 chars and contain only letters, numbers, _ or -")
	}
	if err := validate.Name(node.Name); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if strings.TrimSpace(node.ServerAddr) == "" {
		return httpx.BadRequest("serverAddr is required")
	}
	if strings.Contains(node.ServerAddr, "://") || strings.ContainsAny(node.ServerAddr, " /\\") {
		return httpx.BadRequest("serverAddr must be a host or IP without scheme or path")
	}
	if err := validate.Port(node.FrpsPort); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if node.AuthMethod != "token" {
		return httpx.BadRequest("authMethod must be token")
	}
	if node.WebScheme != "" && node.WebScheme != "http" && node.WebScheme != "https" {
		return httpx.BadRequest("webScheme must be http or https")
	}
	if node.WebBaseDomain != "" {
		if strings.ContainsAny(node.WebBaseDomain, " /\\:") {
			return httpx.BadRequest("webBaseDomain contains invalid characters")
		}
		if err := validate.Port(node.VhostHTTPPort); err != nil {
			return httpx.BadRequest("vhostHTTPPort " + err.Error())
		}
	}
	for _, pr := range node.AllowPorts {
		if err := validate.UserRemotePort(pr.From); err != nil {
			return httpx.BadRequest("allowPorts.from " + err.Error())
		}
		if err := validate.UserRemotePort(pr.To); err != nil {
			return httpx.BadRequest("allowPorts.to " + err.Error())
		}
		if pr.From > pr.To {
			return httpx.BadRequest("allowPorts range from must be <= to")
		}
	}
	return nil
}

func normalizePortRanges(values []control.PortRange) []control.PortRange {
	if len(values) == 0 {
		return nil
	}
	out := make([]control.PortRange, 0, len(values))
	for _, pr := range values {
		out = append(out, control.PortRange{From: pr.From, To: pr.To})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From == out[j].From {
			return out[i].To < out[j].To
		}
		return out[i].From < out[j].From
	})
	return out
}

func sanitizeNodeID(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	lastDash := false
	for _, r := range v {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if (r == '-' || r == '_' || r == ' ' || r == '.') && b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 64 {
		out = strings.TrimRight(out[:64], "-")
	}
	if !nodeIDPattern.MatchString(out) {
		return ""
	}
	return out
}

func enabledRuleUsesNode(data Data, nodeID string) bool {
	for _, rule := range data.PortRules {
		if rule.NodeID == nodeID && rule.Enabled {
			return true
		}
	}
	return false
}

func remotePortAllowed(port int, ranges []control.PortRange) bool {
	if len(ranges) == 0 {
		return true
	}
	for _, pr := range ranges {
		if port >= pr.From && port <= pr.To {
			return true
		}
	}
	return false
}

func cloneData(data Data) (Data, error) {
	var out Data
	raw, err := json.Marshal(data)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	NormalizeData(&out)
	return out, nil
}

func newDirectID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
