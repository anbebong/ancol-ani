package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// toolTmpDir holds temp files created while serving /api/tools/*, mainly the
// video upload path for convert (ffmpeg needs a real file to read from).
var toolTmpDir = filepath.Join("data", "tmp")

var errPoolFull = httpErr{"server dang xu ly qua nhieu yeu cau, vui long thu lai sau"}
var errFFmpegMissing = httpErr{"server thieu ffmpeg/ffprobe"}

// poolFullWait is how long a request waits for a free slot before giving up
// with 429, instead of queueing for the full toolRequestTimeout and only then
// surfacing as a 504 (which would look like the job itself ran too long).
var poolFullWait = 50 * time.Millisecond

// withToolLimits wraps a /api/tools/* handler with a per-request deadline and
// the shared concurrency pool. It disambiguates the two failure shapes a
// caller can see:
//   - 429 Too Many Requests: the pool was already full and stayed full for
//     poolFullWait; the caller gave up quickly rather than queueing.
//   - 504 Gateway Timeout: a slot was acquired (or waited longer than
//     poolFullWait while queued) but the handler itself didn't finish before
//     toolRequestTimeout elapsed.
//
// Any temp file the handler created is cleaned up by the handler's own
// defer; this middleware only owns the slot and the deadline.
func withToolLimits(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), toolRequestTimeout)
		defer cancel()

		waitCtx, waitCancel := context.WithTimeout(ctx, poolFullWait)
		err := acquireToolSlot(waitCtx)
		waitCancel()
		if err != nil {
			if ctx.Err() != nil {
				// The outer deadline itself expired while we were still
				// waiting for a slot - that's a real overrun, not a quick
				// "pool is full" rejection.
				http.Error(w, "qua thoi gian xu ly", http.StatusGatewayTimeout)
			} else {
				http.Error(w, errPoolFull.Error(), http.StatusTooManyRequests)
			}
			return
		}
		defer releaseToolSlot()

		done := make(chan struct{})
		r = r.WithContext(ctx)
		go func() {
			h(w, r)
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			// The handler is expected to observe ctx.Done() (directly, or
			// via a canceled exec.CommandContext) and return on its own;
			// we still report the timeout to the client immediately rather
			// than waiting for it to unwind.
			http.Error(w, "qua thoi gian xu ly", http.StatusGatewayTimeout)
			<-done
		}
	}
}

// decodeAndValidateGIF reads the named multipart field, rejects it as a
// decompression bomb before inflating anything, then fully decodes it.
func decodeAndValidateGIF(r *http.Request, field string) (*gif.GIF, error) {
	if err := r.ParseMultipartForm(maxGifBytes); err != nil {
		return nil, errTooLarge
	}
	file, _, err := r.FormFile(field)
	if err != nil {
		return nil, errMissingFile
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, maxGifBytes+1))
	if err != nil {
		return nil, errTooLarge
	}
	if int64(len(raw)) > maxGifBytes {
		return nil, errTooLarge
	}
	if err := checkGifSize(raw); err != nil {
		return nil, err
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		return nil, errInvalidGif
	}
	return g, nil
}

// encodeAndValidateGIF re-encodes g and makes sure the *output* also stays
// within the size guard - an operation like resize-up or caption can in
// principle grow a GIF past the limits that only applied to the input.
func encodeAndValidateGIF(g *gif.GIF) ([]byte, error) {
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, errInvalidGif
	}
	if err := checkGifSize(buf.Bytes()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeGIFResponse(w http.ResponseWriter, g *gif.GIF) {
	out, err := encodeAndValidateGIF(g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Write(out)
}

func queryInt(r *http.Request, key string, def int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		v = r.FormValue(key)
	}
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

func queryFloat(r *http.Request, key string, def float64) (float64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		v = r.FormValue(key)
	}
	if v == "" {
		return def, nil
	}
	return strconv.ParseFloat(v, 64)
}

func queryStr(r *http.Request, key, def string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		v = r.FormValue(key)
	}
	if v == "" {
		return def
	}
	return v
}

