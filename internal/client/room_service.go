package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"frp-ui-backend/internal/control"
	"frp-ui-backend/internal/httpx"
	"frp-ui-backend/internal/validate"
)

func (s *Service) snapshotForRoomMutation() (Data, error) {
	data, err := s.store.Snapshot()
	if err != nil {
		return Data{}, err
	}
	NormalizeData(&data)
	if strings.TrimSpace(data.Frpc.Path) == "" {
		return Data{}, httpx.BadRequest("frpc path is empty")
	}
	if strings.TrimSpace(data.Frpc.WorkDir) == "" {
		return Data{}, httpx.BadRequest("frpc workdir is empty")
	}
	return data, nil
}

func validateCreateRoomHostRequest(req CreateRoomHostRequest) error {
	if err := validate.Name(strings.TrimSpace(req.Name)); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if err := validateRoomTunnelProtocol(req.TunnelProtocol); err != nil {
		return err
	}
	if err := validateRoomStunServer(req.NatHoleStunServer); err != nil {
		return err
	}
	localIP := defaultString(strings.TrimSpace(req.LocalIP), "127.0.0.1")
	if err := validate.LocalIP(localIP); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if err := validate.Port(req.LocalPort); err != nil {
		return httpx.BadRequest(err.Error())
	}
	return nil
}

func validateJoinRoomRequest(data Data, req JoinRoomRequest) error {
	if err := validateRoomTunnelProtocol(req.TunnelProtocol); err != nil {
		return err
	}
	if err := validateRoomStunServer(req.NatHoleStunServer); err != nil {
		return err
	}
	name := strings.TrimSpace(req.Name)
	if name != "" {
		if err := validate.Name(name); err != nil {
			return httpx.BadRequest(err.Error())
		}
	}
	bindAddr := defaultString(strings.TrimSpace(req.BindAddr), "127.0.0.1")
	if err := validate.LocalIP(bindAddr); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if err := validate.Port(req.BindPort); err != nil {
		return httpx.BadRequest(err.Error())
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if enabled {
		for _, existing := range data.RoomRules {
			if !existing.Enabled || existing.Role != RoomRoleVisitor {
				continue
			}
			if existing.BindAddr == bindAddr && existing.BindPort == req.BindPort {
				return httpx.Conflict("room visitor bind port already in use")
			}
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(bindAddr, fmt.Sprint(req.BindPort)))
		if err != nil {
			return httpx.Conflict("room visitor bind port is not available: " + err.Error())
		}
		_ = ln.Close()
	}
	return nil
}

func validateRoomTunnelProtocol(v RoomTunnelProtocol) error {
	switch normalizeRoomTunnelProtocol(v) {
	case RoomTunnelXTCP, RoomTunnelSTCP:
		return nil
	default:
		return httpx.BadRequest("room tunnelProtocol must be xtcp or stcp")
	}
}

func validateRoomStunServer(v string) error {
	if strings.ContainsAny(strings.TrimSpace(v), " /\\") {
		return httpx.BadRequest("natHoleStunServer must be host:port without scheme or path")
	}
	return nil
}

func (s *Service) deleteRemoteRoomBestEffort(ctx context.Context, baseURL string, created control.CreateRoomResponse) {
	if strings.TrimSpace(created.Room.ID) == "" || strings.TrimSpace(created.Device.ID) == "" || strings.TrimSpace(created.DeviceToken) == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, joinURL(baseURL, "/v1/rooms/"+url.PathEscape(created.Room.ID)), nil)
	if err != nil {
		return
	}
	req.Header.Set("X-Room-ID", created.Room.ID)
	req.Header.Set("X-Room-Device-ID", created.Device.ID)
	req.Header.Set("X-Room-Device-Token", created.DeviceToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}
