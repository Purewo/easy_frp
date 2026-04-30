package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	localclient "frp-ui-backend/internal/client"
)

const defaultCtlAPI = "http://127.0.0.1:7410"

type ctlClient struct {
	baseURL string
	http    *http.Client
	jsonOut bool
}

func runCtl(args []string) error {
	fs := flag.NewFlagSet("ctl", flag.ExitOnError)
	api := fs.String("api", envOrDefault("FRP_PANEL_API", defaultCtlAPI), "local backend API base URL")
	jsonOut := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		ctlUsage()
		return errors.New("missing ctl command")
	}

	c := ctlClient{
		baseURL: strings.TrimRight(*api, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
		jsonOut: *jsonOut,
	}

	switch rest[0] {
	case "nodes":
		return c.runNodes(rest[1:])
	case "ports":
		return c.runPorts(rest[1:])
	case "expose", "create":
		return c.runCreatePort(rest[1:])
	case "delete", "rm", "unexpose":
		return c.runDeletePort(rest[1:])
	case "enable":
		return c.runSetPortEnabled(rest[1:], true)
	case "disable":
		return c.runSetPortEnabled(rest[1:], false)
	case "status":
		return c.runStatus()
	case "reload":
		return c.runReload()
	case "logs":
		return c.runLogs(rest[1:])
	default:
		ctlUsage()
		return fmt.Errorf("unknown ctl command %q", rest[0])
	}
}

func ctlUsage() {
	fmt.Fprintf(os.Stderr, `usage:
  frp-panel ctl [--api %s] nodes
  frp-panel ctl [--api %s] nodes create --id backup --server-addr 1.2.3.4 --token <token>
  frp-panel ctl [--api %s] nodes update backup --server-addr 1.2.3.4 --token <token>
  frp-panel ctl [--api %s] nodes doctor backup
  frp-panel ctl [--api %s] nodes delete backup
  frp-panel ctl [--api %s] ports
  frp-panel ctl [--api %s] expose --local-port 18089 --subdomain cyberstream [--name cyberstream]
  frp-panel ctl [--api %s] expose cyberstream 18089
  frp-panel ctl [--api %s] ports create --protocol tcp --local-port 8080 --remote-port 18080
  frp-panel ctl [--api %s] delete <id|name|domain|subdomain>
  frp-panel ctl [--api %s] enable <id|name|domain|subdomain>
  frp-panel ctl [--api %s] disable <id|name|domain|subdomain>
  frp-panel ctl [--api %s] status
  frp-panel ctl [--api %s] reload
  frp-panel ctl [--api %s] logs [--tail 80]
`, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI, defaultCtlAPI)
}

func (c ctlClient) runNodes(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		return c.runListNodes()
	}
	switch args[0] {
	case "create", "add":
		return c.runCreateNode(args[1:])
	case "update", "edit":
		return c.runUpdateNode(args[1:])
	case "doctor", "check":
		return c.runDoctorNode(args[1:])
	case "delete", "rm":
		return c.runDeleteNode(args[1:])
	default:
		return fmt.Errorf("unknown nodes command %q", args[0])
	}
}

