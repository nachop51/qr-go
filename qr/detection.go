package qr

import (
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
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

func canUseKanji(data []byte) bool {
	enc := japanese.ShiftJIS.NewEncoder()
	sjis, _, err := transform.String(enc, string(data))
	if err != nil {
		return false
	}
	bytes := []byte(sjis)
	// Cada kanji ocupa exactamente 2 bytes en SJIS, así que la cantidad debe ser par
	if len(bytes)%2 != 0 {
		return false
	}
	for i := 0; i < len(bytes); i += 2 {
		v := uint16(bytes[i])<<8 | uint16(bytes[i+1])
		if !((v >= 0x8140 && v <= 0x9FFC) || (v >= 0xE040 && v <= 0xEBBF)) {
			return false
		}
	}
	return true
}

func (b *QrBuilder) detectEncodingMode() QrEncodingMode {
	isNumeric := true
	isAlphanumeric := true

	if canUseKanji(b.data) {
		return QrEncodingModeKanji
	}

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

	return QrEncodingModeByte
}

func (b *QrBuilder) detectVersion(encodingMode QrEncodingMode, isECI bool) (int, error) {
	var payloadBits int

	switch encodingMode {
	case QrEncodingModeNumeric:
		payloadBits = (len(b.data) / 3) * 10
		if len(b.data)%3 == 2 {
			payloadBits += 7
		}
		if len(b.data)%3 == 1 {
			payloadBits += 4
		}
	case QrEncodingModeAlphanumeric:
		payloadBits = (len(b.data)/2)*11 + (len(b.data)%2)*6
	case QrEncodingModeKanji:
		payloadBits = utf8.RuneCountInString(string(b.data)) * 13
	default:
		payloadBits = len(b.data) * 8
	}

	for version, levelCapacity := range capacityTable {
		headerBits := 4 + getBitLengthIndicator(version, encodingMode)
		if isECI {
			headerBits += 12
		}
		totalBits := headerBits + payloadBits

		totalCapacity := (levelCapacity.bytes - levelCapacity.ec[b.errorCorrectionLevel.level]) * 8

		if totalBits <= totalCapacity {
			return version, nil
		}
	}

	return 0, ErrDataTooLong
}
