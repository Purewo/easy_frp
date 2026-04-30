package control

import (
	"encoding/json"
	"fmt"
	"strings"

	"frp-ui-backend/internal/security"
)

type pluginEnvelope struct {
	Op      string          `json:"op"`
	Content json.RawMessage `json:"content"`
}

type pluginContent struct {
	User             pluginUser        `json:"user"`
	UserLegacy       string            `json:"-"`
	ProxyName        string            `json:"proxy_name"`
	ProxyNameAlt     string            `json:"proxyName"`
	ProxyType        string            `json:"proxy_type"`
	ProxyTypeAlt     string            `json:"proxyType"`
	RemotePort       int               `json:"remote_port"`
	RemotePortAlt    int               `json:"remotePort"`
	CustomDomains    []string          `json:"custom_domains"`
	CustomDomainsAlt []string          `json:"customDomains"`
	Metas            map[string]string `json:"metas"`
	Metadatas        map[string]string `json:"metadatas"`
}

type pluginUser struct {
	User  string            `json:"user"`
	Metas map[string]string `json:"metas"`
	RunID string            `json:"run_id"`
}

func (s *Service) PluginDecision(raw json.RawMessage, fallbackOp string) PluginDecision {
	var env pluginEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return reject("invalid plugin request")
	}
	if env.Op == "" {
		env.Op = fallbackOp
	}
	if env.Op == "" {
		return reject("missing plugin op")
	}
	var content pluginContent
	if len(env.Content) > 0 {
		_ = json.Unmarshal(env.Content, &content)
	} else {
		_ = json.Unmarshal(raw, &content)
	}
	content.normalize()

	switch strings.ToLower(env.Op) {
	case "login":
		return s.pluginLogin(content)
	case "newproxy":
		return s.pluginNewProxy(content)
	case "newuserconn":
		return s.pluginNewUserConn(content)
	default:
		return PluginDecision{Reject: false, Unchange: true}
	}
}

func (s *Service) pluginLogin(content pluginContent) PluginDecision {
	groupID := content.Metadatas["group_id"]
	deviceID := content.Metadatas["device_id"]
	deviceToken := content.Metadatas["device_token"]
	if groupID == "" || deviceID == "" || deviceToken == "" {
		return reject("missing device metadata")
	}
	data, err := s.Snapshot()
	if err != nil {
		return reject("%s", err.Error())
	}
	device, ok := data.Devices[deviceID]
	if !ok || device.GroupID != groupID {
		return reject("unknown device")
	}
	if err := security.RequireToken(deviceToken, device.TokenHash); err != nil {
		return reject("invalid device token")
	}
	return PluginDecision{Reject: false, Unchange: true}
}

func (s *Service) pluginNewProxy(content pluginContent) PluginDecision {
	exposureID := content.Metadatas["exposure_id"]
	groupID := content.Metadatas["group_id"]
	deviceID := content.Metadatas["device_id"]
	if exposureID == "" || groupID == "" || deviceID == "" {
		return reject("missing exposure metadata")
	}
	data, err := s.Snapshot()
	if err != nil {
		return reject("%s", err.Error())
	}
	exposure, ok := data.Exposures[exposureID]
	if !ok {
		return reject("unknown exposure")
	}
	if !exposure.Enabled {
		return reject("exposure disabled")
	}
	if exposure.GroupID != groupID || exposure.DeviceID != deviceID {
		return reject("exposure owner mismatch")
	}
	if exposure.Mode == "public" {
		if exposure.Protocol != content.ProxyType {
			return reject("proxy type mismatch")
		}
		if exposure.RemotePort != 0 && exposure.RemotePort != content.RemotePort {
			return reject("remote port mismatch")
		}
		if exposure.Domain != "" && !containsDomain(content.CustomDomains, exposure.Domain) {
			return reject("domain mismatch")
		}
	}
	return PluginDecision{Reject: false, Unchange: true}
}

func (s *Service) pluginNewUserConn(content pluginContent) PluginDecision {
	exposureID := content.Metadatas["exposure_id"]
	if exposureID == "" {
		return PluginDecision{Reject: false, Unchange: true}
	}
	data, err := s.Snapshot()
	if err != nil {
		return reject("%s", err.Error())
	}
	exposure, ok := data.Exposures[exposureID]
	if !ok {
		return reject("unknown exposure")
	}
	if !exposure.Enabled {
		return reject("exposure disabled")
	}
	return PluginDecision{Reject: false, Unchange: true}
}

func (c *pluginContent) normalize() {
	if c.ProxyName == "" {
		c.ProxyName = c.ProxyNameAlt
	}
	if c.ProxyType == "" {
		c.ProxyType = c.ProxyTypeAlt
	}
	if c.RemotePort == 0 {
		c.RemotePort = c.RemotePortAlt
	}
	if len(c.CustomDomains) == 0 {
		c.CustomDomains = c.CustomDomainsAlt
	}
	if len(c.Metadatas) == 0 {
		c.Metadatas = c.Metas
	}
	if c.Metadatas == nil {
		c.Metadatas = map[string]string{}
	}
	for k, v := range c.User.Metas {
		if _, exists := c.Metadatas[k]; !exists {
			c.Metadatas[k] = v
		}
	}
}

func reject(format string, args ...any) PluginDecision {
	return PluginDecision{Reject: true, RejectReason: fmt.Sprintf(format, args...)}
}

func containsDomain(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