func (c ctlClient) runListNodes() error {
	var nodes []localclient.NodeView
	if err := c.getJSON("/v1/nodes", &nodes); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(nodes)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tSERVER\tWEB\tAUTH")
	for _, node := range nodes {
		web := "-"
		if node.WebBaseDomain != "" {
			web = "*." + node.WebBaseDomain
		}
		auth := "missing"
		if node.AuthTokenSet {
			auth = "ok"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s:%d\t%s\t%s\n", node.ID, node.Name, node.ServerAddr, node.FrpsPort, web, auth)
	}
	return tw.Flush()
}

func (c ctlClient) runCreateNode(args []string) error {
	fs := flag.NewFlagSet("nodes create", flag.ExitOnError)
	id := fs.String("id", "", "node id")
	name := fs.String("name", "", "node name")
	serverAddr := fs.String("server-addr", "", "frps server address")
	frpsPort := fs.Int("frps-port", 7000, "frps bind port")
	token := fs.String("token", "", "frps auth token")
	webBaseDomain := fs.String("web-base-domain", "", "wildcard web base domain")
	webScheme := fs.String("web-scheme", "https", "public web scheme")
	vhostHTTPPort := fs.Int("vhost-http-port", 8080, "frps vhost HTTP port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 0 && *id == "" {
		*id = positionals[0]
	}
	if len(positionals) > 1 && *serverAddr == "" {
		*serverAddr = positionals[1]
	}
	req := localclient.CreateNodeRequest{
		ID:            strings.TrimSpace(*id),
		Name:          strings.TrimSpace(cliDefaultString(*name, *id)),
		ServerAddr:    strings.TrimSpace(*serverAddr),
		FrpsPort:      *frpsPort,
		AuthMethod:    "token",
		AuthToken:     strings.TrimSpace(*token),
		WebBaseDomain: strings.TrimSpace(*webBaseDomain),
		WebScheme:     strings.TrimSpace(*webScheme),
		VhostHTTPPort: *vhostHTTPPort,
	}
	var node localclient.NodeView
	if err := c.postJSON("/v1/nodes", req, &node); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(node)
	}
	fmt.Printf("created node %s\n", node.ID)
	fmt.Printf("server: %s:%d\n", node.ServerAddr, node.FrpsPort)
	if node.WebBaseDomain != "" {
		fmt.Printf("web: %s://*.%s\n", node.WebScheme, node.WebBaseDomain)
	}
	fmt.Printf("auth: %s\n", authState(node.AuthTokenSet))
	return nil
}

func (c ctlClient) runUpdateNode(args []string) error {
	fs := flag.NewFlagSet("nodes update", flag.ExitOnError)
	name := fs.String("name", "", "node name")
	serverAddr := fs.String("server-addr", "", "frps server address")
	frpsPort := fs.Int("frps-port", 0, "frps bind port")
	token := fs.String("token", "", "frps auth token")
	clearToken := fs.Bool("clear-token", false, "clear stored auth token")
	webBaseDomain := fs.String("web-base-domain", "", "wildcard web base domain")
	webScheme := fs.String("web-scheme", "", "public web scheme")
	vhostHTTPPort := fs.Int("vhost-http-port", 0, "frps vhost HTTP port")
	selector := ""
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		selector = args[0]
		parseArgs = args[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	positionals := fs.Args()
	if selector == "" && len(positionals) == 1 {
		selector = positionals[0]
	}
	if selector == "" || len(positionals) > 1 || (len(positionals) == 1 && positionals[0] != selector) {
		return errors.New("nodes update requires exactly one node id")
	}
	current, err := c.resolveNode(selector)
	if err != nil {
		return err
	}
	nextName := *name
	if nextName == "" {
		nextName = current.Name
	}
	nextServerAddr := *serverAddr
	if nextServerAddr == "" {
		nextServerAddr = current.ServerAddr
	}
	nextFrpsPort := *frpsPort
	if nextFrpsPort == 0 {
		nextFrpsPort = current.FrpsPort
	}
	nextWebBaseDomain := *webBaseDomain
	if nextWebBaseDomain == "" {
		nextWebBaseDomain = current.WebBaseDomain
	}
	nextWebScheme := *webScheme
	if nextWebScheme == "" {
		nextWebScheme = current.WebScheme
	}
	nextVhostHTTPPort := *vhostHTTPPort
	if nextVhostHTTPPort == 0 {
		nextVhostHTTPPort = current.VhostHTTPPort
	}
	req := localclient.UpdateNodeRequest{
		Name:           strings.TrimSpace(nextName),
		ServerAddr:     strings.TrimSpace(nextServerAddr),
		FrpsPort:       nextFrpsPort,
		AuthMethod:     "token",
		ClearAuthToken: *clearToken,
		WebBaseDomain:  strings.TrimSpace(nextWebBaseDomain),
		WebScheme:      strings.TrimSpace(nextWebScheme),
		VhostHTTPPort:  nextVhostHTTPPort,
		AllowPorts:     current.AllowPorts,
	}
	if *token != "" {
		value := strings.TrimSpace(*token)
		req.AuthToken = &value
	}
	var node localclient.NodeView
	if err := c.putJSON("/v1/nodes/"+current.ID, req, &node); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(node)
	}
	fmt.Printf("updated node %s\n", node.ID)
	fmt.Printf("server: %s:%d\n", node.ServerAddr, node.FrpsPort)
	if node.WebBaseDomain != "" {
		fmt.Printf("web: %s://*.%s\n", node.WebScheme, node.WebBaseDomain)
	}
	fmt.Printf("auth: %s\n", authState(node.AuthTokenSet))
	return nil
}

