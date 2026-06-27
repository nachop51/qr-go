package qr

func (q *QrCode) drawQuietZone() {
	modules := capacityTable[q.Version].modules
	qrWidth := modules * q.pixelSize
	qrHeight := modules * q.pixelSize

	fillRect := func(startX, startY, endX, endY int) {
		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				q.img.Set(x, y, q.whiteColor)
			}
		}
	}

	fillRect(0, 0, q.img.Rect.Max.X, q.quietZoneY)
	fillRect(0, q.quietZoneY+qrHeight, q.img.Rect.Max.X, q.img.Rect.Max.Y)
	fillRect(0, q.quietZoneY, q.quietZoneX, q.quietZoneY+qrHeight)
	fillRect(q.quietZoneX+qrWidth, q.quietZoneY, q.img.Rect.Max.X, q.quietZoneY+qrHeight)
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

func encodeVersion(version uint16) uint32 {
	const g = 0x1F25 // generador BCH(18,6), grado 12

	v := uint32(version) << 12 // dejar lugar para 12 bits de EC
	// limpiar los 6 bits de datos, posiciones 17..12
	for i := 17; i >= 12; i-- {
		if (v>>uint(i))&1 == 1 {
			v ^= g << uint(i-12)
		}
	}
	// ahora v tiene el resto en los 12 bits bajos
	return (uint32(version) << 12) | v
	// sin XOR final
}

func bitOf(f uint16, i int) int {
	return int((f >> uint(i)) & 1)
}

func (q *QrCode) drawPoint(point QrPoint) {
	startingX := point.x*q.pixelSize + q.quietZoneX
	startingY := point.y*q.pixelSize + q.quietZoneY

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
