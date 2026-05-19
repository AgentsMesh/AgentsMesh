package blockstoreservice

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

const allowedHostsEnv = "BLOCKSTORE_WEBHOOK_ALLOW_HOSTS"

func ValidateWebhookURL(raw string) error {
	return validateWebhookURL(raw)
}

// validateWebhookURL is the SSRF guard for Agent-supplied action.url before
// the trigger engine POSTs. Rejects non-http(s), private/loopback/link-local,
// 169.254.169.254 metadata, and unresolvable hosts. Re-applied at fire-time
// for defense-in-depth against rows stored before this check landed.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported scheme %q: only http/https allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url missing host")
	}
	if hostInAllowlist(host) {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("url host %q is a private or reserved address", host)
		}
		return nil
	}
	switch strings.ToLower(host) {
	case "localhost", "localhost.localdomain", "ip6-localhost":
		return fmt.Errorf("url host %q is blocked", host)
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("host %q did not resolve: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("host %q resolved to no addresses", host)
	}
	for _, ip := range addrs {
		if isPrivateIP(ip) {
			return fmt.Errorf("url host %q resolves to private/reserved address %s", host, ip)
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127:
			return true
		case ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255:
			return true
		}
		return false
	}
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}

func hostInAllowlist(host string) bool {
	raw := os.Getenv(allowedHostsEnv)
	if raw == "" {
		return false
	}
	host = strings.ToLower(host)
	for _, entry := range strings.Split(raw, ",") {
		if strings.ToLower(strings.TrimSpace(entry)) == host {
			return true
		}
	}
	return false
}

func validateTriggerDefData(data blockstore.JSONMap) error {
	action, ok := data["action"].(map[string]any)
	if !ok {
		return nil
	}
	kind, _ := action["kind"].(string)
	if kind != "webhook" {
		return nil
	}
	rawURL, _ := action["url"].(string)
	if rawURL == "" {
		return nil
	}
	if err := validateWebhookURL(rawURL); err != nil {
		return fmt.Errorf("%w: action.url %s", blockstore.ErrColumnValueInvalid, err)
	}
	return nil
}
