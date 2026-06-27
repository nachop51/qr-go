package qr

import "rsc.io/qr/gf256"

type BlockRecipe struct {
	EcPerBlock int

	Group1Blocks  int
	Group1DataLen int

	Group2Blocks  int
	Group2DataLen int
}

func (q *QrCode) blockRecipe() BlockRecipe {
	level := q.ErrorCorrectionLevel.level

	g1 := eccTable[q.Version][level][0]
	g2 := eccTable[q.Version][level][1]
	totalBlocks := g1 + g2

	totalEC := capacityTable[q.Version].ec[level]
	totalData := capacityTable[q.Version].bytes - totalEC

	d1 := totalData / totalBlocks
	return BlockRecipe{
		EcPerBlock:    totalEC / totalBlocks,
		Group1Blocks:  g1,
		Group1DataLen: d1,
		Group2Blocks:  g2,
		Group2DataLen: d1 + 1,
	}
}

func (q *QrCode) splitIntoBlocks(data []byte) [][]byte {
	recipe := q.blockRecipe()
	blocks := [][]byte{}
	offset := 0

	for i := 0; i < recipe.Group1Blocks; i++ {
		blocks = append(blocks, data[offset:offset+recipe.Group1DataLen])
		offset += recipe.Group1DataLen
	}
	for i := 0; i < recipe.Group2Blocks; i++ {
		blocks = append(blocks, data[offset:offset+recipe.Group2DataLen])
		offset += recipe.Group2DataLen
	}

	return blocks
}

func (q *QrCode) errorCorrectionPerBlock(blocks [][]byte) [][]byte {
	recipe := q.blockRecipe()
	field := gf256.NewField(0x11d, 0x02)
	enc := gf256.NewRSEncoder(field, recipe.EcPerBlock)

	ecs := make([][]byte, len(blocks))

	for i, b := range blocks {
		ecs[i] = make([]byte, recipe.EcPerBlock)
		enc.ECC(b, ecs[i])
	}

	return ecs
}

func interleave(dataBlocks, ecBlocks [][]byte) []byte {
	result := []byte{}

	maxDataLen := 0
	for _, block := range dataBlocks {
		if len(block) > maxDataLen {
			maxDataLen = len(block)
		}
	}

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
