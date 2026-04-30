package client

import (
	"time"

	"frp-ui-backend/internal/control"
)

type Data struct {
	Server       ServerConfig                   `json:"server"`
	Group        GroupConfig                    `json:"group"`
	Frpc         FrpcConfig                     `json:"frpc"`
	Exposures    map[string]control.Exposure    `json:"exposures"`
	AccessRoutes map[string]control.AccessRoute `json:"accessRoutes"`
	Nodes        map[string]NodeConfig          `json:"nodes"`
	PortRules    map[string]PortRule            `json:"portRules"`
	RoomRules    map[string]RoomRule            `json:"roomRules"`
}

func NormalizeData(data *Data) {
	if data.Server.FrpsPort == 0 {
		data.Server.FrpsPort = 7000
	}
	if data.Frpc.Path == "" {
		data.Frpc.Path = "frpc.exe"
	}
	if data.Frpc.WorkDir == "" {
		data.Frpc.WorkDir = "data/frpc"
	}
	if data.Frpc.AdminAddr == "" {
		data.Frpc.AdminAddr = "127.0.0.1"
	}
	if data.Frpc.AdminPort == 0 {
		data.Frpc.AdminPort = 7400
	}
	if data.Frpc.AdminUser == "" {
		data.Frpc.AdminUser = "admin"
	}
	if data.Exposures == nil {
		data.Exposures = map[string]control.Exposure{}
	}
	if data.AccessRoutes == nil {
		data.AccessRoutes = map[string]control.AccessRoute{}
	}
	if data.Nodes == nil {
		data.Nodes = map[string]NodeConfig{}
	}
	if data.PortRules == nil {
		data.PortRules = map[string]PortRule{}
	}
	if data.RoomRules == nil {
		data.RoomRules = map[string]RoomRule{}
	}
}

const (
	DefaultNodeID               = "default"
	DefaultRoomControlServerURL = "http://149.118.158.112:18080"
)

type NodeConfig struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ServerAddr    string              `json:"serverAddr"`
	FrpsPort      int                 `json:"frpsPort"`
	AuthMethod    string              `json:"authMethod"`
	AuthToken     string              `json:"authToken,omitempty"`
	WebBaseDomain string              `json:"webBaseDomain,omitempty"`
	WebScheme     string              `json:"webScheme,omitempty"`
	VhostHTTPPort int                 `json:"vhostHTTPPort,omitempty"`
	AllowPorts    []control.PortRange `json:"allowPorts,omitempty"`
	CreatedAt     time.Time           `json:"createdAt,omitempty"`
	UpdatedAt     time.Time           `json:"updatedAt,omitempty"`
}

type NodeView struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ServerAddr    string              `json:"serverAddr"`
	FrpsPort      int                 `json:"frpsPort"`
	AuthMethod    string              `json:"authMethod"`
	AuthTokenSet  bool                `json:"authTokenSet"`
	WebBaseDomain string              `json:"webBaseDomain,omitempty"`
	WebScheme     string              `json:"webScheme,omitempty"`
	VhostHTTPPort int                 `json:"vhostHTTPPort,omitempty"`
	AllowPorts    []control.PortRange `json:"allowPorts,omitempty"`
	CreatedAt     time.Time           `json:"createdAt,omitempty"`
	UpdatedAt     time.Time           `json:"updatedAt,omitempty"`
}

type CreateNodeRequest struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ServerAddr    string              `json:"serverAddr"`
	FrpsPort      int                 `json:"frpsPort"`
	AuthMethod    string              `json:"authMethod"`
	AuthToken     string              `json:"authToken"`
	WebBaseDomain string              `json:"webBaseDomain"`
	WebScheme     string              `json:"webScheme"`
	VhostHTTPPort int                 `json:"vhostHTTPPort"`
	AllowPorts    []control.PortRange `json:"allowPorts,omitempty"`
}

type UpdateNodeRequest struct {
	Name           string              `json:"name"`
	ServerAddr     string              `json:"serverAddr"`
	FrpsPort       int                 `json:"frpsPort"`
	AuthMethod     string              `json:"authMethod"`
	AuthToken      *string             `json:"authToken,omitempty"`
	ClearAuthToken bool                `json:"clearAuthToken,omitempty"`
	WebBaseDomain  string              `json:"webBaseDomain"`
	WebScheme      string              `json:"webScheme"`
	VhostHTTPPort  int                 `json:"vhostHTTPPort"`
	AllowPorts     []control.PortRange `json:"allowPorts,omitempty"`
}

type NodeDoctorResult struct {
	Node         NodeView          `json:"node"`
	Overall      string            `json:"overall"`
	TestedDomain string            `json:"testedDomain,omitempty"`
	Checks       []NodeDoctorCheck `json:"checks"`
}

