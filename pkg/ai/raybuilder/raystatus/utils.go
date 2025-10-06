package raystatus

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// returns the first value for any of the keys, case-insensitive
func getEndpointURL(m map[string]string, keys ...string) string {
	if m == nil {
		return ""
	}
	// make a case-insensitive index
	idx := make(map[string]string, len(m))
	for k, v := range m {
		idx[strings.ToLower(k)] = v
	}
	for _, k := range keys {
		if v, ok := idx[strings.ToLower(k)]; ok && v != "" {
			return v
		}
	}
	return ""
}

// parsePort tries to extract a port from a URL or "host:port" string.
// returns 0 if not found / unparseable.
func parsePort(s string) int32 {
	if s == "" {
		return 0
	}
	// try full URL first
	if u, err := url.Parse(s); err == nil {
		// u.Host may be "host:port"
		if _, p, err := net.SplitHostPort(u.Host); err == nil {
			if n, err := strconv.Atoi(p); err == nil && n > 0 && n <= 65535 {
				return int32(n)
			}
		}
		// some endpoints might be just "host:port" without scheme in Path
		if u.Scheme == "" && u.Host == "" && strings.Contains(u.Path, ":") {
			if _, p, err := net.SplitHostPort(u.Path); err == nil {
				if n, err := strconv.Atoi(p); err == nil && n > 0 && n <= 65535 {
					return int32(n)
				}
			}
		}
	}
	// bare "host:port"
	if _, p, err := net.SplitHostPort(s); err == nil {
		if n, err := strconv.Atoi(p); err == nil && n > 0 && n <= 65535 {
			return int32(n)
		}
	}
	return 0
}
