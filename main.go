package main

import (
	"fmt"
	qr "nachop51/qr/qr"
)

type ValidQrInput interface {
	[]byte | string
}

func createQr[T ValidQrInput](data T, filename string) (*qr.QrObject, error) {

	var qrBuilder *qr.QrBuilder

	if s, ok := any(data).(string); ok {
		qrBuilder = qr.NewTextQrBuilder(s)
	} else if b, ok := any(data).([]byte); ok {
		qrBuilder = qr.NewBinaryQrBuilder(b)
	}

	newQr, err := qrBuilder.
		SetErrorCorrectionLevel(qr.QrCorrectionLevelMedium).
		// SetDisableECI(false).
		SetFilename(filename).
		// SetErrorCorrectionLevel(qr.QrCorrectionLevelHigh).
		Build()

	if err != nil {
		return nil, err
	}

	fmt.Printf("Data: %v\n", data)
	fmt.Printf("Version detected: %d\n", newQr.Version)
	fmt.Printf("Error correction level detected: %b\n", newQr.ErrorCorrectionLevel)
	fmt.Printf("Mask detected: %d\n", newQr.Mask)

	// newQr.Debug()

	newQr.Draw()
	newQr.Save()

	return newQr, nil
}

func main() {
	// createQr("1289421489", "numeric.png")
	// createQr("HELLO WORLD", "alphanumeric.png")
	// createQr([]byte("Hola mundo!"), "bytes.png")
	// createQr("日本", "kanji.png")

	// createQr("HELLO123456789012345", "hello123.png")
	// createQr("HELLO1234567", "hello123.png")
	// createQr("Te amooo, XOXOXOXOXO", "love.png")
	createQr("ABC日本123", "test.png")
	// createQr(strings.Repeat("ABC123", 43)+"ho", "strings.png")
}
