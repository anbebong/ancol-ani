package main

import (
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// makeTestGIF returns a GIF with the given number of solid-color frames of
// size w x h, each frame 10 centiseconds long.
func makeTestGIF(t *testing.T, frames int, w, h int) *gif.GIF {
	t.Helper()
	pal := color.Palette{color.Black, color.White}
	anim := &gif.GIF{Config: image.Config{Width: w, Height: h, ColorModel: pal}}
	for i := 0; i < frames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for j := range img.Pix {
			img.Pix[j] = 1
		}
		anim.Image = append(anim.Image, img)
		anim.Delay = append(anim.Delay, 10)
	}
	return anim
}
