package qr

func flip(col QrColor) QrColor {
	if col == QrBlack {
		return QrWhite
	}
	return QrBlack
}

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
	return false
}

func (q *QrCode) applyMask(mask int) error {
	if mask < 0 || mask > 7 {
		return ErrInvalidMask
	}
	size := capacityTable[q.Version].modules

	for y := range size {
		for x := range size {
			p := q.at(x, y)
			if p.protected {
				continue
			}
			if maskCondition(mask, x, y) {
				p.col = flip(p.col)
			}
		}
	}

	return nil
}

// Penalize 5 or more consecutive black/white pixels horizontally or vertically
// 3 + (consecutive - 5)
func (q *QrCode) maskPenalty1() int {
	size := capacityTable[q.Version].modules
	score := 0

	for y := range size {
		run := 1
		for x := 1; x < size; x++ {
			if q.at(x, y).col == q.at(x-1, y).col {
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

	for x := range size {
		run := 1
		for y := 1; y < size; y++ {
			if q.at(x, y).col == q.at(x, y-1).col {
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
func (q *QrCode) maskPenalty2() int {
	score := 0

	for y := 0; y < len(q.points)-1; y++ {
		for x := 0; x < len(q.points[y])-1; x++ {
			c := q.at(x, y).col

			if c == q.at(x+1, y).col &&
				c == q.at(x, y+1).col &&
				c == q.at(x+1, y+1).col {
				score += 3
			}
		}
	}

	return score
}

// TODO: Redo this function, I don't understand it properly
func (q *QrCode) maskPenalty3() int {
	size := capacityTable[q.Version].modules
	penalty := 0

	bit := func(x, y int) bool {
		return q.at(x, y).col == QrBlack
	}

	// Patrón 1:1:3:1:1 = oscuro,claro,oscuro x3,claro,oscuro (7 módulos)
	// Más 4 módulos claros de un lado (no de los dos a la vez)
	check := func(get func(k int) bool) int {
		p := 0
		for i := 0; i <= size-7; i++ {
			// Buscar el patrón finder 7 en (i..i+6): D L DDD L D
			if get(i) && !get(i+1) && get(i+2) && get(i+3) && get(i+4) && !get(i+5) && get(i+6) {
				// Verificar 4 claros antes (i-4..i-1) o después (i+7..i+10)
				before := true
				for k := 1; k <= 4; k++ {
					if i-k < 0 || get(i-k) {
						before = false
						break
					}
				}
				after := true
				for k := range 4 {
					if i+7+k >= size || get(i+7+k) {
						after = false
						break
					}
				}
				if before {
					p += 40
				}
				if after {
					p += 40
				}
			}
		}
		return p
	}

	for y := range size {
		penalty += check(func(x int) bool { return bit(x, y) })
	}
	for x := range size {
		penalty += check(func(y int) bool { return bit(x, y) })
	}
	return penalty
}

// Look for % of black pixels and penalize
func (q *QrCode) maskPenalty4() int {
	dark := 0

	for y := range q.points {
		for x := range q.points[y] {
			if q.at(x, y).col == QrBlack {
				dark++
			}
		}
	}

	size := capacityTable[q.Version].modules
	total := size * size
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

func (q *QrCode) measureMaskScore(mask int) (int, error) {
	err := q.applyMask(mask)
	if err != nil {
		return 0, err
	}
	q.placeMetadata(mask)

	score := q.maskPenalty1() + q.maskPenalty2() + q.maskPenalty3() + q.maskPenalty4()

	err = q.applyMask(mask)
	if err != nil {
		return 0, err
	}

	return score, nil
}
