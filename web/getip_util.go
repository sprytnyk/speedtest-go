package web

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// normalizeCandidateIP validates and normalizes an IP address candidate
// from a request header. It trims whitespace, takes the first comma-separated
// token (for XFF-like headers that may contain a chain), and validates.
func normalizeCandidateIP(raw string, ipv6 bool) string {
	ip := strings.TrimSpace(raw)
	// For XFF-like values, take the first address before a comma
	if idx := strings.Index(ip, ","); idx != -1 {
		ip = strings.TrimSpace(ip[:idx])
	}
	if ip == "" {
		return ""
	}
	if ipv6 {
		parsed := net.ParseIP(ip)
		if parsed != nil && parsed.To16() != nil && parsed.To4() == nil {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ""
	}
	parsed := net.ParseIP(ip)
	if parsed != nil {
		return strings.TrimPrefix(ip, "::ffff:")
	}
	return ""
}

// getClientIP extracts the real client IP from the request using the following
// priority chain, mirroring the PHP getIP_util.php behavior:
//
//  1. CF-Connecting-IPv6 (Cloudflare, must be a valid IPv6)
//  2. Client-IP
//  3. X-Real-IP
//  4. X-Forwarded-For (first address in the chain)
//  5. RemoteAddr (fallback)
func getClientIP(r *http.Request) string {
	// 1. Cloudflare IPv6 header — must be a valid IPv6 address
	if cf := r.Header.Get("CF-Connecting-IPv6"); cf != "" {
		if ip := normalizeCandidateIP(cf, true); ip != "" {
			return ip
		}
	}

	// 2–4. Other forwarding / proxy headers — accept any valid IP
	for _, header := range []string{"Client-IP", "X-Real-IP", "X-Forwarded-For"} {
		if v := r.Header.Get(header); v != "" {
			if ip := normalizeCandidateIP(v, false); ip != "" {
				return ip
			}
		}
	}

	// 5. Fallback: RemoteAddr set by the server
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr may not have a port in some environments
		ip = r.RemoteAddr
	}
	if parsed := net.ParseIP(ip); parsed != nil {
		return strings.TrimPrefix(ip, "::ffff:")
	}

	return ""
}

// specialRanges lists private and special-purpose ranges, mirroring the PHP
// getLocalOrPrivateIpInfo() function. Addresses in these ranges are never sent
// to the ISP lookup API.
var specialRanges = []struct {
	prefix netip.Prefix
	desc   string
}{
	{netip.MustParsePrefix("::1/128"), "localhost IPv6 access"},
	{netip.MustParsePrefix("fe80::/10"), "link-local IPv6 access"},
	{netip.MustParsePrefix("fc00::/7"), "ULA IPv6 access"},
	{netip.MustParsePrefix("127.0.0.0/8"), "localhost IPv4 access"},
	{netip.MustParsePrefix("10.0.0.0/8"), "private IPv4 access"},
	{netip.MustParsePrefix("172.16.0.0/12"), "private IPv4 access"},
	{netip.MustParsePrefix("192.168.0.0/16"), "private IPv4 access"},
	{netip.MustParsePrefix("169.254.0.0/16"), "link-local IPv4 access"},
	{netip.MustParsePrefix("100.64.0.0/10"), "CGNAT IPv4 access"},
}

// classifyPrivateIP returns a human-readable description if the IP is a
// private or special-purpose address, or an empty string otherwise.
func classifyPrivateIP(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	// Prefix.Contains never matches zoned or IPv4-mapped addresses
	addr = addr.WithZone("").Unmap()
	for _, r := range specialRanges {
		if r.prefix.Contains(addr) {
			return r.desc
		}
	}
	return ""
}
