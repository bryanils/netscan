package monitor

import (
	"fmt"
	"netscan/pkg/scanner"
	"strings"
	"time"
)

// MonitorPorts monitors a list of hosts and ports for changes in status
func MonitorPorts(hosts []string, ports []int) {
	fmt.Printf("\n👀 Monitoring %d hosts on %d ports (Ctrl+C to stop)\n", len(hosts), len(ports))
	fmt.Println("⏰ Checking every 30 seconds...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial scan
	CheckHosts(hosts, ports)

	for range ticker.C {
		fmt.Printf("\n⏰ %s - Checking status...\n", time.Now().Format("15:04:05"))
		CheckHosts(hosts, ports)
	}
}

// CheckHosts checks the status of a list of hosts and ports
func CheckHosts(hosts []string, ports []int) {
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		fmt.Printf("🔍 %s: ", host)

		var openPorts []int
		for _, port := range ports {
			if scanner.ScanPort(host, port).Open {
				openPorts = append(openPorts, port)
			}
		}

		if len(openPorts) > 0 {
			fmt.Printf("🟢 UP - Ports: %v\n", openPorts)
		} else {
			fmt.Printf("🔴 DOWN or filtered\n")
		}
	}
}
