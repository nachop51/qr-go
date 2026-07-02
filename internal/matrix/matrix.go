package matrix

import "fmt"

type Color uint8

const (
	White Color = iota
	Black
)

type cell struct {
	col       Color
	protected bool
}

type Matrix struct {
	size  int
	cells []cell
}

func New(size int) *Matrix {
	return &Matrix{size: size, cells: make([]cell, size*size)}
}

func (m *Matrix) Size() int { return m.size }

func (m *Matrix) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < m.size && y < m.size
}
func (m *Matrix) idx(x, y int) int {

	if !m.InBounds(x, y) {
		panic(fmt.Sprintf("matrix: (%d,%d) out of bounds, size %d", x, y, m.size))
	}

	return y*m.size + x
}
func (m *Matrix) Get(x, y int) Color {
	return m.cells[m.idx(x, y)].col
}

func (m *Matrix) Set(x, y int, col Color) {
	m.cells[m.idx(x, y)].col = col
}

func (m *Matrix) IsProtected(x, y int) bool {
	return m.cells[m.idx(x, y)].protected
}

func (m *Matrix) Protect(x, y int, col Color) {
	idx := m.idx(x, y)
	m.cells[idx].protected = true
	m.cells[idx].col = col
}

func (m *Matrix) Square(x, y, size int, col Color) {
	for i := range size {
		for j := range size {
			m.Protect(x+i, y+j, col)
		}
	}
}

func (m *Matrix) Toggle(x, y int) {
	idx := m.idx(x, y)
	m.cells[idx].col ^= 1
}