func (c ctlClient) runDoctorNode(args []string) error {
	if len(args) != 1 {
		return errors.New("nodes doctor requires exactly one node id")
	}
	node, err := c.resolveNode(args[0])
	if err != nil {
		return err
	}
	var result localclient.NodeDoctorResult
	if err := c.postJSON("/v1/nodes/"+node.ID+"/doctor", nil, &result); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(result)
	}
	fmt.Printf("node: %s (%s)\n", result.Node.ID, result.Node.Name)
	fmt.Printf("overall: %s\n", result.Overall)
	if result.TestedDomain != "" {
		fmt.Printf("testedDomain: %s\n", result.TestedDomain)
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tSTATUS\tMS\tMESSAGE")
	for _, check := range result.Checks {
		ms := "-"
		if check.DurationMS > 0 {
			ms = strconv.FormatInt(check.DurationMS, 10)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", check.Name, check.Status, ms, strings.ReplaceAll(check.Message, "\n", " "))
	}
	return tw.Flush()
}

func (c ctlClient) runDeleteNode(args []string) error {
	if len(args) != 1 {
		return errors.New("nodes delete requires exactly one node id")
	}
	node, err := c.resolveNode(args[0])
	if err != nil {
		return err
	}
	if err := c.delete("/v1/nodes/" + node.ID); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(map[string]string{"deleted": node.ID})
	}
	fmt.Printf("deleted node %s (%s)\n", node.ID, node.Name)
	return nil
}

func (c ctlClient) runPorts(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		return c.runListPorts()
	}
	switch args[0] {
	case "create", "add", "expose":
		return c.runCreatePort(args[1:])
	case "delete", "rm", "unexpose":
		return c.runDeletePort(args[1:])
	case "enable":
		return c.runSetPortEnabled(args[1:], true)
	case "disable":
		return c.runSetPortEnabled(args[1:], false)
	default:
		return fmt.Errorf("unknown ports command %q", args[0])
	}
}

func (c ctlClient) runListPorts() error {
	nodes, rules, err := c.nodesAndPorts()
	if err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(rules)
	}

	nodeByID := map[string]localclient.NodeView{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPROTO\tLOCAL\tPUBLIC\tENABLED")
	for _, rule := range rules {
		node := nodeByID[rule.NodeID]
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s:%d\t%s\t%t\n",
			rule.ID,
			dash(rule.Name),
			rule.Protocol,
			rule.LocalIP,
			rule.LocalPort,
			publicAddress(rule, node),
			rule.Enabled,
		)
	}
	return tw.Flush()
}

func (c ctlClient) runCreatePort(args []string) error {
	fs := flag.NewFlagSet("expose", flag.ExitOnError)
	nodeID := fs.String("node-id", localclient.DefaultNodeID, "frps node id")
	name := fs.String("name", "", "rule name")
	protocol := fs.String("protocol", "http", "protocol: http, tcp, or udp")
	localIP := fs.String("local-ip", "127.0.0.1", "local service IP")
	localPort := fs.Int("local-port", 0, "local service port")
	remotePort := fs.Int("remote-port", 0, "remote port for tcp/udp")
	subdomain := fs.String("subdomain", "", "HTTP subdomain under the node web base domain")
	domain := fs.String("domain", "", "full HTTP domain under the node web base domain")
	disabled := fs.Bool("disabled", false, "create the rule disabled")
	if err := fs.Parse(args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 0 && *name == "" {
		*name = positionals[0]
	}
	if len(positionals) > 0 && *protocol == "http" && *subdomain == "" && *domain == "" {
		*subdomain = positionals[0]
	}
	if len(positionals) > 1 && *localPort == 0 {
		parsed, err := strconv.Atoi(positionals[1])
		if err != nil {
			return fmt.Errorf("invalid local port %q", positionals[1])
		}
		*localPort = parsed
	}
	if *localPort == 0 {
		return errors.New("--local-port is required")
	}
	proto := strings.ToLower(strings.TrimSpace(*protocol))
	if proto != "http" && proto != "tcp" && proto != "udp" {
		return errors.New("--protocol must be http, tcp, or udp")
	}
	if proto != "http" && *remotePort == 0 {
		return errors.New("--remote-port is required for tcp/udp")
	}

	enabled := !*disabled
	req := localclient.CreatePortRuleRequest{
		NodeID:     strings.TrimSpace(*nodeID),
		Name:       strings.TrimSpace(*name),
		Protocol:   proto,
		LocalIP:    strings.TrimSpace(*localIP),
		LocalPort:  *localPort,
		RemotePort: *remotePort,
		Subdomain:  strings.TrimSpace(*subdomain),
		Domain:     strings.TrimSpace(*domain),
		Enabled:    &enabled,
	}
	var rule localclient.PortRule
	if err := c.postJSON("/v1/ports", req, &rule); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(rule)
	}

	nodes, err := c.getNodes()
	if err != nil {
		nodes = nil
	}
	node := findNode(nodes, rule.NodeID)
	fmt.Printf("created %s\n", rule.ID)
	fmt.Printf("local:  %s:%d\n", rule.LocalIP, rule.LocalPort)
	fmt.Printf("public: %s\n", publicAddress(rule, node))
	fmt.Printf("enabled: %t\n", rule.Enabled)
	return nil
}

