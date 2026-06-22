package qrimage

import (
	"strings"
)

type QrCorrectionLevel struct {
	level int
	value int
}

var (
	QrCorrectionLevelLow      = QrCorrectionLevel{level: 0, value: 0b01}
	QrCorrectionLevelMedium   = QrCorrectionLevel{level: 1, value: 0b00}
	QrCorrectionLevelQuartile = QrCorrectionLevel{level: 2, value: 0b10}
	QrCorrectionLevelHigh     = QrCorrectionLevel{level: 3, value: 0b11}
)

type QrEncodingMode int

const (
	QrEncodingModeNumeric QrEncodingMode = 1 << iota
	QrEncodingModeAlphanumeric
	QrEncodingModeByte
	QrEncodingModeKanji
)

type QrColor int

const (
	QrWhite QrColor = iota
	QrBlack
)

type QrPoint struct {
	x         int
	y         int
	col       QrColor
	protected bool
	drawn     bool
}

// tables from qrencode-3.1.1/qrspec.c

var capacityTable = [41]struct {
	modules   int
	bytes     int
	remainder int
	// Error correction codewords per block for each error correction level (L, M, Q, H)
	ec [4]int
}{
	{0, 0, 0, [4]int{0, 0, 0, 0}},
	{21, 26, 0, [4]int{7, 10, 13, 17}}, // 1
	{25, 44, 7, [4]int{10, 16, 22, 28}},
	{29, 70, 7, [4]int{15, 26, 36, 44}},
	{33, 100, 7, [4]int{20, 36, 52, 64}},
	{37, 134, 7, [4]int{26, 48, 72, 88}}, // 5
	{41, 172, 7, [4]int{36, 64, 96, 112}},
	{45, 196, 0, [4]int{40, 72, 108, 130}},
	{49, 242, 0, [4]int{48, 88, 132, 156}},
	{53, 292, 0, [4]int{60, 110, 160, 192}},
	{57, 346, 0, [4]int{72, 130, 192, 224}}, //10
	{61, 404, 0, [4]int{80, 150, 224, 264}},
	{65, 466, 0, [4]int{96, 176, 260, 308}},
	{69, 532, 0, [4]int{104, 198, 288, 352}},
	{73, 581, 3, [4]int{120, 216, 320, 384}},
	{77, 655, 3, [4]int{132, 240, 360, 432}}, //15
	{81, 733, 3, [4]int{144, 280, 408, 480}},
	{85, 815, 3, [4]int{168, 308, 448, 532}},
	{89, 901, 3, [4]int{180, 338, 504, 588}},
	{93, 991, 3, [4]int{196, 364, 546, 650}},
	{97, 1085, 3, [4]int{224, 416, 600, 700}}, //20
	{101, 1156, 4, [4]int{224, 442, 644, 750}},
	{105, 1258, 4, [4]int{252, 476, 690, 816}},
	{109, 1364, 4, [4]int{270, 504, 750, 900}},
	{113, 1474, 4, [4]int{300, 560, 810, 960}},
	{117, 1588, 4, [4]int{312, 588, 870, 1050}}, //25
	{121, 1706, 4, [4]int{336, 644, 952, 1110}},
	{125, 1828, 4, [4]int{360, 700, 1020, 1200}},
	{129, 1921, 3, [4]int{390, 728, 1050, 1260}},
	{133, 2051, 3, [4]int{420, 784, 1140, 1350}},
	{137, 2185, 3, [4]int{450, 812, 1200, 1440}}, //30
	{141, 2323, 3, [4]int{480, 868, 1290, 1530}},
	{145, 2465, 3, [4]int{510, 924, 1350, 1620}},
	{149, 2611, 3, [4]int{540, 980, 1440, 1710}},
	{153, 2761, 3, [4]int{570, 1036, 1530, 1800}},
	{157, 2876, 0, [4]int{570, 1064, 1590, 1890}}, //35
	{161, 3034, 0, [4]int{600, 1120, 1680, 1980}},
	{165, 3196, 0, [4]int{630, 1204, 1770, 2100}},
	{169, 3362, 0, [4]int{660, 1260, 1860, 2220}},
	{173, 3532, 0, [4]int{720, 1316, 1950, 2310}},
	{177, 3706, 0, [4]int{750, 1372, 2040, 2430}}, //40
}

const ALPHA_NUMERIC_CHARSET = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func charValue(c byte) int {
	return strings.IndexByte(ALPHA_NUMERIC_CHARSET, c)
}

type QrMask int

const (
	QrMask0 QrMask = iota
	QrMask1
	QrMask2
	QrMask3
	QrMask4
	QrMask5
	QrMask6
	QrMask7
)
