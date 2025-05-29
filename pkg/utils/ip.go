package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// GenerateIPs generates a list of IP addresses from a CIDR notation
func GenerateIPs(network string) []string {
	var ips []string

	// Simple implementation for /24 networks
	if strings.HasSuffix(network, "/24") {
		base := strings.TrimSuffix(network, "/24")
		baseIP := strings.Split(base, ".")
		if len(baseIP) == 4 {
			for i := 1; i < 255; i++ {
				ip := fmt.Sprintf("%s.%s.%s.%d", baseIP[0], baseIP[1], baseIP[2], i)
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

// CompareIPs compares two IP addresses for sorting
func CompareIPs(ip1, ip2 string) bool {
	// Convert IPs to comparable format
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")

	for i := 0; i < 4; i++ {
		var n1, n2 int
		fmt.Sscanf(parts1[i], "%d", &n1)
		fmt.Sscanf(parts2[i], "%d", &n2)
		if n1 != n2 {
			return n1 < n2
		}
	}
	return false
}

// ParsePortRange parses a port range string into a slice of port numbers
func ParsePortRange(portRange string) []int {
	var ports []int

	if strings.Contains(portRange, "-") {
		// Range format: 1-1000
		parts := strings.Split(portRange, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && start <= end {
				for i := start; i <= end; i++ {
					ports = append(ports, i)
				}
			}
		}
	} else {
		// Comma-separated format: 80,443,22
		for _, portStr := range strings.Split(portRange, ",") {
			port, err := strconv.Atoi(strings.TrimSpace(portStr))
			if err == nil && port > 0 && port <= 65535 {
				ports = append(ports, port)
			}
		}
	}

	return ports
}
