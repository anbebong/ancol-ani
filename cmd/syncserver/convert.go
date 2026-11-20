package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
)

// imageToGIF wraps img as a single-frame GIF. The frame is quantized to the
// stdlib Plan9 palette and placed at origin (0,0).
func imageToGIF(img image.Image) *gif.GIF {
	frame := quantizeToPaletted(img)
	b := frame.Bounds()
	return &gif.GIF{
		Image:  []*image.Paletted{frame},
		Delay:  []int{0},
		Config: image.Config{Width: b.Dx(), Height: b.Dy()},
	}
}

// imagesToGIF builds an animated GIF with one frame per image, in the given
// order. Every image is resized (nearest-neighbor) to the first image's bounds
// before quantizing, so mismatched input sizes do not break the encoder. Each
// frame lasts delayMs milliseconds, stored in centiseconds with a minimum of 2.
func imagesToGIF(images []image.Image, delayMs int) (*gif.GIF, error) {
	if len(images) == 0 || len(images) > maxImagesCount {
		return nil, fmt.Errorf("so anh khong hop le: %d (chi nhan tu 1 den %d)", len(images), maxImagesCount)
	}

	first := images[0].Bounds()
	w, h := first.Dx(), first.Dy()

	delay := delayMs / 10
	if delay < 2 {
		delay = 2
	}

	out := &gif.GIF{
		Image:  make([]*image.Paletted, len(images)),
		Delay:  make([]int, len(images)),
		Config: image.Config{Width: w, Height: h},
	}
	for i, img := range images {
		out.Image[i] = quantizeToPaletted(resizeNearest(img, w, h))
		out.Delay[i] = delay
	}
	return out, nil
}

// quantizeToPaletted copies src onto a fresh *image.Paletted at origin (0,0)
// using the stdlib Plan9 palette.
func quantizeToPaletted(src image.Image) *image.Paletted {
	b := src.Bounds()
	dst := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), palette.Plan9)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// resizeNearest returns src nearest-neighbor scaled to w x h, anchored at
// origin (0,0).
func resizeNearest(src image.Image, w, h int) *image.RGBA {
	sb := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y*sb.Dy()/h
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*sb.Dx()/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
