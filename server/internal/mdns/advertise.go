package mdns

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/hashicorp/mdns"
)

// Server wraps an mDNS server for service advertisement
type Server struct {
	server *mdns.Server
}

// Advertise starts advertising the service via mDNS
func Advertise(serviceName string, port int) (*Server, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "pocket-assistant"
	}

	// Get local IP addresses
	ips, err := getLocalIPs()
	if err != nil {
		slog.Warn("Failed to get local IPs, using empty list", "error", err)
		ips = []net.IP{}
	}

	slog.Debug("Advertising mDNS service", "name", serviceName, "port", port, "ips", ips)

	// Create mDNS service
	service, err := mdns.NewMDNSService(
		hostname,                       // Instance name
		fmt.Sprintf("_%s._tcp", serviceName), // Service type
		"",                             // Domain (default: local.)
		"",                             // Host name (default: hostname.local.)
		port,                           // Port
		ips,                            // IPs
		[]string{"pocket-assistant"},   // TXT records
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Create mDNS server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS server: %w", err)
	}

	slog.Info("mDNS service advertised",
		"name", serviceName,
		"port", port,
		"hostname", hostname,
	)

	return &Server{server: server}, nil
}

// Shutdown stops the mDNS server
func (s *Server) Shutdown() error {
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

// getLocalIPs returns all non-loopback IPv4 addresses
func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only use IPv4 addresses
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}
