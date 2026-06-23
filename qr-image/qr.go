package qrimage

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"rsc.io/qr/gf256"
)

var QrMaskError = errors.New("invalid mask: a number between 0 and 7 must be provided")

type QrImage struct {
	Data                 []byte
	EncodingMode         QrEncodingMode
	ErrorCorrectionLevel QrCorrectionLevel
	Version              int
	Mask                 int
	Filename             string
	img                  *image.RGBA
	pixelSize            int
	quietZoneX           int
	quietZoneY           int
	blackColor           color.Color
	whiteColor           color.Color
	points               [][]QrPoint
}

type bitWriter struct {
	data   []byte
	bitPos int
}

func (w *bitWriter) appendBits(value, count int) {
	for i := count - 1; i >= 0; i-- {
		if w.bitPos%8 == 0 {
			w.data = append(w.data, 0)
		}
		bit := byte((value >> i) & 1)
		w.data[len(w.data)-1] |= bit << (7 - (w.bitPos % 8))
		w.bitPos++
	}
}

func (q *QrImage) encodeBytes(bitsData *bitWriter) {
	for _, b := range q.Data {
		bitsData.appendBits(int(b), 8)
	}
}

func (q *QrImage) encodeNumeric(bitsData *bitWriter) {
	i := 0

	for ; i+2 < len(q.Data); i += 3 {
		group := (int(q.Data[i]-'0') * 100) + (int(q.Data[i+1]-'0') * 10) + int(q.Data[i+2]-'0')
		bitsData.appendBits(group, 10)
	}
	switch len(q.Data) - i {
	case 2:
		group := (int(q.Data[i]-'0') * 10) + int(q.Data[i+1]-'0')
		bitsData.appendBits(group, 7)
	case 1:
		group := int(q.Data[i] - '0')
		bitsData.appendBits(group, 4)
	}
}

func (q *QrImage) encodeAlphanumeric(bitsData *bitWriter) {
	i := 0

	for ; i+1 < len(q.Data); i += 2 {
		value := charValue(q.Data[i])*45 + charValue(q.Data[i+1])
		bitsData.appendBits(value, 11)
	}

	if i < len(q.Data) {
		value := charValue(q.Data[i])
		bitsData.appendBits(value, 6)
	}
}

func (q *QrImage) addTerminatorAndPadding(bitsData *bitWriter) {
	dataBytes := capacityTable[q.Version].bytes - capacityTable[q.Version].ec[q.ErrorCorrectionLevel.level]
	capacityBits := dataBytes * 8

	terminatorBits := min(4, capacityBits-bitsData.bitPos)

	bitsData.appendBits(0, terminatorBits)
	// Bit align
	bitsData.appendBits(0, (8-(bitsData.bitPos%8))%8)

	pad := []byte{0xEC, 0x11}
	for i := 0; bitsData.bitPos < capacityBits; i++ {
		bitsData.appendBits(int(pad[i%2]), 8)
	}
}

func (q *QrImage) errorCorrection(data []byte) []byte {
	ecCount := capacityTable[q.Version].ec[q.ErrorCorrectionLevel.level]

	field := gf256.NewField(0x11d, 0x02)
	enc := gf256.NewRSEncoder(field, ecCount)

	ecBytes := make([]byte, ecCount)
	enc.ECC(data, ecBytes)

	return ecBytes
}

type QrDataBlock struct {
	numBlocks  int
	ecPerBlock int
	g1, d1     int
	g2, d2     int
}

func (q *QrImage) blockRecipe() QrDataBlock {
	level := q.ErrorCorrectionLevel.level

	g1 := eccTable[q.Version][level][0]
	g2 := eccTable[q.Version][level][1]
	numBlocks := g1 + g2

	totalEC := capacityTable[q.Version].ec[level]
	totalData := capacityTable[q.Version].bytes - totalEC

	ecPerBlock := totalEC / numBlocks
	d1 := totalData / numBlocks
	d2 := d1 + 1

	return QrDataBlock{numBlocks, ecPerBlock, g1, d1, g2, d2}
}

func (q *QrImage) encode() []byte {

	var bitsData bitWriter

	bitsData.appendBits(int(q.EncodingMode), 4)
	bitsData.appendBits(len(q.Data), getBitLengthIndicator(q.Version, q.EncodingMode))

	switch q.EncodingMode {
	case QrEncodingModeByte:
		q.encodeBytes(&bitsData)
	case QrEncodingModeNumeric:
		q.encodeNumeric(&bitsData)
	case QrEncodingModeAlphanumeric:
		q.encodeAlphanumeric(&bitsData)
	}

	q.addTerminatorAndPadding(&bitsData)

	recipe := q.blockRecipe()

	fmt.Printf("%+v\n", recipe)

	ecBytes := q.errorCorrection(bitsData.data)

	// Combine data and error correction bytes
	fullData := append(bitsData.data, ecBytes...)

	return fullData
}

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

func (q *QrImage) applyMask(mask int) error {
	if mask < 0 || mask > 7 {
		return QrMaskError
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
func (q *QrImage) maskPenalty1() int {
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
func (q *QrImage) maskPenalty2() int {
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
func (q *QrImage) maskPenalty3() int {
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
func (q *QrImage) maskPenalty4() int {
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

func (q *QrImage) measureMaskScore(mask int) (int, error) {
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

func (q *QrImage) Debug() {
	for y := range q.points {
		for x := range q.points[y] {
			if q.points[y][x].col == QrBlack {
				if q.points[y][x].protected {
					print("X")
				} else {
					print("x")
				}
			} else {
				if q.points[y][x].protected {
					print(",")
				} else {
					print(".")
				}
			}
		}
		println()
	}
}

func (q *QrImage) debugMasks() {
	originalFile := q.Filename
	originalMask := q.Mask

	for mask := range 8 {
		q.Filename = fmt.Sprintf("mask%d.png", mask)
		q.applyMask(mask)
		q.placeMetadata(mask)
		q.Draw()
		q.Save()
		q.applyMask(mask)
	}
	q.Filename = originalFile
	q.Mask = originalMask
}

func (q *QrImage) Save() error {
	f, err := os.Create(q.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	err = png.Encode(f, q.img)
	return err
}
