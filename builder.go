package qr

import (
	"image"
	"image/color"
	"unicode/utf8"
)

type QrBuilder struct {
	data                 []byte
	dataKind             QrDataKind
	textECIPolicy        QrTextECIPolicy
	width                int
	height               int
	filename             string
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

func (b *QrBuilder) SetTextECIPolicy(policy QrTextECIPolicy) *QrBuilder {
	b.textECIPolicy = policy
	return b
}

func (b *QrBuilder) segmentize() ([]Segment, bool, error) {
	ranges := []VersionRange{
		VersionRangeSmall,
		VersionRangeMedium,
		VersionRangeLarge,
	}

	for _, vr := range ranges {
		segs, totalSixths := segmentizeOptimal(b.data, vr)
		totalBits := (totalSixths + 5) / 6

		needsECI := b.textECIPolicy != QrTextECIPolicyDisabled && segmentsNeedsECI(segs)
		if needsECI {
			totalBits += 12
		}

		if fitsInRange(totalBits, vr, b.errorCorrectionLevel.level) {
			return segs, needsECI, nil
		}
	}

	return nil, false, ErrDataTooLong
}

func (b *QrBuilder) Build() (*QrCode, error) {
	if b.width <= 0 || b.height <= 0 {
		return nil, ErrInvalidDimensions
	}

	var isECI bool = false
	var segments []Segment
	var err error

	switch b.dataKind {
	case QrDataKindText:
		if !utf8.Valid(b.data) {
			return nil, ErrInvalidUTF8Text
		}

		segments, isECI, err = b.segmentize()

		if err != nil {
			return nil, err
		}
	case QrDataKindBinary:
		segments = []Segment{{Mode: EncodingModeByte, Data: b.data}}
		isECI = false
	default:
		return nil, ErrInvalidDataKind
	}

	version, err := b.detectVersion(segments, isECI)

	if err != nil {
		return nil, err
	}

	modules := capacityTable[version].modules
	pixelSize, quietZoneX, quietZoneY := measurePixelAndQuietZone(b.width, b.height, version)

	qrObj := &QrCode{
		Segments:             segments,
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
