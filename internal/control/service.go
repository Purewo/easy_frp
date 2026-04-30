package control

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"frp-ui-backend/internal/httpx"
	"frp-ui-backend/internal/security"
	"frp-ui-backend/internal/storage"
	"frp-ui-backend/internal/validate"
)

type Service struct {
	store *storage.JSONFile[Data]
	now   func() time.Time
}

type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	frpsAddr string
	frpsPort int
}

func WithFrpsEndpoint(addr string, port int) ServiceOption {
	return func(opts *serviceOptions) {
		opts.frpsAddr = strings.TrimSpace(addr)
		opts.frpsPort = port
	}
}

func OpenService(path string, options ...ServiceOption) (*Service, error) {
	opts := serviceOptions{}
	for _, option := range options {
		option(&opts)
	}
	store, err := storage.OpenJSONFile(path, Data{}, NormalizeData)
	if err != nil {
		return nil, err
	}
	svc := &Service{store: store, now: time.Now}
	if opts.frpsAddr != "" || opts.frpsPort != 0 {
		err = store.Update(func(data *Data) error {
			if opts.frpsAddr != "" {
				data.Config.FrpsAddr = opts.frpsAddr
			}
			if opts.frpsPort != 0 {
				data.Config.FrpsPort = opts.frpsPort
			}
			return nil
		})
	}
	return svc, err
}

