package qr

import (
	"fmt"
	"github.com/nachop51/qr-go/internal/coding"
	"github.com/nachop51/qr-go/internal/matrix"
	"github.com/nachop51/qr-go/internal/spec"
)

func createQrBase(version int) *matrix.Matrix {
	m := matrix.New(spec.Modules(version))

	placeFinders(m)
	placeTimingPattern(m)
	placeAlignmentPatterns(m, version)
	reserveMetadata(m, version)

	return m
}

func placeFinders(m *matrix.Matrix) {
	// Top left
	m.Square(0, 0, 8, matrix.White)
	m.Square(0, 0, 7, matrix.Black)
	m.Square(1, 1, 5, matrix.White)
	m.Square(2, 2, 3, matrix.Black)

	// Bottom left
	m.Square(0, m.Size()-8, 8, matrix.White)
	m.Square(0, m.Size()-7, 7, matrix.Black)
	m.Square(1, m.Size()-6, 5, matrix.White)
	m.Square(2, m.Size()-5, 3, matrix.Black)

	// Top right
	m.Square(m.Size()-8, 0, 8, matrix.White)
	m.Square(m.Size()-7, 0, 7, matrix.Black)
	m.Square(m.Size()-6, 1, 5, matrix.White)
	m.Square(m.Size()-5, 2, 3, matrix.Black)
}

func placeTimingPattern(m *matrix.Matrix) {
	colors := []matrix.Color{matrix.Black, matrix.White}

	for i := 8; i < m.Size()-8; i++ {
		// Vertical stripe
		m.Protect(i, 6, colors[i%2])
		// Horizontal stripe
		m.Protect(6, i, colors[i%2])
	}
}

func placeAlignmentPatterns(m *matrix.Matrix, version int) {
	coords := spec.AlignmentCoords(version)
	last := m.Size() - 1

	isFinderCorner := func(x, y int) bool {
		return (x == 6 && y == 6) || (x == 6 && y == last-6) || (x == last-6 && y == 6)
	}

	for _, x := range coords {
		for _, y := range coords {
			if isFinderCorner(x, y) {
				continue
			}

			m.Square(x-2, y-2, 5, matrix.Black)
			m.Square(x-1, y-1, 3, matrix.White)
			m.Protect(x, y, matrix.Black)
		}
	}
}

func reserveMetadata(m *matrix.Matrix, version int) {
	x, y := spec.DarkModule(version)
	m.Protect(x, y, matrix.Black)

	for _, fm := range spec.FormatModules(version) {
		m.Protect(fm.X, fm.Y, matrix.White)
	}

	if version >= 7 {
		for _, fm := range spec.VersionModules(version) {
			m.Protect(fm.X, fm.Y, matrix.White)
		}
	}
}

func placeData(m *matrix.Matrix, data []byte) {
	r := coding.NewBitReader(data)

	upward := true

	for col := m.Size() - 1; col > 0; col -= 2 {
		// TODO: Dynamic timing column
		if col == 6 {
			col = 5
		}
		for vert := range m.Size() {
			y := vert

			if upward {
				y = m.Size() - 1 - vert
			}

			for i := range 2 {
				x := col - i

				if !m.IsProtected(x, y) {
					m.Set(x, y, matrix.Color(r.Next()))
				}
			}
		}

		upward = !upward
	}

	if r.Pos() != len(r.Data())*8 {
		panic(fmt.Sprintf("data does not fit in QR code: pos=%d, len=%d", r.Pos(), len(r.Data())*8))
	}
}

func bitOf(f uint32, i int) int {
	return int((f >> uint(i)) & 1)
}

func placeMetadata(m *matrix.Matrix, version, mask int, ec CorrectionLevel) {
	group := (ec.formatBits() << 3) | mask
	encFormat := encodeFormat(uint16(group))
	encVersion := encodeVersion(uint16(version))

	for _, fm := range spec.FormatModules(version) {
		bit := bitOf(uint32(encFormat), fm.Bit)
		m.Set(fm.X, fm.Y, matrix.Color(bit))
	}

	if version < 7 {
		return
	}

	for _, vm := range spec.VersionModules(version) {
		bit := bitOf(encVersion, vm.Bit)
		m.Set(vm.X, vm.Y, matrix.Color(bit))
	}
}
