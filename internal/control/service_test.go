package control

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestControlServicePublicExposureAndPlugin(t *testing.T) {
	svc, err := OpenService(filepath.Join(t.TempDir(), "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateGroup(CreateGroupRequest{GroupID: "team_123", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	joined, err := svc.JoinGroup("team_123", JoinGroupRequest{Password: "password123", DeviceName: "winbox"})
	if err != nil {
		t.Fatal(err)
	}
	node, err := svc.CreateNode(CreateNodeRequest{
		Name:       "public-a",
		ServerAddr: "frps.example.com",
		FrpsPort:   7000,
		AllowPorts: []PortRange{{From: 20000, To: 20010}},
	})
	if err != nil {
		t.Fatal(err)
	}
	exposure, err := svc.CreatePublicExposure(CreatePublicExposureRequest{
		DeviceAuth: DeviceAuth{GroupID: joined.GroupID, DeviceID: joined.DeviceID, DeviceToken: joined.DeviceToken},
		NodeID:     node.ID,
		Name:       "web",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 20001,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreatePublicExposure(CreatePublicExposureRequest{
		DeviceAuth: DeviceAuth{GroupID: joined.GroupID, DeviceID: joined.DeviceID, DeviceToken: joined.DeviceToken},
		NodeID:     node.ID,
		Name:       "web2",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8081,
		RemotePort: 20001,
	})
	if err == nil {
		t.Fatal("expected duplicate remote port to fail")
	}

	raw, _ := json.Marshal(map[string]any{
		"op": "NewProxy",
		"content": map[string]any{
			"proxy_name":  "public." + exposure.ID,
			"proxy_type":  "tcp",
			"remote_port": 20001,
			"metadatas": map[string]string{
				"group_id":     joined.GroupID,
				"device_id":    joined.DeviceID,
				"device_token": joined.DeviceToken,
				"exposure_id":  exposure.ID,
			},
		},
	})
	decision := svc.PluginDecision(raw, "")
	if decision.Reject {
		t.Fatalf("expected plugin to accept proxy, got %#v", decision)
	}

	withoutBodyOp, _ := json.Marshal(map[string]any{
		"content": map[string]any{
			"metas": map[string]string{
				"group_id":     joined.GroupID,
				"device_id":    joined.DeviceID,
				"device_token": joined.DeviceToken,
			},
		},
	})
	decision = svc.PluginDecision(withoutBodyOp, "Login")
	if decision.Reject || !decision.Unchange {
		t.Fatalf("expected query-op plugin login to pass, got %#v", decision)
	}
}

func TestControlServiceRoomCreateJoinAndPlugin(t *testing.T) {
	svc, err := OpenService(filepath.Join(t.TempDir(), "server.json"), WithFrpsEndpoint("frps.example.com", 7000))
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateRoom(CreateRoomRequest{Name: "private api", DeviceName: "host"})
	if err != nil {
		t.Fatal(err)
	}
	if created.RoomCode == "" || created.Room.FrpsAddr != "frps.example.com" || created.Device.Role != "host" {
		t.Fatalf("unexpected created room: %#v", created)
	}
	joined, err := svc.JoinRoom(JoinRoomRequest{RoomCode: created.RoomCode, DeviceName: "visitor"})
	if err != nil {
		t.Fatal(err)
	}
	if joined.Room.ID != created.Room.ID || joined.Device.Role != "visitor" {
		t.Fatalf("unexpected joined room: %#v", joined)
	}

	loginRaw, _ := json.Marshal(map[string]any{
		"op": "Login",
		"content": map[string]any{
			"metadatas": map[string]string{
				"room_id":           created.Room.ID,
				"room_device_id":    created.Device.ID,
				"room_device_token": created.DeviceToken,
				"room_role":         "host",
			},
		},
	})
	decision := svc.PluginDecision(loginRaw, "")
	if decision.Reject {
		t.Fatalf("expected room login to pass, got %#v", decision)
	}

	proxyRaw, _ := json.Marshal(map[string]any{
		"op": "NewProxy",
		"content": map[string]any{
			"proxy_name": created.Room.ServerName,
			"proxy_type": "xtcp",
			"metadatas": map[string]string{
				"room_id":           created.Room.ID,
				"room_device_id":    created.Device.ID,
				"room_device_token": created.DeviceToken,
				"room_role":         "host",
			},
		},
	})
	decision = svc.PluginDecision(proxyRaw, "")
	if decision.Reject {
		t.Fatalf("expected room proxy to pass, got %#v", decision)
	}

	stcpProxyRaw, _ := json.Marshal(map[string]any{
		"op": "NewProxy",
		"content": map[string]any{
			"proxy_name": created.Room.ServerName,
			"proxy_type": "stcp",
			"metadatas": map[string]string{
				"room_id":           created.Room.ID,
				"room_device_id":    created.Device.ID,
				"room_device_token": created.DeviceToken,
				"room_role":         "host",
			},
		},
	})
	decision = svc.PluginDecision(stcpProxyRaw, "")
	if decision.Reject {
		t.Fatalf("expected room stcp proxy to pass, got %#v", decision)
	}

	badVisitorProxyRaw, _ := json.Marshal(map[string]any{
		"op": "NewProxy",
		"content": map[string]any{
			"proxy_name": created.Room.ServerName,
			"proxy_type": "xtcp",
			"metadatas": map[string]string{
				"room_id":           created.Room.ID,
				"room_device_id":    joined.Device.ID,
				"room_device_token": joined.DeviceToken,
				"room_role":         "visitor",
			},
		},
	})
	decision = svc.PluginDecision(badVisitorProxyRaw, "")
	if !decision.Reject {
		t.Fatalf("expected visitor NewProxy to be rejected")
	}

	if _, err := svc.JoinRoom(JoinRoomRequest{RoomCode: created.Room.ID + ".wrong", DeviceName: "bad"}); err == nil {
		t.Fatal("expected bad room code to fail")
	}
}