type NodeDoctorCheck struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Message    string            `json:"message"`
	DurationMS int64             `json:"durationMs,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

type PortRule struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"nodeId"`
	Name       string    `json:"name"`
	Protocol   string    `json:"protocol"`
	LocalIP    string    `json:"localIP"`
	LocalPort  int       `json:"localPort"`
	RemotePort int       `json:"remotePort,omitempty"`
	Subdomain  string    `json:"subdomain,omitempty"`
	Domain     string    `json:"domain,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type RoomRole string

const (
	RoomRoleHost    RoomRole = "host"
	RoomRoleVisitor RoomRole = "visitor"
)

type RoomTunnelProtocol string

const (
	RoomTunnelXTCP RoomTunnelProtocol = "xtcp"
	RoomTunnelSTCP RoomTunnelProtocol = "stcp"
)

type RoomRule struct {
	ID                string             `json:"id"`
	RoomID            string             `json:"roomId"`
	Name              string             `json:"name"`
	Role              RoomRole           `json:"role"`
	TunnelProtocol    RoomTunnelProtocol `json:"tunnelProtocol,omitempty"`
	NatHoleStunServer string             `json:"natHoleStunServer,omitempty"`
	ServerName        string             `json:"serverName"`
	ServerAddr        string             `json:"serverAddr"`
	ServerPort        int                `json:"serverPort"`
	DeviceID          string             `json:"deviceId"`
	DeviceToken       string             `json:"deviceToken,omitempty"`
	SecretKey         string             `json:"secretKey,omitempty"`
	LocalIP           string             `json:"localIP,omitempty"`
	LocalPort         int                `json:"localPort,omitempty"`
	BindAddr          string             `json:"bindAddr,omitempty"`
	BindPort          int                `json:"bindPort,omitempty"`
	Enabled           bool               `json:"enabled"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

type RoomRuleView struct {
	ID                string             `json:"id"`
	RoomID            string             `json:"roomId"`
	RoomCode          string             `json:"roomCode,omitempty"`
	Name              string             `json:"name"`
	Role              RoomRole           `json:"role"`
	TunnelProtocol    RoomTunnelProtocol `json:"tunnelProtocol"`
	NatHoleStunServer string             `json:"natHoleStunServer,omitempty"`
	ServerName        string             `json:"serverName"`
	ServerAddr        string             `json:"serverAddr"`
	ServerPort        int                `json:"serverPort"`
	DeviceID          string             `json:"deviceId"`
	DeviceTokenSet    bool               `json:"deviceTokenSet"`
	SecretKeySet      bool               `json:"secretKeySet"`
	LocalIP           string             `json:"localIP,omitempty"`
	LocalPort         int                `json:"localPort,omitempty"`
	BindAddr          string             `json:"bindAddr,omitempty"`
	BindPort          int                `json:"bindPort,omitempty"`
	Enabled           bool               `json:"enabled"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

type CreatePortRuleRequest struct {
	NodeID     string `json:"nodeId"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	Subdomain  string `json:"subdomain"`
	Domain     string `json:"domain"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

type UpdatePortRuleRequest struct {
	NodeID     string `json:"nodeId"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	Subdomain  string `json:"subdomain"`
	Domain     string `json:"domain"`
	Enabled    bool   `json:"enabled"`
}

type PatchPortRuleRequest struct {
	Enabled *bool `json:"enabled"`
}

type CreateRoomHostRequest struct {
	Name              string             `json:"name"`
	DeviceName        string             `json:"deviceName"`
	ServerBaseURL     string             `json:"serverBaseURL,omitempty"`
	TunnelProtocol    RoomTunnelProtocol `json:"tunnelProtocol,omitempty"`
	NatHoleStunServer string             `json:"natHoleStunServer,omitempty"`
	LocalIP           string             `json:"localIP"`
	LocalPort         int                `json:"localPort"`
	Enabled           *bool              `json:"enabled,omitempty"`
}

type JoinRoomRequest struct {
	RoomCode          string             `json:"roomCode"`
	Name              string             `json:"name"`
	DeviceName        string             `json:"deviceName"`
	ServerBaseURL     string             `json:"serverBaseURL,omitempty"`
	TunnelProtocol    RoomTunnelProtocol `json:"tunnelProtocol,omitempty"`
	NatHoleStunServer string             `json:"natHoleStunServer,omitempty"`
	BindAddr          string             `json:"bindAddr"`
	BindPort          int                `json:"bindPort"`
	Enabled           *bool              `json:"enabled,omitempty"`
}

type PatchRoomRuleRequest struct {
	Enabled *bool `json:"enabled"`
}

