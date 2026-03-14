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
		hostname = "nexus"
	}

	// Get local IP addresses
	ips, err := getLocalIPs()
	if err != nil {
		slog.Warn("Failed to get local IPs, using empty list", "error", err)
		ips = []net.IP{}
	}

	slog.Info("Advertising mDNS service", "name", serviceName, "port", port, "ips", ips)

	// Create mDNS service
	service, err := mdns.NewMDNSService(
		hostname,                       // Instance name
		fmt.Sprintf("_%s._tcp", serviceName), // Service type
		"",                             // Domain (default: local.)
		"",                             // Host name (default: hostname.local.)
		port,                           // Port
		ips,                            // IPs
		[]string{"nexus"},              // TXT records
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

// getLocalIPs returns LAN IPv4 addresses (filters out VPN/Docker/CGNAT)
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

		// Skip known VPN/virtual interfaces
		if isVirtualInterface(iface.Name) {
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

			// Only use IPv4 addresses that are LAN IPs
			if ip != nil && ip.To4() != nil && isLANIP(ip) {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}

// isVirtualInterface returns true for known VPN/virtual interface names
func isVirtualInterface(name string) bool {
	// Tailscale, Docker, VirtualBox, VMware, etc.
	virtualPrefixes := []string{"tailscale", "docker", "br-", "veth", "virbr", "vmnet", "vboxnet", "tun", "tap"}
	for _, prefix := range virtualPrefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isLANIP returns true for typical LAN IP ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
// and filters out CGNAT (100.64-127.x.x used by Tailscale) and Docker (172.17.x.x)
func isLANIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	// 192.168.0.0/16 - common home LAN
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}

	// 10.0.0.0/8 - private network
	if ip4[0] == 10 {
		return true
	}

	// 172.16.0.0/12 - but NOT 172.17.x.x (Docker default)
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 && ip4[1] != 17 {
		return true
	}

	return false
}
