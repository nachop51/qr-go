package qr

import (
	"nachop51/qr/internal/coding"
	"nachop51/qr/internal/spec"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const costImpossible = 1 << 30

func (s *Segment) payloadBits() int {
	n := len(s.data)
	switch s.mode {
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
		return utf8.RuneCountInString(string(s.data)) * 13
	default:
		return n * 8
	}
}

func (s *Segment) dataLength() int {
	if s.mode == EncodingModeKanji {
		sjis, _, _ := transform.String(japanese.ShiftJIS.NewEncoder(), string(s.data))
		return len(sjis) / 2
	}
	return len(s.data)
}

func (s *Segment) encodeBytes(bitsData *coding.BitWriter) {
	for _, b := range s.data {
		bitsData.AppendBits(int(b), 8)
	}
}

func (s *Segment) encodeNumeric(bitsData *coding.BitWriter) {
	i := 0

	for ; i+2 < len(s.data); i += 3 {
		group := (int(s.data[i]-'0') * 100) + (int(s.data[i+1]-'0') * 10) + int(s.data[i+2]-'0')
		bitsData.AppendBits(group, 10)
	}
	switch len(s.data) - i {
	case 2:
		group := (int(s.data[i]-'0') * 10) + int(s.data[i+1]-'0')
		bitsData.AppendBits(group, 7)
	case 1:
		group := int(s.data[i] - '0')
		bitsData.AppendBits(group, 4)
	}
}

func (s *Segment) encodeAlphanumeric(bitsData *coding.BitWriter) {
	i := 0

	for ; i+1 < len(s.data); i += 2 {
		value := spec.CharValue(s.data[i])*45 + spec.CharValue(s.data[i+1])
		bitsData.AppendBits(value, 11)
	}

	if i < len(s.data) {
		value := spec.CharValue(s.data[i])
		bitsData.AppendBits(value, 6)
	}
}

func (s *Segment) encodeKanji(bitsData *coding.BitWriter) {
	enc := japanese.ShiftJIS.NewEncoder()
	sjis, _, _ := transform.String(enc, string(s.data))
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

		bitsData.AppendBits(packed, 13)
	}
}

// Encode writes the segment payload bits (mode header + count are written by the caller).
func (s Segment) encode(w *coding.BitWriter) {
	switch s.mode {
	case spec.EncodingModeByte:
		s.encodeBytes(w)
	case spec.EncodingModeNumeric:
		s.encodeNumeric(w)
	case spec.EncodingModeAlphanumeric:
		s.encodeAlphanumeric(w)
	case spec.EncodingModeKanji:
		s.encodeKanji(w)
	}
}

func detectVersion(segments []Segment, ec QrCorrectionLevel, isECI bool) (int, error) {
	for version := 1; version <= spec.MaxVersion(); version++ {
		totalBits := 0
		if isECI {
			// Header bits
			totalBits += 12
		}

		for _, seg := range segments {
			totalBits += 4 + spec.CharCountBits(seg.mode, spec.VersionRangeFor(version))
			totalBits += seg.payloadBits()
		}

		totalCapacity := spec.DataCodewords(version, ec.level) * 8

		if totalBits <= totalCapacity {
			return version, nil
		}
	}

	return 0, spec.ErrDataTooLong
}

// ------------- DP -------------

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
		if strings.ContainsRune(spec.ALPHA_NUMERIC_CHARSET, r) {
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

func headerCost(mode EncodingMode, vr spec.VersionRange) int {
	return (4 + spec.CharCountBits(mode, vr)) * 6
}

func segmentizeOptimal(data []byte, vr spec.VersionRange) ([]Segment, int) {
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
				mode: modePerChar[start],
				data: []byte(chunk),
			})
			start = i
		}
	}

	return segments
}

func segmentsNeedsECI(segs []Segment) bool {
	for _, seg := range segs {
		if seg.mode != EncodingModeByte {
			continue
		}

		if spec.HasNonASCII(seg.data) {
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

func (b *QrBuilder) segmentize() ([]Segment, bool, error) {
	ranges := []spec.VersionRange{spec.VersionRangeSmall, spec.VersionRangeMedium, spec.VersionRangeLarge}
	for _, vr := range ranges {
		segs, sixths := segmentizeOptimal(b.data, vr)
		totalBits := (sixths + 5) / 6 // costs are in sixth-bits; ceil to bits

		needsECI := b.textECIPolicy != QrTextECIPolicyDisabled && segmentsNeedsECI(segs)
		if needsECI {
			totalBits += 12
		}

		if totalBits <= spec.DataCodewords(spec.MaxVersionForRange(vr), b.errorCorrectionLevel.level)*8 {
			return segs, needsECI, nil
		}
	}
	return nil, false, spec.ErrDataTooLong
}
