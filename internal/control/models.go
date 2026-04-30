package control

import "time"

type Data struct {
	Groups       map[string]Group       `json:"groups"`
	Devices      map[string]Device      `json:"devices"`
	Nodes        map[string]Node        `json:"nodes"`
	Exposures    map[string]Exposure    `json:"exposures"`
	AccessRoutes map[string]AccessRoute `json:"accessRoutes"`
}

func NormalizeData(data *Data) {
	if data.Groups == nil {
		data.Groups = map[string]Group{}
	}
	if data.Devices == nil {
		data.Devices = map[string]Device{}
	}
	if data.Nodes == nil {
		data.Nodes = map[string]Node{}
	}
	if data.Exposures == nil {
		data.Exposures = map[string]Exposure{}
	}
	if data.AccessRoutes == nil {
		data.AccessRoutes = map[string]AccessRoute{}
	}
}

type Group struct {
	ID           string    `json:"id"`
	PasswordHash string    `json:"passwordHash,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type GroupView struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type Device struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"groupId"`
	Name       string    `json:"name"`
	TokenHash  string    `json:"tokenHash,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

type Node struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	ServerAddr     string      `json:"serverAddr"`
	FrpsPort       int         `json:"frpsPort"`
	AllowPorts     []PortRange `json:"allowPorts"`
	DomainSuffixes []string    `json:"domainSuffixes"`
	CreatedAt      time.Time   `json:"createdAt"`
}

type PortRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type Exposure struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"groupId"`
	DeviceID   string    `json:"deviceId"`
	NodeID     string    `json:"nodeId,omitempty"`
	Name       string    `json:"name"`
	Mode       string    `json:"mode"`
	Protocol   string    `json:"protocol"`
	LocalIP    string    `json:"localIP"`
	LocalPort  int       `json:"localPort"`
	RemotePort int       `json:"remotePort,omitempty"`
	Domain     string    `json:"domain,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type AccessRoute struct {
	ID               string    `json:"id"`
	GroupID          string    `json:"groupId"`
	DeviceID         string    `json:"deviceId"`
	ExposureID       string    `json:"exposureId"`
	BindAddr         string    `json:"bindAddr"`
	BindPort         int       `json:"bindPort"`
	FallbackBindPort int       `json:"fallbackBindPort,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type DeviceAuth struct {
	GroupID     string `json:"groupId"`
	DeviceID    string `json:"deviceId"`
	DeviceToken string `json:"deviceToken"`
}

type CreateGroupRequest struct {
	GroupID  string `json:"groupId"`
	Password string `json:"password"`
}

type JoinGroupRequest struct {
	Password   string `json:"password"`
	DeviceName string `json:"deviceName"`
}

type JoinGroupResponse struct {
	GroupID     string `json:"groupId"`
	DeviceID    string `json:"deviceId"`
	DeviceToken string `json:"deviceToken"`
}

type CreateNodeRequest struct {
	Name           string      `json:"name"`
	ServerAddr     string      `json:"serverAddr"`
	FrpsPort       int         `json:"frpsPort"`
	AllowPorts     []PortRange `json:"allowPorts"`
	DomainSuffixes []string    `json:"domainSuffixes"`
}

type CreatePrivateExposureRequest struct {
	DeviceAuth
	Name      string `json:"name"`
	LocalIP   string `json:"localIP"`
	LocalPort int    `json:"localPort"`
}

type CreatePublicExposureRequest struct {
	DeviceAuth
	NodeID     string `json:"nodeId"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	Domain     string `json:"domain"`
}

type UpdateExposureRequest struct {
	Enabled *bool `json:"enabled"`
}

type CreateAccessRouteRequest struct {
	DeviceAuth
	ExposureID       string `json:"exposureId"`
	BindAddr         string `json:"bindAddr"`
	BindPort         int    `json:"bindPort"`
	FallbackBindPort int    `json:"fallbackBindPort"`
}

type PluginDecision struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason,omitempty"`
	Unchange     bool   `json:"unchange,omitempty"`
}
