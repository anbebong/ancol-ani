package main

import (
	"image"
	"image/color"
	"testing"
)

func TestImageToGIF_SingleFrame(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	out := imageToGIF(img)
	if len(out.Image) != 1 {
		t.Fatal("want 1 frame")
	}
}

func TestImagesToGIF_OrderAndDelay(t *testing.T) {
	imgs := []image.Image{
		image.NewRGBA(image.Rect(0, 0, 20, 20)),
		image.NewRGBA(image.Rect(0, 0, 20, 20)),
	}
	out, err := imagesToGIF(imgs, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 2 || out.Delay[0] != 20 { // 200ms -> 20 centiseconds
		t.Fatalf("got frames=%d delay=%v", len(out.Image), out.Delay)
	}
}

func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestImagesToGIF_PreservesFrameOrder(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	imgs := []image.Image{
		solidRGBA(20, 20, red),
		solidRGBA(20, 20, blue),
		// Mismatched size: must be resized to the first image's bounds.
		solidRGBA(40, 10, red),
	}
	out, err := imagesToGIF(imgs, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 3 {
		t.Fatalf("want 3 frames, got %d", len(out.Image))
	}
	want := []color.RGBA{red, blue, red}
	for i, frame := range out.Image {
		if frame.Bounds().Dx() != 20 || frame.Bounds().Dy() != 20 {
			t.Fatalf("frame %d bounds=%v, want 20x20", i, frame.Bounds())
		}
		r, g, b, _ := frame.At(10, 10).RGBA()
		got := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
		if (want[i].R == 255) != (got.R > got.B) || (want[i].B == 255) != (got.B > got.R) {
			t.Fatalf("frame %d color=%v, want dominant channel of %v", i, got, want[i])
		}
	}
}

func TestImagesToGIF_RejectsOverCount(t *testing.T) {
	imgs := make([]image.Image, maxImagesCount+1)
	for i := range imgs {
		imgs[i] = image.NewRGBA(image.Rect(0, 0, 5, 5))
	}
	if _, err := imagesToGIF(imgs, 100); err == nil {
		t.Fatal("want error over maxImagesCount")
	}
}
