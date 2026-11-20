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
