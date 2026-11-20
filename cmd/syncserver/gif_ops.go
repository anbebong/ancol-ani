package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"sort"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
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

// speedGIF returns a copy of g with every frame's delay divided by factor, so
// factor 2 plays twice as fast. factor must be greater than 0. Each new delay is
// rounded to the nearest centisecond and clamped to at least 2 centiseconds, the
// de facto minimum most GIF renderers honor. Pixel data is not touched.
func speedGIF(g *gif.GIF, factor float64) (*gif.GIF, error) {
	if !(factor > 0) {
		return nil, fmt.Errorf("he so toc do phai lon hon 0: %v", factor)
	}

	out := &gif.GIF{
		Image:           append([]*image.Paletted(nil), g.Image...),
		Delay:           make([]int, len(g.Delay)),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	for i, d := range g.Delay {
		nd := int(math.Round(float64(d) / factor))
		if nd < 2 {
			nd = 2
		}
		out.Delay[i] = nd
	}
	return out, nil
}

// optimizeGIF returns a copy of g with every frame's palette reduced to cut
// encoded size. level must be 1 (light), 2 (medium) or 3 (aggressive),
// mapping to a target palette size of 128, 64 or 32 colors respectively.
// Fewer palette entries let the GIF encoder use a narrower LZW code size
// (image/gif picks ceil(log2(len(palette))) bits per pixel code), which is
// what actually shrinks the output; this only pays off for frames whose
// palette already exceeds the target. Each frame's reduced palette is built
// by sorting its existing palette by luma and sampling it at even intervals
// (a simple uniform quantization, not full median-cut), then every pixel is
// re-mapped to its nearest color in that smaller palette via
// image/draw.Draw with draw.Src onto a fresh *image.Paletted. This keeps the
// dependency footprint to the standard library only.
func optimizeGIF(g *gif.GIF, level int) (*gif.GIF, error) {
	var target int
	switch level {
	case 1:
		target = 128
	case 2:
		target = 64
	case 3:
		target = 32
	default:
		return nil, fmt.Errorf("muc do toi uu khong hop le: %d (chi nhan 1, 2, 3)", level)
	}

	out := &gif.GIF{
		Image:           make([]*image.Paletted, len(g.Image)),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}

	for i, frame := range g.Image {
		pal := reducePalette(frame.Palette, target)
		dst := image.NewPaletted(frame.Bounds(), pal)
		draw.Draw(dst, dst.Bounds(), frame, frame.Bounds().Min, draw.Src)
		out.Image[i] = dst
	}
	return out, nil
}

// reducePalette returns a palette of at most target colors derived from src.
// If src already fits within target, it is returned unchanged. Otherwise src
// is sorted by approximate luma and sampled at even intervals across that
// sorted order, giving a deterministic, stdlib-only approximation of palette
// reduction without pulling in a quantization library.
func reducePalette(src color.Palette, target int) color.Palette {
	if len(src) <= target {
		return append(color.Palette(nil), src...)
	}

	sorted := append(color.Palette(nil), src...)
	sort.Slice(sorted, func(i, j int) bool {
		return luma(sorted[i]) < luma(sorted[j])
	})

	pal := make(color.Palette, target)
	for i := 0; i < target; i++ {
		idx := i * (len(sorted) - 1) / (target - 1)
		pal[i] = sorted[idx]
	}
	return pal
}

// luma returns an approximate perceptual brightness of c, used only to order
// colors before sampling in reducePalette.
func luma(c color.Color) int {
	r, g, b, _ := c.RGBA()
	return int(r)*299/1000 + int(g)*587/1000 + int(b)*114/1000
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

// zippedFrame is a single decoded GIF frame encoded as a standalone PNG,
// named for inclusion in a zip archive.
type zippedFrame struct {
	Name string
	PNG  []byte
}

// splitGIFFrames decodes every frame of g onto the GIF's full logical canvas
// and re-encodes each as a standalone PNG. Frames in test fixtures (and any
// GIF whose frames already cover the whole canvas) convert directly; partial,
// disposal-based sub-rectangle compositing across frames is out of scope here.
func splitGIFFrames(g *gif.GIF) ([]zippedFrame, error) {
	canvas := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	frames := make([]zippedFrame, len(g.Image))
	for i, frame := range g.Image {
		rgba := image.NewRGBA(canvas)
		draw.Draw(rgba, frame.Bounds(), frame, frame.Bounds().Min, draw.Src)

		var buf bytes.Buffer
		if err := png.Encode(&buf, rgba); err != nil {
			return nil, fmt.Errorf("khong the ma hoa khung %d thanh png: %w", i, err)
		}
		frames[i] = zippedFrame{
			Name: fmt.Sprintf("frame_%03d.png", i+1),
			PNG:  buf.Bytes(),
		}
	}
	return frames, nil
}

// captionGIF returns a copy of g with text burned into every frame at a fixed
// margin near the top or bottom of the canvas (position must be "top" or
// "bottom"). text must contain non-whitespace content. hexColor is a
// "#rrggbb" string. size is the desired glyph height in pixels; text is
// rendered with basicfont.Face7x13 (its native size is 13px tall) and then
// nearest-neighbor scaled up by size/13 to approximate the requested size.
// Text that would exceed the frame's width wraps onto additional lines rather
// than being clipped.
func captionGIF(g *gif.GIF, text, position, hexColor string, size int) (*gif.GIF, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("noi dung chu thich khong duoc de trong")
	}
	if position != "top" && position != "bottom" {
		return nil, fmt.Errorf("vi tri chu thich khong hop le: %q (chi nhan top, bottom)", position)
	}
	col, err := parseHexColor(hexColor)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		return nil, fmt.Errorf("kich thuoc chu khong hop le: %d", size)
	}

	const margin = 4
	scale := size / 13
	if scale < 1 {
		scale = 1
	}

	maxLineWidthNative := (g.Config.Width - 2*margin) / scale
	if maxLineWidthNative < 1 {
		maxLineWidthNative = 1
	}
	lines := wrapCaptionText(text, maxLineWidthNative)

	textImg := renderCaptionLines(lines, col, scale)
	tw, th := textImg.Bounds().Dx(), textImg.Bounds().Dy()

	x := margin
	var y int
	if position == "top" {
		y = margin
	} else {
		y = g.Config.Height - margin - th
		if y < 0 {
			y = 0
		}
	}

	out := &gif.GIF{
		Image:           make([]*image.Paletted, len(g.Image)),
		Delay:           append([]int(nil), g.Delay...),
		LoopCount:       g.LoopCount,
		Disposal:        append([]byte(nil), g.Disposal...),
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}

	for i, frame := range g.Image {
		bounds := frame.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, frame, bounds.Min, draw.Src)
		draw.Draw(rgba, image.Rect(bounds.Min.X+x, bounds.Min.Y+y, bounds.Min.X+x+tw, bounds.Min.Y+y+th), textImg, image.Point{}, draw.Over)

		pal := extendPalette(frame.Palette, col)
		dst := image.NewPaletted(bounds, pal)
		draw.Draw(dst, bounds, rgba, bounds.Min, draw.Src)
		out.Image[i] = dst
	}
	return out, nil
}

