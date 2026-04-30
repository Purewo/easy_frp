package validate

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,63}$`)

func GroupID(v string) error {
	if !idPattern.MatchString(v) {
		return fmt.Errorf("groupId must be 3-64 chars and contain only letters, numbers, _ or -")
	}
	return nil
}

func Name(v string) error {
	v = strings.TrimSpace(v)
	if len(v) < 1 || len(v) > 80 {
		return fmt.Errorf("name must be 1-80 chars")
	}
	return nil
}

func Password(v string) error {
	if len(v) < 8 {
		return fmt.Errorf("password must be at least 8 chars")
	}
	return nil
}

func Port(v int) error {
	if v < 1 || v > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}
	return nil
}

func UserRemotePort(v int) error {
	if v < 1024 || v > 65535 {
		return fmt.Errorf("remote port must be 1024-65535")
	}
	return nil
}

func LocalIP(v string) error {
	if v == "" {
		return fmt.Errorf("localIP is required")
	}
	if net.ParseIP(v) == nil && v != "localhost" {
		return fmt.Errorf("localIP must be an IP address or localhost")
	}
	return nil
}

func Protocol(v string) error {
	switch v {
	case "tcp", "udp", "http", "https":
		return nil
	default:
		return fmt.Errorf("protocol must be tcp, udp, http or https")
	}
}

func Domain(domain string, suffixes []string) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if strings.ContainsAny(domain, " /\\:") {
		return fmt.Errorf("domain contains invalid characters")
	}
	if len(suffixes) == 0 {
		return nil
	}
	for _, suffix := range suffixes {
		suffix = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(suffix)), ".")
		d := strings.ToLower(domain)
		if d == suffix || strings.HasSuffix(d, "."+suffix) {
			return nil
		}
	}
	return fmt.Errorf("domain is not under an allowed suffix")
}
