package qr

type QrCorrectionLevel struct {
	level int
	value int
}

var (
	QrCorrectionLevelLow      = QrCorrectionLevel{level: 0, value: 0b01}
	QrCorrectionLevelMedium   = QrCorrectionLevel{level: 1, value: 0b00}
	QrCorrectionLevelQuartile = QrCorrectionLevel{level: 2, value: 0b11}
	QrCorrectionLevelHigh     = QrCorrectionLevel{level: 3, value: 0b10}
)

type EncodingMode int

const (
	EncodingModeNumeric EncodingMode = 1 << iota
	EncodingModeAlphanumeric
	EncodingModeByte
	EncodingModeKanji
)

type QrColor int

const (
	QrWhite QrColor = iota
	QrBlack
)

type Segment struct {
	Mode EncodingMode
	Data []byte
}

type QrDataKind int

const (
	QrDataKindText QrDataKind = iota
	QrDataKindBinary
)

type QrTextECIPolicy int

const (
	QrTextECIPolicyAuto QrTextECIPolicy = iota
	QrTextECIPolicyDisabled
)
