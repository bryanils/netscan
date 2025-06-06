package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"netscan/scanner"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PortResult struct {
	Port    int
	Open    bool
	Service string
	Banner  string
}

type HostResult struct {
	IP      string
	Alive   bool
	Ports   []PortResult
	Latency time.Duration
}

// Common services for port identification
var commonServices = map[int]string{
	21:   "FTP",
	22:   "SSH",
	23:   "Telnet",
	25:   "SMTP",
	53:   "DNS",
	80:   "HTTP",
	110:  "POP3",
	135:  "RPC",
	139:  "NetBIOS",
	143:  "IMAP",
	443:  "HTTPS",
	445:  "SMB",
	993:  "IMAPS",
	995:  "POP3S",
	1433: "MSSQL",
	3306: "MySQL",
	3389: "RDP",
	5432: "PostgreSQL",
	5900: "VNC",
	6379: "Redis",
	8080: "HTTP-Alt",
	9200: "Elasticsearch",
}

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
			pingSweep(network)
		case "2":
			fmt.Print("Enter target IP: ")
			usrIn.Scan()
			target := strings.TrimSpace(usrIn.Text())
			fmt.Print("Enter port range (e.g., 1-1000 or 80,443,22): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := parsePortRange(portRange)
			scanPorts(target, ports)
		case "3":
			fmt.Print("Enter network (e.g., 192.168.1.0/24): ")
			usrIn.Scan()
			network := strings.TrimSpace(usrIn.Text())
			fmt.Print("Enter port range (e.g., 22,80,443): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := parsePortRange(portRange)
			scanner.NetworkDiscovery(network, ports)
		case "4":
			fmt.Print("Enter hosts to monitor (comma-separated): ")
			usrIn.Scan()
			hosts := strings.Split(strings.TrimSpace(usrIn.Text()), ",")
			fmt.Print("Enter ports to monitor (comma-separated): ")
			usrIn.Scan()
			portRange := strings.TrimSpace(usrIn.Text())
			ports := parsePortRange(portRange)
			monitorPorts(hosts, ports)
		case "5":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice!")
		}
	}
}

// Batch processing version for very large networks
func pingSweep(network string) {
	fmt.Printf("\n🔍 Batch scanning network: %s\n", network)

	ips := generateIPs(network)
	const batchSize = 254 // Process one subnet at a time
	const maxConcurrent = 500

	var allHosts []HostResult
	var resultsMutex sync.Mutex

	start := time.Now()

	for i := 0; i < len(ips); i += batchSize {
		end := i + batchSize
		if end > len(ips) {
			end = len(ips)
		}

		batch := ips[i:end]
		batchStart := time.Now()

		var wg sync.WaitGroup
		results := make(chan HostResult, len(batch))
		sem := make(chan struct{}, maxConcurrent)

		for _, ip := range batch {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				pingStart := time.Now()
				alive := pingHostFast(ip)
				latency := time.Since(pingStart)

				if alive {
					results <- HostResult{
						IP:      ip,
						Alive:   alive,
						Latency: latency,
					}
				}
			}(ip)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		var batchHosts []HostResult
		for result := range results {
			batchHosts = append(batchHosts, result)
		}

		resultsMutex.Lock()
		allHosts = append(allHosts, batchHosts...)
		resultsMutex.Unlock()

		batchElapsed := time.Since(batchStart)
		fmt.Printf("📈 Batch %d/%d: %d hosts found in %v\n",
			(i/batchSize)+1, (len(ips)+batchSize-1)/batchSize,
			len(batchHosts), batchElapsed)
	}

	elapsed := time.Since(start)

	sort.Slice(allHosts, func(i, j int) bool {
		return compareIPs(allHosts[i].IP, allHosts[j].IP)
	})

	fmt.Printf("\n✅ Batch scan completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned:\n\n", len(allHosts), len(ips))

	for _, host := range allHosts {
		fmt.Printf("🟢 %-15s (%.2fms)\n", host.IP, float64(host.Latency.Nanoseconds())/1000000)
	}
}

