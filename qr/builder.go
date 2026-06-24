package qr

import (
	"errors"
	"image"
	"image/color"
)

var (
	ErrInvalidDimensions = errors.New("invalid dimensions")
)

type QrBuilder struct {
	data                 []byte
	width                int
	height               int
	filename             string
	version              int
	errorCorrectionLevel QrCorrectionLevel
	blackColor           color.Color
	whiteColor           color.Color
}

func NewQrBuilder(data []byte) *QrBuilder {
	return &QrBuilder{
		data:                 data,
		errorCorrectionLevel: QrCorrectionLevelMedium,
		blackColor:           color.Black,
		whiteColor:           color.White,
		filename:             "image.png",
		width:                400,
		height:               400,
	}
}

func (b *QrBuilder) SetWidth(width int) *QrBuilder {
	b.width = width
	return b
}

func (b *QrBuilder) SetHeight(height int) *QrBuilder {
	b.height = height
	return b
}

func (b *QrBuilder) SetFilename(filename string) *QrBuilder {
	b.filename = filename
	return b
}

func (b *QrBuilder) SetBlackColor(blackColor color.Color) *QrBuilder {
	b.blackColor = blackColor
	return b
}

func (b *QrBuilder) SetWhiteColor(whiteColor color.Color) *QrBuilder {
	b.whiteColor = whiteColor
	return b
}

func (b *QrBuilder) SetErrorCorrectionLevel(level QrCorrectionLevel) *QrBuilder {
	b.errorCorrectionLevel = level
	return b
}

func (b *QrBuilder) Build() (*QrObject, error) {
	if b.width <= 0 || b.height <= 0 {
		return nil, ErrInvalidDimensions
	}
	img := image.NewRGBA(image.Rect(0, 0, b.width, b.height))

	encodingMode := b.detectEncodingMode()
	isECI := encodingMode == QrEncodingModeByte && needsECI(b.data)
	version, err := b.detectVersion(encodingMode, isECI)

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

	pixelSize, quietZoneX, quietZoneY := measurePixelAndQuietZone(b.width, b.height, version)

	qrObj := &QrObject{
		Data:                 b.data,
		EncodingMode:         encodingMode,
		Version:              version,
		ErrorCorrectionLevel: b.errorCorrectionLevel,
		img:                  img,
		isECI:                isECI,
		pixelSize:            pixelSize,
		quietZoneX:           quietZoneX,
		quietZoneY:           quietZoneY,
		Filename:             b.filename,
		blackColor:           b.blackColor,
		whiteColor:           b.whiteColor,
		points:               points,
	}

	qrObj.drawQuietZone()
	qrObj.placeFinders()
	qrObj.placeTimingPattern()
	qrObj.placeReserved()
	qrObj.placeAlignmentPatterns()

	data := qrObj.encode()
	err = qrObj.placeData(data)
	if err != nil {
		return nil, err
	}

	scores := make([]int, 8)

	for mask := range 8 {
		score, err := qrObj.measureMaskScore(mask)
		if err != nil {
			return nil, err
		}
		scores[mask] = score
	}

	maskIdx := 0
	// get lowest mask score
	for i, score := range scores {
		if score < scores[maskIdx] {
			maskIdx = i
		}
	}
	qrObj.Mask = maskIdx

	// qr.debugMasks()

	qrObj.applyMask(maskIdx)
	qrObj.placeMetadata(maskIdx)

	return qrObj, nil
}
