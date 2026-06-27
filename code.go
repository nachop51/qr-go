package qr

import (
	"fmt"
	"image"
	"image/color"
)

type QrCode struct {
	Segments             []Segment
	ErrorCorrectionLevel QrCorrectionLevel
	Version              int
	Mask                 int
	Filename             string
	isECI                bool
	img                  *image.RGBA
	pixelSize            int
	quietZoneX           int
	quietZoneY           int
	blackColor           color.Color
	whiteColor           color.Color
	points               [][]QrPoint
}

func (q *QrCode) Debug() {
	for y := range q.points {
		for x := range q.points[y] {
			if q.points[y][x].col == QrBlack {
				if q.points[y][x].protected {
					print("X")
				} else {
					print("x")
				}
			} else {
				if q.points[y][x].protected {
					print(",")
				} else {
					print(".")
				}
			}
		}
		println()
	}
}

func (q *QrCode) debugMasks() {
	originalFile := q.Filename
	originalMask := q.Mask

	for mask := range 8 {
		q.Filename = fmt.Sprintf("mask%d.png", mask)
		q.applyMask(mask)
		q.placeMetadata(mask)
		q.Draw()
		q.Save()
		q.applyMask(mask)
	}
	q.Filename = originalFile
	q.Mask = originalMask
}
