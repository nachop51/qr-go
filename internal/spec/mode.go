package spec

// EncodingMode identifies a QR segment encoding mode.
// The values are the 4-bit QR mode indicators (0001, 0010, 0100, 1000).
type EncodingMode int

const (
	EncodingModeNumeric EncodingMode = 1 << iota
	EncodingModeAlphanumeric
	EncodingModeByte
	EncodingModeKanji
)

// VersionRange groups QR versions by character-count indicator sizes.
type VersionRange int

const (
	VersionRangeSmall  VersionRange = iota // versions 1-9
	VersionRangeMedium                     // versions 10-26
	VersionRangeLarge                      // versions 27-40
)

// VersionRangeFor returns the range a version belongs to.
func VersionRangeFor(version int) VersionRange {
	switch {
	case version <= 9:
		return VersionRangeSmall
	case version <= 26:
		return VersionRangeMedium
	default:
		return VersionRangeLarge
	}
}

func MaxVersionForRange(vr VersionRange) int {
	switch vr {
	case VersionRangeSmall:
		return 9
	case VersionRangeMedium:
		return 26
	default:
		return 40
	}
}

// CharCountBits returns the character-count indicator length in bits
// for the given mode and version range.
func CharCountBits(mode EncodingMode, vr VersionRange) int {
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
