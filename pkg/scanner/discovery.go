package scanner

import (
	"fmt"
	"netscan/pkg/models"
	"netscan/pkg/utils"
	"sort"
	"sync"
	"time"
)

// NetworkDiscovery performs network discovery and port scanning on a network
func NetworkDiscovery(network string, ports []int) []models.HostResult {
	fmt.Printf("\n🔍 Network discovery on %s\n", network)

	ips := utils.GenerateIPs(network)

	// Increased concurrency limits for better performance
	const maxHostConcurrency = 100 // More hosts scanned simultaneously
	const maxPortConcurrency = 50  // More ports per host
	const batchSize = 50           // Process hosts in batches for better memory management

	var allHosts []models.HostResult
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
		results := make(chan models.HostResult, len(batch))
		sem := make(chan struct{}, maxHostConcurrency)

		for _, ip := range batch {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// Use the faster ping method first
				if !PingHostFast(ip) {
					return
				}

				// Scan ports concurrently for this host
				var portWg sync.WaitGroup
				portResults := make(chan models.PortResult, len(ports))
				portSem := make(chan struct{}, maxPortConcurrency)

				for _, port := range ports {
					portWg.Add(1)
					go func(port int) {
						defer portWg.Done()
						portSem <- struct{}{}
						defer func() { <-portSem }()

						result := ScanPortFast(ip, port)
						if result.Open {
							portResults <- result
						}
					}(port)
				}

				go func() {
					portWg.Wait()
					close(portResults)
				}()

				var openPorts []models.PortResult
				for result := range portResults {
					openPorts = append(openPorts, result)
				}

				if len(openPorts) > 0 || len(ports) == 0 {
					results <- models.HostResult{
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

	// Sort results by IP
	sort.Slice(allHosts, func(i, j int) bool {
		return utils.CompareIPs(allHosts[i].IP, allHosts[j].IP)
	})

	fmt.Printf("\n✅ Discovery completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned\n", len(allHosts), len(ips))

	return allHosts
}

// NetworkDiscoveryWorkerPool is an alternative implementation using worker pools
func NetworkDiscoveryWorkerPool(network string, ports []int) []models.HostResult {
	fmt.Printf("\n🔍 Network discovery on %s (Worker Pool)\n", network)

	ips := utils.GenerateIPs(network)

	const numWorkers = 50
	const bufferSize = 100

	// Create channels
	jobs := make(chan string, bufferSize)
	results := make(chan models.HostResult, bufferSize)

	var wg sync.WaitGroup

	start := time.Now()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if !PingHostFast(ip) {
					continue
				}

				var portResults []models.PortResult
				var portWg sync.WaitGroup
				portChan := make(chan models.PortResult, len(ports))

				for _, port := range ports {
					portWg.Add(1)
					go func(port int) {
						defer portWg.Done()
						result := ScanPortFast(ip, port)
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
					results <- models.HostResult{
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

	var hosts []models.HostResult
	for result := range results {
		hosts = append(hosts, result)
	}

	elapsed := time.Since(start)

	sort.Slice(hosts, func(i, j int) bool {
		return utils.CompareIPs(hosts[i].IP, hosts[j].IP)
	})

	fmt.Printf("\n✅ Discovery completed in %v\n", elapsed)
	fmt.Printf("📊 Found %d live hosts out of %d scanned\n", len(hosts), len(ips))

	return hosts
}