func (c ctlClient) runDeletePort(args []string) error {
	if len(args) != 1 {
		return errors.New("delete requires exactly one selector")
	}
	rule, err := c.resolveRule(args[0])
	if err != nil {
		return err
	}
	if err := c.delete("/v1/ports/" + rule.ID); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(map[string]string{"deleted": rule.ID})
	}
	fmt.Printf("deleted %s (%s)\n", rule.ID, rule.Name)
	return nil
}

func (c ctlClient) runSetPortEnabled(args []string, enabled bool) error {
	if len(args) != 1 {
		if enabled {
			return errors.New("enable requires exactly one selector")
		}
		return errors.New("disable requires exactly one selector")
	}
	rule, err := c.resolveRule(args[0])
	if err != nil {
		return err
	}
	var updated localclient.PortRule
	if err := c.patchJSON("/v1/ports/"+rule.ID, localclient.PatchPortRuleRequest{Enabled: &enabled}, &updated); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(updated)
	}
	state := "disabled"
	if updated.Enabled {
		state = "enabled"
	}
	fmt.Printf("%s %s (%s)\n", state, updated.ID, updated.Name)
	return nil
}

func (c ctlClient) runStatus() error {
	var status localclient.FrpcStatus
	if err := c.getJSON("/v1/frpc/status", &status); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(status)
	}
	fmt.Printf("running: %t\n", status.Running)
	if status.PID != 0 {
		fmt.Printf("pid: %d\n", status.PID)
	}
	fmt.Printf("config: %s\n", status.ConfigPath)
	if len(status.Nodes) > 0 {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NODE\tRUNNING\tPID\tCONFIG\tERROR")
		for _, node := range status.Nodes {
			pid := "-"
			if node.PID != 0 {
				pid = strconv.Itoa(node.PID)
			}
			fmt.Fprintf(tw, "%s\t%t\t%s\t%s\t%s\n", node.NodeID, node.Running, pid, node.ConfigPath, dash(node.LastError))
		}
		_ = tw.Flush()
	}
	if status.LastError != "" {
		fmt.Printf("lastError: %s\n", status.LastError)
	}
	return nil
}

func (c ctlClient) runReload() error {
	var status localclient.FrpcStatus
	if err := c.postJSON("/v1/frpc/reload", nil, &status); err != nil {
		return err
	}
	if c.jsonOut {
		return printJSON(status)
	}
	fmt.Printf("running: %t\n", status.Running)
	if status.PID != 0 {
		fmt.Printf("pid: %d\n", status.PID)
	}
	fmt.Printf("config: %s\n", status.ConfigPath)
	if len(status.Nodes) > 0 {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NODE\tRUNNING\tPID\tCONFIG\tERROR")
		for _, node := range status.Nodes {
			pid := "-"
			if node.PID != 0 {
				pid = strconv.Itoa(node.PID)
			}
			fmt.Fprintf(tw, "%s\t%t\t%s\t%s\t%s\n", node.NodeID, node.Running, pid, node.ConfigPath, dash(node.LastError))
		}
		_ = tw.Flush()
	}
	return nil
}

func (c ctlClient) runLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	tail := fs.Int("tail", 80, "number of lines to print; 0 prints all")
	if err := fs.Parse(args); err != nil {
		return err
	}
	raw, err := c.getText("/v1/logs")
	if err != nil {
		return err
	}
	if *tail > 0 {
		raw = tailLines(raw, *tail)
	}
	fmt.Print(raw)
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		fmt.Println()
	}
	return nil
}

