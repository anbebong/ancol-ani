package main

import (
	"fmt"
	"image"
	"image/gif"
)

// resizeGIF returns a copy of g with every frame scaled to width x height
// using nearest-neighbor sampling. Each frame keeps its own palette, delay and
// disposal method.
func resizeGIF(g *gif.GIF, width, height int) (*gif.GIF, error) {
	if width <= 0 || height <= 0 || width > maxGifSide || height > maxGifSide {
		return nil, fmt.Errorf("kich thuoc khong hop le: %dx%d (toi da %dx%d)", width, height, maxGifSide, maxGifSide)
	}

	out := &gif.GIF{
		Image:           make([]*image.Paletted, len(g.Image)),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	out.Config.Width = width
	out.Config.Height = height

	for i, frame := range g.Image {
		out.Image[i] = scalePaletted(frame, width, height)
	}
	return out, nil
}

// scalePaletted nearest-neighbor scales src to width x height. Pixels are
// copied as palette indices, so the result shares src's palette exactly.
func scalePaletted(src *image.Paletted, width, height int) *image.Paletted {
	srcB := src.Bounds()
	dst := image.NewPaletted(image.Rect(0, 0, width, height), src.Palette)
	for y := 0; y < height; y++ {
		sy := srcB.Min.Y + y*srcB.Dy()/height
		for x := 0; x < width; x++ {
			sx := srcB.Min.X + x*srcB.Dx()/width
			dst.SetColorIndex(x, y, src.ColorIndexAt(sx, sy))
		}
	}
	return dst
}
