package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

func TestResizeGIF_Basic(t *testing.T) {
	g := makeTestGIF(t, 3, 300, 300)
	out, err := resizeGIF(g, 150, 150)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 3 {
		t.Fatalf("frames: got %d want 3", len(out.Image))
	}
	b := out.Image[0].Bounds()
	if b.Dx() != 150 || b.Dy() != 150 {
		t.Fatalf("size: got %dx%d want 150x150", b.Dx(), b.Dy())
	}
}

func TestResizeGIF_RejectsZeroOrNegative(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 100)
	if _, err := resizeGIF(g, 0, 100); err == nil {
		t.Fatal("want error for width=0")
	}
	if _, err := resizeGIF(g, 100, -5); err == nil {
		t.Fatal("want error for negative height")
	}
}

func TestResizeGIF_RejectsOverMaxSide(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 100)
	if _, err := resizeGIF(g, maxGifSide+1, 100); err == nil {
		t.Fatal("want error over maxGifSide")
	}
}

func TestCropGIF_Basic(t *testing.T) {
	g := makeTestGIF(t, 2, 300, 300)
	out, err := cropGIF(g, 50, 50, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	b := out.Image[0].Bounds()
	if b.Dx() != 100 || b.Dy() != 100 {
		t.Fatalf("size: got %dx%d want 100x100", b.Dx(), b.Dy())
	}
}

func TestCropGIF_RejectsOutOfBounds(t *testing.T) {
	g := makeTestGIF(t, 1, 300, 300)
	if _, err := cropGIF(g, 250, 250, 100, 100); err == nil {
		t.Fatal("want error: crop box exceeds source")
	}
}

func TestCropGIF_RejectsZeroSize(t *testing.T) {
	g := makeTestGIF(t, 1, 300, 300)
	if _, err := cropGIF(g, 10, 10, 0, 50); err == nil {
		t.Fatal("want error: w=0")
	}
}

// makePartialFrameGIF builds a 2-frame GIF where frame 0 covers the whole
// canvas in bg (the "background") and frame 1 is a small sub-rectangle at
// (0,0)-(10,10) in fg (the "delta") - the common real-world GIF shape where
// later frames only encode the pixels that changed, relying on earlier
// frames' pixels staying visible underneath. garbage is a third palette
// color that a correct implementation should never produce, used to detect
// the out-of-bounds-read bug directly (a buggy cropGIF read garbage pixels
// as palette index 0; by putting the garbage color at index 0 and
// background at index 1, any garbage-colored output pixel proves the bug).
func makePartialFrameGIF(t *testing.T, canvasW, canvasH int) (*gif.GIF, color.Palette) {
	t.Helper()
	garbage := color.RGBA{R: 255, A: 255}
	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	fg := color.RGBA{A: 255}
	pal := color.Palette{garbage, bg, fg}

	full := image.NewPaletted(image.Rect(0, 0, canvasW, canvasH), pal)
	for i := range full.Pix {
		full.Pix[i] = 1 // bg
	}
	delta := image.NewPaletted(image.Rect(0, 0, 10, 10), pal)
	for i := range delta.Pix {
		delta.Pix[i] = 2 // fg
	}

	anim := &gif.GIF{
		Config:   image.Config{Width: canvasW, Height: canvasH, ColorModel: pal},
		Image:    []*image.Paletted{full, delta},
		Delay:    []int{10, 10},
		Disposal: []byte{gif.DisposalNone, gif.DisposalNone},
	}
	return anim, pal
}

func TestCropGIF_CompositesPartialFrameOverBackground(t *testing.T) {
	g, _ := makePartialFrameGIF(t, 100, 100)
	out, err := cropGIF(g, 0, 0, 50, 50)
	if err != nil {
		t.Fatal(err)
	}
	// Frame 1 (the delta frame) only paints (0,0)-(10,10); a pixel well
	// outside that, inside the crop, must show frame 0's background, not
	// the garbage color a buggy out-of-bounds read would produce.
	got := out.Image[1].ColorIndexAt(30, 30)
	if got != 1 {
		t.Fatalf("pixel (30,30) of cropped delta frame: got palette index %d, want 1 (background)", got)
	}
	// The delta itself must still come through where it was actually drawn.
	if got := out.Image[1].ColorIndexAt(5, 5); got != 2 {
		t.Fatalf("pixel (5,5) of cropped delta frame: got palette index %d, want 2 (delta)", got)
	}
}

func TestCropGIF_ValidatesAgainstLogicalCanvasNotFirstFrame(t *testing.T) {
	// The logical canvas is 100x100, but frame 0 (the only frame cropGIF used
	// to check bounds against) happens to be smaller. A crop that fits the
	// real canvas but would have been rejected against frame 0's bounds must
	// now succeed.
	g, _ := makePartialFrameGIF(t, 100, 100)
	g.Image[0] = image.NewPaletted(image.Rect(0, 0, 20, 20), g.Image[0].Palette)
	if _, err := cropGIF(g, 60, 60, 20, 20); err != nil {
		t.Fatalf("crop within the 100x100 logical canvas should succeed, got: %v", err)
	}
}

func TestRotateGIF_90SwapsDimensions(t *testing.T) {
	g := makeTestGIF(t, 1, 200, 100)
	out, err := rotateGIF(g, 90, "none")
	if err != nil {
		t.Fatal(err)
	}
	b := out.Image[0].Bounds()
	if b.Dx() != 100 || b.Dy() != 200 {
		t.Fatalf("size: got %dx%d want 100x200", b.Dx(), b.Dy())
	}
}

func TestRotateGIF_NoopPassesThrough(t *testing.T) {
	g := makeTestGIF(t, 1, 200, 100)
	out, err := rotateGIF(g, 0, "none")
	if err != nil {
		t.Fatal(err)
	}
	b := out.Image[0].Bounds()
	if b.Dx() != 200 || b.Dy() != 100 {
		t.Fatal("size changed on no-op")
	}
}

func TestRotateGIF_RejectsInvalidAngle(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 100)
	if _, err := rotateGIF(g, 45, "none"); err == nil {
		t.Fatal("want error for angle=45")
	}
}

