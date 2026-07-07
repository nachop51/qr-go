package qr

import (
	"unicode/utf8"

	"github.com/nachop51/qr-go/internal/matrix"
	"github.com/nachop51/qr-go/internal/spec"
	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/terminal"
)

type Code struct {
	ErrorCorrectionLevel CorrectionLevel
	Version              int
	Mask                 int
	IsECI                bool
	Segments             []Segment
	Matrix               *matrix.Matrix
	renderer             render.Renderer
}

type Builder struct {
	data                 []byte
	dataKind             DataKind
	textECIPolicy        TextECIPolicy
	errorCorrectionLevel CorrectionLevel
	renderer             render.Renderer
}

func NewTextBuilder(text string) *Builder {
	return newBuilder([]byte(text), DataKindText)
}

func NewBinaryBuilder(data []byte) *Builder {
	return newBuilder(data, DataKindBinary)
}

func newBuilder(data []byte, inputKind DataKind) *Builder {
	return &Builder{
		data:                 append([]byte(nil), data...),
		dataKind:             inputKind,
		errorCorrectionLevel: CorrectionLevelMedium,
		renderer:             terminal.New(),
	}
}

func (b *Builder) SetErrorCorrectionLevel(level CorrectionLevel) *Builder {
	b.errorCorrectionLevel = level
	return b
}

func (b *Builder) SetTextECIPolicy(policy TextECIPolicy) *Builder {
	b.textECIPolicy = policy
	return b
}

func (b *Builder) SetRenderer(r render.Renderer) *Builder { b.renderer = r; return b }

func (b *Builder) Build() (*Code, error) {
	var isECI bool = false
	var segments []Segment
	var err error

	switch b.dataKind {
	case DataKindText:
		if !utf8.Valid(b.data) {
			return nil, spec.ErrInvalidUTF8Text
		}

		segments, isECI, err = b.segmentize()

		if err != nil {
			return nil, err
		}
	case DataKindBinary:
		segments = []Segment{{mode: EncodingModeByte, data: b.data}}
		isECI = false
	default:
		return nil, spec.ErrInvalidDataKind
	}

	version, err := detectVersion(segments, b.errorCorrectionLevel, isECI)

	if err != nil {
		return nil, err
	}

	qrObj := &Code{
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
