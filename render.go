package qr

import (
	"image/png"
	"os"
)

func (q *QrCode) Draw() {
	for _, points := range q.points {
		for _, point := range points {
			if !point.drawn {
				q.drawPoint(point)
				point.drawn = true
			}
		}
	}
}

func (q *QrCode) Save() error {
	f, err := os.Create(q.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	err = png.Encode(f, q.img)
	return err
}
