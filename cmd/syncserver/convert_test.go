package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"strings"
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

func TestFfmpegAvailable_ReflectsPath(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed in this environment; function exercised in Task 11's handler test with a stub")
	}
}

func TestProbeVideoDuration_RejectsMissingFile(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}
	_, err := probeVideoDuration(context.Background(), "/no/such/file.mp4")
	if err == nil {
		t.Fatal("want error for missing file")
	}
}

// --- I1: fps cap + scale filter ---

func TestVideoToGIF_RejectsBadFPS(t *testing.T) {
	// No ffmpeg/ffprobe call needed: the fps check runs before probing the
	// video, so this must fail even for a nonexistent path.
	if _, err := videoToGIF(context.Background(), "/no/such/file.mp4", 0); err == nil {
		t.Fatal("want error for fps=0")
	}
	if _, err := videoToGIF(context.Background(), "/no/such/file.mp4", -1); err == nil {
		t.Fatal("want error for fps<0")
	}
	if _, err := videoToGIF(context.Background(), "/no/such/file.mp4", maxVideoFPS+1); err == nil {
		t.Fatal("want error for fps over maxVideoFPS")
	}
}

func TestFfmpegGifArgs_IncludesFpsAndScaleFilter(t *testing.T) {
	args := ffmpegGifArgs("in.mp4", 12)
	var vf string
	for i, a := range args {
		if a == "-vf" && i+1 < len(args) {
			vf = args[i+1]
		}
	}
	if vf == "" {
		t.Fatalf("no -vf argument found in %v", args)
	}
	if !strings.Contains(vf, "fps=12") {
		t.Fatalf("-vf %q missing fps=12", vf)
	}
	wantScale := fmt.Sprintf("scale='min(iw\\,%d)'", maxGifSide)
	if !strings.Contains(vf, wantScale) {
		t.Fatalf("-vf %q missing scale filter using maxGifSide (%s)", vf, wantScale)
	}
}

func TestVideoToGIF_HappyPath(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}
	data, err := videoToGIF(context.Background(), "testdata/tiny.mp4", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
		t.Fatalf("output not a valid gif: %v", err)
	}
}
