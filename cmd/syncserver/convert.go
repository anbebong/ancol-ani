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

// maxVideoFPS caps the fps parameter accepted by videoToGIF. Anything
// higher multiplies ffmpeg's output frame count (and so its CPU/RAM cost and
// the resulting GIF size) for no real benefit at GIF-typical playback rates.
const maxVideoFPS = 30

// maxFfmpegOutputBytes bounds how much ffmpeg's stdout can buffer into
// memory for the video->gif convert path (I1): without this, a pathological
// input (or a disconnect that leaves ffmpeg running long) could grow an
// unbounded in-memory buffer before checkGifSize ever gets a chance to
// reject the result. Set well above maxGifBytes so a legitimately
// not-quite-fitting GIF still surfaces as checkGifSize's own clear "too big"
// error rather than this lower-level cutoff.
const maxFfmpegOutputBytes = maxGifBytes * 4

// ffmpegGifArgs builds the ffmpeg argument list for the video->gif convert
// path. Split out from videoToGIF so the command (in particular the scale
// filter) can be asserted on directly in tests without running ffmpeg.
//
// The -vf filter does two things:
//   - fps=N samples the video at the requested rate.
//   - scale clamps width to maxGifSide (preserving aspect ratio) so
//     real-world video resolutions (720p/1080p, common on phones) don't
//     get fully transcoded only to blow maxGifSide/maxGifTotalPixels
//     afterward. "-2" keeps the scaled height even, which some codecs
//     require.
func ffmpegGifArgs(path string, fps int) []string {
	vf := fmt.Sprintf("fps=%d,scale='min(iw\\,%d)':-2:flags=lanczos", fps, maxGifSide)
	return []string{
		"-i", path,
		"-vf", vf,
		"-f", "gif",
		"-",
	}
}

// limitedWriter is an io.Writer that errors (and, if set, calls kill) once
// more than limit bytes have been written to it. Used to cap how much of
// ffmpeg's stdout videoToGIF will buffer in memory (I1).
type limitedWriter struct {
	buf   bytes.Buffer
	limit int64
	kill  func()
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if int64(lw.buf.Len())+int64(len(p)) > lw.limit {
		if lw.kill != nil {
			lw.kill()
		}
		return 0, fmt.Errorf("dau ra ffmpeg vuot qua gioi han %d bytes", lw.limit)
	}
	return lw.buf.Write(p)
}

// videoToGIF transcodes the video at path into an animated GIF sampled at
// fps frames per second. fps is capped at maxVideoFPS and the video's
// duration is checked against maxVideoDurationSec before ffmpeg is ever
// invoked, so an invalid request never reaches the transcode step.
func videoToGIF(ctx context.Context, path string, fps int) ([]byte, error) {
	if fps <= 0 || fps > maxVideoFPS {
		return nil, fmt.Errorf("fps khong hop le: %d (chi nhan 1-%d)", fps, maxVideoFPS)
	}

	dur, err := probeVideoDuration(ctx, path)
	if err != nil {
		return nil, err
	}
	if dur > time.Duration(maxVideoDurationSec)*time.Second {
		return nil, fmt.Errorf("video qua dai: %.1fs (toi da %ds)", dur.Seconds(), maxVideoDurationSec)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegGifArgs(path, fps)...)
	out := &limitedWriter{limit: maxFfmpegOutputBytes}
	var stderr bytes.Buffer
	cmd.Stdout = out
	cmd.Stderr = &stderr
	out.kill = func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chuyen doi video sang gif that bai: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return out.buf.Bytes(), nil
}
