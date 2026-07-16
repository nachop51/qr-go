package qr

import (
	"math"

	"github.com/nachop51/qr-go/internal/matrix"
)

func maskCondition(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return (x*y)%2+(x*y)%3 == 0
	case 6:
		return ((x*y)%2+(x*y)%3)%2 == 0
	case 7:
		return ((x+y)%2+(x*y)%3)%2 == 0
	}
	panic("invalid mask")
}

func applyMask(m *matrix.Matrix, mask int) {
	for y := range m.Size() {
		for x := range m.Size() {
			if m.IsProtected(x, y) {
				continue
			}
			if maskCondition(mask, x, y) {
				m.Toggle(x, y)
			}
		}
	}
}

// Penalize 5 or more consecutive black/white pixels horizontally or vertically
// 3 + (consecutive - 5)
func maskPenalty1(m *matrix.Matrix) int {
	score := 0

	for y := range m.Size() {
		run := 1
		for x := 1; x < m.Size(); x++ {

			if m.Get(x, y) == m.Get(x-1, y) {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}

	for x := range m.Size() {
		run := 1
		for y := 1; y < m.Size(); y++ {
			if m.Get(x, y) == m.Get(x, y-1) {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}

	return score
}

// Penalize 2x2 black/white pixel blocks
func maskPenalty2(m *matrix.Matrix) int {
	score := 0

	for y := 0; y < m.Size()-1; y++ {
		for x := 0; x < m.Size()-1; x++ {
			c := m.Get(x, y)

			if c == m.Get(x+1, y) &&
				c == m.Get(x, y+1) &&
				c == m.Get(x+1, y+1) {
				score += 3
			}
		}
	}

	return score
}

func maskPenalty3(m *matrix.Matrix) int {
	size := m.Size()
	dark := func(x, y int) bool {
		return m.Get(x, y) == matrix.Black
	}

	check := func(get func(k int) bool) int {
		lightRun := func(from, to int) bool {
			if from < 0 || to > size {
				return false
			}
			for k := from; k < to; k++ {
				if get(k) {
					return false
				}
			}
			return true
		}
		p := 0
		for i := 0; i+6 < size; i++ {
			if get(i) && !get(i+1) && get(i+2) && get(i+3) && get(i+4) && !get(i+5) && get(i+6) &&
				(lightRun(i-4, i) || lightRun(i+7, i+11)) {
				p += 40
			}
		}
		return p
	}

	penalty := 0
	for y := range size {
		penalty += check(func(x int) bool { return dark(x, y) })
	}
	for x := range size {
		penalty += check(func(y int) bool { return dark(x, y) })
	}
	return penalty
}

// Look for % of black pixels and penalize
func maskPenalty4(m *matrix.Matrix) int {
	dark := 0

	for y := range m.Size() {
		for x := range m.Size() {
			if m.Get(x, y) == matrix.Black {
				dark++
			}
		}
	}

	total := m.Size() * m.Size()
	percent := float64(dark) * 100 / float64(total)

	prev := int(percent/5) * 5
	next := prev
	if float64(prev) < percent {
		next += 5
	}

	abs := func(n int) int {
		if n < 0 {
			return -n
		}
		return n
	}

	prevPenalty := abs(prev-50) / 5 * 10
	nextPenalty := abs(next-50) / 5 * 10

	return min(prevPenalty, nextPenalty)
}

func maskPenalty(m *matrix.Matrix, version, mask int, ec CorrectionLevel) int {
	placeMetadata(m, version, mask, ec)

	score := maskPenalty1(m) + maskPenalty2(m) + maskPenalty3(m) + maskPenalty4(m)

	return score
}

func bestMask(m *matrix.Matrix, version int, ec CorrectionLevel) int {
	best, bestScore := 0, math.MaxInt

	for mask := range 8 {
		applyMask(m, mask)
		score := maskPenalty(m, version, mask, ec)
		applyMask(m, mask)

		if bestScore > score {
			best, bestScore = mask, score
		}
	}

	return best
}
