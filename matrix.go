package qr

import (
	"fmt"
	"image"
)

type QrPoint struct {
	x         int
	y         int
	col       QrColor
	protected bool
	drawn     bool
}

type PlaceOptions struct {
	Protected bool
}

func (q *QrCode) at(x, y int) *QrPoint {
	return &q.points[y][x]
}

func (q *QrCode) PlacePoint(point image.Point, col QrColor, options ...PlaceOptions) {

	var protected bool
	if len(options) > 0 {
		protected = options[0].Protected
	}

	for y := range q.points {
		if y == point.Y {
			for x := range q.points[y] {
				if x == point.X {
					p := q.at(x, y)
					p.col = col
					p.protected = protected
					return
				}
			}
		}
	}
}

func (q *QrCode) placeSquare(point image.Point, size int, col QrColor, fill bool, options ...PlaceOptions) {
	for i := point.X; i < point.X+size; i++ {
		for j := point.Y; j < point.Y+size; j++ {
			if fill || i == point.X || i == point.X+size-1 || j == point.Y || j == point.Y+size-1 {
				q.PlacePoint(image.Point{
					X: i,
					Y: j,
				}, col, options...)
			}
		}
	}
}

func createPoints(modules int) [][]QrPoint {
	points := make([][]QrPoint, modules)

	for i := range points {
		points[i] = make([]QrPoint, modules)
	}

	for y := range modules {
		for x := range modules {
			points[y][x] = QrPoint{
				x:   x,
				y:   y,
				col: QrWhite,
			}
		}
	}

	return points
}

func (q *QrCode) placeFinders() {
	q.placeSquare(image.Point{0, 0}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{0, 0}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{1, 1}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{2, 2}, 3, QrBlack, true, PlaceOptions{Protected: true})

	modules := capacityTable[q.Version].modules

	q.placeSquare(image.Point{0, modules - 8}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{0, modules - 7}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{1, modules - 6}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{2, modules - 5}, 3, QrBlack, true, PlaceOptions{Protected: true})

	q.placeSquare(image.Point{modules - 8, 0}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{modules - 7, 0}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{modules - 6, 1}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{modules - 5, 2}, 3, QrBlack, true, PlaceOptions{Protected: true})
}

func (q *QrCode) placeTimingPattern() {
	modules := capacityTable[q.Version].modules
	colors := []QrColor{QrBlack, QrWhite}

	for i := 8; i < modules-8; i++ {
		if i%2 == 0 {
			q.PlacePoint(image.Point{i, 6}, colors[0], PlaceOptions{Protected: true})
		} else {
			q.PlacePoint(image.Point{i, 6}, colors[1], PlaceOptions{Protected: true})
		}
	}

	for i := 8; i < modules-8; i++ {
		if i%2 == 0 {
			q.PlacePoint(image.Point{6, i}, colors[0], PlaceOptions{Protected: true})
		} else {
			q.PlacePoint(image.Point{6, i}, colors[1], PlaceOptions{Protected: true})
		}
	}
}

func (q *QrCode) formatModules() [][3]int {
	size := capacityTable[q.Version].modules

	mods := [][3]int{
		// Copia 1, alrededor del finder superior izquierdo
		{8, 0, 0}, {8, 1, 1}, {8, 2, 2}, {8, 3, 3}, {8, 4, 4}, {8, 5, 5},
		{8, 7, 6}, // salta la fila 6 (timing)
		{8, 8, 7},
		{7, 8, 8},
		{5, 8, 9}, {4, 8, 10}, {3, 8, 11}, {2, 8, 12}, {1, 8, 13}, {0, 8, 14}, // salta col 6
	}

	// Copia 2: bits 0-7 por la fila 8 desde la derecha
	for i := range 8 {
		mods = append(mods, [3]int{size - 1 - i, 8, i})
	}
	// Copia 2: bits 8-14 por la columna 8 desde abajo (arranca en size-7 → saltea el dark module en size-8)
	for i := 8; i < 15; i++ {
		mods = append(mods, [3]int{8, size - 15 + i, i})
	}

	return mods
}

