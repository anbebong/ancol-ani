package main

import "testing"

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
