package frp

import (
	"fmt"
	"sort"
	"strings"
)

type ClientConfig struct {
	ServerAddr    string
	ServerPort    int
	AuthMethod    string
	AuthToken     string
	AdminAddr     string
	AdminPort     int
	AdminUser     string
	AdminPassword string
	GroupID       string
	DeviceID      string
	DeviceToken   string
	Metadatas     map[string]string
	Proxies       []Proxy
	Visitors      []Visitor
}

type Proxy struct {
	Name              string
	Type              string
	LocalIP           string
	LocalPort         int
	RemotePort        int
	SecretKey         string
	CustomDomains     []string
	HostHeaderRewrite string
	Metadatas         map[string]string
}

type Visitor struct {
	Name       string
	Type       string
	ServerName string
	SecretKey  string
	BindAddr   string
	BindPort   int
}

func RenderClientConfig(cfg ClientConfig) string {
	var b strings.Builder
	writeKV(&b, "serverAddr", cfg.ServerAddr)
	writeKVInt(&b, "serverPort", cfg.ServerPort)
	b.WriteString("\n[auth]\n")
	writeKV(&b, "method", defaultString(cfg.AuthMethod, "token"))
	writeKV(&b, "token", cfg.AuthToken)
	b.WriteString("\n[webServer]\n")
	writeKV(&b, "addr", defaultString(cfg.AdminAddr, "127.0.0.1"))
	writeKVInt(&b, "port", defaultInt(cfg.AdminPort, 7400))
	writeKV(&b, "user", defaultString(cfg.AdminUser, "admin"))
	writeKV(&b, "password", cfg.AdminPassword)

	globalMetas := cfg.Metadatas
	if len(globalMetas) == 0 && (cfg.GroupID != "" || cfg.DeviceID != "" || cfg.DeviceToken != "") {
		globalMetas = map[string]string{
			"group_id":     cfg.GroupID,
			"device_id":    cfg.DeviceID,
			"device_token": cfg.DeviceToken,
		}
	}
	if len(globalMetas) > 0 {
		b.WriteString("\n[metadatas]\n")
		writeMap(&b, globalMetas)
	}

	for _, p := range cfg.Proxies {
		b.WriteString("\n[[proxies]]\n")
		writeKV(&b, "name", p.Name)
		writeKV(&b, "type", p.Type)
		if p.LocalIP != "" {
			writeKV(&b, "localIP", p.LocalIP)
		}
		if p.LocalPort > 0 {
			writeKVInt(&b, "localPort", p.LocalPort)
		}
		if p.RemotePort > 0 {
			writeKVInt(&b, "remotePort", p.RemotePort)
		}
		if p.SecretKey != "" {
			writeKV(&b, "secretKey", p.SecretKey)
		}
		if len(p.CustomDomains) > 0 {
			writeArray(&b, "customDomains", p.CustomDomains)
		}
		if p.HostHeaderRewrite != "" {
			writeKV(&b, "hostHeaderRewrite", p.HostHeaderRewrite)
		}
		if len(p.Metadatas) > 0 {
			b.WriteString("[proxies.metadatas]\n")
			writeMap(&b, p.Metadatas)
		}
	}

	for _, v := range cfg.Visitors {
		b.WriteString("\n[[visitors]]\n")
		writeKV(&b, "name", v.Name)
		writeKV(&b, "type", v.Type)
		writeKV(&b, "serverName", v.ServerName)
		writeKV(&b, "secretKey", v.SecretKey)
		writeKV(&b, "bindAddr", defaultString(v.BindAddr, "127.0.0.1"))
		writeKVInt(&b, "bindPort", v.BindPort)
	}

	return b.String()
}

func PrivateProxyNames(groupID, exposureID string) (xtcp string, stcp string) {
	base := sanitizeName(groupID + "." + exposureID)
	return base + ".xtcp", base + ".stcp"
}

func sanitizeName(v string) string {
	v = strings.TrimSpace(v)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(v)
}

func writeMap(b *strings.Builder, values map[string]string) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeKV(b, k, values[k])
	}
}

func writeKV(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s = %q\n", key, value)
}

func writeKVInt(b *strings.Builder, key string, value int) {
	fmt.Fprintf(b, "%s = %d\n", key, value)
}

func writeArray(b *strings.Builder, key string, values []string) {
	b.WriteString(key)
	b.WriteString(" = [")
	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%q", value)
	}
	b.WriteString("]\n")
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func defaultInt(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}
