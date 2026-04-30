package control

import "time"

type Data struct {
	Config       ServerConfig           `json:"config"`
	Groups       map[string]Group       `json:"groups"`
	Devices      map[string]Device      `json:"devices"`
	Nodes        map[string]Node        `json:"nodes"`
	Exposures    map[string]Exposure    `json:"exposures"`
	AccessRoutes map[string]AccessRoute `json:"accessRoutes"`
	Rooms        map[string]Room        `json:"rooms"`
	RoomDevices  map[string]RoomDevice  `json:"roomDevices"`
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
	if data.Rooms == nil {
		data.Rooms = map[string]Room{}
	}
	if data.RoomDevices == nil {
		data.RoomDevices = map[string]RoomDevice{}
	}
}

type ServerConfig struct {
	FrpsAddr string `json:"frpsAddr"`
	FrpsPort int    `json:"frpsPort"`
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

type Room struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ServerName   string    `json:"serverName"`
	CodeHash     string    `json:"codeHash,omitempty"`
	HostDeviceID string    `json:"hostDeviceId"`
	FrpsAddr     string    `json:"frpsAddr"`
	FrpsPort     int       `json:"frpsPort"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type RoomView struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ServerName   string    `json:"serverName"`
	HostDeviceID string    `json:"hostDeviceId"`
	FrpsAddr     string    `json:"frpsAddr"`
	FrpsPort     int       `json:"frpsPort"`
	Enabled      bool      `json:"enabled"`
	MemberCount  int       `json:"memberCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type RoomDevice struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"roomId"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	TokenHash  string    `json:"tokenHash,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

type RoomDeviceView struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"roomId"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
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

type CreateRoomRequest struct {
	Name       string `json:"name"`
	DeviceName string `json:"deviceName"`
}

type CreateRoomResponse struct {
	Room        RoomView       `json:"room"`
	RoomCode    string         `json:"roomCode"`
	Device      RoomDeviceView `json:"device"`
	DeviceToken string         `json:"deviceToken"`
}

type JoinRoomRequest struct {
	RoomCode   string `json:"roomCode"`
	DeviceName string `json:"deviceName"`
}

type JoinRoomResponse struct {
	Room        RoomView       `json:"room"`
	Device      RoomDeviceView `json:"device"`
	DeviceToken string         `json:"deviceToken"`
}

type UpdateRoomRequest struct {
	Enabled *bool `json:"enabled"`
}

type PluginDecision struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason,omitempty"`
	Unchange     bool   `json:"unchange,omitempty"`
}
