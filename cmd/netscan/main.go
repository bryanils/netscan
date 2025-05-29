package main

import (
	"bufio"
	"fmt"
	"netscan/pkg/models"
	"netscan/pkg/monitor"
	"netscan/pkg/scanner"
	"netscan/pkg/utils"
	"os"
	"strings"
)

func main() {
	fmt.Println("🔍 Network Discovery & Port Scanner")
	fmt.Println("-===================================-")

	usrIn := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\nSelect an option:")
		fmt.Println("1. Ping sweep (discover live hosts)")
		fmt.Println("2. Port scan single host")
		fmt.Println("3. Network discovery + port scan")
		fmt.Println("4. Monitor specific ports")
		fmt.Println("5. Exit")
		fmt.Print("Choice: ")

		usrIn.Scan()
		choice := strings.TrimSpace(usrIn.Text())

		switch choice {
		case "1":
			fmt.Print("Enter network (e.g., 192.168.1.0/24): ")
			usrIn.Scan()
			network := strings.TrimSpace(usrIn.Text())
			displayPingSweepResults(scanner.PingSweep(network))
		case "2":
			fmt.Print("Enter target IP: ")
			usrIn.Scan()
			target := strings.TrimSpace(usrIn.Text())
			fmt.Print("Enter port range (e.g., 1-1000 or 80,443,22): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := utils.ParsePortRange(portRange)
			displayPortScanResults(target, scanner.ScanPorts(target, ports))
		case "3":
			fmt.Print("Enter network (e.g., 192.168.1.0/24): ")
			usrIn.Scan()
			network := strings.TrimSpace(usrIn.Text())
			fmt.Print("Enter port range (e.g., 22,80,443): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := utils.ParsePortRange(portRange)
			displayNetworkDiscoveryResults(scanner.NetworkDiscovery(network, ports))
		case "4":
			fmt.Print("Enter hosts to monitor (comma-separated): ")
			usrIn.Scan()
			hosts := strings.Split(strings.TrimSpace(usrIn.Text()), ",")
			fmt.Print("Enter ports to monitor (comma-separated): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := utils.ParsePortRange(portRange)
			monitor.MonitorPorts(hosts, ports)
		case "5":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice!")
		}
	}
}

// displayPingSweepResults displays the results of a ping sweep
func displayPingSweepResults(hosts []models.HostResult) {
	for _, host := range hosts {
		fmt.Printf("🟢 %-15s (%.2fms)\n", host.IP, float64(host.Latency.Nanoseconds())/1000000)
	}
}

// displayPortScanResults displays the results of a port scan
func displayPortScanResults(target string, ports []models.PortResult) {
	for _, port := range ports {
		service := port.Service
		if service == "" {
			service = "Unknown"
		}
		fmt.Printf("🟢 Port %-5d %-12s", port.Port, service)
		if port.Banner != "" {
			fmt.Printf(" - %s", port.Banner)
		}
		fmt.Println()
	}
}

// displayNetworkDiscoveryResults displays the results of a network discovery
func displayNetworkDiscoveryResults(hosts []models.HostResult) {
	for _, host := range hosts {
		fmt.Printf("🖥️  %s\n", host.IP)
		if len(host.Ports) > 0 {
			for _, port := range host.Ports {
				service := port.Service
				if service == "" {
					service = "Unknown"
				}
				fmt.Printf("   🟢 %-5d %-12s", port.Port, service)
				if port.Banner != "" {
					fmt.Printf(" - %s", port.Banner)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("   📝 Host alive but no open ports found in scanned range\n")
		}
		fmt.Println()
	}
}
