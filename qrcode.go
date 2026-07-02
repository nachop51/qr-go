package qr

import (
	"errors"

	"nachop51/qr/internal/matrix"
)

// ErrNoRenderer is returned by Render when no renderer was set on the builder.
var ErrNoRenderer = errors.New("qr: no renderer set")

// Render delegates entirely to the chosen renderer — core does no output logic.
func (q *QrCode) Render() error {
	if q.renderer == nil {
		return ErrNoRenderer
	}
	return q.renderer.Render(q)
}

// Size returns the module count per side.
func (q *QrCode) Size() int { return q.Matrix.Size() }

// IsDark reports whether the module at (x, y) is black.
func (q *QrCode) IsDark(x, y int) bool { return q.Matrix.Get(x, y) == matrix.Black }
