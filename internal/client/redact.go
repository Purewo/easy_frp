package client

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"frp-ui-backend/internal/httpx"
)

var (
	roomCodeRedactionPattern       = regexp.MustCompile(`room_[A-Za-z0-9]+\.[A-Za-z0-9._~+-]+`)
	jsonSecretRedactionPattern     = regexp.MustCompile(`(?i)("(?:roomCode|deviceToken|secretKey|authToken|adminPassword)"\s*:\s*)"[^"]*"`)
	assignmentSecretRedactionRegex = regexp.MustCompile(`(?i)\b(token|deviceToken|device_token|room_device_token|secretKey|secret_key|authToken|auth_token|adminPassword|admin_password)(\s*[:=]\s*)("[^"]*"|[^\s,}]+)`)
)

func redactDataSecrets(raw string, data Data) string {
	values := []string{
		data.Server.AuthToken,
		data.Group.DeviceToken,
		data.Group.SecretKey,
		data.Frpc.AdminPassword,
	}
	for _, node := range data.Nodes {
		values = append(values, node.AuthToken)
	}
	for _, rule := range data.RoomRules {
		values = append(values, rule.DeviceToken, rule.SecretKey)
	}
	return redactSecrets(raw, values...)
}

func redactSecrets(raw string, values ...string) string {
	out := raw
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) < 6 {
			continue
		}
		out = strings.ReplaceAll(out, value, "[redacted]")
	}
	out = roomCodeRedactionPattern.ReplaceAllString(out, "room_[redacted]")
	out = jsonSecretRedactionPattern.ReplaceAllString(out, `${1}"[redacted]"`)
	out = assignmentSecretRedactionRegex.ReplaceAllString(out, `${1}${2}"[redacted]"`)
	return out
}

func redactError(err error, values ...string) error {
	if err == nil {
		return nil
	}
	message := redactSecrets(err.Error(), values...)
	var statusErr *httpx.StatusError
	if errors.As(err, &statusErr) {
		return &httpx.StatusError{Code: statusErr.Code, Message: message}
	}
	return errors.New(message)
}

func sensitiveValuesFromPayload(payload any) []string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	var values []string
	collectSensitiveValues(v, &values)
	return values
}

func collectSensitiveValues(v any, values *[]string) {
	switch typed := v.(type) {
	case map[string]any:
		for key, value := range typed {
			if isSensitiveKey(key) {
				if s, ok := value.(string); ok {
					*values = append(*values, s)
				}
				continue
			}
			collectSensitiveValues(value, values)
		}
	case []any:
		for _, item := range typed {
			collectSensitiveValues(item, values)
		}
	}
}

func isSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "roomcode", "devicetoken", "secretkey", "authtoken", "adminpassword":
		return true
	default:
		return false
	}
}
