package qr

import (
	"fmt"
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
	data     []byte
	dataKind DataKind
	// -1 means auto, 0-7 means specific mask
	mask int
	// -1 means auto, 1-40 means specific version
	version              int
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
		mask:                 -1,
		version:              -1,
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

func (b *Builder) SetVersion(version int) *Builder {
	b.version = version
	return b
}

func (b *Builder) SetMask(mask int) *Builder {
	b.mask = mask
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

	if b.version != -1 {
		if b.version < 1 || b.version > 40 {
			return nil, spec.ErrInvalidVersion
		}
		if b.version < version {
			return nil, fmt.Errorf("%w: data requires version %d, requested %d",
				spec.ErrVersionTooSmall, version, b.version)
		}
		version = b.version
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

	if b.mask == -1 {
		qrObj.Mask = bestMask(qrObj.Matrix, version, b.errorCorrectionLevel)
	} else if b.mask >= 0 && b.mask <= 7 {
		qrObj.Mask = b.mask
	} else {
		return nil, spec.ErrInvalidMask
	}

	applyMask(qrObj.Matrix, qrObj.Mask)
	placeMetadata(qrObj.Matrix, version, qrObj.Mask, b.errorCorrectionLevel)

	return qrObj, nil
}
