// Command qrgo generates QR codes from the terminal. It exposes the whole
// qr-go library. Every content helper, error-correction level, and renderer
// through a single flat command:
//
//	qrgo "HELLO WORLD"                       # QR to the terminal
//	qrgo -t url https://example.com -o q.png # PNG file
//	qrgo -t wifi --ssid home --pass s3cr3t   # Wi-Fi join code
//
// Run "qrgo --help" for the full flag reference.
package main

import "os"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
