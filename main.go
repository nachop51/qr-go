package main

import (
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
		SetErrorCorrectionLevel(qrimage.QrCorrectionLevelLow).
		Build()

	if err != nil {
		return nil, err
	}

	// img.Debug()

	img.Draw()
	img.Save()

	return img, nil
}

func main() {
	var data = []byte("Hola mi amor, como estas?")

	_, err := createQrImage(data, 400, 400, "image.png")
	if err != nil {
		log.Fatal(err)
	}

	// _, err = createQrImage([]byte("Hola mundo como estas espe"), 400, 400, "image.png")

	// if err != nil {
	// 	log.Fatal(err)
	// }

}
