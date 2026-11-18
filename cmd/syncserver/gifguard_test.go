package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"testing"
)

func encodeGif(t *testing.T, w, h, frames int) []byte {
	t.Helper()
	pal := color.Palette{color.Black, color.White}
	anim := &gif.GIF{Config: image.Config{Width: w, Height: h, ColorModel: pal}}
	for i := 0; i < frames; i++ {
		anim.Image = append(anim.Image, image.NewPaletted(image.Rect(0, 0, w, h), pal))
		anim.Delay = append(anim.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, anim); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCheckGifSize(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"small ok", encodeGif(t, 100, 100, 10), nil},
		{"max side ok", encodeGif(t, maxGifSide, maxGifSide, 2), nil},
		{"too wide", encodeGif(t, maxGifSide+1, 10, 1), errGifTooBig},
		{"too tall", encodeGif(t, 10, maxGifSide+1, 1), errGifTooBig},
		{"too many frames", encodeGif(t, 2, 2, maxGifFrames+1), errGifTooBig},
		// 70 frames of 1024x1024 = 73M px > maxGifTotalPixels, yet only a few KB compressed
		{"too many pixels", encodeGif(t, maxGifSide, maxGifSide, 70), errGifTooBig},
		{"not a gif", []byte("hello world, definitely not a gif"), errInvalidGif},
		{"truncated", encodeGif(t, 50, 50, 3)[:40], errInvalidGif},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := checkGifSize(c.data); got != c.want {
				t.Fatalf("checkGifSize = %v, want %v (size %d bytes)", got, c.want, len(c.data))
			}
		})
	}
}

// The real mascot shipped with the app must keep passing.
func TestCheckGifSizeMascot(t *testing.T) {
	data, err := os.ReadFile("../../mascot.gif")
	if err != nil {
		t.Skip("mascot.gif not found:", err)
	}
	if err := checkGifSize(data); err != nil {
		t.Fatalf("mascot rejected: %v", err)
	}
}
