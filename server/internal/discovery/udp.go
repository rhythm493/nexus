package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// StartUDPBroadcast starts a goroutine that broadcasts the server's identity on UDP port 9999.
// The goroutine stops when the provided context is cancelled.
func StartUDPBroadcast(ctx context.Context, port int) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		slog.Error("Failed to start UDP broadcast", "error", err)
		return
	}

	addr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
	if err != nil {
		slog.Error("Failed to resolve UDP broadcast address", "error", err)
		conn.Close()
		return
	}

	go func() {
		defer conn.Close()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("UDP broadcast stopped")
				return
			case <-ticker.C:
				ip := getLocalIP()
				message := fmt.Sprintf("NEXUS:%s:%d", ip, port)
				_, err := conn.WriteTo([]byte(message), addr)
				if err != nil {
					slog.Debug("Failed to send UDP broadcast", "error", err)
				}
			}
		}
	}()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			// Only return LAN IPs — filter out Tailscale (100.x), Docker (172.17.x)
			if ip4[0] == 192 && ip4[1] == 168 {
				return ip4.String()
			}
			if ip4[0] == 10 {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}
