package qrimage

import (
	"errors"
	"strings"
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

func (b *QrImageBuilder) detectEncodingMode() QrEncodingMode {
	isNumeric := true
	isAlphanumeric := true
	// TODO: Implement Kanji mode detection
	// isKanji := true

	for _, c := range b.data {
		if c < '0' || c > '9' {
			isNumeric = false
		}
		if !strings.Contains(ALPHA_NUMERIC_CHARSET, string(c)) {
			isAlphanumeric = false
		}
	}

	if isNumeric {
		return QrEncodingModeNumeric
	}

	if isAlphanumeric {
		return QrEncodingModeAlphanumeric
	}

	// Kanji mode is not implemented, default to byte mode

	return QrEncodingModeByte
}

func (b *QrImageBuilder) detectVersion(encodingMode QrEncodingMode) (int, error) {
	contentLength := len(b.data) * 8

	switch encodingMode {
	case QrEncodingModeNumeric:
		contentLength = (len(b.data) / 3) * 10
		if len(b.data)%3 == 2 {
			contentLength += 7
		}
		if len(b.data)%3 == 1 {
			contentLength += 4
		}
	case QrEncodingModeAlphanumeric:
		contentLength = (len(b.data)/2)*11 + (len(b.data) % 2 * 6)
	}

	for version, levelCapacity := range capacityTable {
		bitLength := getBitLengthIndicator(version, encodingMode)
		totalBits := 4 + bitLength + contentLength

		totalCapacity := (levelCapacity.bytes - levelCapacity.ec[b.errorCorrectionLevel.level]) * 8

		if totalBits <= totalCapacity {
			return version, nil
		}
	}

	return 0, ErrDataTooLong
}
