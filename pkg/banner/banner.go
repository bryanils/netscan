package banner

import (
	"net"
	"strings"
	"time"
)

// GrabBanner attempts to grab a service banner from an open port
func GrabBanner(conn net.Conn, port int) string {
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

// GrabBannerFast is an optimized version of GrabBanner with shorter timeouts
func GrabBannerFast(conn net.Conn, port int) string {
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
