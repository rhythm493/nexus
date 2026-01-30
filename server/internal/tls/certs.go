package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
)

// LoadServerConfig loads TLS configuration for the server
func LoadServerConfig(certsDir string) (*tls.Config, error) {
	// Load server certificate and key
	certFile := filepath.Join(certsDir, "server.crt")
	keyFile := filepath.Join(certsDir, "server.key")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caFile := filepath.Join(certsDir, "ca.crt")
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.NoClientCert, // No client cert required for LAN use
		MinVersion:   tls.VersionTLS12, // TLS 1.2 for better compatibility
	}

	return config, nil
}

// LoadClientConfig loads TLS configuration for a client
func LoadClientConfig(certsDir string) (*tls.Config, error) {
	// Load client certificate and key
	certFile := filepath.Join(certsDir, "client.crt")
	keyFile := filepath.Join(certsDir, "client.key")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate for server verification
	caFile := filepath.Join(certsDir, "ca.crt")
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	return config, nil
}
