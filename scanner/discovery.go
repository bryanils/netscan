package scanner

import (
	"context"
	"fmt"
	"net"

	"sort"
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

func NetworkDiscovery(network string, ports []int) {
	fmt.Printf("\n🔍 Network discovery on %s\n", network)

	ips := generateIPs(network)

	// Increased concurrency limits for better performance
	const maxHostConcurrency = 100 // More hosts scanned simultaneously
	const maxPortConcurrency = 50  // More ports per host
	const batchSize = 50           // Process hosts in batches for better memory management

	var allHosts []HostResult
	var resultsMutex sync.Mutex

	start := time.Now()

	// Process IPs in batches to manage memory and provide progress feedback
	for i := 0; i < len(ips); i += batchSize {
		end := i + batchSize
		if end > len(ips) {
			end = len(ips)
		}

		batch := ips[i:end]
		batchStart := time.Now()

		var wg sync.WaitGroup
		results := make(chan HostResult, len(batch))
		sem := make(chan struct{}, maxHostConcurrency)

		for _, ip := range batch {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// Use the faster ping method first
				if !pingHostFast(ip) {
					return
				}

				// Scan ports concurrently for this host
				var portWg sync.WaitGroup
				portResults := make(chan PortResult, len(ports))
				portSem := make(chan struct{}, maxPortConcurrency)

				for _, port := range ports {
					portWg.Add(1)
					go func(port int) {
						defer portWg.Done()
						portSem <- struct{}{}
						defer func() { <-portSem }()

						result := scanPortFast(ip, port)
						if result.Open {
							portResults <- result
						}
					}(port)
				}

				go func() {
					portWg.Wait()
					close(portResults)
				}()

				var openPorts []PortResult
				for result := range portResults {
					openPorts = append(openPorts, result)
				}

				if len(openPorts) > 0 || len(ports) == 0 {
					results <- HostResult{
						IP:    ip,
						Alive: true,
						Ports: openPorts,
					}
				}
			}(ip)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect batch results
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

	// Sort results by IP
	sort.Slice(allHosts, func(i, j int) bool {
		return compareIPs(allHosts[i].IP, allHosts[j].IP)
	})

	fmt.Printf("\n✅ Discovery completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned:\n\n", len(allHosts), len(ips))

	for _, host := range allHosts {
		fmt.Printf("🖥️  %s\n", host.IP)
		if len(host.Ports) > 0 {
			// Sort ports for consistent output
			sort.Slice(host.Ports, func(i, j int) bool {
				return host.Ports[i].Port < host.Ports[j].Port
			})

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

// Optimized port scanning function with shorter timeouts
func scanPortFast(host string, port int) PortResult {
	timeout := 1 * time.Second // Reduced from 3 seconds
	target := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return PortResult{Port: port, Open: false}
	}
	defer conn.Close()

	service := commonServices[port]
	banner := grabBannerFast(conn, port)

	return PortResult{
		Port:    port,
		Open:    true,
		Service: service,
		Banner:  banner,
	}
}

// Faster banner grabbing with shorter timeout
func grabBannerFast(conn net.Conn, port int) string {
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) // Reduced timeout

	// Send appropriate probe based on port
	switch port {
	case 22:
		// SSH typically sends banner immediately
	case 80, 8080:
		conn.Write([]byte("GET / HTTP/1.1\r\nHost: \r\nConnection: close\r\n\r\n"))
	case 25:
		// SMTP sends banner immediately
	case 21:
		// FTP sends banner immediately
	case 443:
		// HTTPS - don't try to grab banner as it requires TLS handshake
		return ""
	}

	buffer := make([]byte, 512) // Smaller buffer
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	banner := string(buffer[:n])
	banner = strings.ReplaceAll(banner, "\r\n", " ")
	banner = strings.ReplaceAll(banner, "\n", " ")
	banner = strings.TrimSpace(banner)

	if len(banner) > 40 { // Shorter banner limit
		banner = banner[:40] + "..."
	}

	return banner
}

// Alternative implementation using worker pools for even better performance
func networkDiscoveryWorkerPool(network string, ports []int) {
	fmt.Printf("\n🔍 Network discovery on %s (Worker Pool)\n", network)

	ips := generateIPs(network)

	const numWorkers = 50
	const bufferSize = 100

	// Create channels
	jobs := make(chan string, bufferSize)
	results := make(chan HostResult, bufferSize)

	var wg sync.WaitGroup

	start := time.Now()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if !pingHostFast(ip) {
					continue
				}

				var portResults []PortResult
				var portWg sync.WaitGroup
				portChan := make(chan PortResult, len(ports))

				for _, port := range ports {
					portWg.Add(1)
					go func(port int) {
						defer portWg.Done()
						result := scanPortFast(ip, port)
						if result.Open {
							portChan <- result
						}
					}(port)
				}

				go func() {
					portWg.Wait()
					close(portChan)
				}()

				for result := range portChan {
					portResults = append(portResults, result)
				}

				if len(portResults) > 0 {
					results <- HostResult{
						IP:    ip,
						Alive: true,
						Ports: portResults,
					}
				}
			}
		}()
	}

	// Send jobs
	go func() {
		for _, ip := range ips {
			jobs <- ip
		}
		close(jobs)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var hosts []HostResult
	for result := range results {
		hosts = append(hosts, result)
	}

	elapsed := time.Since(start)

	sort.Slice(hosts, func(i, j int) bool {
		return compareIPs(hosts[i].IP, hosts[j].IP)
	})

	fmt.Printf("\n✅ Discovery completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned:\n\n", len(hosts), len(ips))

	for _, host := range hosts {
		fmt.Printf("🖥️  %s\n", host.IP)
		if len(host.Ports) > 0 {
			sort.Slice(host.Ports, func(i, j int) bool {
				return host.Ports[i].Port < host.Ports[j].Port
			})

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
