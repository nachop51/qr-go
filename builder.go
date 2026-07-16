package qr

import (
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/nachop51/qr-go/internal/matrix"
	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/terminal"
)

type Code struct {
	correctionLevel CorrectionLevel
	version         int
	mask            int
	usesECI         bool
	segments        []Segment
	matrix          *matrix.Matrix
	renderer        render.Renderer
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
	if !b.errorCorrectionLevel.valid() {
		return nil, fmt.Errorf("%w: invalid correction level %d", ErrInvalidOptions, b.errorCorrectionLevel)
	}
	if b.textECIPolicy != TextECIPolicyAuto && b.textECIPolicy != TextECIPolicyDisabled {
		return nil, fmt.Errorf("%w: invalid text ECI policy %d", ErrInvalidOptions, b.textECIPolicy)
	}
	if b.version != -1 && (b.version < 1 || b.version > 40) {
		return nil, fmt.Errorf("%w: version must be auto or between 1 and 40", ErrInvalidOptions)
	}
	if b.mask != -1 && (b.mask < 0 || b.mask > 7) {
		return nil, fmt.Errorf("%w: mask must be auto or between 0 and 7", ErrInvalidOptions)
	}
	if b.renderer == nil || (reflect.ValueOf(b.renderer).Kind() == reflect.Pointer && reflect.ValueOf(b.renderer).IsNil()) {
		return nil, fmt.Errorf("%w: renderer is nil", ErrInvalidOptions)
	}
	if b.dataKind == DataKindText && len(b.data) > 7089 {
		return nil, ErrDataTooLong
	}

	var isECI bool
	var segments []Segment
	var err error

	switch b.dataKind {
	case DataKindText:
		if !utf8.Valid(b.data) {
			return nil, ErrInvalidUTF8
		}

		segments, isECI, err = b.segmentize()

		if err != nil {
			return nil, err
		}
	case DataKindBinary:
		segments = []Segment{{mode: EncodingModeByte, data: b.data}}
		isECI = false
	default:
		return nil, fmt.Errorf("%w: invalid data kind", ErrInvalidOptions)
	}

	version, err := detectVersion(segments, b.errorCorrectionLevel, isECI)

	if err != nil {
		return nil, err
	}

	if b.version != -1 {
		if b.version < version {
			return nil, fmt.Errorf("%w: data requires version %d, requested %d",
				ErrVersionTooSmall, version, b.version)
		}
		version = b.version
	}

	qrObj := &Code{
		segments:        cloneSegments(segments),
		version:         version,
		correctionLevel: b.errorCorrectionLevel,
		usesECI:         isECI,
		matrix:          createQrBase(version),
		renderer:        b.renderer,
	}

	data := buildCodewords(segments, version, b.errorCorrectionLevel, isECI)

	if err := placeData(qrObj.matrix, data); err != nil {
		return nil, err
	}

	if b.mask == -1 {
		qrObj.mask = bestMask(qrObj.matrix, version, b.errorCorrectionLevel)
	} else {
		qrObj.mask = b.mask
	}

	applyMask(qrObj.matrix, qrObj.mask)
	placeMetadata(qrObj.matrix, version, qrObj.mask, b.errorCorrectionLevel)

	return qrObj, nil
}

func cloneSegments(in []Segment) []Segment {
	out := make([]Segment, len(in))
	for i, s := range in {
		out[i] = Segment{mode: s.mode, data: append([]byte(nil), s.data...)}
	}
	return out
}
