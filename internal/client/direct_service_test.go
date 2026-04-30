package client

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectPortRulesCreatePatchDeleteAndRenderConfig(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		Name:       "Public frps",
		ServerAddr: "149.118.158.112",
		FrpsPort:   7000,
		AuthMethod: "token",
		AuthToken:  "test-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var verifyCount int
	var applyCount int
	svc.applyAll = nil
	svc.verify = func(ctx context.Context, frpcPath, configPath string) error {
		verifyCount++
		raw, err := os.ReadFile(configPath)
		if err != nil {
			return err
		}
		config := string(raw)
		for _, want := range []string{
			`serverAddr = "149.118.158.112"`,
			`serverPort = 7000`,
			`method = "token"`,
			`token = "test-token"`,
		} {
			if !strings.Contains(config, want) {
				t.Fatalf("candidate config missing %q:\n%s", want, config)
			}
		}
		return nil
	}
	svc.apply = func(ctx context.Context, frpcPath, configPath string) error {
		applyCount++
		return nil
	}

	nodes, err := svc.ListNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].ID != DefaultNodeID || !nodes[0].AuthTokenSet {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}

	rule, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		NodeID:     DefaultNodeID,
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 18080,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.Name != "tcp-18080" || !rule.Enabled {
		t.Fatalf("unexpected rule defaults: %#v", rule)
	}
	raw, err := os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	config := string(raw)
	for _, want := range []string{
		`[[proxies]]`,
		`name = "port.` + rule.ID + `"`,
		`type = "tcp"`,
		`localIP = "127.0.0.1"`,
		`localPort = 8080`,
		`remotePort = 18080`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
	if strings.Contains(config, `[metadatas]`) {
		t.Fatalf("direct config should not include legacy metadata:\n%s", config)
	}

	enabled := false
	if _, err := svc.PatchPortRule(context.Background(), rule.ID, PatchPortRuleRequest{Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `[[proxies]]`) {
		t.Fatalf("disabled rule should not render a proxy:\n%s", raw)
	}

	if err := svc.DeletePortRule(context.Background(), rule.ID); err != nil {
		t.Fatal(err)
	}
	rules, err := svc.ListPortRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected all rules deleted, got %#v", rules)
	}
	if verifyCount != 3 || applyCount != 3 {
		t.Fatalf("expected verify/apply for create, patch and delete; got %d/%d", verifyCount, applyCount)
	}
}

func TestDirectPortRuleRejectsDuplicateRemotePort(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		ServerAddr: "149.118.158.112",
		FrpsPort:   7000,
		AuthMethod: "token",
		AuthToken:  "test-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	svc.applyAll = nil
	svc.verify = func(context.Context, string, string) error { return nil }
	svc.apply = func(context.Context, string, string) error { return nil }

	req := CreatePortRuleRequest{Protocol: "udp", LocalIP: "127.0.0.1", LocalPort: 9000, RemotePort: 19000}
	if _, err := svc.CreatePortRule(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePortRule(context.Background(), req); err == nil {
		t.Fatal("expected duplicate remote port to fail")
	}
}

func TestDirectNodeDoctorReportsConnectivityAndMissingToken(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		ServerAddr: "127.0.0.1",
		FrpsPort:   port,
		AuthMethod: "token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateNode(CreateNodeRequest{
		ID:         "local-node",
		Name:       "Local node",
		ServerAddr: "127.0.0.1",
		FrpsPort:   port,
		AuthMethod: "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.DoctorNode(context.Background(), "local-node")
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]NodeDoctorCheck{}
	for _, check := range result.Checks {
		checks[check.ID] = check
	}
	if result.Overall != "fail" {
		t.Fatalf("expected fail because token is missing, got %#v", result)
	}
	if checks["frps-tcp"].Status != "pass" {
		t.Fatalf("expected tcp connectivity to pass: %#v", checks["frps-tcp"])
	}
	if checks["auth-token"].Status != "fail" {
		t.Fatalf("expected auth token check to fail: %#v", checks["auth-token"])
	}
	if checks["web-dns"].Status != "skipped" || checks["web-http"].Status != "skipped" {
		t.Fatalf("expected web checks to be skipped: %#v", result.Checks)
	}
}