func (s *Service) CreateGroup(req CreateGroupRequest) (GroupView, error) {
	if err := validate.GroupID(req.GroupID); err != nil {
		return GroupView{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Password(req.Password); err != nil {
		return GroupView{}, httpx.BadRequest(err.Error())
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return GroupView{}, err
	}

	var group Group
	err = s.store.Update(func(data *Data) error {
		if _, ok := data.Groups[req.GroupID]; ok {
			return httpx.Conflict("group already exists")
		}
		group = Group{ID: req.GroupID, PasswordHash: hash, CreatedAt: s.now().UTC()}
		data.Groups[group.ID] = group
		return nil
	})
	if err != nil {
		return GroupView{}, err
	}
	return toGroupView(group), nil
}

func (s *Service) JoinGroup(groupID string, req JoinGroupRequest) (JoinGroupResponse, error) {
	if err := validate.GroupID(groupID); err != nil {
		return JoinGroupResponse{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Name(req.DeviceName); err != nil {
		return JoinGroupResponse{}, httpx.BadRequest(err.Error())
	}

	var resp JoinGroupResponse
	err := s.store.Update(func(data *Data) error {
		group, ok := data.Groups[groupID]
		if !ok {
			return httpx.NotFound("group not found")
		}
		if !security.VerifyPassword(group.PasswordHash, req.Password) {
			return httpx.Unauthorized("invalid group password")
		}
		token, err := security.NewToken()
		if err != nil {
			return err
		}
		device := Device{
			ID:         newID("dev"),
			GroupID:    groupID,
			Name:       strings.TrimSpace(req.DeviceName),
			TokenHash:  security.HashToken(token),
			CreatedAt:  s.now().UTC(),
			LastSeenAt: s.now().UTC(),
		}
		data.Devices[device.ID] = device
		resp = JoinGroupResponse{GroupID: groupID, DeviceID: device.ID, DeviceToken: token}
		return nil
	})
	return resp, err
}

func (s *Service) ListNodes() ([]Node, error) {
	var nodes []Node
	err := s.store.View(func(data Data) error {
		nodes = make([]Node, 0, len(data.Nodes))
		for _, node := range data.Nodes {
			nodes = append(nodes, node)
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
		return nil
	})
	return nodes, err
}

func (s *Service) CreateNode(req CreateNodeRequest) (Node, error) {
	if err := validate.Name(req.Name); err != nil {
		return Node{}, httpx.BadRequest(err.Error())
	}
	if strings.TrimSpace(req.ServerAddr) == "" {
		return Node{}, httpx.BadRequest("serverAddr is required")
	}
	if err := validate.Port(req.FrpsPort); err != nil {
		return Node{}, httpx.BadRequest(err.Error())
	}
	for _, pr := range req.AllowPorts {
		if pr.From > pr.To {
			return Node{}, httpx.BadRequest("allowPorts from must be <= to")
		}
		if err := validate.UserRemotePort(pr.From); err != nil {
			return Node{}, httpx.BadRequest(err.Error())
		}
		if err := validate.UserRemotePort(pr.To); err != nil {
			return Node{}, httpx.BadRequest(err.Error())
		}
	}

	node := Node{
		ID:             newID("node"),
		Name:           strings.TrimSpace(req.Name),
		ServerAddr:     strings.TrimSpace(req.ServerAddr),
		FrpsPort:       req.FrpsPort,
		AllowPorts:     req.AllowPorts,
		DomainSuffixes: req.DomainSuffixes,
		CreatedAt:      s.now().UTC(),
	}
	err := s.store.Update(func(data *Data) error {
		data.Nodes[node.ID] = node
		return nil
	})
	return node, err
}

func (s *Service) DeleteNode(nodeID string) error {
	return s.store.Update(func(data *Data) error {
		if _, ok := data.Nodes[nodeID]; !ok {
			return httpx.NotFound("node not found")
		}
		for _, exposure := range data.Exposures {
			if exposure.NodeID == nodeID {
				return httpx.Conflict("node has active exposures")
			}
		}
		delete(data.Nodes, nodeID)
		return nil
	})
}

func (s *Service) ListExposures(groupID string) ([]Exposure, error) {
	var out []Exposure
	err := s.store.View(func(data Data) error {
		for _, exposure := range data.Exposures {
			if groupID == "" || exposure.GroupID == groupID {
				out = append(out, exposure)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
		return nil
	})
	return out, err
}

func (s *Service) CreatePrivateExposure(req CreatePrivateExposureRequest) (Exposure, error) {
	if err := s.validateDevice(req.DeviceAuth); err != nil {
		return Exposure{}, err
	}
	if err := validate.Name(req.Name); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}
	if err := validate.LocalIP(req.LocalIP); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Port(req.LocalPort); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}

	now := s.now().UTC()
	exposure := Exposure{
		ID:        newID("exp"),
		GroupID:   req.GroupID,
		DeviceID:  req.DeviceID,
		Name:      strings.TrimSpace(req.Name),
		Mode:      "private",
		Protocol:  "xtcp",
		LocalIP:   req.LocalIP,
		LocalPort: req.LocalPort,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := s.store.Update(func(data *Data) error {
		data.Exposures[exposure.ID] = exposure
		return nil
	})
	return exposure, err
}

func (s *Service) CreatePublicExposure(req CreatePublicExposureRequest) (Exposure, error) {
	if err := s.validateDevice(req.DeviceAuth); err != nil {
		return Exposure{}, err
	}
	if err := validate.Name(req.Name); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Protocol(req.Protocol); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}
	if err := validate.LocalIP(req.LocalIP); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Port(req.LocalPort); err != nil {
		return Exposure{}, httpx.BadRequest(err.Error())
	}

	var exposure Exposure
	err := s.store.Update(func(data *Data) error {
		node, ok := data.Nodes[req.NodeID]
		if !ok {
			return httpx.NotFound("node not found")
		}
		if req.Protocol == "tcp" || req.Protocol == "udp" {
			if err := validate.UserRemotePort(req.RemotePort); err != nil {
				return httpx.BadRequest(err.Error())
			}
			if !portAllowed(req.RemotePort, node.AllowPorts) {
				return httpx.BadRequest("remote port is not allowed on this node")
			}
			if portInUse(data.Exposures, req.NodeID, req.Protocol, req.RemotePort) {
				return httpx.Conflict("remote port already in use")
			}
		} else {
			if err := validate.Domain(req.Domain, node.DomainSuffixes); err != nil {
				return httpx.BadRequest(err.Error())
			}
			if domainInUse(data.Exposures, req.NodeID, req.Domain) {
				return httpx.Conflict("domain already in use")
			}
		}
		now := s.now().UTC()
		exposure = Exposure{
			ID:         newID("exp"),
			GroupID:    req.GroupID,
			DeviceID:   req.DeviceID,
			NodeID:     req.NodeID,
			Name:       strings.TrimSpace(req.Name),
			Mode:       "public",
			Protocol:   req.Protocol,
			LocalIP:    req.LocalIP,
			LocalPort:  req.LocalPort,
			RemotePort: req.RemotePort,
			Domain:     req.Domain,
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		data.Exposures[exposure.ID] = exposure
		return nil
	})
	return exposure, err
}

func (s *Service) UpdateExposure(id string, auth DeviceAuth, req UpdateExposureRequest) (Exposure, error) {
	if err := s.validateDevice(auth); err != nil {
		return Exposure{}, err
	}
	var exposure Exposure
	err := s.store.Update(func(data *Data) error {
		var ok bool
		exposure, ok = data.Exposures[id]
		if !ok {
			return httpx.NotFound("exposure not found")
		}
		if exposure.GroupID != auth.GroupID || exposure.DeviceID != auth.DeviceID {
			return httpx.Unauthorized("exposure owner mismatch")
		}
		if req.Enabled != nil {
			exposure.Enabled = *req.Enabled
		}
		exposure.UpdatedAt = s.now().UTC()
		data.Exposures[id] = exposure
		return nil
	})
	return exposure, err
}

func (s *Service) DeleteExposure(id string, auth DeviceAuth) error {
	if err := s.validateDevice(auth); err != nil {
		return err
	}
	return s.store.Update(func(data *Data) error {
		exposure, ok := data.Exposures[id]
		if !ok {
			return httpx.NotFound("exposure not found")
		}
		if exposure.GroupID != auth.GroupID || exposure.DeviceID != auth.DeviceID {
			return httpx.Unauthorized("exposure owner mismatch")
		}
		delete(data.Exposures, id)
		for routeID, route := range data.AccessRoutes {
			if route.ExposureID == id {
				delete(data.AccessRoutes, routeID)
			}
		}
		return nil
	})
}

func (s *Service) CreateAccessRoute(req CreateAccessRouteRequest) (AccessRoute, error) {
	if err := s.validateDevice(req.DeviceAuth); err != nil {
		return AccessRoute{}, err
	}
	if err := validate.Port(req.BindPort); err != nil {
		return AccessRoute{}, httpx.BadRequest(err.Error())
	}
	if req.FallbackBindPort != 0 {
		if err := validate.Port(req.FallbackBindPort); err != nil {
			return AccessRoute{}, httpx.BadRequest(err.Error())
		}
		if req.FallbackBindPort == req.BindPort {
			return AccessRoute{}, httpx.BadRequest("fallbackBindPort must differ from bindPort")
		}
	}
	if req.BindAddr == "" {
		req.BindAddr = "127.0.0.1"
	}

	var route AccessRoute
	err := s.store.Update(func(data *Data) error {
		exposure, ok := data.Exposures[req.ExposureID]
		if !ok {
			return httpx.NotFound("exposure not found")
		}
		if exposure.GroupID != req.GroupID {
			return httpx.Unauthorized("exposure is not in this group")
		}
		route = AccessRoute{
			ID:               newID("route"),
			GroupID:          req.GroupID,
			DeviceID:         req.DeviceID,
			ExposureID:       req.ExposureID,
			BindAddr:         req.BindAddr,
			BindPort:         req.BindPort,
			FallbackBindPort: req.FallbackBindPort,
			CreatedAt:        s.now().UTC(),
		}
		data.AccessRoutes[route.ID] = route
		return nil
	})
	return route, err
}

func (s *Service) ListRooms() ([]RoomView, error) {
	out := []RoomView{}
	err := s.store.View(func(data Data) error {
		rooms := make([]Room, 0, len(data.Rooms))
		for _, room := range data.Rooms {
			rooms = append(rooms, room)
		}
		sort.Slice(rooms, func(i, j int) bool { return rooms[i].CreatedAt.Before(rooms[j].CreatedAt) })
		for _, room := range rooms {
			out = append(out, toRoomView(room, data.RoomDevices))
		}
		return nil
	})
	return out, err
}

func (s *Service) CreateRoom(req CreateRoomRequest) (CreateRoomResponse, error) {
	if err := validate.Name(req.Name); err != nil {
		return CreateRoomResponse{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Name(req.DeviceName); err != nil {
		return CreateRoomResponse{}, httpx.BadRequest(err.Error())
	}

	roomSecret, err := security.NewToken()
	if err != nil {
		return CreateRoomResponse{}, err
	}
	deviceToken, err := security.NewToken()
	if err != nil {
		return CreateRoomResponse{}, err
	}
	now := s.now().UTC()
	roomID := newID("room")
	roomCode := roomID + "." + roomSecret
	device := RoomDevice{
		ID:         newID("rdev"),
		RoomID:     roomID,
		Name:       strings.TrimSpace(req.DeviceName),
		Role:       "host",
		TokenHash:  security.HashToken(deviceToken),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	room := Room{
		ID:           roomID,
		Name:         strings.TrimSpace(req.Name),
		ServerName:   "room." + roomID + ".xtcp",
		CodeHash:     security.HashToken(roomCode),
		HostDeviceID: device.ID,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err = s.store.Update(func(data *Data) error {
		frpsAddr := strings.TrimSpace(data.Config.FrpsAddr)
		if frpsAddr == "" {
			return httpx.BadRequest("frps endpoint is not configured")
		}
		frpsPort := data.Config.FrpsPort
		if frpsPort == 0 {
			frpsPort = 7000
		}
		if err := validate.Port(frpsPort); err != nil {
			return httpx.BadRequest("frps " + err.Error())
		}
		room.FrpsAddr = frpsAddr
		room.FrpsPort = frpsPort
		data.Rooms[room.ID] = room
		data.RoomDevices[device.ID] = device
		return nil
	})
	if err != nil {
		return CreateRoomResponse{}, err
	}
	return CreateRoomResponse{
		Room:        toRoomView(room, map[string]RoomDevice{device.ID: device}),
		RoomCode:    roomCode,
		Device:      toRoomDeviceView(device),
		DeviceToken: deviceToken,
	}, nil
}

func (s *Service) JoinRoom(req JoinRoomRequest) (JoinRoomResponse, error) {
	roomID, _, err := ParseRoomCode(req.RoomCode)
	if err != nil {
		return JoinRoomResponse{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Name(req.DeviceName); err != nil {
		return JoinRoomResponse{}, httpx.BadRequest(err.Error())
	}
	deviceToken, err := security.NewToken()
	if err != nil {
		return JoinRoomResponse{}, err
	}
	now := s.now().UTC()
	device := RoomDevice{
		ID:         newID("rdev"),
		RoomID:     roomID,
		Name:       strings.TrimSpace(req.DeviceName),
		Role:       "visitor",
		TokenHash:  security.HashToken(deviceToken),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	var room Room
	var memberCount int
	err = s.store.Update(func(data *Data) error {
		var ok bool
		room, ok = data.Rooms[roomID]
		if !ok {
			return httpx.NotFound("room not found")
		}
		if !room.Enabled {
			return httpx.BadRequest("room is disabled")
		}
		if security.HashToken(strings.TrimSpace(req.RoomCode)) != room.CodeHash {
			return httpx.Unauthorized("invalid room code")
		}
		data.RoomDevices[device.ID] = device
		memberCount = roomMemberCount(room.ID, data.RoomDevices)
		return nil
	})
	if err != nil {
		return JoinRoomResponse{}, err
	}
	view := toRoomView(room, nil)
	view.MemberCount = memberCount
	return JoinRoomResponse{Room: view, Device: toRoomDeviceView(device), DeviceToken: deviceToken}, nil
}

func (s *Service) GetRoom(roomID string) (RoomView, error) {
	var view RoomView
	err := s.store.View(func(data Data) error {
		room, ok := data.Rooms[roomID]
		if !ok {
			return httpx.NotFound("room not found")
		}
		view = toRoomView(room, data.RoomDevices)
		return nil
	})
	return view, err
}

func (s *Service) UpdateRoom(roomID string, auth RoomDeviceAuth, req UpdateRoomRequest) (RoomView, error) {
	if req.Enabled == nil {
		return RoomView{}, httpx.BadRequest("enabled is required")
	}
	if err := s.validateRoomDevice(auth, "host"); err != nil {
		return RoomView{}, err
	}
	var room Room
	var devices map[string]RoomDevice
	err := s.store.Update(func(data *Data) error {
		var ok bool
		room, ok = data.Rooms[roomID]
		if !ok {
			return httpx.NotFound("room not found")
		}
		if room.ID != auth.RoomID {
			return httpx.Unauthorized("room auth mismatch")
		}
		room.Enabled = *req.Enabled
		room.UpdatedAt = s.now().UTC()
		data.Rooms[room.ID] = room
		devices = data.RoomDevices
		return nil
	})
	if err != nil {
		return RoomView{}, err
	}
	return toRoomView(room, devices), nil
}

func (s *Service) DeleteRoom(roomID string, auth RoomDeviceAuth) error {
	if err := s.validateRoomDevice(auth, "host"); err != nil {
		return err
	}
	return s.store.Update(func(data *Data) error {
		room, ok := data.Rooms[roomID]
		if !ok {
			return httpx.NotFound("room not found")
		}
		if room.ID != auth.RoomID {
			return httpx.Unauthorized("room auth mismatch")
		}
		delete(data.Rooms, roomID)
		for deviceID, device := range data.RoomDevices {
			if device.RoomID == roomID {
				delete(data.RoomDevices, deviceID)
			}
		}
		return nil
	})
}

type RoomDeviceAuth struct {
	RoomID      string
	DeviceID    string
	DeviceToken string
}

func ParseRoomCode(roomCode string) (roomID string, roomSecret string, err error) {
	parts := strings.SplitN(strings.TrimSpace(roomCode), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("roomCode must be <roomId>.<secret>")
	}
	return parts[0], parts[1], nil
}

func (s *Service) validateDevice(auth DeviceAuth) error {
	if auth.GroupID == "" || auth.DeviceID == "" || auth.DeviceToken == "" {
		return httpx.Unauthorized("missing device auth")
	}
	return s.store.Update(func(data *Data) error {
		device, ok := data.Devices[auth.DeviceID]
		if !ok || device.GroupID != auth.GroupID {
			return httpx.Unauthorized("unknown device")
		}
		if err := security.RequireToken(auth.DeviceToken, device.TokenHash); err != nil {
			return httpx.Unauthorized(err.Error())
		}
		device.LastSeenAt = s.now().UTC()
		data.Devices[auth.DeviceID] = device
		return nil
	})
}

func (s *Service) validateRoomDevice(auth RoomDeviceAuth, requiredRole string) error {
	if auth.RoomID == "" || auth.DeviceID == "" || auth.DeviceToken == "" {
		return httpx.Unauthorized("missing room device auth")
	}
	return s.store.Update(func(data *Data) error {
		room, ok := data.Rooms[auth.RoomID]
		if !ok {
			return httpx.Unauthorized("unknown room")
		}
		if !room.Enabled && requiredRole != "host" {
			return httpx.Unauthorized("room disabled")
		}
		device, ok := data.RoomDevices[auth.DeviceID]
		if !ok || device.RoomID != auth.RoomID {
			return httpx.Unauthorized("unknown room device")
		}
		if requiredRole != "" && device.Role != requiredRole {
			return httpx.Unauthorized("room role mismatch")
		}
		if err := security.RequireToken(auth.DeviceToken, device.TokenHash); err != nil {
			return httpx.Unauthorized(err.Error())
		}
		device.LastSeenAt = s.now().UTC()
		data.RoomDevices[auth.DeviceID] = device
		return nil
	})
}

func (s *Service) Snapshot() (Data, error) {
	return s.store.Snapshot()
}

func toGroupView(group Group) GroupView {
	return GroupView{ID: group.ID, CreatedAt: group.CreatedAt}
}

func toRoomView(room Room, devices map[string]RoomDevice) RoomView {
	return RoomView{
		ID:           room.ID,
		Name:         room.Name,
		ServerName:   room.ServerName,
		HostDeviceID: room.HostDeviceID,
		FrpsAddr:     room.FrpsAddr,
		FrpsPort:     room.FrpsPort,
		Enabled:      room.Enabled,
		MemberCount:  roomMemberCount(room.ID, devices),
		CreatedAt:    room.CreatedAt,
		UpdatedAt:    room.UpdatedAt,
	}
}

func toRoomDeviceView(device RoomDevice) RoomDeviceView {
	return RoomDeviceView{
		ID:         device.ID,
		RoomID:     device.RoomID,
		Name:       device.Name,
		Role:       device.Role,
		CreatedAt:  device.CreatedAt,
		LastSeenAt: device.LastSeenAt,
	}
}

func roomMemberCount(roomID string, devices map[string]RoomDevice) int {
	count := 0
	for _, device := range devices {
		if device.RoomID == roomID {
			count++
		}
	}
	return count
}

func portAllowed(port int, ranges []PortRange) bool {
	if len(ranges) == 0 {
		return port >= 1024 && port <= 65535
	}
	for _, pr := range ranges {
		if port >= pr.From && port <= pr.To {
			return true
		}
	}
	return false
}

func portInUse(exposures map[string]Exposure, nodeID, protocol string, port int) bool {
	for _, exposure := range exposures {
		if exposure.NodeID == nodeID && exposure.Protocol == protocol && exposure.RemotePort == port {
			return true
		}
	}
	return false
}

func domainInUse(exposures map[string]Exposure, nodeID, domain string) bool {
	for _, exposure := range exposures {
		if exposure.NodeID == nodeID && strings.EqualFold(exposure.Domain, domain) {
			return true
		}
	}
	return false
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
