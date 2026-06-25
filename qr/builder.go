package qr

import (
	"errors"
	"image"
	"image/color"
	"unicode/utf8"
)

var (
	ErrInvalidDimensions = errors.New("invalid dimensions")
	ErrInvalidDataKind   = errors.New("invalid data kind")
	ErrInvalidUTF8Text   = errors.New("invalid utf8 text")
)

type QrDataKind int

const (
	QrDataKindText QrDataKind = iota
	QrDataKindBinary
)

type TextECIPolicy int

const (
	TextECIPolicyAuto TextECIPolicy = iota
	TextECIPolicyDisabled
)

type QrBuilder struct {
	data                 []byte
	dataKind             QrDataKind
	textECIPolicy        TextECIPolicy
	width                int
	height               int
	filename             string
	version              int
	errorCorrectionLevel QrCorrectionLevel
	blackColor           color.Color
	whiteColor           color.Color
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

func (b *QrBuilder) SetDisableECI(disable bool) *QrBuilder {
	if disable {
		b.textECIPolicy = TextECIPolicyDisabled
	} else {
		b.textECIPolicy = TextECIPolicyAuto
	}
	return b
}

func createPoints(modules int) [][]QrPoint {
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

	return points
}

func (b *QrBuilder) Build() (*QrObject, error) {
	if b.width <= 0 || b.height <= 0 {
		return nil, ErrInvalidDimensions
	}

	var encodingMode QrEncodingMode
	var isECI bool

	switch b.dataKind {
	case QrDataKindText:
		if !utf8.Valid(b.data) {
			return nil, ErrInvalidUTF8Text
		}

		encodingMode = b.detectEncodingMode()

		if b.textECIPolicy == TextECIPolicyDisabled {
			isECI = false
		} else {
			isECI = encodingMode == QrEncodingModeByte && hasNonASCII(b.data)
		}
	case QrDataKindBinary:
		encodingMode = QrEncodingModeByte
		isECI = false
	default:
		return nil, ErrInvalidDataKind
	}

	version, err := b.detectVersion(encodingMode, isECI)

	if err != nil {
		return nil, err
	}

	modules := capacityTable[version].modules
	pixelSize, quietZoneX, quietZoneY := measurePixelAndQuietZone(b.width, b.height, version)

	qrObj := &QrObject{
		Data:                 b.data,
		EncodingMode:         encodingMode,
		Version:              version,
		ErrorCorrectionLevel: b.errorCorrectionLevel,
		img:                  image.NewRGBA(image.Rect(0, 0, b.width, b.height)),
		isECI:                isECI,
		pixelSize:            pixelSize,
		quietZoneX:           quietZoneX,
		quietZoneY:           quietZoneY,
		Filename:             b.filename,
		blackColor:           b.blackColor,
		whiteColor:           b.whiteColor,
		points:               createPoints(modules),
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

func NewTextQrBuilder(text string) *QrBuilder {
	return newQrBuilder([]byte(text), QrDataKindText)
}

func NewBinaryQrBuilder(data []byte) *QrBuilder {
	return newQrBuilder(data, QrDataKindBinary)
}

func newQrBuilder(data []byte, inputKind QrDataKind) *QrBuilder {
	return &QrBuilder{
		data:                 append([]byte(nil), data...),
		dataKind:             inputKind,
		errorCorrectionLevel: QrCorrectionLevelMedium,
		blackColor:           color.Black,
		whiteColor:           color.White,
		filename:             "image.png",
		width:                400,
		height:               400,
	}
}