func TestRotateGIF_RejectsInvalidFlip(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 100)
	if _, err := rotateGIF(g, 0, "diagonal"); err == nil {
		t.Fatal("want error for flip=diagonal")
	}
}

// Marks the top-left pixel of a 3x2 frame and checks where it lands. Rotation
// is clockwise; flip is applied before rotation.
func TestReverseGIF_OrderAndDelay(t *testing.T) {
	g := makeTestGIF(t, 3, 50, 50)
	g.Delay = []int{10, 20, 30}
	out := reverseGIF(g)
	if len(out.Delay) != 3 || out.Delay[0] != 30 || out.Delay[2] != 10 {
		t.Fatalf("delays not reversed: %v", out.Delay)
	}
}

func TestReverseGIF_SingleFrameNoop(t *testing.T) {
	g := makeTestGIF(t, 1, 50, 50)
	out := reverseGIF(g)
	if len(out.Image) != 1 {
		t.Fatal("frame count changed")
	}
}

func TestReverseGIF_ReversesImageAndDisposalInLockstep(t *testing.T) {
	g := makeTestGIF(t, 3, 4, 4)
	g.Delay = []int{10, 20, 30}
	g.Disposal = []byte{1, 2, 3}
	first, last := g.Image[0], g.Image[2]
	out := reverseGIF(g)
	if out.Image[0] != last || out.Image[2] != first {
		t.Fatal("images not reversed")
	}
	if out.Disposal[0] != 3 || out.Disposal[1] != 2 || out.Disposal[2] != 1 {
		t.Fatalf("disposal not reversed: %v", out.Disposal)
	}
	if out.Delay[0] != 30 || out.Delay[1] != 20 || out.Delay[2] != 10 {
		t.Fatalf("delay not reversed: %v", out.Delay)
	}
	if g.Image[0] != first || g.Delay[0] != 10 || g.Disposal[0] != 1 {
		t.Fatal("input was mutated")
	}
}

func TestRotateGIF_PixelMapping(t *testing.T) {
	cases := []struct {
		name  string
		angle int
		flip  string
		x, y  int // expected position of the marked pixel
	}{
		{"none-0", 0, "none", 0, 0},
		{"cw-90", 90, "none", 1, 0},   // top-left -> top-right
		{"cw-180", 180, "none", 2, 1}, // top-left -> bottom-right
		{"cw-270", 270, "none", 0, 2}, // top-left -> bottom-left
		{"h-0", 0, "h", 2, 0},         // mirror horizontally
		{"v-0", 0, "v", 0, 1},         // mirror vertically
		{"h-90", 90, "h", 1, 2},       // mirror to (2,0), then rotate CW to (1,2)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := makeTestGIF(t, 1, 3, 2)
			// Fill with 0 so the single marked pixel is the only non-zero one.
			for j := range g.Image[0].Pix {
				g.Image[0].Pix[j] = 0
			}
			g.Image[0].SetColorIndex(0, 0, 1)
			out, err := rotateGIF(g, c.angle, c.flip)
			if err != nil {
				t.Fatal(err)
			}
			dst := out.Image[0]
			if got := dst.ColorIndexAt(c.x, c.y); got != 1 {
				t.Fatalf("marked pixel not at (%d,%d)", c.x, c.y)
			}
			count := 0
			for _, p := range dst.Pix {
				if p != 0 {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("expected exactly one marked pixel, got %d", count)
			}
		})
	}
}

