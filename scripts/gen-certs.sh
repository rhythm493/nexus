#!/bin/bash
# Generate mTLS certificates for pocket-assistant
# Usage: ./gen-certs.sh [output_dir]

set -e

OUTPUT_DIR="${1:-../certs}"
DAYS_VALID=3650  # 10 years
KEY_SIZE=4096

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Generating mTLS certificates for pocket-assistant${NC}"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"
cd "$OUTPUT_DIR"

# Generate CA private key and certificate
echo -e "${YELLOW}1. Generating CA certificate...${NC}"
openssl genrsa -out ca.key $KEY_SIZE 2>/dev/null
openssl req -new -x509 -days $DAYS_VALID -key ca.key -out ca.crt \
    -subj "/CN=pocket-assistant-ca/O=pocket-assistant" 2>/dev/null
echo -e "${GREEN}   Created: ca.key, ca.crt${NC}"

# Generate server private key and CSR
echo -e "${YELLOW}2. Generating server certificate...${NC}"
openssl genrsa -out server.key $KEY_SIZE 2>/dev/null

# Create server config with SANs
cat > server.cnf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = pocket-assistant-server
O = pocket-assistant

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = pocket-assistant.local
DNS.3 = *.local
IP.1 = 127.0.0.1
EOF

# Add local IPs to SANs
IP_COUNT=2
for IP in $(hostname -I 2>/dev/null || echo ""); do
    echo "IP.$IP_COUNT = $IP" >> server.cnf
    IP_COUNT=$((IP_COUNT + 1))
done

openssl req -new -key server.key -out server.csr -config server.cnf 2>/dev/null
openssl x509 -req -days $DAYS_VALID -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -extensions v3_req -extfile server.cnf 2>/dev/null
echo -e "${GREEN}   Created: server.key, server.crt${NC}"

# Generate client private key and certificate
echo -e "${YELLOW}3. Generating client certificate...${NC}"
openssl genrsa -out client.key $KEY_SIZE 2>/dev/null

cat > client.cnf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = pocket-assistant-client
O = pocket-assistant

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

openssl req -new -key client.key -out client.csr -config client.cnf 2>/dev/null
openssl x509 -req -days $DAYS_VALID -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -extensions v3_req -extfile client.cnf 2>/dev/null
echo -e "${GREEN}   Created: client.key, client.crt${NC}"

# Clean up temporary files
rm -f server.cnf client.cnf server.csr client.csr ca.srl

# Set permissions
chmod 600 *.key
chmod 644 *.crt

echo ""
echo -e "${GREEN}Certificate generation complete!${NC}"
echo ""
echo "Files created:"
echo "  ca.crt       - CA certificate (share with clients)"
echo "  ca.key       - CA private key (keep secure!)"
echo "  server.crt   - Server certificate"
echo "  server.key   - Server private key"
echo "  client.crt   - Client certificate (transfer to phone)"
echo "  client.key   - Client private key (transfer to phone)"
echo ""
echo -e "${YELLOW}To transfer to phone:${NC}"
echo "  1. Copy ca.crt, client.crt, client.key to your phone"
echo "  2. Import them in the pocket-assistant app"
echo ""
echo -e "${RED}Keep ca.key and server.key secure!${NC}"
