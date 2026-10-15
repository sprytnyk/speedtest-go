package web

import "testing"

func TestClassifyPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want string
	}{
		{"::1", "localhost IPv6 access"},
		{"fe80::1%eth0", "link-local IPv6 access"},
		{"febf::1", "link-local IPv6 access"},
		{"fd12:3456::1", "ULA IPv6 access"},
		{"127.0.0.1", "localhost IPv4 access"},
		{"::ffff:10.1.2.3", "private IPv4 access"},
		{"172.31.255.255", "private IPv4 access"},
		{"192.168.1.1", "private IPv4 access"},
		{"169.254.10.10", "link-local IPv4 access"},
		{"100.64.0.1", "CGNAT IPv4 access"},
		{"100.108.0.1", "CGNAT IPv4 access"},
		{"100.127.255.255", "CGNAT IPv4 access"},
		{"100.63.255.255", ""},
		{"100.128.0.1", ""},
		{"172.32.0.1", ""},
		{"203.0.113.42", ""},
		{"2001:db8::1", ""},
		{"not an ip", ""},
	}

	for _, tt := range tests {
		if got := classifyPrivateIP(tt.ip); got != tt.want {
			t.Errorf("classifyPrivateIP(%q) = %q, want %q", tt.ip, got, tt.want)
		}
	}
}
