# pocket-assistant Setup Guide

This guide walks you through setting up pocket-assistant from scratch.

## Prerequisites

### On your PC (server)

- **Go 1.21+**: Install from https://go.dev/dl/
- **Python 3.7+**: Usually pre-installed on Linux
- **uv**: Python package manager (installed automatically by scripts)
- **Avahi**: mDNS service (usually pre-installed on Linux)
- **OpenSSL**: For certificate generation

Check prerequisites:
```bash
go version          # Should show 1.21+
python3 --version   # Should show 3.7+
avahi-daemon -V     # Should show version
openssl version     # Should show version
```

### On your phone

- **Android**: 5.0+ or **iOS**: 12+
- **Flutter**: 3.x (only for development)

## Step 1: Clone the Repository

```bash
cd ~/Projects/Local
git clone <repo-url> pocket-assistant
cd pocket-assistant
```

## Step 2: Get a Gemini API Key

1. Go to https://makersuite.google.com/app/apikey
2. Create a new API key (free tier available)
3. Save the key securely

## Step 3: Generate Certificates

```bash
make certs
```

This creates the following in `certs/`:
- `ca.crt` - CA certificate (share with phone)
- `ca.key` - CA private key (keep secure!)
- `server.crt` - Server certificate
- `server.key` - Server private key
- `client.crt` - Client certificate (transfer to phone)
- `client.key` - Client private key (transfer to phone)

## Step 4: Install Sonos MCP (Optional)

If you have Sonos speakers:

```bash
make install-sonos-mcp
```

## Step 5: Configure the Server

Create a config file:
```bash
cp server/config/config.example.yaml server/config/config.yaml
```

Or just use environment variables:
```bash
export GEMINI_API_KEY="your-api-key-here"
```

## Step 6: Start the Server

```bash
make server
# Or for debug mode:
make dev
```

You should see:
```
INFO Starting pocket-assistant server port=8443 service=pocket-assistant
INFO mDNS service advertised name=pocket-assistant port=8443 hostname=yourpc
```

## Step 7: Verify Server

In another terminal:
```bash
make check-health
```

Or manually:
```bash
curl -k --cert certs/client.crt --key certs/client.key \
  https://localhost:8443/api/v1/health
```

Should return:
```json
{"status":"ok","mcp_servers":["sonos"]}
```

## Step 8: Transfer Certificates to Phone

You need to copy these files to your phone:
- `certs/ca.crt`
- `certs/client.crt`
- `certs/client.key`

Methods:
1. **USB cable**: Copy to phone storage
2. **Wireless**: Use `python3 -m http.server 8000` and download via browser
3. **QR code**: Encode the small files as QR codes
4. **Cloud**: Upload to Google Drive/Dropbox (less secure)

## Step 9: Build and Install the App

### For Development

```bash
cd app
flutter pub get
flutter run
```

### For Release (Android)

```bash
cd app
flutter build apk --release
# Install the APK from build/app/outputs/flutter-apk/app-release.apk
```

### For Release (iOS)

```bash
cd app
flutter build ios --release
# Open in Xcode and archive
```

## Step 10: Configure the App

1. Open the app
2. Tap the settings icon (gear)
3. Import certificates:
   - Tap "CA Certificate" and select `ca.crt`
   - Tap "Client Certificate" and select `client.crt`
   - Tap "Client Key" and select `client.key`
4. The app should auto-discover the server via mDNS
5. If not found, use "Manual" to enter your PC's IP and port 8443

## Troubleshooting

### Server not discovered

1. Check Avahi is running: `systemctl status avahi-daemon`
2. Check mDNS: `avahi-browse -a | grep pocket-assistant`
3. Use manual connection with your PC's IP address

### Certificate errors

1. Regenerate certs: `make certs`
2. Re-import on phone
3. Check cert dates: `openssl x509 -in certs/ca.crt -noout -dates`

### Voice not working

1. Grant microphone permission in app settings
2. Check platform speech recognition is enabled
3. On Android, ensure Google app is installed

### MCP server not starting

1. Check Python is available: `python3 --version`
2. Check uv is available: `uv --version`
3. Test MCP manually: `uvx sonos-mcp-server`

### Sonos not responding

1. Ensure Sonos speakers are on the same network
2. Check Sonos app can control speakers
3. Test MCP directly:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | uvx sonos-mcp-server
   ```

## Network Requirements

- Phone and PC must be on the same local network
- Port 8443 must be accessible (not firewalled)
- mDNS (port 5353) should be open for auto-discovery

## Security Notes

1. **Keep private keys secure**: Never share `ca.key`, `server.key`, or `client.key` publicly
2. **Regenerate if compromised**: Run `make certs` to generate new certificates
3. **Local network only**: The server is only accessible on your LAN
4. **mTLS**: All connections require client certificates
