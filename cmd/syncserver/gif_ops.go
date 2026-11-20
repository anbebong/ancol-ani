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

// cropGIF returns a copy of g with every frame cropped to the rectangle
// (x, y, w, h) of the logical screen. Each output frame is re-materialized at
// origin (0,0) so the encoder sees the same layout as any other GIF. Each frame
// keeps its own palette, delay and disposal method.
func cropGIF(g *gif.GIF, x, y, w, h int) (*gif.GIF, error) {
	srcB := g.Image[0].Bounds()
	if w <= 0 || h <= 0 || x < 0 || y < 0 || x+w > srcB.Dx() || y+h > srcB.Dy() {
		return nil, fmt.Errorf("vung cat khong hop le: (%d,%d) %dx%d vuot qua nguon %dx%d", x, y, w, h, srcB.Dx(), srcB.Dy())
	}

	rect := image.Rect(srcB.Min.X+x, srcB.Min.Y+y, srcB.Min.X+x+w, srcB.Min.Y+y+h)
	out := &gif.GIF{
		Image:           make([]*image.Paletted, len(g.Image)),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	out.Config.Width = w
	out.Config.Height = h

	for i, frame := range g.Image {
		sub := frame.SubImage(rect).(*image.Paletted)
		dst := image.NewPaletted(image.Rect(0, 0, w, h), sub.Palette)
		for py := 0; py < h; py++ {
			for px := 0; px < w; px++ {
				dst.SetColorIndex(px, py, sub.ColorIndexAt(sub.Rect.Min.X+px, sub.Rect.Min.Y+py))
			}
		}
		out.Image[i] = dst
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
