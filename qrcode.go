package qr

import (
	"errors"

	"github.com/nachop51/qr-go/internal/matrix"
	"github.com/nachop51/qr-go/internal/spec"
)

var ErrNoRenderer = errors.New("qr: no renderer set")

var (
	ErrInvalidUTF8     = spec.ErrInvalidUTF8Text
	ErrDataTooLong     = spec.ErrDataTooLong
	ErrInvalidOptions  = errors.New("qr: invalid options")
	ErrVersionTooSmall = spec.ErrVersionTooSmall
)

// Compatibility aliases keep the stable error categories discoverable under
// both concise and descriptive names.
var (
	ErrInvalidUTF8Text       = ErrInvalidUTF8
	ErrExcessiveData         = ErrDataTooLong
	ErrInvalidOption         = ErrInvalidOptions
	ErrForcedVersionTooSmall = ErrVersionTooSmall
)

// Render delegates entirely to the chosen renderer
func (q *Code) Render() error {
	if q.renderer == nil {
		return ErrNoRenderer
	}
	return q.renderer.Render(q)
}

func (q *Code) Bytes() ([]byte, error) {
	if q.renderer == nil {
		return nil, ErrNoRenderer
	}
	return q.renderer.Bytes(q)
}

// Module count
func (q *Code) Size() int { return q.matrix.Size() }

func (q *Code) IsDark(x, y int) bool {
	return q != nil && q.matrix != nil && q.matrix.InBounds(x, y) && q.matrix.Get(x, y) == matrix.Black
}

func (q *Code) Version() int                     { return q.version }
func (q *Code) Mask() int                        { return q.mask }
func (q *Code) CorrectionLevel() CorrectionLevel { return q.correctionLevel }
func (q *Code) UsesECI() bool                    { return q.usesECI }
func (q *Code) Segments() []Segment              { return cloneSegments(q.segments) }

func (q *Code) MaxLogoModules() int {
	size := q.Size()
	switch q.correctionLevel {
	case CorrectionLevelHigh:
		return size / 4
	case CorrectionLevelQuartile:
		return size / 5
	case CorrectionLevelMedium:
		return size / 6
	default: // Low
		return size / 7
	}
}
