package qrimage

import (
	"image"
	"log"
)

type PlaceOptions struct {
	Protected bool
}

func (q *QrImage) at(x, y int) *QrPoint {
	return &q.points[y][x]
}

func (q *QrImage) PlacePoint(point image.Point, col QrColor, options ...PlaceOptions) {

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

func (q *QrImage) placeSquare(point image.Point, size int, col QrColor, fill bool, options ...PlaceOptions) {
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

func (q *QrImage) placeMarkers() {
	q.placeSquare(image.Point{0, 0}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{0, 0}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{1, 1}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{2, 2}, 3, QrBlack, true, PlaceOptions{Protected: true})

	q.placeSquare(image.Point{0, 13}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{0, 14}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{1, 15}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{2, 16}, 3, QrBlack, true, PlaceOptions{Protected: true})

	q.placeSquare(image.Point{13, 0}, 8, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{14, 0}, 7, QrBlack, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{15, 1}, 5, QrWhite, false, PlaceOptions{Protected: true})
	q.placeSquare(image.Point{16, 2}, 3, QrBlack, true, PlaceOptions{Protected: true})
}

func (q *QrImage) placeTimingMarkers() {
	q.PlacePoint(image.Point{8, 6}, QrBlack, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{9, 6}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{10, 6}, QrBlack, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{11, 6}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{12, 6}, QrBlack, PlaceOptions{Protected: true})

	q.PlacePoint(image.Point{6, 8}, QrBlack, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{6, 9}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{6, 10}, QrBlack, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{6, 11}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{6, 12}, QrBlack, PlaceOptions{Protected: true})
}

func (q *QrImage) placeFormatAndReserved() {
	q.PlacePoint(image.Point{8, 13}, QrBlack, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 0}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 1}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 2}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 3}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 4}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 5}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 7}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 8}, QrWhite, PlaceOptions{Protected: true})

	q.PlacePoint(image.Point{0, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{1, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{2, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{3, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{4, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{5, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{7, 8}, QrWhite, PlaceOptions{Protected: true})

	q.PlacePoint(image.Point{13, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{14, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{15, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{16, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{17, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{18, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{19, 8}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{20, 8}, QrWhite, PlaceOptions{Protected: true})

	q.PlacePoint(image.Point{8, 14}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 15}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 16}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 17}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 18}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 19}, QrWhite, PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 20}, QrWhite, PlaceOptions{Protected: true})
}

type bitReader struct {
	data []byte
	pos  int
}

func (r *bitReader) next() QrColor {
	byteIdx := r.pos / 8
	if byteIdx >= len(r.data) {
		return 0
	}
	bit := int((r.data[byteIdx] >> (7 - r.pos%8)) & 1)
	r.pos++

	if bit == 1 {
		return QrBlack
	}
	return QrWhite
}

func (q *QrImage) placeData(data []byte) {
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
		log.Fatalf("bitReader position (%d) does not match data length (%d)", r.pos, len(r.data)*8)
	}
}

func encodeFormat(d uint16) uint16 {
	const g = 0x537 // 10100110111, generador grado 10

	v := d << 10
	// limpio los 5 bits de datos, posiciones 14..10
	for i := 14; i >= 10; i-- {
		if (v>>uint(i))&1 == 1 {
			v ^= g << uint(i-10)
		}
	}
	// ahora v tiene solo el resto (10 bits bajos)
	code := (d << 10) | v
	return code ^ 0x5412
}

func bitOf(f uint16, i int) int {
	return int((f >> uint(i)) & 1)
}

func (q *QrImage) placeMetadata(mask int) {
	group := (int(q.ErrorCorrectionLevel.value) << 3) | mask
	size := capacityTable[q.Version].modules
	encoded := encodeFormat(uint16(group))

	colors := []QrColor{QrWhite, QrBlack}

	for i := 0; i <= 5; i++ {
		q.PlacePoint(image.Point{8, i}, colors[bitOf(encoded, i)], PlaceOptions{Protected: true})
	}
	q.PlacePoint(image.Point{8, 7}, colors[bitOf(encoded, 6)], PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{8, 8}, colors[bitOf(encoded, 7)], PlaceOptions{Protected: true})
	q.PlacePoint(image.Point{7, 8}, colors[bitOf(encoded, 8)], PlaceOptions{Protected: true})

	for i := 9; i < 15; i++ {
		q.PlacePoint(image.Point{14 - i, 8}, colors[bitOf(encoded, i)], PlaceOptions{Protected: true})
	}

	// ---- Copia 2: bajo el finder de arriba-derecha y al lado del de abajo-izquierda ----
	// bits 0-7 por la fila 8, desde la columna size-1 hacia la izquierda (20..13)
	for i := 0; i < 8; i++ {
		q.PlacePoint(image.Point{size - 1 - i, 8}, colors[bitOf(encoded, i)], PlaceOptions{Protected: true})
	}
	// bits 8-14 por la columna 8, filas size-7 a size-1 (14..20)
	for i := 8; i < 15; i++ {
		q.PlacePoint(image.Point{8, size - 15 + i}, colors[bitOf(encoded, i)], PlaceOptions{Protected: true})
	}
	// el módulo oscuro fijo (8, size-8) ya lo pusiste en el esqueleto
}

func (q *QrImage) drawPoint(point QrPoint) {
	startingX := point.x * q.pixelSize
	startingY := point.y * q.pixelSize

	color := q.blackColor

	if point.col == QrWhite {
		color = q.whiteColor
	}

	for x := startingX; x < startingX+q.pixelSize; x++ {
		for y := startingY; y < startingY+q.pixelSize; y++ {
			q.img.Set(x, y, color)
		}
	}
}

func (q *QrImage) Draw() {
	for _, points := range q.points {
		for _, point := range points {
			if !point.drawn {
				q.drawPoint(point)
				point.drawn = true
			}
		}
	}
}