func (q *QrCode) versionModules() [][3]int {
	if q.Version < 7 {
		return nil
	}

	size := capacityTable[q.Version].modules
	mods := [][3]int{}

	bit := 0
	// Copia 1: 6 columnas (0..5) × 3 filas (size-11..size-9)
	for col := range 6 {
		for row := size - 11; row <= size-9; row++ {
			mods = append(mods, [3]int{col, row, bit})
			bit++
		}
	}

	bit = 0
	// Copia 2: transpuesta — 6 filas (0..5) × 3 columnas (size-11..size-9)
	for row := range 6 {
		for col := size - 11; col <= size-9; col++ {
			mods = append(mods, [3]int{col, row, bit})
			bit++
		}
	}

	return mods
}

func (q *QrCode) placeReserved() {
	reservedY := 4*q.Version + 9
	// Dark reserved module
	q.PlacePoint(image.Point{8, reservedY}, QrBlack, PlaceOptions{Protected: true})

	for _, m := range q.formatModules() {
		q.PlacePoint(image.Point{m[0], m[1]}, QrWhite, PlaceOptions{Protected: true})
	}

	if q.Version >= 7 {
		for _, m := range q.versionModules() {
			q.PlacePoint(image.Point{m[0], m[1]}, QrWhite, PlaceOptions{Protected: true})
		}
	}
}

func (q *QrCode) placeAlignmentPatterns() {
	if q.Version == 1 {
		return
	}

	coords := alignmentCoords(q.Version)
	size := capacityTable[q.Version].modules
	last := size - 1

	isFinderCorner := func(x, y int) bool {
		return (x == 6 && y == 6) || (x == 6 && y == last-6) || (x == last-6 && y == 6)
	}

	for _, x := range coords {
		for _, y := range coords {
			if isFinderCorner(x, y) {
				continue
			}

			q.placeSquare(image.Point{x - 2, y - 2}, 5, QrBlack, false, PlaceOptions{Protected: true})
			q.placeSquare(image.Point{x - 1, y - 1}, 3, QrWhite, false, PlaceOptions{Protected: true})
			q.PlacePoint(image.Point{x, y}, QrBlack, PlaceOptions{Protected: true})
		}
	}
}

func (q *QrCode) placeData(data []byte) error {
	modules := capacityTable[q.Version].modules
	r := bitReader{data: data}

	upward := true

	for col := modules - 1; col > 0; col -= 2 {
		// TODO: Dynamic timing column
		if col == 6 {
			col = 5
		}
		for vert := range modules {
			y := vert

			if upward {
				y = modules - 1 - vert
			}

			for i := range 2 {
				x := col - i
				p := q.at(x, y)

				if !p.protected {
					p.col = r.next()
				}
			}
		}

		upward = !upward
	}

	if r.pos != len(r.data)*8 {
		return fmt.Errorf("data does not fit in QR code: pos=%d, len=%d", r.pos, len(r.data)*8)
	}

	return nil
}

func (q *QrCode) placeMetadata(mask int) {
	group := (int(q.ErrorCorrectionLevel.value) << 3) | mask
	encFormat := encodeFormat(uint16(group))
	encVersion := encodeVersion(uint16(q.Version))
	colors := []QrColor{QrWhite, QrBlack}

	for _, m := range q.formatModules() {
		bit := bitOf(encFormat, m[2])
		col := colors[bit]
		q.PlacePoint(image.Point{m[0], m[1]}, col, PlaceOptions{Protected: true})
	}

	if q.Version < 7 {
		return
	}

	for _, m := range q.versionModules() {
		bit := bitOf(uint16(encVersion), m[2])
		col := colors[bit]
		q.PlacePoint(image.Point{m[0], m[1]}, col, PlaceOptions{Protected: true})
	}
}
