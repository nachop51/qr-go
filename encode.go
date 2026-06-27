package qr

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

func (q *QrCode) addTerminatorAndPadding(bitsData *bitWriter) {
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

func (q *QrCode) encode() []byte {
	var bitsData bitWriter

	if q.isECI {
		bitsData.appendBits(0b0111, 4)
		bitsData.appendBits(26, 8)
	}

	for _, seg := range q.Segments {
		bitsData.appendBits(int(seg.Mode), 4)
		bitsData.appendBits(seg.dataLength(), getBitLengthIndicator(q.Version, seg.Mode))

		switch seg.Mode {
		case EncodingModeByte:
			seg.encodeBytes(&bitsData)
		case EncodingModeNumeric:
			seg.encodeNumeric(&bitsData)
		case EncodingModeAlphanumeric:
			seg.encodeAlphanumeric(&bitsData)
		case EncodingModeKanji:
			seg.encodeKanji(&bitsData)
		}
	}

	q.addTerminatorAndPadding(&bitsData)

	blocks := q.splitIntoBlocks(bitsData.data)
	ecs := q.errorCorrectionPerBlock(blocks)

	// ecBytes := q.errorCorrection(bitsData.data)

	// Combine data and error correction bytes
	// fullData := append(bitsData.data, ecBytes...)

	return interleave(blocks, ecs)
}
