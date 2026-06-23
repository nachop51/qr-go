package main

import (
	"fmt"
	"image/color"
	"log"
	qrimage "nachop51/qr/qr-image"
)

func createQrImage(data []byte, width, height int, filename string) (*qrimage.QrImage, error) {
	img, err := qrimage.NewQrImageBuilder(data).
		SetWidth(width).
		SetHeight(height).
		SetFilename(filename).
		SetBlackColor(color.Black).
		SetWhiteColor(color.White).
		SetErrorCorrectionLevel(qrimage.QrCorrectionLevelMedium).
		SetErrorCorrectionLevel(qrimage.QrCorrectionLevelHigh).
		Build()

	if err != nil {
		return nil, err
	}

	fmt.Printf("Version detected: %d\n", img.Version)
	fmt.Printf("Encoding mode detected: %b\n", img.EncodingMode)
	fmt.Printf("Error correction level detected: %b\n", img.ErrorCorrectionLevel)
	fmt.Printf("Mask detected: %d\n", img.Mask)

	// img.Debug()

	img.Draw()
	img.Save()

	return img, nil
}

func main() {
	var data = []byte("Hola mi amor, te amo, espero que estee ")

	_, err := createQrImage(data, 400, 400, "image.png")
	if err != nil {
		log.Fatal(err)
	}

	// _, err = createQrImage([]byte("Hola mundo como estas espe"), 400, 400, "image.png")

	// if err != nil {
	// 	log.Fatal(err)
	// }

}
