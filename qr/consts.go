package qr

import (
	"math"
	"strings"
)

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

type QrSegment struct {
	Mode QrEncodingMode
	Data []byte
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

// Taken from https://github.com/rsc/qr/blob/master/coding/gen.go
var eccTable = [41][4][2]int{
	{{0, 0}, {0, 0}, {0, 0}, {0, 0}},
	{{1, 0}, {1, 0}, {1, 0}, {1, 0}}, // 1
	{{1, 0}, {1, 0}, {1, 0}, {1, 0}},
	{{1, 0}, {1, 0}, {2, 0}, {2, 0}},
	{{1, 0}, {2, 0}, {2, 0}, {4, 0}},
	{{1, 0}, {2, 0}, {2, 2}, {2, 2}}, // 5
	{{2, 0}, {4, 0}, {4, 0}, {4, 0}},
	{{2, 0}, {4, 0}, {2, 4}, {4, 1}},
	{{2, 0}, {2, 2}, {4, 2}, {4, 2}},
	{{2, 0}, {3, 2}, {4, 4}, {4, 4}},
	{{2, 2}, {4, 1}, {6, 2}, {6, 2}}, //10
	{{4, 0}, {1, 4}, {4, 4}, {3, 8}},
	{{2, 2}, {6, 2}, {4, 6}, {7, 4}},
	{{4, 0}, {8, 1}, {8, 4}, {12, 4}},
	{{3, 1}, {4, 5}, {11, 5}, {11, 5}},
	{{5, 1}, {5, 5}, {5, 7}, {11, 7}}, //15
	{{5, 1}, {7, 3}, {15, 2}, {3, 13}},
	{{1, 5}, {10, 1}, {1, 15}, {2, 17}},
	{{5, 1}, {9, 4}, {17, 1}, {2, 19}},
	{{3, 4}, {3, 11}, {17, 4}, {9, 16}},
	{{3, 5}, {3, 13}, {15, 5}, {15, 10}}, //20
	{{4, 4}, {17, 0}, {17, 6}, {19, 6}},
	{{2, 7}, {17, 0}, {7, 16}, {34, 0}},
	{{4, 5}, {4, 14}, {11, 14}, {16, 14}},
	{{6, 4}, {6, 14}, {11, 16}, {30, 2}},
	{{8, 4}, {8, 13}, {7, 22}, {22, 13}}, //25
	{{10, 2}, {19, 4}, {28, 6}, {33, 4}},
	{{8, 4}, {22, 3}, {8, 26}, {12, 28}},
	{{3, 10}, {3, 23}, {4, 31}, {11, 31}},
	{{7, 7}, {21, 7}, {1, 37}, {19, 26}},
	{{5, 10}, {19, 10}, {15, 25}, {23, 25}}, //30
	{{13, 3}, {2, 29}, {42, 1}, {23, 28}},
	{{17, 0}, {10, 23}, {10, 35}, {19, 35}},
	{{17, 1}, {14, 21}, {29, 19}, {11, 46}},
	{{13, 6}, {14, 23}, {44, 7}, {59, 1}},
	{{12, 7}, {12, 26}, {39, 14}, {22, 41}}, //35
	{{6, 14}, {6, 34}, {46, 10}, {2, 64}},
	{{17, 4}, {29, 14}, {49, 10}, {24, 46}},
	{{4, 18}, {13, 32}, {48, 14}, {42, 32}},
	{{20, 4}, {40, 7}, {43, 22}, {10, 67}},
	{{19, 6}, {18, 31}, {34, 34}, {20, 61}}, //40
}

const ALPHA_NUMERIC_CHARSET = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func charValue(c byte) int {
	return strings.IndexByte(ALPHA_NUMERIC_CHARSET, c)
}

func alignmentCoords(version int) []int {
	if version == 1 {
		return nil
	}
	first := 6
	last := (17 + 4*version) - 7
	count := int(math.Ceil(float64(last-first)/28)) + 1

	result := make([]int, count)
	result[0] = first
	result[count-1] = last
	if count > 2 {
		step := int(math.Ceil(float64(last-first) / float64(count-1)))
		if step%2 == 1 { // corrección de paridad
			frac := float64(last-first) / float64(count-1)
			r := math.Floor(frac)
			if frac-r >= 0.5 {
				r = math.Ceil(frac)
			}
			if int(r)%2 == 0 {
				step--
			} else {
				step++
			}
		}
		for i := 1; i < count-1; i++ {
			result[i] = last - step*(count-1-i) // se llena desde el final
		}
	}
	return result
}

func measurePixelAndQuietZone(width, height, version int) (int, int, int) {
	modules := capacityTable[version].modules
	pixelSize := min(width, height) / (modules + 8)
	quietZoneX := (width - modules*pixelSize) / 2
	quietZoneY := (height - modules*pixelSize) / 2

	return pixelSize, quietZoneX, quietZoneY
}

func needsECI(data []byte) bool {
	for _, b := range data {
		if b >= 0x80 {
			return true
		}
	}
	return false
}
