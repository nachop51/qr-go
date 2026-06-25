package qr

import (
	"errors"
)

var (
	ErrDataTooLong = errors.New("Data too long for the selected version and error correction level")
)

func getBitLengthIndicator(version int, mode QrEncodingMode) int {
	group := 0
	if version <= 9 {
		group = 0
	} else if version >= 10 && version <= 26 {
		group = 1
	} else if version >= 27 && version <= 40 {
		group = 2
	}

	switch mode {
	case QrEncodingModeNumeric:
		return []int{10, 12, 14}[group]
	case QrEncodingModeAlphanumeric:
		return []int{9, 11, 13}[group]
	case QrEncodingModeKanji:
		return []int{8, 10, 12}[group]
	case QrEncodingModeByte:
		return []int{8, 16, 16}[group]
	default:
		return 0
	}

}

func (b *QrBuilder) detectVersion(segments []QrSegment, isECI bool) (int, error) {
	for version := 1; version < len(capacityTable); version++ {
		levelCapacity := capacityTable[version]
		totalBits := 0
		if isECI {
			// Header bits
			totalBits += 12
		}

		for _, seg := range segments {
			totalBits += 4 + getBitLengthIndicator(version, seg.Mode)
			totalBits += seg.payloadBits()
		}

		totalCapacity := (levelCapacity.bytes - levelCapacity.ec[b.errorCorrectionLevel.level]) * 8

		if totalBits <= totalCapacity {
			return version, nil
		}
	}

	return 0, ErrDataTooLong
}
