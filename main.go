package main

import (
	"fmt"
	qr "nachop51/qr/qr"
)

type ValidQr interface {
	[]byte | string
}

func createQr[T ValidQr](data T, filename string) (*qr.QrObject, error) {

	var qrBuilder *qr.QrBuilder

	if s, ok := any(data).(string); ok {
		qrBuilder = qr.NewTextQrBuilder(s)
	} else if b, ok := any(data).([]byte); ok {
		qrBuilder = qr.NewBinaryQrBuilder(b)
	}

	newQr, err := qrBuilder.
		SetErrorCorrectionLevel(qr.QrCorrectionLevelMedium).
		SetFilename(filename).
		// SetErrorCorrectionLevel(qr.QrCorrectionLevelHigh).
		Build()

	if err != nil {
		return nil, err
	}

	fmt.Printf("Version detected: %d\n", newQr.Version)
	fmt.Printf("Encoding mode detected: %b\n", newQr.EncodingMode)
	fmt.Printf("Error correction level detected: %b\n", newQr.ErrorCorrectionLevel)
	fmt.Printf("Mask detected: %d\n", newQr.Mask)

	// newQr.Debug()

	newQr.Draw()
	newQr.Save()

	return newQr, nil
}

func main() {
	createQr("1289421489", "numeric.png")
	createQr("HELLO WORLD", "alphanumeric.png")
	createQr([]byte("Hola mundo!"), "bytes.png")
	createQr("日本", "kanji.png")
}
