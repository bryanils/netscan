package models

import "time"

// PortResult represents the result of scanning a single port
type PortResult struct {
	Port    int
	Open    bool
	Service string
	Banner  string
}

// HostResult represents the result of scanning a host
type HostResult struct {
	IP      string
	Alive   bool
	Ports   []PortResult
	Latency time.Duration
}

// Common services for port identification
var CommonServices = map[int]string{
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
