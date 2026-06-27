package qr

type VersionRange int

const (
	VersionRangeSmall VersionRange = iota
	VersionRangeMedium
	VersionRangeLarge
)

func fitsInRange(totalBits int, vr VersionRange, level int) bool {
	var maxV int

	switch vr {
	case VersionRangeSmall:
		maxV = 9
	case VersionRangeMedium:
		maxV = 26
	case VersionRangeLarge:
		maxV = 40
	}

	maxCapacity := (capacityTable[maxV].bytes - capacityTable[maxV].ec[level]) * 8

	return totalBits <= maxCapacity
}

func getBitLengthIndicator(version int, mode EncodingMode) int {
	group := 0
	if version <= 9 {
		group = 0
	} else if version >= 10 && version <= 26 {
		group = 1
	} else if version >= 27 && version <= 40 {
		group = 2
	}

	switch mode {
	case EncodingModeNumeric:
		return []int{10, 12, 14}[group]
	case EncodingModeAlphanumeric:
		return []int{9, 11, 13}[group]
	case EncodingModeKanji:
		return []int{8, 10, 12}[group]
	case EncodingModeByte:
		return []int{8, 16, 16}[group]
	default:
		return 0
	}

}

func (b *QrBuilder) detectVersion(segments []Segment, isECI bool) (int, error) {
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
