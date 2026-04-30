package client

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"frp-ui-backend/internal/control"
)

func TestClientServiceJoinCreateExposureAndWriteConfig(t *testing.T) {
	controlSvc, err := control.OpenService(filepath.Join(t.TempDir(), "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controlSvc.CreateGroup(control.CreateGroupRequest{GroupID: "team_abc", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	node, err := controlSvc.CreateNode(control.CreateNodeRequest{
		Name:       "public-a",
		ServerAddr: "frps.example.com",
		FrpsPort:   7000,
		AllowPorts: []control.PortRange{{From: 30000, To: 30010}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(control.NewHandler(controlSvc))
	defer ts.Close()

	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConfigureServer(ConfigureServerRequest{
		BaseURL:  ts.URL,
		FrpsHost: "frps.example.com",
		FrpsPort: 7000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.JoinGroup(context.Background(), ClientJoinGroupRequest{
		GroupID: "team_abc", Password: "password123", DeviceName: "winbox",
	}); err != nil {
		t.Fatal(err)
	}
	exposure, err := svc.CreateExposure(context.Background(), ClientCreateExposureRequest{
		Mode:       "public",
		NodeID:     node.ID,
		Name:       "web",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 30001,
	})
	if err != nil {
		t.Fatal(err)
	}
	if exposure.ID == "" {
		t.Fatal("expected exposure id")
	}
	raw, err := os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	config := string(raw)
	for _, want := range []string{
		`serverAddr = "frps.example.com"`,
		`remotePort = 30001`,
		`exposure_id = "` + exposure.ID + `"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
}
