package coding

import "rsc.io/qr/gf256"

func NewRSEncoder(ecCount int) *gf256.RSEncoder {
	field := gf256.NewField(0x11d, 0x02)
	enc := gf256.NewRSEncoder(field, ecCount)

	return enc
}

func ReedSolomon(data []byte, ecCount int, enc *gf256.RSEncoder) []byte {
	out := make([]byte, ecCount)
	enc.ECC(data, out)
	return out
}

func Interleave(dataBlocks, ecBlocks [][]byte) []byte {
	total := 0
	maxDataLen := 0
	for _, block := range dataBlocks {
		total += len(block)
		if len(block) > maxDataLen {
			maxDataLen = len(block)
		}
	}
	for _, block := range ecBlocks {
		total += len(block)
	}
	result := make([]byte, 0, total)

	for col := 0; col < maxDataLen; col++ {
		for _, block := range dataBlocks {
			if col < len(block) {
				result = append(result, block[col])
			}
		}
	}

	ecLen := len(ecBlocks[0])
	for col := range ecLen {
		for _, block := range ecBlocks {
			result = append(result, block[col])
		}
	}

	return result
}
