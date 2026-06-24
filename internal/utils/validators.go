package utils

import (
	"net"
	"strconv"
)

func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func IsValidIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}

func IsValidIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() == nil
}

func IsValidASN(asn int) bool {
	return asn > 0 && asn <= 4294967295
}

func IsValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func ParseASN(asnStr string) (int, error) {
	asn, err := strconv.Atoi(asnStr)
	if err != nil {
		return 0, err
	}
	if !IsValidASN(asn) {
		return 0, err
	}
	return asn, nil
}

func GetIPType(ip string) string {
	if IsValidIPv4(ip) {
		return "IPv4"
	}
	if IsValidIPv6(ip) {
		return "IPv6"
	}
	return "unknown"
}

func GetCIDRInfo(cidr string) (network string, netmask string, ipCount int64, err error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", 0, err
	}

	ones, bits := ipNet.Mask.Size()
	if bits == 32 {
		ipCount = 1 << (32 - ones)
	} else {
		ipCount = 1 << (128 - ones)
	}

	return ipNet.IP.String(), net.IP(ipNet.Mask).String(), ipCount, nil
}
