package qrimage

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

var (
	ErrInvalidDimensions = errors.New("invalid dimensions")
)

type QrImageBuilder struct {
	data                 []byte
	width                int
	height               int
	filename             string
	version              int
	errorCorrectionLevel QrCorrectionLevel
	blackColor           color.Color
	whiteColor           color.Color
}

func NewQrImageBuilder(data []byte) *QrImageBuilder {
	return &QrImageBuilder{
		data:                 data,
		errorCorrectionLevel: QrCorrectionLevelMedium,
		blackColor:           color.Black,
		whiteColor:           color.White,
		filename:             "image.png",
		width:                400,
		height:               400,
	}
}

func (b *QrImageBuilder) SetWidth(width int) *QrImageBuilder {
	b.width = width
	return b
}

func (b *QrImageBuilder) SetHeight(height int) *QrImageBuilder {
	b.height = height
	return b
}

func (b *QrImageBuilder) SetFilename(filename string) *QrImageBuilder {
	b.filename = filename
	return b
}

func (b *QrImageBuilder) SetBlackColor(blackColor color.Color) *QrImageBuilder {
	b.blackColor = blackColor
	return b
}

func (b *QrImageBuilder) SetWhiteColor(whiteColor color.Color) *QrImageBuilder {
	b.whiteColor = whiteColor
	return b
}

func (b *QrImageBuilder) SetVersion(version int) *QrImageBuilder {
	b.version = version
	return b
}

func (b *QrImageBuilder) SetErrorCorrectionLevel(level QrCorrectionLevel) *QrImageBuilder {
	b.errorCorrectionLevel = level
	return b
}

func (b *QrImageBuilder) Build() (*QrImage, error) {
	if b.width <= 0 || b.height <= 0 {
		return nil, ErrInvalidDimensions
	}
	img := image.NewRGBA(image.Rect(0, 0, b.width, b.height))

	encodingMode := b.detectEncodingMode()
	version, err := b.detectVersion(encodingMode)

	fmt.Printf("Version detected: %d\n", version)

	if err != nil {
		return nil, err
	}

	modules := capacityTable[version].modules

	points := make([][]QrPoint, modules)

	for i := range points {
		points[i] = make([]QrPoint, modules)
	}

	for y := range modules {
		for x := range modules {
			points[y][x] = QrPoint{
				x:   x,
				y:   y,
				col: QrWhite,
			}
		}
	}

	qrImage := &QrImage{
		Data:                 b.data,
		EncodingMode:         encodingMode,
		Version:              version,
		ErrorCorrectionLevel: b.errorCorrectionLevel,
		img:                  img,
		pixelSize:            img.Rect.Max.X / modules,
		Filename:             b.filename,
		blackColor:           b.blackColor,
		whiteColor:           b.whiteColor,
		points:               points,
	}

	qrImage.placeFinders()
	qrImage.placeTimingPattern()
	qrImage.placeReserved()
	qrImage.placeAlignmentPatterns()

	data := qrImage.encode()
	qrImage.placeData(data)

	scores := make([]int, 8)

	for mask := range 8 {
		err := qrImage.applyMask(mask)
		if err != nil {
			return nil, err
		}
		// 		qrImage.Filename = fmt.Sprintf("mask_%d.png", mask)
		// qrImage.Draw()
		// qrImage.Save()
		scores[mask] = qrImage.measureMaskScore()
		err = qrImage.applyMask(mask)
		if err != nil {
			return nil, err
		}
	}

	fmt.Printf("Scores: %v\n", scores)

	maskIdx := 0
	// get lowest mask score
	for i, score := range scores {
		if score < scores[maskIdx] {
			maskIdx = i
		}
	}

	qrImage.debugMasks()

	qrImage.applyMask(maskIdx)
	qrImage.placeMetadata(maskIdx)

	return qrImage, nil
}
