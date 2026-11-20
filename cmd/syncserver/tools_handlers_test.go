package main

import (
	"bytes"
	"image"
	"image/gif"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// multipartGifRequest builds a multipart/form-data body with field name
// holding data as a .gif file, returning the body and its Content-Type.
func multipartGifRequest(t *testing.T, field string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, "upload.gif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

// multipartBombRequest wraps the gifguard_test.go bomb fixture (70 real
// frames at maxGifSide, small compressed bytes, huge declared pixel total)
// in a multipart body under field.
func multipartBombRequest(t *testing.T, field string) (*bytes.Buffer, string) {
	t.Helper()
	return multipartGifRequest(t, field, encodeGif(t, maxGifSide, maxGifSide, 70))
}

// encodeTestGIF encodes frames solid-color frames of w x h into GIF bytes,
// reusing the testfixtures_test.go helper for the pixel data.
func encodeTestGIF(t *testing.T, frames, w, h int) []byte {
	t.Helper()
	anim := makeTestGIF(t, frames, w, h)
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, anim); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHandleToolResize_HappyPath(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 300, 300))
	req := httptest.NewRequest("POST", "/api/tools/resize?width=150&height=150", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolResize)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	if _, err := gif.DecodeAll(w.Body); err != nil {
		t.Fatalf("not a valid gif: %v", err)
	}
}

func TestHandleToolResize_RejectsGifBomb(t *testing.T) {
	body, ct := multipartBombRequest(t, "file")
	req := httptest.NewRequest("POST", "/api/tools/resize?width=50&height=50", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolResize)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400", w.Code)
	}
}

func TestWithToolLimits_RejectsOverConcurrency(t *testing.T) {
	// The slot is now acquired inside the handler (runWithToolSlot), around
	// the processing step only - see I3. Simulate that explicitly here
	// since withToolLimits itself no longer touches the pool.
	slow := func(w http.ResponseWriter, r *http.Request) {
		runWithToolSlot(w, r, func() {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})
	}
	wrapped := withToolLimits(slow)
	results := make(chan int, maxConcurrentToolJobs+1)
	for i := 0; i < maxConcurrentToolJobs+1; i++ {
		go func() {
			w := httptest.NewRecorder()
			wrapped(w, httptest.NewRequest("POST", "/api/tools/resize", nil))
			results <- w.Code
		}()
	}
	codes := map[int]int{}
	for i := 0; i < maxConcurrentToolJobs+1; i++ {
		codes[<-results]++
	}
	if codes[http.StatusTooManyRequests] == 0 {
		t.Fatalf("want at least one 429 among: %v", codes)
	}
}

func TestWithToolLimits_TimesOutAndCleansTempFiles(t *testing.T) {
	if err := os.MkdirAll(toolTmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hang := func(w http.ResponseWriter, r *http.Request) {
		tmp, err := os.CreateTemp(toolTmpDir, "hang-*")
		if err != nil {
			return
		}
		defer func() {
			tmp.Close() // Windows can't remove an open file
			os.Remove(tmp.Name())
		}()
		<-r.Context().Done()
	}
	saved := toolRequestTimeout
	toolRequestTimeout = 20 * time.Millisecond
	defer func() { toolRequestTimeout = saved }()

	w := httptest.NewRecorder()
	withToolLimits(hang)(w, httptest.NewRequest("POST", "/api/tools/resize", nil))
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("status: %d want 504", w.Code)
	}
	leftover, _ := os.ReadDir(toolTmpDir)
	if len(leftover) != 0 {
		t.Fatalf("temp files left behind: %v", leftover)
	}
}

// --- Smoke tests: one per remaining route, confirming it's wired and does
// something sane on trivial valid input (or the right error on invalid). ---

func TestHandleToolCrop_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 100, 100))
	req := httptest.NewRequest("POST", "/api/tools/crop?x=0&y=0&width=50&height=50", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolCrop)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolOptimize_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/optimize?level=2", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolOptimize)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolRotate_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/rotate?angle=90&flip=none", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolRotate)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolRotate_InvalidAngle(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/rotate?angle=45", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolRotate)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400", w.Code)
	}
}

func TestHandleToolReverse_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 3, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/reverse", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolReverse)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolSpeed_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 3, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/speed?factor=2", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolSpeed)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolSplit_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 3, 50, 50))
	req := httptest.NewRequest("POST", "/api/tools/split", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolSplit)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("content-type: %q want application/zip", ct)
	}
}

func TestHandleToolCaption_Smoke(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 100, 50))
	req := httptest.NewRequest("POST", "/api/tools/caption?text=hi&position=top&color=%23ffffff&size=13", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolCaption)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolCaption_EmptyTextRejected(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 2, 100, 50))
	req := httptest.NewRequest("POST", "/api/tools/caption?text=&position=top&color=%23ffffff&size=13", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolCaption)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400", w.Code)
	}
}

