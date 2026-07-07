package qr

import (
	"errors"

	"github.com/nachop51/qr-go/internal/matrix"
)

var ErrNoRenderer = errors.New("qr: no renderer set")

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
func (q *Code) Size() int { return q.Matrix.Size() }

func (q *Code) IsDark(x, y int) bool { return q.Matrix.Get(x, y) == matrix.Black }

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
