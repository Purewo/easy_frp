package frp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderClientConfig(t *testing.T) {
	cfg := ClientConfig{
		ServerAddr:    "frps.example.com",
		ServerPort:    7000,
		AuthToken:     "server-token",
		AdminAddr:     "127.0.0.1",
		AdminPort:     7400,
		AdminUser:     "admin",
		AdminPassword: "admin-pass",
		GroupID:       "group1",
		DeviceID:      "dev1",
		DeviceToken:   "dev-token",
		Proxies: []Proxy{{
			Name:       "public.exp1",
			Type:       "tcp",
			LocalIP:    "127.0.0.1",
			LocalPort:  8080,
			RemotePort: 18080,
			Metadatas: map[string]string{
				"group_id":    "group1",
				"exposure_id": "exp1",
			},
		}},
		Visitors: []Visitor{{
			Name:       "visitor.route1.xtcp",
			Type:       "xtcp",
			ServerName: "group1.exp1.xtcp",
			SecretKey:  "secret",
			BindAddr:   "127.0.0.1",
			BindPort:   28080,
		}},
	}

	got := RenderClientConfig(cfg)
	for _, want := range []string{
		`serverAddr = "frps.example.com"`,
		`remotePort = 18080`,
		`[proxies.metadatas]`,
		`exposure_id = "exp1"`,
		`[[visitors]]`,
		`serverName = "group1.exp1.xtcp"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
}

func TestRenderedClientConfigAcceptedByFrpc(t *testing.T) {
	frpcExe := os.Getenv("FRPC_EXE")
	if frpcExe == "" {
		t.Skip("FRPC_EXE is not set")
	}

	cfg := ClientConfig{
		ServerAddr:    "127.0.0.1",
		ServerPort:    7000,
		AuthToken:     "token",
		AdminAddr:     "127.0.0.1",
		AdminPort:     7400,
		AdminUser:     "admin",
		AdminPassword: "admin-pass",
		GroupID:       "group1",
		DeviceID:      "dev1",
		DeviceToken:   "dev-token",
		Proxies: []Proxy{
			{
				Name:       "public.exp1",
				Type:       "tcp",
				LocalIP:    "127.0.0.1",
				LocalPort:  8080,
				RemotePort: 18080,
				Metadatas: map[string]string{
					"group_id":     "group1",
					"device_id":    "dev1",
					"device_token": "dev-token",
					"exposure_id":  "exp1",
				},
			},
			{
				Name:      "group1.exp2.xtcp",
				Type:      "xtcp",
				LocalIP:   "127.0.0.1",
				LocalPort: 8081,
				SecretKey: "secret",
				Metadatas: map[string]string{"exposure_id": "exp2"},
			},
			{
				Name:              "web.exp3",
				Type:              "http",
				LocalIP:           "127.0.0.1",
				LocalPort:         8082,
				CustomDomains:     []string{"dev.example.com"},
				HostHeaderRewrite: "127.0.0.1",
			},
		},
		Visitors: []Visitor{{
			Name:       "visitor.route1.xtcp",
			Type:       "xtcp",
			ServerName: "group1.exp2.xtcp",
			SecretKey:  "secret",
			BindAddr:   "127.0.0.1",
			BindPort:   28080,
		}},
	}

	configPath := filepath.Join(t.TempDir(), "frpc.toml")
	if err := os.WriteFile(configPath, []byte(RenderClientConfig(cfg)), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(frpcExe, "verify", "-c", configPath).CombinedOutput()
	if err != nil {
		t.Fatalf("frpc verify failed: %v\n%s", err, out)
	}
}
