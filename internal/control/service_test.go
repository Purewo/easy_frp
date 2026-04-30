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
