package qr

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const costImpossible = 1 << 30

func (s *Segment) payloadBits() int {
	n := len(s.Data)
	switch s.Mode {
	case EncodingModeNumeric:
		bits := (n / 3) * 10
		if n%3 == 2 {
			bits += 7
		}
		if n%3 == 1 {
			bits += 4
		}
		return bits
	case EncodingModeAlphanumeric:
		return (n/2)*11 + (n%2)*6
	case EncodingModeKanji:
		return utf8.RuneCountInString(string(s.Data)) * 13
	default:
		return n * 8
	}
}

func (s *Segment) dataLength() int {
	if s.Mode == EncodingModeKanji {
		sjis, _, _ := transform.String(japanese.ShiftJIS.NewEncoder(), string(s.Data))
		return len(sjis) / 2
	}
	return len(s.Data)
}

func (s *Segment) encodeBytes(bitsData *bitWriter) {
	for _, b := range s.Data {
		bitsData.appendBits(int(b), 8)
	}
}

func (s *Segment) encodeNumeric(bitsData *bitWriter) {
	i := 0

	for ; i+2 < len(s.Data); i += 3 {
		group := (int(s.Data[i]-'0') * 100) + (int(s.Data[i+1]-'0') * 10) + int(s.Data[i+2]-'0')
		bitsData.appendBits(group, 10)
	}
	switch len(s.Data) - i {
	case 2:
		group := (int(s.Data[i]-'0') * 10) + int(s.Data[i+1]-'0')
		bitsData.appendBits(group, 7)
	case 1:
		group := int(s.Data[i] - '0')
		bitsData.appendBits(group, 4)
	}
}

func (s *Segment) encodeAlphanumeric(bitsData *bitWriter) {
	i := 0

	for ; i+1 < len(s.Data); i += 2 {
		value := charValue(s.Data[i])*45 + charValue(s.Data[i+1])
		bitsData.appendBits(value, 11)
	}

	if i < len(s.Data) {
		value := charValue(s.Data[i])
		bitsData.appendBits(value, 6)
	}
}

func (s *Segment) encodeKanji(bitsData *bitWriter) {
	enc := japanese.ShiftJIS.NewEncoder()
	sjis, _, _ := transform.String(enc, string(s.Data))
	bytes := []byte(sjis)

	for i := 0; i < len(bytes); i += 2 {
		v := uint16(bytes[i])<<8 | uint16(bytes[i+1])

		var sub uint16
		if v >= 0x8140 && v <= 0x9FFC {
			sub = v - 0x8140
		} else {
			sub = v - 0xC140
		}

		high := sub >> 8
		low := sub & 0xFF
		packed := int(high)*0xC0 + int(low)

		bitsData.appendBits(packed, 13)
	}
}

func isKanjiRune(r rune) bool {
	sjis, _, err := transform.String(japanese.ShiftJIS.NewEncoder(), string(r))
	if err != nil {
		return false
	}
	bytes := []byte(sjis)
	if len(bytes) != 2 {
		return false
	}
	v := uint16(bytes[0])<<8 | uint16(bytes[1])
	return (v >= 0x8140 && v <= 0x9FFC) || (v >= 0xE040 && v <= 0xEBBF)
}

func charCost(r rune, mode EncodingMode) int {
	switch mode {
	case EncodingModeNumeric:
		if r >= '0' && r <= '9' {
			return 20
		}
	case EncodingModeAlphanumeric:
		if strings.ContainsRune(ALPHA_NUMERIC_CHARSET, r) {
			return 33
		}
		return costImpossible
	case EncodingModeByte:
		return utf8.RuneLen(r) * 48
	case EncodingModeKanji:
		if isKanjiRune(r) {
			return 78
		}
		return costImpossible
	}

	return costImpossible
}

func headerCost(mode EncodingMode, vr VersionRange) int {
	countBits := charCountIndicatorBits(mode, vr)
	totalBits := 4 + countBits

	return totalBits * 6
}

func charCountIndicatorBits(mode EncodingMode, vr VersionRange) int {
	switch mode {
	case EncodingModeNumeric:
		return []int{10, 12, 14}[vr]
	case EncodingModeAlphanumeric:
		return []int{9, 11, 13}[vr]
	case EncodingModeByte:
		return []int{8, 16, 16}[vr]
	case EncodingModeKanji:
		return []int{8, 10, 12}[vr]
	}
	return 0
}

func segmentizeOptimal(data []byte, vr VersionRange) ([]Segment, int) {
	runes := []rune(string(data))
	n := len(runes)

	if n == 0 {
		return nil, 0
	}

	modes := []EncodingMode{
		EncodingModeNumeric,
		EncodingModeAlphanumeric,
		EncodingModeByte,
		EncodingModeKanji,
	}

	cost := make([][]int, n+1)
	from := make([][]EncodingMode, n+1)

	for i := range cost {
		cost[i] = make([]int, len(modes))
		from[i] = make([]EncodingMode, len(modes))

		for j := range cost[i] {
			cost[i][j] = costImpossible
		}
	}

	for i, m := range modes {
		cost[0][i] = headerCost(m, vr)
	}

	for i := 1; i <= n; i++ {
		r := runes[i-1]

		for modeIdx, m := range modes {
			cc := charCost(r, m)
			if cc == costImpossible {
				continue
			}
			best := cost[i-1][modeIdx] + cc
			bestFrom := m

			for piPrev, mPrev := range modes {
				if mPrev == m {
					continue
				}
				candidate := cost[i-1][piPrev] + headerCost(m, vr) + cc
				if candidate < best {
					best = candidate
					bestFrom = mPrev
				}
			}

			cost[i][modeIdx] = best
			from[i][modeIdx] = bestFrom
		}
	}

	bestFinal := 0
	for modeIdx := 1; modeIdx < len(modes); modeIdx++ {
		if cost[n][modeIdx] < cost[n][bestFinal] {
			bestFinal = modeIdx
		}
	}

	return reconstructSegments(runes, from, modes, bestFinal), cost[n][bestFinal]
}

func reconstructSegments(runes []rune, from [][]EncodingMode, modes []EncodingMode, finalMode int) []Segment {
	n := len(runes)

	modePerChar := make([]EncodingMode, n)
	currentMode := modes[finalMode]

	for i := n; i >= 1; i-- {
		modePerChar[i-1] = currentMode
		currentModeIdx := indexOfMode(modes, currentMode)
		currentMode = from[i][currentModeIdx]
	}

	segments := []Segment{}
	start := 0

	for i := 1; i <= n; i++ {
		if i == n || modePerChar[i] != modePerChar[start] {
			chunk := string(runes[start:i])
			segments = append(segments, Segment{
				Mode: modePerChar[start],
				Data: []byte(chunk),
			})
			start = i
		}
	}

	return segments
}

func segmentsNeedsECI(segs []Segment) bool {
	for _, seg := range segs {
		if seg.Mode != EncodingModeByte {
			continue
		}

		if hasNonASCII(seg.Data) {
			return true
		}
	}
	return false
}

func indexOfMode(modes []EncodingMode, m EncodingMode) int {
	for i, mm := range modes {
		if mm == m {
			return i
		}
	}

	return -1
}