func TestDirectNodesCreateUpdateDeleteAndApplyActiveRules(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		ServerAddr: "149.118.158.112",
		FrpsPort:   7000,
		AuthMethod: "token",
		AuthToken:  "test-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var applyCount int
	svc.applyAll = nil
	svc.verify = func(context.Context, string, string) error { return nil }
	svc.apply = func(context.Context, string, string) error {
		applyCount++
		return nil
	}

	node, err := svc.CreateNode(CreateNodeRequest{
		ID:         "backup",
		Name:       "Backup frps",
		ServerAddr: "backup.example.com",
		FrpsPort:   7001,
		AuthMethod: "token",
		AuthToken:  "backup-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if node.ID != "backup" || !node.AuthTokenSet {
		t.Fatalf("unexpected node: %#v", node)
	}
	if _, err := svc.CreateNode(CreateNodeRequest{ID: "backup", Name: "Duplicate", ServerAddr: "dup.example.com"}); err == nil {
		t.Fatal("expected duplicate node to fail")
	}

	rule, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		NodeID:     "backup",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 18080,
	})
	if err != nil {
		t.Fatal(err)
	}

	nextToken := "new-token"
	updated, err := svc.UpdateNode(context.Background(), "backup", UpdateNodeRequest{
		Name:       "Backup frps updated",
		ServerAddr: "backup2.example.com",
		FrpsPort:   7002,
		AuthMethod: "token",
		AuthToken:  &nextToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ServerAddr != "backup2.example.com" || applyCount < 2 {
		t.Fatalf("expected active node update to apply, got node=%#v applyCount=%d", updated, applyCount)
	}
	raw, err := os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	config := string(raw)
	for _, want := range []string{
		`serverAddr = "backup2.example.com"`,
		`serverPort = 7002`,
		`token = "new-token"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("updated config missing %q:\n%s", want, config)
		}
	}

	if err := svc.DeleteNode("backup"); err == nil {
		t.Fatal("expected deleting node used by a rule to fail")
	}
	if err := svc.DeletePortRule(context.Background(), rule.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteNode("backup"); err != nil {
		t.Fatal(err)
	}
	nodes, err := svc.ListNodes()
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.ID == "backup" {
			t.Fatalf("backup node should have been deleted: %#v", nodes)
		}
	}
}

func TestDirectPortRulesRenderMultipleNodes(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		ServerAddr: "149.118.158.112",
		FrpsPort:   7000,
		AuthMethod: "token",
		AuthToken:  "test-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var applied []runtimeFrpcConfig
	svc.verify = func(context.Context, string, string) error { return nil }
	svc.applyAll = func(ctx context.Context, frpcPath string, configs []runtimeFrpcConfig) error {
		applied = append([]runtimeFrpcConfig(nil), configs...)
		return nil
	}

	if _, err := svc.CreateNode(CreateNodeRequest{
		ID:         "backup",
		Name:       "Backup frps",
		ServerAddr: "backup.example.com",
		FrpsPort:   7001,
		AuthMethod: "token",
		AuthToken:  "backup-token",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		NodeID:     DefaultNodeID,
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 18080,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		NodeID:     "backup",
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8081,
		RemotePort: 18080,
	}); err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected two runtime configs, got %#v", applied)
	}

	defaultRaw, err := os.ReadFile(filepath.Join(dir, "frpc", "frpc.toml"))
	if err != nil {
		t.Fatal(err)
	}
	backupRaw, err := os.ReadFile(filepath.Join(dir, "frpc", "frpc.backup.toml"))
	if err != nil {
		t.Fatal(err)
	}
	defaultConfig := string(defaultRaw)
	backupConfig := string(backupRaw)
	for _, want := range []string{
		`serverAddr = "149.118.158.112"`,
		`serverPort = 7000`,
		`localPort = 8080`,
		`remotePort = 18080`,
		`port = 7400`,
	} {
		if !strings.Contains(defaultConfig, want) {
			t.Fatalf("default config missing %q:\n%s", want, defaultConfig)
		}
	}
	for _, want := range []string{
		`serverAddr = "backup.example.com"`,
		`serverPort = 7001`,
		`token = "backup-token"`,
		`localPort = 8081`,
		`remotePort = 18080`,
	} {
		if !strings.Contains(backupConfig, want) {
			t.Fatalf("backup config missing %q:\n%s", want, backupConfig)
		}
	}
	if strings.Contains(backupConfig, `port = 7400`) {
		t.Fatalf("backup config should use a separate admin port:\n%s", backupConfig)
	}
}

func TestDirectWebPortRulesRenderCustomDomains(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:            DefaultNodeID,
		ServerAddr:    "149.118.158.112",
		FrpsPort:      7000,
		AuthMethod:    "token",
		AuthToken:     "test-token",
		WebBaseDomain: "ma1.gameuniverse.top",
		WebScheme:     "https",
		VhostHTTPPort: 8080,
	}))
	if err != nil {
		t.Fatal(err)
	}
	svc.applyAll = nil
	svc.verify = func(context.Context, string, string) error { return nil }
	svc.apply = func(context.Context, string, string) error { return nil }

	blog, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		Name:      "blog",
		Protocol:  "http",
		LocalIP:   "127.0.0.1",
		LocalPort: 3000,
		Subdomain: "blog",
	})
	if err != nil {
		t.Fatal(err)
	}
	if blog.Domain != "blog.ma1.gameuniverse.top" || blog.RemotePort != 0 {
		t.Fatalf("unexpected blog rule: %#v", blog)
	}
	dev, err := svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		Protocol:  "http",
		LocalIP:   "127.0.0.1",
		LocalPort: 5173,
		Domain:    "dev.ma1.gameuniverse.top",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dev.Subdomain != "dev" {
		t.Fatalf("expected subdomain to be derived from domain, got %#v", dev)
	}
	raw, err := os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	config := string(raw)
	for _, want := range []string{
		`name = "web.` + blog.ID + `"`,
		`type = "http"`,
		`localPort = 3000`,
		`customDomains = ["blog.ma1.gameuniverse.top"]`,
		`hostHeaderRewrite = "127.0.0.1"`,
		`name = "web.` + dev.ID + `"`,
		`localPort = 5173`,
		`customDomains = ["dev.ma1.gameuniverse.top"]`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
	if strings.Contains(config, `remotePort`) {
		t.Fatalf("http rules should not render remotePort:\n%s", config)
	}

	_, err = svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		Protocol:  "http",
		LocalIP:   "127.0.0.1",
		LocalPort: 8080,
		Subdomain: "blog",
	})
	if err == nil {
		t.Fatal("expected duplicate web domain to fail")
	}
}

func TestDirectPortRuleRollbackWhenApplyFails(t *testing.T) {
	dir := t.TempDir()
	svc, err := OpenService(filepath.Join(dir, "client.json"), "frpc.exe", filepath.Join(dir, "frpc"), WithDefaultNode(NodeConfig{
		ID:         DefaultNodeID,
		ServerAddr: "149.118.158.112",
		FrpsPort:   7000,
		AuthMethod: "token",
		AuthToken:  "test-token",
	}))
	if err != nil {
		t.Fatal(err)
	}
	svc.applyAll = nil
	svc.verify = func(context.Context, string, string) error { return nil }
	svc.apply = func(context.Context, string, string) error { return errors.New("reload failed") }

	_, err = svc.CreatePortRule(context.Background(), CreatePortRuleRequest{
		Protocol:   "tcp",
		LocalIP:    "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 18080,
	})
	if err == nil {
		t.Fatal("expected apply failure")
	}
	rules, err := svc.ListPortRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected rollback to remove candidate rule, got %#v", rules)
	}
	raw, err := os.ReadFile(svc.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `[[proxies]]`) {
		t.Fatalf("expected config rollback without proxies:\n%s", raw)
	}
}
