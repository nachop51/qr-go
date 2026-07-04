package qr

import (
	"github.com/nachop51/qr-go/internal/coding"
	"github.com/nachop51/qr-go/internal/spec"
)

func addTerminatorAndPadding(bitsData *coding.BitWriter, version int, ec CorrectionLevel) {
	dataBytes := spec.DataCodewords(version, ec.level)
	capacityBits := dataBytes * 8

	terminatorBits := min(4, capacityBits-bitsData.BitLen())

	bitsData.AppendBits(0, terminatorBits)
	// Bit align
	bitsData.AppendBits(0, (8-(bitsData.BitLen()%8))%8)

	pad := []byte{0xEC, 0x11}
	for i := 0; bitsData.BitLen() < capacityBits; i++ {
		bitsData.AppendBits(int(pad[i%2]), 8)
	}
}

type BlockRecipe struct {
	EcPerBlock int

	Group1Blocks  int
	Group1DataLen int

	Group2Blocks  int
	Group2DataLen int
}

func blockRecipe(version int, ec CorrectionLevel) BlockRecipe {
	g1, g2 := spec.ECBlocks(version, ec.level)
	totalBlocks := g1 + g2

	totalEC := spec.ECCodewords(version, ec.level)
	totalData := spec.DataCodewords(version, ec.level)

	d1 := totalData / totalBlocks
	return BlockRecipe{
		EcPerBlock:    totalEC / totalBlocks,
		Group1Blocks:  g1,
		Group1DataLen: d1,
		Group2Blocks:  g2,
		Group2DataLen: d1 + 1,
	}
}

func splitIntoBlocks(data []byte, version int, ec CorrectionLevel) [][]byte {
	recipe := blockRecipe(version, ec)
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

func errorCorrectionPerBlock(blocks [][]byte, version int, ec CorrectionLevel) [][]byte {
	recipe := blockRecipe(version, ec)
	enc := coding.NewRSEncoder(recipe.EcPerBlock)

	ecs := make([][]byte, len(blocks))

	for i, b := range blocks {
		ecs[i] = coding.ReedSolomon(b, recipe.EcPerBlock, enc)
	}

	return ecs
}

func buildCodewords(segments []Segment, version int, ec CorrectionLevel, isECI bool) []byte {
	var bitsData coding.BitWriter

	if isECI {
		bitsData.AppendBits(0b0111, 4)
		bitsData.AppendBits(26, 8)
	}

	for _, seg := range segments {
		bitsData.AppendBits(int(seg.mode), 4)
		bitsData.AppendBits(seg.dataLength(), spec.CharCountBits(seg.mode, spec.VersionRangeFor(version)))

		seg.encode(&bitsData)
	}

	addTerminatorAndPadding(&bitsData, version, ec)

	blocks := splitIntoBlocks(bitsData.Data(), version, ec)
	ecs := errorCorrectionPerBlock(blocks, version, ec)

	return coding.Interleave(blocks, ecs)
}

func encodeFormat(d uint16) uint16 {
	const g = 0x537 // 10100110111, generador grado 10

	v := d << 10
	// limpio los 5 bits de datos, posiciones 14..10
	for i := 14; i >= 10; i-- {
		if (v>>uint(i))&1 == 1 {
			v ^= g << uint(i-10)
		}
	}
	// ahora v tiene solo el resto (10 bits bajos)
	code := (d << 10) | v
	return code ^ 0x5412
}

func encodeVersion(version uint16) uint32 {
	const g = 0x1F25 // generador BCH(18,6), grado 12

	v := uint32(version) << 12 // dejar lugar para 12 bits de EC
	// limpiar los 6 bits de datos, posiciones 17..12
	for i := 17; i >= 12; i-- {
		if (v>>uint(i))&1 == 1 {
			v ^= g << uint(i-12)
		}
	}
	// ahora v tiene el resto en los 12 bits bajos
	return (uint32(version) << 12) | v
	// sin XOR final
}