// multipartImageRequest builds a multipart body with one or more "file"
// fields holding PNG-encoded images.
func multipartImageRequest(t *testing.T, images [][]byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for i, data := range images {
		fw, err := mw.CreateFormFile("file", "img"+string(rune('0'+i))+".png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func encodeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHandleToolConvert_SingleImageSmoke(t *testing.T) {
	body, ct := multipartImageRequest(t, [][]byte{encodeTestPNG(t, 40, 40)})
	req := httptest.NewRequest("POST", "/api/tools/convert", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	if _, err := gif.DecodeAll(w.Body); err != nil {
		t.Fatalf("not a valid gif: %v", err)
	}
}

func TestHandleToolConvert_MultiImageSmoke(t *testing.T) {
	body, ct := multipartImageRequest(t, [][]byte{encodeTestPNG(t, 40, 40), encodeTestPNG(t, 40, 40)})
	req := httptest.NewRequest("POST", "/api/tools/convert?fps=5", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	g, err := gif.DecodeAll(w.Body)
	if err != nil {
		t.Fatalf("not a valid gif: %v", err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames: %d want 2", len(g.Image))
	}
}

func TestHandleToolConvert_OversizedImageRejected(t *testing.T) {
	body, ct := multipartImageRequest(t, [][]byte{encodeTestPNG(t, maxGifSide+10, 10)})
	req := httptest.NewRequest("POST", "/api/tools/convert", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400 body: %s", w.Code, w.Body.String())
	}
}

func TestHandleToolConvert_MissingFile(t *testing.T) {
	body, ct := multipartImageRequest(t, nil)
	req := httptest.NewRequest("POST", "/api/tools/convert", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400", w.Code)
	}
}

// --- C1: multipart body cap + explicit temp-file cleanup ---

func TestParseToolMultipart_CleansSpilledTempFiles(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "big.bin")
	if err != nil {
		t.Fatal(err)
	}
	// Bigger than toolMultipartMemThreshold, so ParseMultipartForm spills
	// the extra content to an on-disk temp file.
	big := bytes.Repeat([]byte{0xAB}, int(toolMultipartMemThreshold)+(1<<16))
	if _, err := fw.Write(big); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	bodyLen := buf.Len()

	req := httptest.NewRequest("POST", "/x", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	before, _ := os.ReadDir(os.TempDir())
	beforeNames := map[string]bool{}
	for _, e := range before {
		beforeNames[e.Name()] = true
	}

	cleanup, err := parseToolMultipart(req, int64(bodyLen)+4096)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()

	after, _ := os.ReadDir(os.TempDir())
	for _, e := range after {
		if beforeNames[e.Name()] {
			continue
		}
		if strings.HasPrefix(e.Name(), "multipart-") {
			t.Fatalf("leftover multipart temp file after cleanup: %s", e.Name())
		}
	}
}

func TestParseToolMultipart_RejectsOverBodyCap(t *testing.T) {
	body, ct := multipartGifRequest(t, "file", encodeTestGIF(t, 1, 50, 50))
	req := httptest.NewRequest("POST", "/x", body)
	req.Header.Set("Content-Type", ct)
	cleanup, err := parseToolMultipart(req, 4) // way under the body's actual size
	defer cleanup()
	if err == nil {
		t.Fatal("want error when body exceeds the MaxBytesReader cap")
	}
}

// --- C2 / I4: image(s)->GIF convert rejects oversized/over-budget input
// before decoding pixels ---

func TestHandleConvertImages_RejectsOversizedDimensionsBeforeDecode(t *testing.T) {
	// encodeTestPNG(maxGifSide+10, 10) already covers this via
	// TestHandleToolConvert_OversizedImageRejected above; this test name
	// documents explicitly which finding it proves (C2).
	body, ct := multipartImageRequest(t, [][]byte{encodeTestPNG(t, maxGifSide+10, 10)})
	req := httptest.NewRequest("POST", "/api/tools/convert", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400 body: %s", w.Code, w.Body.String())
	}
}

func TestHandleConvertImages_RejectsBatchPixelBudget(t *testing.T) {
	// Each image is within maxGifSide on its own, but len(files)*w*h must
	// also fit maxGifTotalPixels (I4) - use a frame size and count whose
	// product clearly exceeds it.
	w, h := maxGifSide, maxGifSide
	count := maxGifTotalPixels/(w*h) + 2
	if count > maxImagesCount {
		count = maxImagesCount
	}
	imgs := make([][]byte, count)
	png := encodeTestPNG(t, w, h)
	for i := range imgs {
		imgs[i] = png
	}
	body, ct := multipartImageRequest(t, imgs)
	req := httptest.NewRequest("POST", "/api/tools/convert", body)
	req.Header.Set("Content-Type", ct)
	wr := httptest.NewRecorder()
	withToolLimits(handleToolConvert)(wr, req)
	if wr.Code != http.StatusBadRequest {
		t.Fatalf("status: %d want 400 body: %s", wr.Code, wr.Body.String())
	}
}

// --- I2: video/image routing is case-insensitive on file extension ---

func TestLooksLikeVideo_CaseInsensitive(t *testing.T) {
	cases := map[string]bool{
		"clip.mp4":  true,
		"CLIP.MP4":  true,
		"Video.MOV": true,
		"photo.PNG": false,
		"photo.png": false,
	}
	for name, want := range cases {
		if got := looksLikeVideo(name); got != want {
			t.Errorf("looksLikeVideo(%q) = %v, want %v", name, got, want)
		}
	}
}
