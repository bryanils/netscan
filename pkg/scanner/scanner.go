package scanner

import (
	"context"
	"fmt"
	"net"
	"netscan/pkg/banner"
	"netscan/pkg/models"
	"netscan/pkg/utils"
	"sort"
	"sync"
	"time"
)

// PingHost checks if a host is alive using TCP connections to common ports
func PingHost(ip string) bool {
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

// PingHostFast is an optimized version of PingHost that tries multiple ports concurrently
func PingHostFast(ip string) bool {
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

// ScanPort checks if a specific port is open on a host
func ScanPort(host string, port int) models.PortResult {
	timeout := 3 * time.Second
	target := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return models.PortResult{Port: port, Open: false}
	}
	defer conn.Close()

	service := models.CommonServices[port]
	bannerStr := banner.GrabBanner(conn, port)

	return models.PortResult{
		Port:    port,
		Open:    true,
		Service: service,
		Banner:  bannerStr,
	}
}

// ScanPortFast is an optimized version of ScanPort with shorter timeouts
func ScanPortFast(host string, port int) models.PortResult {
	timeout := 1 * time.Second // Reduced from 3 seconds
	target := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return models.PortResult{Port: port, Open: false}
	}
	defer conn.Close()

	service := models.CommonServices[port]
	bannerStr := banner.GrabBannerFast(conn, port)

	return models.PortResult{
		Port:    port,
		Open:    true,
		Service: service,
		Banner:  bannerStr,
	}
}

// PingSweep performs a ping sweep on a network to discover live hosts
func PingSweep(network string) []models.HostResult {
	fmt.Printf("\n🔍 Batch scanning network: %s\n", network)

	ips := utils.GenerateIPs(network)
	const batchSize = 254 // Process one subnet at a time
	const maxConcurrent = 500

	var allHosts []models.HostResult
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
		results := make(chan models.HostResult, len(batch))
		sem := make(chan struct{}, maxConcurrent)

		for _, ip := range batch {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				pingStart := time.Now()
				alive := PingHostFast(ip)
				latency := time.Since(pingStart)

				if alive {
					results <- models.HostResult{
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

		var batchHosts []models.HostResult
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
		return utils.CompareIPs(allHosts[i].IP, allHosts[j].IP)
	})

	fmt.Printf("\n✅ Batch scan completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned\n", len(allHosts), len(ips))

	return allHosts
}

// ScanPorts scans a list of ports on a specific host
func ScanPorts(target string, ports []int) []models.PortResult {
	fmt.Printf("\n🔍 Scanning %s for %d ports...\n", target, len(ports))

	const batchSize = 1000
	const maxConcurrent = 5000

	var allResults []models.PortResult
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
		results := make(chan models.PortResult, len(batch))
		sem := make(chan struct{}, maxConcurrent)

		for _, port := range batch {
			wg.Add(1)
			go func(port int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				result := ScanPort(target, port)
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
	fmt.Printf("📊 Found %d open ports\n", len(allResults))

	return allResults
}
