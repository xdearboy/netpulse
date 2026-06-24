package tests

import (
	"net"
	"testing"

	"netpulse/internal/utils"
)

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"2001:4860:4860::8888", true},
		{"invalid", false},
		{"", false},
		{"999.999.999.999", false},
		{"192.168.1.1", true},
		{"::1", true},
		{"10.0.0.1", true},
	}
	for _, tt := range tests {
		got := utils.IsValidIP(tt.ip)
		if got != tt.want {
			t.Errorf("IsValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIsValidIPv4(t *testing.T) {
	if !utils.IsValidIPv4("8.8.8.8") {
		t.Error("expected true for 8.8.8.8")
	}
	if utils.IsValidIPv4("2001:4860::8888") {
		t.Error("expected false for IPv6")
	}
}

func TestIsValidIPv6(t *testing.T) {
	if !utils.IsValidIPv6("2001:4860:4860::8888") {
		t.Error("expected true for IPv6")
	}
	if utils.IsValidIPv6("8.8.8.8") {
		t.Error("expected false for IPv4")
	}
}

func TestIsValidASN(t *testing.T) {
	tests := []struct {
		asn  int
		want bool
	}{
		{15169, true},
		{0, false},
		{-1, false},
		{4294967295, true},
		{4294967296, false},
	}
	for _, tt := range tests {
		got := utils.IsValidASN(tt.asn)
		if got != tt.want {
			t.Errorf("IsValidASN(%d) = %v, want %v", tt.asn, got, tt.want)
		}
	}
}

func TestIsValidCIDR(t *testing.T) {
	tests := []struct {
		cidr string
		want bool
	}{
		{"8.8.8.0/24", true},
		{"10.0.0.0/8", true},
		{"invalid", false},
		{"8.8.8.8", false},
		{"2001:db8::/32", true},
	}
	for _, tt := range tests {
		got := utils.IsValidCIDR(tt.cidr)
		if got != tt.want {
			t.Errorf("IsValidCIDR(%q) = %v, want %v", tt.cidr, got, tt.want)
		}
	}
}

func TestParseASN(t *testing.T) {
	asn, err := utils.ParseASN("15169")
	if err != nil || asn != 15169 {
		t.Errorf("ParseASN(15169) = %d, %v", asn, err)
	}
	_, err = utils.ParseASN("invalid")
	if err == nil {
		t.Error("expected error for invalid ASN")
	}
}

func TestGetIPType(t *testing.T) {
	if utils.GetIPType("8.8.8.8") != "IPv4" {
		t.Error("expected IPv4")
	}
	if utils.GetIPType("2001:4860::8888") != "IPv6" {
		t.Error("expected IPv6")
	}
	if utils.GetIPType("invalid") != "unknown" {
		t.Error("expected unknown")
	}
}

func TestGetCIDRInfo(t *testing.T) {
	network, netmask, ipCount, err := utils.GetCIDRInfo("8.8.8.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if network != "8.8.8.0" {
		t.Errorf("network = %s, want 8.8.8.0", network)
	}
	if netmask != net.IPv4(255, 255, 255, 0).String() {
		t.Errorf("netmask = %s, want 255.255.255.0", netmask)
	}
	if ipCount != 256 {
		t.Errorf("ipCount = %d, want 256", ipCount)
	}
}