// handleToolResize: POST /api/tools/resize, multipart "file" + width/height.
func handleToolResize(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	width, err1 := queryInt(r, "width", 0)
	height, err2 := queryInt(r, "height", 0)
	if err1 != nil || err2 != nil {
		http.Error(w, "width/height khong hop le", http.StatusBadRequest)
		return
	}
	out, err := resizeGIF(g, width, height)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolCrop: POST /api/tools/crop, multipart "file" + x/y/width/height.
func handleToolCrop(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	x, e1 := queryInt(r, "x", 0)
	y, e2 := queryInt(r, "y", 0)
	width, e3 := queryInt(r, "width", 0)
	height, e4 := queryInt(r, "height", 0)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		http.Error(w, "x/y/width/height khong hop le", http.StatusBadRequest)
		return
	}
	out, err := cropGIF(g, x, y, width, height)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolOptimize: POST /api/tools/optimize, multipart "file" + level (1-3).
func handleToolOptimize(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	level, errLvl := queryInt(r, "level", 2)
	if errLvl != nil {
		http.Error(w, "level khong hop le", http.StatusBadRequest)
		return
	}
	out, err := optimizeGIF(g, level)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolRotate: POST /api/tools/rotate, multipart "file" + angle/flip.
func handleToolRotate(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	angle, errAngle := queryInt(r, "angle", 0)
	if errAngle != nil {
		http.Error(w, "angle khong hop le", http.StatusBadRequest)
		return
	}
	flip := queryStr(r, "flip", "none")
	out, err := rotateGIF(g, angle, flip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolReverse: POST /api/tools/reverse, multipart "file" only.
func handleToolReverse(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, reverseGIF(g))
}

// handleToolSpeed: POST /api/tools/speed, multipart "file" + factor.
func handleToolSpeed(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	factor, errFactor := queryFloat(r, "factor", 1)
	if errFactor != nil {
		http.Error(w, "factor khong hop le", http.StatusBadRequest)
		return
	}
	out, err := speedGIF(g, factor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolSplit: POST /api/tools/split, multipart "file" only. Returns a
// .zip with one PNG per frame.
func handleToolSplit(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	frames, err := splitGIFFrames(g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range frames {
		fw, err := zw.Create(f.Name)
		if err != nil {
			http.Error(w, "khong the tao zip", http.StatusInternalServerError)
			return
		}
		if _, err := fw.Write(f.PNG); err != nil {
			http.Error(w, "khong the tao zip", http.StatusInternalServerError)
			return
		}
	}
	if err := zw.Close(); err != nil {
		http.Error(w, "khong the tao zip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Write(buf.Bytes())
}

// handleToolCaption: POST /api/tools/caption, multipart "file" + text/position/color/size.
func handleToolCaption(w http.ResponseWriter, r *http.Request) {
	g, err := decodeAndValidateGIF(r, "file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text := queryStr(r, "text", "")
	position := queryStr(r, "position", "bottom")
	hexColor := queryStr(r, "color", "#ffffff")
	size, errSize := queryInt(r, "size", 16)
	if errSize != nil {
		http.Error(w, "size khong hop le", http.StatusBadRequest)
		return
	}
	out, err := captionGIF(g, text, position, hexColor, size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}

// handleToolConvert: POST /api/tools/convert.
//   - multipart "file" = a video -> transcoded with ffmpeg at the given fps.
//   - multipart "file" (one or more) = image(s) -> combined into a GIF.
//
// Decided by Content-Type sniffing of the first "file" field, not a
// separate "kind" parameter, since the brief's own example payloads never
// send one explicitly and the upload's own bytes are authoritative anyway.
func handleToolConvert(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxVideoBytes); err != nil {
		http.Error(w, errTooLarge.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, errMissingFile.Error(), http.StatusBadRequest)
		return
	}

	if len(files) == 1 && looksLikeVideo(files[0].Filename) {
		handleConvertVideo(w, r, files[0])
		return
	}
	handleConvertImages(w, r, files)
}

func looksLikeVideo(filename string) bool {
	switch filepath.Ext(filename) {
	case ".mp4", ".mov", ".webm", ".mkv", ".avi":
		return true
	}
	return false
}

func handleConvertVideo(w http.ResponseWriter, r *http.Request, fh *multipart.FileHeader) {
	if !ffmpegAvailable() {
		http.Error(w, errFFmpegMissing.Error(), http.StatusServiceUnavailable)
		return
	}
	file, err := fh.Open()
	if err != nil {
		http.Error(w, errMissingFile.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if fh.Size > maxVideoBytes {
		http.Error(w, errTooLarge.Error(), http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(toolTmpDir, 0o755); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tmp, err := os.CreateTemp(toolTmpDir, "convert-*.mp4")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	n, err := io.Copy(tmp, io.LimitReader(file, maxVideoBytes+1))
	tmp.Close()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n > maxVideoBytes {
		http.Error(w, errTooLarge.Error(), http.StatusBadRequest)
		return
	}

	fps, errFps := queryInt(r, "fps", 10)
	if errFps != nil || fps <= 0 {
		http.Error(w, "fps khong hop le", http.StatusBadRequest)
		return
	}

	data, err := videoToGIF(r.Context(), tmpPath, fps)
	if err != nil {
		if errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			http.Error(w, "qua thoi gian xu ly", http.StatusGatewayTimeout)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := checkGifSize(data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Write(data)
}

func handleConvertImages(w http.ResponseWriter, r *http.Request, files []*multipart.FileHeader) {
	if len(files) > maxImagesCount {
		http.Error(w, fmt.Sprintf("qua nhieu anh: toi da %d", maxImagesCount), http.StatusBadRequest)
		return
	}

	images := make([]image.Image, 0, len(files))
	for _, fh := range files {
		if fh.Size > maxImageBytes {
			http.Error(w, errTooLarge.Error(), http.StatusBadRequest)
			return
		}
		f, err := fh.Open()
		if err != nil {
			http.Error(w, errMissingFile.Error(), http.StatusBadRequest)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
		f.Close()
		if err != nil || int64(len(raw)) > maxImageBytes {
			http.Error(w, errTooLarge.Error(), http.StatusBadRequest)
			return
		}
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			http.Error(w, "anh khong hop le: "+fh.Filename, http.StatusBadRequest)
			return
		}
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > maxGifSide || b.Dy() > maxGifSide {
			http.Error(w, fmt.Sprintf("kich thuoc anh khong hop le: %dx%d (toi da %dx%d)", b.Dx(), b.Dy(), maxGifSide, maxGifSide), http.StatusBadRequest)
			return
		}
		images = append(images, img)
	}

	if len(images) == 1 {
		writeGIFResponse(w, imageToGIF(images[0]))
		return
	}

	delayMs, errDelay := queryInt(r, "fps", 0)
	if errDelay != nil {
		http.Error(w, "fps khong hop le", http.StatusBadRequest)
		return
	}
	delay := 200
	if delayMs > 0 {
		delay = 1000 / delayMs
	}
	out, err := imagesToGIF(images, delay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGIFResponse(w, out)
}