type RoomDoctorResult struct {
	Rule    RoomRuleView      `json:"rule"`
	Overall string            `json:"overall"`
	Checks  []NodeDoctorCheck `json:"checks"`
}

type RoomRuleStatus struct {
	Rule    RoomRuleView   `json:"rule"`
	Process FrpcNodeStatus `json:"process"`
}

type NatHoleDiscoverRequest struct {
	StunServer string `json:"stunServer,omitempty"`
	LocalAddr  string `json:"localAddr,omitempty"`
}

type NatHoleDiscoverResult struct {
	Success           bool     `json:"success"`
	StunServer        string   `json:"stunServer,omitempty"`
	NatType           string   `json:"natType,omitempty"`
	Behavior          string   `json:"behavior,omitempty"`
	ExternalAddresses []string `json:"externalAddresses,omitempty"`
	LocalAddress      string   `json:"localAddress,omitempty"`
	PublicNetwork     *bool    `json:"publicNetwork,omitempty"`
	RawOutput         string   `json:"rawOutput"`
	Error             string   `json:"error,omitempty"`
	DurationMS        int64    `json:"durationMs,omitempty"`
}

type NetworkInterfaceView struct {
	Name      string `json:"name"`
	Index     int    `json:"index"`
	Address   string `json:"address"`
	Loopback  bool   `json:"loopback"`
	Multicast bool   `json:"multicast"`
}

type ServerConfig struct {
	BaseURL   string `json:"baseURL"`
	FrpsHost  string `json:"frpsHost"`
	FrpsPort  int    `json:"frpsPort"`
	AuthToken string `json:"authToken,omitempty"`
}

type GroupConfig struct {
	GroupID     string    `json:"groupId"`
	DeviceID    string    `json:"deviceId"`
	DeviceToken string    `json:"deviceToken"`
	SecretKey   string    `json:"secretKey"`
	JoinedAt    time.Time `json:"joinedAt,omitempty"`
}

type FrpcConfig struct {
	Path          string `json:"path"`
	WorkDir       string `json:"workDir"`
	AdminAddr     string `json:"adminAddr"`
	AdminPort     int    `json:"adminPort"`
	AdminUser     string `json:"adminUser"`
	AdminPassword string `json:"adminPassword"`
}

type ConfigureServerRequest struct {
	BaseURL   string `json:"baseURL"`
	FrpsHost  string `json:"frpsHost"`
	FrpsPort  int    `json:"frpsPort"`
	AuthToken string `json:"authToken"`
}

type ClientJoinGroupRequest struct {
	GroupID    string `json:"groupId"`
	Password   string `json:"password"`
	DeviceName string `json:"deviceName"`
}

type ClientCreateExposureRequest struct {
	Mode       string `json:"mode"`
	NodeID     string `json:"nodeId"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	Domain     string `json:"domain"`
}

type ClientCreateAccessRouteRequest struct {
	ExposureID       string `json:"exposureId"`
	BindAddr         string `json:"bindAddr"`
	BindPort         int    `json:"bindPort"`
	FallbackBindPort int    `json:"fallbackBindPort"`
}

type ClientState struct {
	Server       ServerConfig                   `json:"server"`
	Group        GroupConfig                    `json:"group"`
	Frpc         FrpcConfig                     `json:"frpc"`
	Exposures    map[string]control.Exposure    `json:"exposures"`
	AccessRoutes map[string]control.AccessRoute `json:"accessRoutes"`
	Nodes        []NodeView                     `json:"nodes"`
	PortRules    []PortRule                     `json:"portRules"`
	RoomRules    []RoomRuleView                 `json:"roomRules"`
}

type FrpcStatus struct {
	Running    bool             `json:"running"`
	PID        int              `json:"pid,omitempty"`
	ConfigPath string           `json:"configPath"`
	LastError  string           `json:"lastError,omitempty"`
	Nodes      []FrpcNodeStatus `json:"nodes,omitempty"`
}

type FrpcNodeStatus struct {
	NodeID         string             `json:"nodeId"`
	Running        bool               `json:"running"`
	PID            int                `json:"pid,omitempty"`
	ConfigPath     string             `json:"configPath"`
	LogPath        string             `json:"logPath,omitempty"`
	LastError      string             `json:"lastError,omitempty"`
	RoomRuleID     string             `json:"roomRuleId,omitempty"`
	RoomID         string             `json:"roomId,omitempty"`
	RoomName       string             `json:"roomName,omitempty"`
	RoomRole       RoomRole           `json:"roomRole,omitempty"`
	TunnelProtocol RoomTunnelProtocol `json:"tunnelProtocol,omitempty"`
	LocalEndpoint  string             `json:"localEndpoint,omitempty"`
}