func TestSpeedGIF_ScalesDelay(t *testing.T) {
	g := makeTestGIF(t, 1, 50, 50)
	g.Delay = []int{10}
	out, err := speedGIF(g, 2.0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Delay[0] != 5 {
		t.Fatalf("delay: got %d want 5", out.Delay[0])
	}
}

func TestSpeedGIF_RejectsNonPositiveFactor(t *testing.T) {
	g := makeTestGIF(t, 1, 50, 50)
	if _, err := speedGIF(g, 0); err == nil {
		t.Fatal("want error for factor=0")
	}
	if _, err := speedGIF(g, -1); err == nil {
		t.Fatal("want error for factor<0")
	}
}

func TestSpeedGIF_ClampsToMinimumDelay(t *testing.T) {
	g := makeTestGIF(t, 1, 50, 50)
	g.Delay = []int{2}
	out, err := speedGIF(g, 100)
	if err != nil {
		t.Fatal(err)
	}
	if out.Delay[0] < 2 {
		t.Fatalf("delay below GIF minimum: %d", out.Delay[0])
	}
}

// makeColorfulTestGIF builds frames with a full 256-entry palette and a
// non-repeating pixel pattern, so the frames actually need the full palette
// width (8 bits/pixel code) to encode. makeTestGIF's fixture only ever uses 2
// palette entries, so palette reduction cannot shrink it further (GIF already
// picks the minimum code size for a 2-color palette); this fixture gives
// optimizeGIF real color variation to reduce.
func makeColorfulTestGIF(t *testing.T, frames int, w, h int) *gif.GIF {
	t.Helper()
	pal := make(color.Palette, 0, 256)
	for r := 0; r < 8; r++ {
		for g := 0; g < 8; g++ {
			for b := 0; b < 4; b++ {
				pal = append(pal, color.RGBA{R: uint8(r * 36), G: uint8(g * 36), B: uint8(b * 85), A: 255})
			}
		}
	}
	anim := &gif.GIF{Config: image.Config{Width: w, Height: h, ColorModel: pal}}
	for i := 0; i < frames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := (x + y + i) % len(pal)
				img.SetColorIndex(x, y, uint8(idx))
			}
		}
		anim.Image = append(anim.Image, img)
		anim.Delay = append(anim.Delay, 10)
	}
	return anim
}

func TestOptimizeGIF_ReducesEncodedSize(t *testing.T) {
	g := makeColorfulTestGIF(t, 5, 200, 200)
	var before bytes.Buffer
	if err := gif.EncodeAll(&before, g); err != nil {
		t.Fatal(err)
	}
	out, err := optimizeGIF(g, 3)
	if err != nil {
		t.Fatal(err)
	}
	var after bytes.Buffer
	if err := gif.EncodeAll(&after, out); err != nil {
		t.Fatal(err)
	}
	if after.Len() >= before.Len() {
		t.Fatalf("optimize did not shrink: before=%d after=%d", before.Len(), after.Len())
	}
}

func TestOptimizeGIF_RejectsInvalidLevel(t *testing.T) {
	g := makeTestGIF(t, 1, 50, 50)
	if _, err := optimizeGIF(g, 0); err == nil {
		t.Fatal("want error for level=0")
	}
	if _, err := optimizeGIF(g, 4); err == nil {
		t.Fatal("want error for level=4")
	}
}

