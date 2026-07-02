package coding

type BitWriter struct {
	data   []byte
	bitPos int
}

func (w *BitWriter) AppendBits(value, count int) {
	for i := count - 1; i >= 0; i-- {
		if w.bitPos%8 == 0 {
			w.data = append(w.data, 0)
		}
		bit := byte((value >> i) & 1)
		w.data[len(w.data)-1] |= bit << (7 - (w.bitPos % 8))
		w.bitPos++
	}
}
func (w *BitWriter) Data() []byte { return w.data }
func (w *BitWriter) BitLen() int  { return w.bitPos }

type BitReader struct {
	data []byte
	pos  int
}

func NewBitReader(data []byte) *BitReader { return &BitReader{data: data} }

func (r *BitReader) Pos() int     { return r.pos }
func (r *BitReader) Data() []byte { return r.data }

// Next returns next bit (0/1). Past end: returns 0, does NOT advance pos.
func (r *BitReader) Next() int {
	byteIdx := r.pos / 8
	if byteIdx >= len(r.data) {
		return 0
	}
	bit := int((r.data[byteIdx] >> (7 - r.pos%8)) & 1)
	r.pos++
	return bit
}
