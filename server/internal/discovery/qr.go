package discovery

import (
	"fmt"
	"net/http"

	"github.com/skip2/go-qrcode"
)

// HandleQRCode returns a handler that serves a QR code image for discovery
func HandleQRCode(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getLocalIP()
		url := fmt.Sprintf("https://%s:%d", ip, port)

		png, err := qrcode.Encode(url, qrcode.Medium, 256)
		if err != nil {
			http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Discovery-URL", url)
		w.Write(png)
	}
}