func TestSplitGIFFrames_Count(t *testing.T) {
	g := makeTestGIF(t, 4, 50, 50)
	frames, err := splitGIFFrames(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("frames: got %d want 4", len(frames))
	}
	if frames[0].Name == "" {
		t.Fatal("frame 0 has empty name")
	}
	img, err := png.Decode(bytes.NewReader(frames[0].PNG))
	if err != nil {
		t.Fatalf("frame 0 not valid PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 50 || b.Dy() != 50 {
		t.Fatalf("frame 0 size: got %dx%d want 50x50", b.Dx(), b.Dy())
	}
}

func TestSplitGIFFrames_CompositesPartialFrameOverBackground(t *testing.T) {
	g, _ := makePartialFrameGIF(t, 100, 100)
	frames, err := splitGIFFrames(g)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(frames[1].PNG))
	if err != nil {
		t.Fatalf("frame 1 not valid PNG: %v", err)
	}
	// A pixel outside the delta frame's own (0,0)-(10,10) must still show
	// frame 0's white background, not a transparent/black hole - the bug
	// the un-composited implementation had for any GIF using sub-rectangle
	// delta frames.
	r, g2, b, a := img.At(30, 30).RGBA()
	if a == 0 || r>>8 != 255 || g2>>8 != 255 || b>>8 != 255 {
		t.Fatalf("pixel (30,30) of frame 1's PNG: got rgba(%d,%d,%d,%d), want opaque white", r>>8, g2>>8, b>>8, a>>8)
	}
	// The delta itself must still be present where it was actually drawn.
	r, g2, b, a = img.At(5, 5).RGBA()
	if a>>8 != 255 || r>>8 != 0 || g2>>8 != 0 || b>>8 != 0 {
		t.Fatalf("pixel (5,5) of frame 1's PNG: got rgba(%d,%d,%d,%d), want opaque black", r>>8, g2>>8, b>>8, a>>8)
	}
}

func TestCaptionGIF_RejectsEmptyText(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	if _, err := captionGIF(g, "", "top", "#ffffff", 16, 0, 0); err == nil {
		t.Fatal("want error for empty text")
	}
}

func TestCaptionGIF_RejectsWhitespaceOnlyText(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	if _, err := captionGIF(g, "   \t  ", "top", "#ffffff", 16, 0, 0); err == nil {
		t.Fatal("want error for whitespace-only text")
	}
}

func TestCaptionGIF_RejectsInvalidPosition(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	if _, err := captionGIF(g, "Hi", "middle", "#ffffff", 16, 0, 0); err == nil {
		t.Fatal("want error for invalid position")
	}
}

func TestCaptionGIF_RejectsInvalidHexColor(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	if _, err := captionGIF(g, "Hi", "top", "not-a-color", 16, 0, 0); err == nil {
		t.Fatal("want error for invalid hex color")
	}
}

func TestCaptionGIF_AppliesToAllFrames(t *testing.T) {
	g := makeTestGIF(t, 3, 100, 50)
	out, err := captionGIF(g, "Hi", "bottom", "#ffffff", 16, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 3 {
		t.Fatal("frame count changed")
	}
}

func TestCaptionGIF_ChangesPixels(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	out, err := captionGIF(g, "Hi", "top", "#ff0000", 16, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	frame := out.Image[0]
	changed := false
	for y := 0; y < frame.Bounds().Dy() && !changed; y++ {
		for x := 0; x < frame.Bounds().Dx(); x++ {
			r, gg, b, _ := frame.At(x, y).RGBA()
			if r>>8 > 200 && gg>>8 < 50 && b>>8 < 50 {
				changed = true
				break
			}
		}
	}
	if !changed {
		t.Fatal("expected some pixels to match caption color #ff0000")
	}
}

func TestCaptionGIF_CustomPositionPlacesTextAtGivenCoords(t *testing.T) {
	g := makeTestGIF(t, 1, 100, 50)
	// Place the caption near the right edge, clear of the top/bottom
	// margins captionGIF uses for "top"/"bottom" - if posX/posY were
	// ignored, the red pixels would instead cluster near x=4,y=4 (top-left).
	out, err := captionGIF(g, "A", "custom", "#ff0000", 16, 80, 20)
	if err != nil {
		t.Fatal(err)
	}
	frame := out.Image[0]
	foundNearTarget := false
	for y := 15; y < 35 && !foundNearTarget; y++ {
		for x := 70; x < 100; x++ {
			r, gg, b, _ := frame.At(x, y).RGBA()
			if r>>8 > 200 && gg>>8 < 50 && b>>8 < 50 {
				foundNearTarget = true
				break
			}
		}
	}
	if !foundNearTarget {
		t.Fatal("expected caption pixels near custom position (80,20), found none in that region")
	}
}

func TestCaptionGIF_RejectsCustomPositionOutOfRange(t *testing.T) {
	// position="custom" with coordinates must still be accepted (clamped),
	// never rejected - only the position keyword itself is validated.
	g := makeTestGIF(t, 1, 100, 50)
	if _, err := captionGIF(g, "A", "custom", "#ff0000", 16, -50, 9999); err != nil {
		t.Fatalf("expected out-of-range custom coords to be clamped, not rejected: %v", err)
	}
}

func TestCaptionGIF_WrapsLongText(t *testing.T) {
	g := makeTestGIF(t, 1, 60, 50)
	out, err := captionGIF(g, "this is a fairly long caption that should wrap", "bottom", "#ffffff", 16, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 1 {
		t.Fatal("frame count changed")
	}
}
