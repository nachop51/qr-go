package qr

import (
	"errors"

	"github.com/nachop51/qr-go/internal/matrix"
)

// ErrNoRenderer is returned by Render when no renderer was set on the builder.
var ErrNoRenderer = errors.New("qr: no renderer set")

// Render delegates entirely to the chosen renderer — core does no output logic.
func (q *Code) Render() error {
	if q.renderer == nil {
		return ErrNoRenderer
	}
	return q.renderer.Render(q)
}

// Size returns the module count per side.
func (q *Code) Size() int { return q.Matrix.Size() }

// IsDark reports whether the module at (x, y) is black.
func (q *Code) IsDark(x, y int) bool { return q.Matrix.Get(x, y) == matrix.Black }

// MaxLogoModules returns the largest centred logo span, in modules, expected to
// remain scannable at this code's error-correction level. It's a conservative
// rule of thumb — roughly size/3 at High down to size/6 at Low — that overlay
// renderers use to warn about oversized logos.
func (q *Code) MaxLogoModules() int {
	size := q.Size()
	switch q.ErrorCorrectionLevel.level {
	case 3: // High
		return size / 3
	case 2: // Quartile
		return size / 4
	case 1: // Medium
		return size / 5
	default: // Low
		return size / 6
	}
}
