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

// reverseGIF returns a copy of g with its frames played in reverse order. Image,
// Delay and Disposal are reordered together so each frame keeps its own timing
// and disposal method. Frame pixels are not touched. A GIF with one frame comes
// back unchanged in content. Delay and Disposal are reversed only when they
// have one entry per frame; otherwise they are copied as they are.
func reverseGIF(g *gif.GIF) *gif.GIF {
	n := len(g.Image)
	out := &gif.GIF{
		Image:           make([]*image.Paletted, n),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	for i, frame := range g.Image {
		out.Image[n-1-i] = frame
	}
	if len(g.Delay) == n {
		for i, d := range g.Delay {
			out.Delay[n-1-i] = d
		}
	}
	if len(g.Disposal) == n {
		for i, d := range g.Disposal {
			out.Disposal[n-1-i] = d
		}
	}
	return out
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

// rotateGIF returns a copy of g with every frame flipped and then rotated
// clockwise by angle degrees. angle must be 0, 90, 180 or 270. flip must be
// "none", "h" (mirror left-right) or "v" (mirror top-bottom); the flip is
// applied before the rotation. Each frame keeps its own palette, delay and
// disposal method.
func rotateGIF(g *gif.GIF, angle int, flip string) (*gif.GIF, error) {
	switch angle {
	case 0, 90, 180, 270:
	default:
		return nil, fmt.Errorf("goc xoay khong hop le: %d (chi nhan 0, 90, 180, 270)", angle)
	}
	switch flip {
	case "none", "h", "v":
	default:
		return nil, fmt.Errorf("kieu lat khong hop le: %q (chi nhan none, h, v)", flip)
	}

	out := &gif.GIF{
		Image:           make([]*image.Paletted, len(g.Image)),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	if angle == 90 || angle == 270 {
		out.Config.Width = g.Config.Height
		out.Config.Height = g.Config.Width
	}

	for i, frame := range g.Image {
		out.Image[i] = rotatePaletted(frame, angle, flip)
	}
	return out, nil
}

// rotatePaletted flips src according to flip, then rotates it clockwise by
// angle. Pixels are copied as palette indices, so the result shares src's
// palette exactly. Each destination pixel is computed by inverting the rotation
// and then the flip, so every output pixel reads exactly one source pixel.
func rotatePaletted(src *image.Paletted, angle int, flip string) *image.Paletted {
	srcB := src.Bounds()
	w, h := srcB.Dx(), srcB.Dy()
	dw, dh := w, h
	if angle == 90 || angle == 270 {
		dw, dh = h, w
	}
	dst := image.NewPaletted(image.Rect(0, 0, dw, dh), src.Palette)
	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			// Undo the rotation: destination (dx,dy) -> flipped-image (fx,fy).
			var fx, fy int
			switch angle {
			case 0:
				fx, fy = dx, dy
			case 90:
				fx, fy = dy, h-1-dx
			case 180:
				fx, fy = w-1-dx, h-1-dy
			case 270:
				fx, fy = w-1-dy, dx
			}
			// Undo the flip (a flip is its own inverse).
			sx, sy := fx, fy
			switch flip {
			case "h":
				sx = w - 1 - fx
			case "v":
				sy = h - 1 - fy
			}
			dst.SetColorIndex(dx, dy, src.ColorIndexAt(srcB.Min.X+sx, srcB.Min.Y+sy))
		}
	}
	return dst
}