// parseHexColor parses a "#rrggbb" string into an opaque color.RGBA.
func parseHexColor(s string) (color.RGBA, error) {
	trimmed := strings.TrimPrefix(s, "#")
	if len(trimmed) != 6 {
		return color.RGBA{}, fmt.Errorf("ma mau khong hop le: %q (dang #rrggbb)", s)
	}
	b, err := hex.DecodeString(trimmed)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("ma mau khong hop le: %q (dang #rrggbb)", s)
	}
	return color.RGBA{R: b[0], G: b[1], B: b[2], A: 255}, nil
}

// wrapCaptionText greedily packs words of text into lines no wider than
// maxWidthNative pixels (measured with basicfont.Face7x13 at its native
// scale). A single word wider than maxWidthNative still gets its own line
// rather than being split mid-word.
func wrapCaptionText(text string, maxWidthNative int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		trial := cur + " " + w
		if font.MeasureString(basicfont.Face7x13, trial).Ceil() <= maxWidthNative {
			cur = trial
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}

// renderCaptionLines draws lines (top to bottom) in col using
// basicfont.Face7x13, then nearest-neighbor scales the result up by scale.
func renderCaptionLines(lines []string, col color.RGBA, scale int) *image.RGBA {
	face := basicfont.Face7x13
	metrics := face.Metrics()
	lineHeight := metrics.Height.Ceil()
	ascent := metrics.Ascent.Ceil()

	maxW := 1
	lineImgs := make([]*image.RGBA, len(lines))
	for i, ln := range lines {
		w := font.MeasureString(face, ln).Ceil()
		if w < 1 {
			w = 1
		}
		if w > maxW {
			maxW = w
		}
		img := image.NewRGBA(image.Rect(0, 0, w, lineHeight))
		d := font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(col),
			Face: face,
			Dot:  fixed.P(0, ascent),
		}
		d.DrawString(ln)
		lineImgs[i] = img
	}

	native := image.NewRGBA(image.Rect(0, 0, maxW, lineHeight*len(lineImgs)))
	for i, img := range lineImgs {
		dr := image.Rect(0, i*lineHeight, img.Bounds().Dx(), (i+1)*lineHeight)
		draw.Draw(native, dr, img, image.Point{}, draw.Over)
	}

	return upscaleRGBA(native, scale)
}

// upscaleRGBA nearest-neighbor scales src up by the integer factor scale.
func upscaleRGBA(src *image.RGBA, scale int) *image.RGBA {
	if scale <= 1 {
		return src
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		sy := b.Min.Y + y/scale
		for x := 0; x < dst.Bounds().Dx(); x++ {
			sx := b.Min.X + x/scale
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// extendPalette returns pal with c appended if pal doesn't already contain an
// exact match for c and has room (GIF palettes are capped at 256 colors). If
// pal is already full, it is returned unchanged and c will be matched to its
// nearest existing entry when drawn.
func extendPalette(pal color.Palette, c color.Color) color.Palette {
	cr, cg, cb, ca := c.RGBA()
	for _, e := range pal {
		er, eg, eb, ea := e.RGBA()
		if er == cr && eg == cg && eb == cb && ea == ca {
			return pal
		}
	}
	if len(pal) >= 256 {
		return pal
	}
	return append(append(color.Palette(nil), pal...), c)
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
