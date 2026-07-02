package qr

import (
	"unicode/utf8"

	"nachop51/qr/internal/matrix"
	"nachop51/qr/internal/spec"
	"nachop51/qr/render"
	"nachop51/qr/render/terminal"
)

type QrCode struct {
	ErrorCorrectionLevel QrCorrectionLevel
	Version              int
	Mask                 int
	IsECI                bool
	Segments             []Segment
	Matrix               *matrix.Matrix
	renderer             render.Renderer
}

type QrBuilder struct {
	data                 []byte
	dataKind             QrDataKind
	textECIPolicy        QrTextECIPolicy
	errorCorrectionLevel QrCorrectionLevel
	renderer             render.Renderer
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
		renderer:             terminal.New(), // default: lightweight terminal output
	}
}

func (b *QrBuilder) SetErrorCorrectionLevel(level QrCorrectionLevel) *QrBuilder {
	b.errorCorrectionLevel = level
	return b
}

func (b *QrBuilder) SetTextECIPolicy(policy QrTextECIPolicy) *QrBuilder {
	b.textECIPolicy = policy
	return b
}

func (b *QrBuilder) SetRenderer(r render.Renderer) *QrBuilder { b.renderer = r; return b }

func (b *QrBuilder) Build() (*QrCode, error) {
	var isECI bool = false
	var segments []Segment
	var err error

	switch b.dataKind {
	case QrDataKindText:
		if !utf8.Valid(b.data) {
			return nil, spec.ErrInvalidUTF8Text
		}

		segments, isECI, err = b.segmentize()

		if err != nil {
			return nil, err
		}
	case QrDataKindBinary:
		segments = []Segment{{mode: EncodingModeByte, data: b.data}}
		isECI = false
	default:
		return nil, spec.ErrInvalidDataKind
	}

	version, err := detectVersion(segments, b.errorCorrectionLevel, isECI)

	if err != nil {
		return nil, err
	}

	qrObj := &QrCode{
		Segments:             segments,
		Version:              version,
		ErrorCorrectionLevel: b.errorCorrectionLevel,
		IsECI:                isECI,
		Matrix:               createQrBase(version),
		renderer:             b.renderer,
	}

	data := buildCodewords(segments, version, b.errorCorrectionLevel, isECI)

	placeData(qrObj.Matrix, data)

	qrObj.Mask = bestMask(qrObj.Matrix, version, b.errorCorrectionLevel)
	applyMask(qrObj.Matrix, qrObj.Mask)
	placeMetadata(qrObj.Matrix, version, qrObj.Mask, b.errorCorrectionLevel)

	return qrObj, nil
}