// another helper
// Fast ping using TCP connect instead of ICMP
func pingHostFast(ip string) bool {
	// Try multiple common ports quickly
	ports := []int{80, 443, 22, 21, 23, 25, 53, 135, 139, 445}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Use a channel to return as soon as any port responds
	success := make(chan bool, len(ports))

	for _, port := range ports {
		go func(p int) {
			address := fmt.Sprintf("%s:%d", ip, p)
			conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				select {
				case success <- true:
				default:
				}
			}
		}(port)
	}

	select {
	case <-success:
		return true
	case <-ctx.Done():
		return false
	}
}

// Helper function to properly sort IPs numerically
func compareIPs(ip1, ip2 string) bool {
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

func scanPorts(target string, ports []int) {
	fmt.Printf("\n🔍 Scanning %s for %d ports...\n", target, len(ports))

	const batchSize = 1000
	const maxConcurrent = 5000

	var allResults []PortResult
	var resultsMutex sync.Mutex

	start := time.Now()

	// Process ports in batches
	for i := 0; i < len(ports); i += batchSize {
		end := i + batchSize
		if end > len(ports) {
			end = len(ports)
		}

		batch := ports[i:end]

		var wg sync.WaitGroup
		results := make(chan PortResult, len(batch))
		sem := make(chan struct{}, maxConcurrent)

		for _, port := range batch {
			wg.Add(1)
			go func(port int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				result := scanPort(target, port)
				if result.Open {
					results <- result
				}
			}(port)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect batch results
		for result := range results {
			resultsMutex.Lock()
			allResults = append(allResults, result)
			resultsMutex.Unlock()
		}

		fmt.Printf("📈 Processed batch %d/%d\n", (i/batchSize)+1, (len(ports)+batchSize-1)/batchSize)
	}

	elapsed := time.Since(start)

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Port < allResults[j].Port
	})

	fmt.Printf("\n✅ Scan completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d open ports:\n\n", len(allResults))

	for _, port := range allResults {
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

func monitorPorts(hosts []string, ports []int) {
	fmt.Printf("\n👀 Monitoring %d hosts on %d ports (Ctrl+C to stop)\n", len(hosts), len(ports))
	fmt.Println("⏰ Checking every 30 seconds...\n")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial scan
	checkHosts(hosts, ports)

	for range ticker.C {
		fmt.Printf("\n⏰ %s - Checking status...\n", time.Now().Format("15:04:05"))
		checkHosts(hosts, ports)
	}
}

func checkHosts(hosts []string, ports []int) {
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		fmt.Printf("🔍 %s: ", host)

		var openPorts []int
		for _, port := range ports {
			if scanPort(host, port).Open {
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

func pingHost(ip string) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", ip+":80", timeout)
	if err != nil {
		// Try port 443 if 80 fails
		conn, err = net.DialTimeout("tcp", ip+":443", timeout)
		if err != nil {
			return false
		}
	}
	conn.Close()
	return true
}

func scanPort(host string, port int) PortResult {
	timeout := 3 * time.Second
	target := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return PortResult{Port: port, Open: false}
	}
	defer conn.Close()

	service := commonServices[port]
	banner := grabBanner(conn, port)

	return PortResult{
		Port:    port,
		Open:    true,
		Service: service,
		Banner:  banner,
	}
}

func grabBanner(conn net.Conn, port int) string {
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Send appropriate probe based on port
	switch port {
	case 22:
		// SSH typically sends banner immediately
	case 80, 8080:
		conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	case 25:
		// SMTP sends banner immediately
	case 21:
		// FTP sends banner immediately
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	banner := string(buffer[:n])
	banner = strings.ReplaceAll(banner, "\r\n", " ")
	banner = strings.ReplaceAll(banner, "\n", " ")
	banner = strings.TrimSpace(banner)

	if len(banner) > 50 {
		banner = banner[:50] + "..."
	}

	return banner
}

func parsePortRange(portRange string) []int {
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

func generateIPs(network string) []string {
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