func (c ctlClient) nodesAndPorts() ([]localclient.NodeView, []localclient.PortRule, error) {
	nodes, err := c.getNodes()
	if err != nil {
		return nil, nil, err
	}
	var rules []localclient.PortRule
	if err := c.getJSON("/v1/ports", &rules); err != nil {
		return nil, nil, err
	}
	return nodes, rules, nil
}

func (c ctlClient) getNodes() ([]localclient.NodeView, error) {
	var nodes []localclient.NodeView
	if err := c.getJSON("/v1/nodes", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (c ctlClient) resolveRule(selector string) (localclient.PortRule, error) {
	nodes, rules, err := c.nodesAndPorts()
	if err != nil {
		return localclient.PortRule{}, err
	}
	nodeByID := map[string]localclient.NodeView{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	selector = strings.TrimSpace(selector)
	var matches []localclient.PortRule
	for _, rule := range rules {
		node := nodeByID[rule.NodeID]
		if rule.ID == selector ||
			rule.Name == selector ||
			rule.Domain == selector ||
			rule.Subdomain == selector ||
			publicAddress(rule, node) == selector {
			matches = append(matches, rule)
		}
	}
	if len(matches) == 0 {
		return localclient.PortRule{}, fmt.Errorf("no port rule matches %q", selector)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return localclient.PortRule{}, fmt.Errorf("selector %q matched multiple rules: %s", selector, strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func (c ctlClient) resolveNode(selector string) (localclient.NodeView, error) {
	nodes, err := c.getNodes()
	if err != nil {
		return localclient.NodeView{}, err
	}
	selector = strings.TrimSpace(selector)
	var matches []localclient.NodeView
	for _, node := range nodes {
		if node.ID == selector || node.Name == selector || node.ServerAddr == selector {
			matches = append(matches, node)
		}
	}
	if len(matches) == 0 {
		return localclient.NodeView{}, fmt.Errorf("no node matches %q", selector)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return localclient.NodeView{}, fmt.Errorf("selector %q matched multiple nodes: %s", selector, strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func (c ctlClient) getJSON(path string, out any) error {
	return c.doJSON(http.MethodGet, path, nil, out)
}

func (c ctlClient) postJSON(path string, in any, out any) error {
	return c.doJSON(http.MethodPost, path, in, out)
}

func (c ctlClient) patchJSON(path string, in any, out any) error {
	return c.doJSON(http.MethodPatch, path, in, out)
}

func (c ctlClient) putJSON(path string, in any, out any) error {
	return c.doJSON(http.MethodPut, path, in, out)
}

func (c ctlClient) doJSON(method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s %s returned %s: %s", method, path, resp.Status, responseError(raw))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

func (c ctlClient) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("DELETE %s returned %s: %s", path, resp.Status, responseError(raw))
	}
	return nil
}

func (c ctlClient) getText(path string) (string, error) {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GET %s returned %s: %s", path, resp.Status, responseError(raw))
	}
	return string(raw), nil
}

func responseError(raw []byte) string {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		if payload.Error != "" {
			return payload.Error
		}
		if payload.Message != "" {
			return payload.Message
		}
	}
	return strings.TrimSpace(string(raw))
}

func printJSON(v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func publicAddress(rule localclient.PortRule, node localclient.NodeView) string {
	if rule.Protocol == "http" {
		if rule.Domain == "" {
			return "-"
		}
		scheme := node.WebScheme
		if scheme == "" {
			scheme = "https"
		}
		return scheme + "://" + rule.Domain
	}
	if rule.RemotePort == 0 {
		return "-"
	}
	host := node.ServerAddr
	if host == "" {
		host = rule.NodeID
	}
	return fmt.Sprintf("%s:%d", host, rule.RemotePort)
}

func findNode(nodes []localclient.NodeView, id string) localclient.NodeView {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
	}
	return localclient.NodeView{}
}

func dash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func authState(ok bool) string {
	if ok {
		return "ok"
	}
	return "missing"
}

func cliDefaultString(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func tailLines(v string, n int) string {
	lines := strings.Split(v, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if n <= 0 || len(lines) <= n {
		return v
	}
	return strings.Join(lines[len(lines)-n:], "\n") + "\n"
}
