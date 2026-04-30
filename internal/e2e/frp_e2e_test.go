package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"frp-ui-backend/internal/control"
	"frp-ui-backend/internal/frp"
)

func TestFrpPublicTCPExposureEndToEnd(t *testing.T) {
	frpcExe := os.Getenv("FRPC_EXE")
	frpsExe := os.Getenv("FRPS_EXE")
	if frpcExe == "" || frpsExe == "" {
		t.Skip("FRPC_EXE and FRPS_EXE are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dir := t.TempDir()
	controlSvc, err := control.OpenService(filepath.Join(dir, "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controlSvc.CreateGroup(control.CreateGroupRequest{GroupID: "team_e2e", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	joined, err := controlSvc.JoinGroup("team_e2e", control.JoinGroupRequest{Password: "password123", DeviceName: "e2e-client"})
	if err != nil {
		t.Fatal(err)
	}

	frpsPort := freePort(t)
	remotePort := freePort(t)
	adminPort := freePort(t)
	localListener := startLineServer(t, "frp-e2e-ok")
	defer localListener.Close()
	localPort := localListener.Addr().(*net.TCPAddr).Port

	node, err := controlSvc.CreateNode(control.CreateNodeRequest{
		Name:       "e2e-node",
		ServerAddr: "127.0.0.1",
		FrpsPort:   frpsPort,
		AllowPorts: []control.PortRange{{From: remotePort, To: remotePort}},
	})
	if err != nil {
		t.Fatal(err)
	}
	exposure, err := controlSvc.CreatePublicExposure(control.CreatePublicExposureRequest{
		DeviceAuth: control.DeviceAuth{
			GroupID:     joined.GroupID,
			DeviceID:    joined.DeviceID,
			DeviceToken: joined.DeviceToken,
		},
		NodeID:     node.ID,
		Name:       "e2e-tcp",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  localPort,
		RemotePort: remotePort,
	})
	if err != nil {
		t.Fatal(err)
	}

	controlHTTP := httptest.NewServer(control.NewHandler(controlSvc))
	defer controlHTTP.Close()
	controlURL, err := url.Parse(controlHTTP.URL)
	if err != nil {
		t.Fatal(err)
	}

	frpsConfig := filepath.Join(dir, "frps.toml")
	writeFile(t, frpsConfig, fmt.Sprintf(`
bindAddr = "127.0.0.1"
bindPort = %d
proxyBindAddr = "127.0.0.1"

[[httpPlugins]]
name = "frp-ui-backend-e2e"
addr = %q
path = "/internal/frps/plugin"
ops = ["Login", "NewProxy", "NewUserConn"]
`, frpsPort, controlURL.Host))

	frpsCmd := startProcess(t, ctx, frpsExe, "-c", frpsConfig)
	defer stopProcess(frpsCmd)
	waitTCP(t, ctx, "127.0.0.1", frpsPort)

	frpcConfig := filepath.Join(dir, "frpc.toml")
	writeFile(t, frpcConfig, frp.RenderClientConfig(frp.ClientConfig{
		ServerAddr:    "127.0.0.1",
		ServerPort:    frpsPort,
		AdminAddr:     "127.0.0.1",
		AdminPort:     adminPort,
		AdminUser:     "admin",
		AdminPassword: "admin-pass",
		GroupID:       joined.GroupID,
		DeviceID:      joined.DeviceID,
		DeviceToken:   joined.DeviceToken,
		Proxies: []frp.Proxy{{
			Name:       "public." + exposure.ID,
			Type:       "tcp",
			LocalIP:    "127.0.0.1",
			LocalPort:  localPort,
			RemotePort: remotePort,
			Metadatas: map[string]string{
				"group_id":     joined.GroupID,
				"device_id":    joined.DeviceID,
				"device_token": joined.DeviceToken,
				"exposure_id":  exposure.ID,
			},
		}},
	}))

	verifyOut, err := exec.CommandContext(ctx, frpcExe, "verify", "-c", frpcConfig).CombinedOutput()
	if err != nil {
		t.Fatalf("frpc verify failed: %v\n%s", err, verifyOut)
	}

	frpcCmd := startProcess(t, ctx, frpcExe, "-c", frpcConfig)
	defer stopProcess(frpcCmd)
	waitTCP(t, ctx, "127.0.0.1", remotePort)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", remotePort), 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(line) != "frp-e2e-ok" {
		t.Fatalf("unexpected response %q", line)
	}
}

func startLineServer(t *testing.T, response string) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = bufio.NewReader(conn).ReadString('\n')
				_, _ = conn.Write([]byte(response + "\n"))
			}()
		}
	}()
	return ln
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startProcess(t *testing.T, ctx context.Context, exe string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(ctx, exe, args...)
	outPath := filepath.Join(t.TempDir(), filepath.Base(exe)+".log")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		_ = out.Close()
		t.Fatalf("start %s failed: %v", exe, err)
	}
	t.Cleanup(func() {
		_ = out.Close()
		if raw, err := os.ReadFile(outPath); err == nil && len(raw) > 0 && t.Failed() {
			t.Logf("%s output:\n%s", filepath.Base(exe), raw)
		}
	})
	return cmd
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func waitTCP(t *testing.T, ctx context.Context, host string, port int) {
	t.Helper()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	deadline := time.Now().Add(8 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s: %v", addr, err)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", addr, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func writeFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
