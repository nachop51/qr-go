package main

import (
	"fmt"
	"image/color"
	qr "nachop51/qr/qr"
)

func createQr(data []byte, width, height int, filename string) (*qr.QrObject, error) {
	newQr, err := qr.NewQrBuilder(data).
		SetWidth(width).
		SetHeight(height).
		SetFilename(filename).
		SetBlackColor(color.Black).
		SetWhiteColor(color.White).
		SetErrorCorrectionLevel(qr.QrCorrectionLevelMedium).
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
	createQr([]byte("1289421489"), 400, 400, "numeric.png")
	createQr([]byte("HELLO WORLD"), 400, 400, "alphanumeric.png")
	createQr([]byte("Hola mundo!"), 400, 400, "bytes.png")
	createQr([]byte("日本"), 400, 400, "kanji.png")
}
