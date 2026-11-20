package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

// ffmpegAvailable reports whether both ffmpeg and ffprobe are reachable on
// PATH. Callers that depend on video transcoding should check this first and
// degrade gracefully (e.g. return a clear error) when it is false.
func ffmpegAvailable() bool {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return false
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return false
	}
	return true
}

// probeVideoDuration runs ffprobe against the file at path and returns its
// duration. It errors if the file is missing/unreadable or ffprobe's output
// cannot be parsed.
func probeVideoDuration(ctx context.Context, path string) (time.Duration, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		path,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("khong doc duoc thoi luong video: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	secStr := strings.TrimSpace(stdout.String())
	secs, err := strconv.ParseFloat(secStr, 64)
	if err != nil {
		return 0, fmt.Errorf("khong doc duoc thoi luong video: ffprobe tra ve gia tri khong hop le %q", secStr)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// videoToGIF transcodes the video at path into an animated GIF sampled at
// fps frames per second. The video's duration is checked against
// maxVideoDurationSec before ffmpeg is ever invoked, so an over-limit video
// never reaches the transcode step.
func videoToGIF(ctx context.Context, path string, fps int) ([]byte, error) {
	dur, err := probeVideoDuration(ctx, path)
	if err != nil {
		return nil, err
	}
	if dur > time.Duration(maxVideoDurationSec)*time.Second {
		return nil, fmt.Errorf("video qua dai: %.1fs (toi da %ds)", dur.Seconds(), maxVideoDurationSec)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", path,
		"-vf", fmt.Sprintf("fps=%d", fps),
		"-f", "gif",
		"-",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chuyen doi video sang gif that bai: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
