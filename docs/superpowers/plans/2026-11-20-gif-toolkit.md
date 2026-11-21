# GIF Toolkit (backend API) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 9 GIF-processing HTTP endpoints to `cmd/syncserver` (convert video/image(s) to GIF, resize, crop, optimize, rotate/flip, reverse, speed, split, caption), public (no auth), each guarded by size/structure limits and a shared concurrency+timeout pool.

**Architecture:** Each tool is a pure Go function operating on a decoded `*gif.GIF` (or raw image/video for convert), unit-tested directly with in-memory fixtures — no HTTP needed to test the core logic. A thin HTTP handler layer decodes multipart input, validates against the shared limits, calls the pure function, and streams the result back. All 9 handlers are wrapped by one concurrency+timeout middleware instead of each reimplementing it.

**Tech Stack:** Go stdlib `image`, `image/gif`, `image/draw`, `archive/zip`, `os/exec` (ffmpeg/ffprobe), `net/http`. No new Go modules required except possibly a bitmap font for caption text (stdlib `golang.org/x/image/font/basicfont` — already an indirect dependency class used by `x/image`, confirm via `go.mod` in Task 1).

**Spec:** `docs/BA-gif-toolkit.md`, `docs/UC-gif-toolkit.md`, `docs/TC-gif-toolkit.md`

## Global Constraints

- `maxGifSide` = 1024 px, `maxGifFrames` = 500, `maxGifTotalPixels` = 64<<20 — existing constants in `cmd/syncserver/gifguard.go`; every GIF **output** must also satisfy these, not just input.
- `maxVideoBytes` = 20MB, `maxVideoDurationSec` = 30 — new, for `/api/tools/convert` video path.
- `maxImageBytes` = 20MB per image, `maxImagesCount` = 100 — new, for `/api/tools/convert` image path(s).
- `maxConcurrentToolJobs` = 2 — new pool, separate from the existing `gifDecodeSlots` used by the sync/share feature.
- `toolRequestTimeout` = 60 seconds, measured from request entry (including time spent waiting for a slot).
- All `/api/tools/*` routes are public: no `requireAuth` middleware.
- No file written to disk may outlive its request; anything written to `data/tmp/` must be removed via `defer`, success or failure.
- No logging of uploaded file contents — only size, type, and success/failure.

## Review Focus

- **Structurally malicious GIF input reused across every tool** (not just convert): a GIF whose header claims small frames but whose body encodes 10000 frames must be rejected by `checkGifSize`/`inspectGif` *before* `gif.DecodeAll`, for all of resize/crop/rotate/reverse/speed/split/caption/optimize — a reasonable person expects the same bomb protection on every endpoint, not only the one it was first built for. Tested in each tool's own task.
- **Video duration must be rejected before transcoding starts**, not after: `ffprobe` runs first, and a 60s video on a 30s limit never reaches `ffmpeg`. Tested in Task 10.
- **Output that itself violates the GIF limits** (e.g. resizing up past `maxGifSide`, or an image-list convert that produces more than `maxGifFrames` frames) must return a clean 400, not a truncated/corrupt file or a panic. Tested in Tasks 2 and 9.
- **Concurrency ceiling and timeout actually bound work**, including cleanup: a request that queues past `maxConcurrentToolJobs` either waits or gets 429; a request that runs past `toolRequestTimeout` is canceled via context and gets 504; any temp file from that canceled request is still removed. Tested in Task 11.
- **Zero/negative/empty parameters must 400, not panic or silently no-op into a crash**: `width=0`, `factor<=0`, empty caption text, a crop box outside the source frame. Tested in each tool's own task (2–8).

---

## File Structure

- Create `cmd/syncserver/limits.go` — all new constants from Global Constraints, plus the shared `toolJobSlots` channel and `acquireToolSlot`/`releaseToolSlot` helpers (mirrors the existing pattern in `gifguard.go`).
- Create `cmd/syncserver/gif_ops.go` — pure functions: `resizeGIF`, `cropGIF`, `optimizeGIF`, `rotateGIF`, `reverseGIF`, `speedGIF`, `splitGIFFrames`, `captionGIF`.
- Create `cmd/syncserver/gif_ops_test.go` — unit tests for the above, using small in-memory fixture GIFs built with a shared test helper.
- Create `cmd/syncserver/convert.go` — `imageToGIF`, `imagesToGIF`, `videoToGIF` (ffmpeg/ffprobe via `os/exec`).
- Create `cmd/syncserver/convert_test.go` — unit tests for the above.
- Create `cmd/syncserver/tools_handlers.go` — 9 HTTP handlers (`handleToolConvert`, `handleToolResize`, `handleToolCrop`, `handleToolOptimize`, `handleToolRotate`, `handleToolReverse`, `handleToolSpeed`, `handleToolSplit`, `handleToolCaption`) plus the `withToolLimits` middleware (concurrency + timeout, Task 11) and a shared `decodeAndValidateGIF(r *http.Request, field string) (*gif.GIF, error)` helper.
- Create `cmd/syncserver/tools_handlers_test.go` — `httptest`-based tests per handler.
- Modify `cmd/syncserver/main.go` — register the 9 routes (wrapped by `withToolLimits`), check `ffmpeg`/`ffprobe` are on `PATH` at startup (log warning, don't crash), ensure `data/tmp/` exists.

---

### Task 1: Limits, shared slot pool, and test fixture helper

**Files:**
- Create: `cmd/syncserver/limits.go`
- Create: `cmd/syncserver/testfixtures_test.go`
- Test: `cmd/syncserver/limits_test.go`

**Interfaces:**
- Produces: constants `maxVideoBytes int64`, `maxVideoDurationSec int`, `maxImageBytes int64`, `maxImagesCount int`, `maxConcurrentToolJobs int`, `toolRequestTimeout time.Duration`; functions `acquireToolSlot(ctx context.Context) error`, `releaseToolSlot()`; test helper `makeTestGIF(t *testing.T, frames int, w, h int) *gif.GIF` (solid-color frames, used by every later test file).

- [ ] **Step 1: Write the failing test for the slot pool**

```go
func TestToolSlotPool_LimitsConcurrency(t *testing.T) {
    ctx := context.Background()
    for i := 0; i < maxConcurrentToolJobs; i++ {
        if err := acquireToolSlot(ctx); err != nil {
            t.Fatalf("acquire %d: %v", i, err)
        }
    }
    cctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
    defer cancel()
    if err := acquireToolSlot(cctx); err == nil {
        t.Fatal("expected acquire to block past the limit and time out")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/syncserver/... -run TestToolSlotPool_LimitsConcurrency -v`
Expected: FAIL (`maxConcurrentToolJobs`/`acquireToolSlot` undefined)

- [ ] **Step 3: Implement `limits.go`**

Constants exactly as listed in Global Constraints. `acquireToolSlot`/`releaseToolSlot` mirror `acquireGifSlot`/`releaseGifSlot` in `gifguard.go` (buffered channel of size `maxConcurrentToolJobs`, select on channel send vs `ctx.Done()`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/syncserver/... -run TestToolSlotPool_LimitsConcurrency -v`
Expected: PASS

- [ ] **Step 5: Implement `makeTestGIF` in `testfixtures_test.go`**

Builds a `*gif.GIF` with `frames` images of size `w x h`, each a single solid `color.Palette` frame, `Delay` = 10 (centiseconds) each. No test step needed by itself — verified indirectly when Task 2 uses it.

- [ ] **Step 6: Commit**

```bash
git add cmd/syncserver/limits.go cmd/syncserver/limits_test.go cmd/syncserver/testfixtures_test.go
git commit -m "feat(syncserver): add gif-toolkit limits and shared slot pool"
```

---

### Task 2: Resize

**Files:**
- Create/modify: `cmd/syncserver/gif_ops.go` (add `resizeGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Consumes: `makeTestGIF` (Task 1), `maxGifSide` (`gifguard.go`).
- Produces: `func resizeGIF(g *gif.GIF, width, height int) (*gif.GIF, error)` — used by the resize handler in Task 11's handler file.

- [ ] **Step 1: Write failing tests**

```go
func TestResizeGIF_Basic(t *testing.T) {
    g := makeTestGIF(t, 3, 300, 300)
    out, err := resizeGIF(g, 150, 150)
    if err != nil { t.Fatal(err) }
    if len(out.Image) != 3 { t.Fatalf("frames: got %d want 3", len(out.Image)) }
    b := out.Image[0].Bounds()
    if b.Dx() != 150 || b.Dy() != 150 { t.Fatalf("size: got %dx%d want 150x150", b.Dx(), b.Dy()) }
}

func TestResizeGIF_RejectsZeroOrNegative(t *testing.T) {
    g := makeTestGIF(t, 1, 100, 100)
    if _, err := resizeGIF(g, 0, 100); err == nil { t.Fatal("want error for width=0") }
    if _, err := resizeGIF(g, 100, -5); err == nil { t.Fatal("want error for negative height") }
}

func TestResizeGIF_RejectsOverMaxSide(t *testing.T) {
    g := makeTestGIF(t, 1, 100, 100)
    if _, err := resizeGIF(g, maxGifSide+1, 100); err == nil { t.Fatal("want error over maxGifSide") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestResizeGIF -v`
Expected: FAIL (`resizeGIF` undefined)

- [ ] **Step 3: Implement `resizeGIF`**

Validate `width > 0 && height > 0 && width <= maxGifSide && height <= maxGifSide` first, return a plain `error` otherwise. Then resize each `*image.Paletted` frame (nearest-neighbor via `image/draw.NearestNeighbor.Scale`, reusing each frame's existing palette), preserving `Delay`/`Disposal` per frame.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestResizeGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add resizeGIF"
```

---

### Task 3: Crop

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `cropGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Consumes: `makeTestGIF`.
- Produces: `func cropGIF(g *gif.GIF, x, y, w, h int) (*gif.GIF, error)`.

- [ ] **Step 1: Write failing tests**

```go
func TestCropGIF_Basic(t *testing.T) {
    g := makeTestGIF(t, 2, 300, 300)
    out, err := cropGIF(g, 50, 50, 100, 100)
    if err != nil { t.Fatal(err) }
    b := out.Image[0].Bounds()
    if b.Dx() != 100 || b.Dy() != 100 { t.Fatalf("size: got %dx%d want 100x100", b.Dx(), b.Dy()) }
}

func TestCropGIF_RejectsOutOfBounds(t *testing.T) {
    g := makeTestGIF(t, 1, 300, 300)
    if _, err := cropGIF(g, 250, 250, 100, 100); err == nil { t.Fatal("want error: crop box exceeds source") }
}

func TestCropGIF_RejectsZeroSize(t *testing.T) {
    g := makeTestGIF(t, 1, 300, 300)
    if _, err := cropGIF(g, 10, 10, 0, 50); err == nil { t.Fatal("want error: w=0") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestCropGIF -v`
Expected: FAIL

- [ ] **Step 3: Implement `cropGIF`**

Validate `w > 0 && h > 0 && x >= 0 && y >= 0 && x+w <= srcWidth && y+h <= srcHeight` against `g.Image[0].Bounds()` (the logical screen); crop every frame to the same `image.Rect(x, y, x+w, y+h)` via `(*image.Paletted).SubImage`, then re-materialize each as its own `*image.Paletted` at origin `(0,0)` so downstream GIF encoding is unaffected by the sub-image offset.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestCropGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add cropGIF"
```

---

### Task 4: Rotate / Flip

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `rotateGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Produces: `func rotateGIF(g *gif.GIF, angle int, flip string) (*gif.GIF, error)` — `angle` one of `0, 90, 180, 270`; `flip` one of `"none", "h", "v"`.

- [ ] **Step 1: Write failing tests**

```go
func TestRotateGIF_90SwapsDimensions(t *testing.T) {
    g := makeTestGIF(t, 1, 200, 100)
    out, err := rotateGIF(g, 90, "none")
    if err != nil { t.Fatal(err) }
    b := out.Image[0].Bounds()
    if b.Dx() != 100 || b.Dy() != 200 { t.Fatalf("size: got %dx%d want 100x200", b.Dx(), b.Dy()) }
}

func TestRotateGIF_NoopPassesThrough(t *testing.T) {
    g := makeTestGIF(t, 1, 200, 100)
    out, err := rotateGIF(g, 0, "none")
    if err != nil { t.Fatal(err) }
    b := out.Image[0].Bounds()
    if b.Dx() != 200 || b.Dy() != 100 { t.Fatal("size changed on no-op") }
}

func TestRotateGIF_RejectsInvalidAngle(t *testing.T) {
    g := makeTestGIF(t, 1, 100, 100)
    if _, err := rotateGIF(g, 45, "none"); err == nil { t.Fatal("want error for angle=45") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestRotateGIF -v`
Expected: FAIL

- [ ] **Step 3: Implement `rotateGIF`**

Validate `angle` is one of the 4 allowed values and `flip` is one of the 3 allowed values first. Apply flip then rotation per frame via pixel-index remapping into a new `*image.Paletted` (same palette, swapped bounds for 90/270).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestRotateGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add rotateGIF"
```

---

### Task 5: Reverse

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `reverseGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Produces: `func reverseGIF(g *gif.GIF) *gif.GIF` (no error — always succeeds, including the 1-frame no-op case).

- [ ] **Step 1: Write failing tests**

```go
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
    if len(out.Image) != 1 { t.Fatal("frame count changed") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestReverseGIF -v`
Expected: FAIL

- [ ] **Step 3: Implement `reverseGIF`**

Reverse `g.Image`, `g.Delay`, and `g.Disposal` slices in lockstep (new slices; don't mutate the input).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestReverseGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add reverseGIF"
```

---

### Task 6: Speed

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `speedGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Produces: `func speedGIF(g *gif.GIF, factor float64) (*gif.GIF, error)`.

- [ ] **Step 1: Write failing tests**

```go
func TestSpeedGIF_ScalesDelay(t *testing.T) {
    g := makeTestGIF(t, 1, 50, 50)
    g.Delay = []int{10}
    out, err := speedGIF(g, 2.0)
    if err != nil { t.Fatal(err) }
    if out.Delay[0] != 5 { t.Fatalf("delay: got %d want 5", out.Delay[0]) }
}

func TestSpeedGIF_RejectsNonPositiveFactor(t *testing.T) {
    g := makeTestGIF(t, 1, 50, 50)
    if _, err := speedGIF(g, 0); err == nil { t.Fatal("want error for factor=0") }
    if _, err := speedGIF(g, -1); err == nil { t.Fatal("want error for factor<0") }
}

func TestSpeedGIF_ClampsToMinimumDelay(t *testing.T) {
    g := makeTestGIF(t, 1, 50, 50)
    g.Delay = []int{2}
    out, err := speedGIF(g, 100)
    if err != nil { t.Fatal(err) }
    if out.Delay[0] < 2 { t.Fatalf("delay below GIF minimum: %d", out.Delay[0]) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestSpeedGIF -v`
Expected: FAIL

- [ ] **Step 3: Implement `speedGIF`**

Validate `factor > 0`. New delay per frame = `max(2, round(delay / factor))` (2 centiseconds is the de facto minimum most GIF renderers honor — document this as the clamp floor in a one-line comment, since the spec leaves the exact floor to the implementer).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestSpeedGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add speedGIF"
```

---

### Task 7: Optimize

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `optimizeGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Produces: `func optimizeGIF(g *gif.GIF, level int) (*gif.GIF, error)` — `level` 1..3; higher level = fewer palette colors.

- [ ] **Step 1: Write failing tests**

```go
func TestOptimizeGIF_ReducesEncodedSize(t *testing.T) {
    g := makeTestGIF(t, 5, 200, 200)
    var before bytes.Buffer
    if err := gif.EncodeAll(&before, g); err != nil { t.Fatal(err) }
    out, err := optimizeGIF(g, 3)
    if err != nil { t.Fatal(err) }
    var after bytes.Buffer
    if err := gif.EncodeAll(&after, out); err != nil { t.Fatal(err) }
    if after.Len() >= before.Len() {
        t.Fatalf("optimize did not shrink: before=%d after=%d", before.Len(), after.Len())
    }
}

func TestOptimizeGIF_RejectsInvalidLevel(t *testing.T) {
    g := makeTestGIF(t, 1, 50, 50)
    if _, err := optimizeGIF(g, 0); err == nil { t.Fatal("want error for level=0") }
    if _, err := optimizeGIF(g, 4); err == nil { t.Fatal("want error for level=4") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run TestOptimizeGIF -v`
Expected: FAIL

- [ ] **Step 3: Implement `optimizeGIF`**

Validate `level` in `{1,2,3}`. Map level to a target palette size (e.g. 1→128, 2→64, 3→32 colors) and re-quantize each frame with `image/draw.Draw` into a new `*image.Paletted` built via `color.Palette` reduction (stdlib `image/color/palette.Plan9` subsampled to the target size, or a simple median-cut if already hand-rolling — implementer's choice, no new dependency required).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run TestOptimizeGIF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add optimizeGIF"
```

---

### Task 8: Split and Caption

**Files:**
- Modify: `cmd/syncserver/gif_ops.go` (add `splitGIFFrames`, `captionGIF`)
- Test: `cmd/syncserver/gif_ops_test.go`

**Interfaces:**
- Produces: `type zippedFrame struct { Name string; PNG []byte }`; `func splitGIFFrames(g *gif.GIF) ([]zippedFrame, error)`; `func captionGIF(g *gif.GIF, text, position, hexColor string, size int) (*gif.GIF, error)` — `position` one of `"top", "bottom"`.

- [ ] **Step 1: Write failing tests**

```go
func TestSplitGIFFrames_Count(t *testing.T) {
    g := makeTestGIF(t, 4, 50, 50)
    frames, err := splitGIFFrames(g)
    if err != nil { t.Fatal(err) }
    if len(frames) != 4 { t.Fatalf("frames: got %d want 4", len(frames)) }
    if _, err := png.Decode(bytes.NewReader(frames[0].PNG)); err != nil {
        t.Fatalf("frame 0 not valid PNG: %v", err)
    }
}

func TestCaptionGIF_RejectsEmptyText(t *testing.T) {
    g := makeTestGIF(t, 1, 100, 50)
    if _, err := captionGIF(g, "", "top", "#ffffff", 16); err == nil {
        t.Fatal("want error for empty text")
    }
}

func TestCaptionGIF_AppliesToAllFrames(t *testing.T) {
    g := makeTestGIF(t, 3, 100, 50)
    out, err := captionGIF(g, "Hi", "bottom", "#ffffff", 16)
    if err != nil { t.Fatal(err) }
    if len(out.Image) != 3 { t.Fatal("frame count changed") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run "TestSplitGIFFrames|TestCaptionGIF" -v`
Expected: FAIL

- [ ] **Step 3: Implement `splitGIFFrames`**

For each frame, draw it onto an `image.RGBA` (frames may be partial/disposal-based) and `png.Encode` into `zippedFrame{Name: fmt.Sprintf("frame_%03d.png", i+1)}`.

- [ ] **Step 4: Implement `captionGIF`**

Validate `strings.TrimSpace(text) != ""`. Parse `hexColor` (`#rrggbb`) into `color.RGBA`. Draw the text onto every frame at the same pixel position using `golang.org/x/image/font/basicfont.Face7x13` (scale by nearest multiple for `size`) via `golang.org/x/image/font` + `image/draw`; `position` picks top vs bottom margin. If text width exceeds frame width, wrap onto a second line rather than clipping.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run "TestSplitGIFFrames|TestCaptionGIF" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/syncserver/gif_ops.go cmd/syncserver/gif_ops_test.go
git commit -m "feat(syncserver): add splitGIFFrames and captionGIF"
```

---

### Task 9: Convert — image(s) to GIF

**Files:**
- Create: `cmd/syncserver/convert.go`
- Create: `cmd/syncserver/convert_test.go`

**Interfaces:**
- Consumes: `maxImageBytes`, `maxImagesCount`, `maxGifFrames`, `maxGifSide` (Task 1 / `gifguard.go`).
- Produces: `func imageToGIF(img image.Image) *gif.GIF`; `func imagesToGIF(images []image.Image, delayMs int) (*gif.GIF, error)`.

- [ ] **Step 1: Write failing tests**

```go
func TestImageToGIF_SingleFrame(t *testing.T) {
    img := image.NewRGBA(image.Rect(0, 0, 20, 20))
    out := imageToGIF(img)
    if len(out.Image) != 1 { t.Fatal("want 1 frame") }
}

func TestImagesToGIF_OrderAndDelay(t *testing.T) {
    imgs := []image.Image{
        image.NewRGBA(image.Rect(0, 0, 20, 20)),
        image.NewRGBA(image.Rect(0, 0, 20, 20)),
    }
    out, err := imagesToGIF(imgs, 200)
    if err != nil { t.Fatal(err) }
    if len(out.Image) != 2 || out.Delay[0] != 20 { // 200ms -> 20 centiseconds
        t.Fatalf("got frames=%d delay=%v", len(out.Image), out.Delay)
    }
}

func TestImagesToGIF_RejectsOverCount(t *testing.T) {
    imgs := make([]image.Image, maxImagesCount+1)
    for i := range imgs { imgs[i] = image.NewRGBA(image.Rect(0, 0, 5, 5)) }
    if _, err := imagesToGIF(imgs, 100); err == nil {
        t.Fatal("want error over maxImagesCount")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run "TestImageToGIF|TestImagesToGIF" -v`
Expected: FAIL

- [ ] **Step 3: Implement `imageToGIF` and `imagesToGIF`**

`imageToGIF`: quantize the image into a `*image.Paletted` (median-cut or `palette.Plan9`), wrap as a 1-frame `*gif.GIF`, `Delay: []int{0}`. `imagesToGIF`: validate `len(images) <= maxImagesCount` and `len(images) > 0`; resize every image to the first image's bounds (nearest-neighbor) before quantizing, so mismatched input sizes don't panic; `delayMs/10` centiseconds per frame (minimum 2, same floor as Task 6).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run "TestImageToGIF|TestImagesToGIF" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/convert.go cmd/syncserver/convert_test.go
git commit -m "feat(syncserver): add image(s)-to-gif conversion"
```

---

### Task 10: Convert — video to GIF

**Files:**
- Modify: `cmd/syncserver/convert.go` (add `videoToGIF`, `probeVideoDuration`, `ffmpegAvailable`)
- Modify: `cmd/syncserver/convert_test.go`

**Interfaces:**
- Consumes: `maxVideoDurationSec` (Task 1).
- Produces: `func ffmpegAvailable() bool`; `func probeVideoDuration(ctx context.Context, path string) (time.Duration, error)`; `func videoToGIF(ctx context.Context, path string, fps int) ([]byte, error)` — `path` is a temp file already on disk (the HTTP handler in Task 11 owns writing/deleting it).

- [ ] **Step 1: Write failing tests**

```go
func TestFfmpegAvailable_ReflectsPath(t *testing.T) {
    if !ffmpegAvailable() {
        t.Skip("ffmpeg not installed in this environment; function exercised in Task 11's handler test with a stub")
    }
}

func TestProbeVideoDuration_RejectsMissingFile(t *testing.T) {
    if !ffmpegAvailable() { t.Skip("ffmpeg not available") }
    _, err := probeVideoDuration(context.Background(), "/no/such/file.mp4")
    if err == nil { t.Fatal("want error for missing file") }
}
```

Note: full video→GIF happy-path is covered by an integration test gated behind `ffmpegAvailable()` (skips cleanly in CI/dev machines without ffmpeg), using a tiny generated test clip (`ffmpeg -f lavfi -i color=c=red:size=32x32:d=1 testdata/tiny.mp4`, committed once as a fixture) — add this as a third test:

```go
func TestVideoToGIF_HappyPath(t *testing.T) {
    if !ffmpegAvailable() { t.Skip("ffmpeg not available") }
    data, err := videoToGIF(context.Background(), "testdata/tiny.mp4", 10)
    if err != nil { t.Fatal(err) }
    if _, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
        t.Fatalf("output not a valid gif: %v", err)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run "TestFfmpegAvailable|TestProbeVideoDuration|TestVideoToGIF" -v`
Expected: FAIL (functions undefined)

- [ ] **Step 3: Generate the fixture clip**

Run: `ffmpeg -f lavfi -i color=c=red:size=32x32:d=1 -t 1 cmd/syncserver/testdata/tiny.mp4` (one-time, commit the resulting ~1KB file).

- [ ] **Step 4: Implement `ffmpegAvailable`, `probeVideoDuration`, `videoToGIF`**

`ffmpegAvailable`: `exec.LookPath("ffmpeg")` and `exec.LookPath("ffprobe")` both succeed. `probeVideoDuration`: run `ffprobe -v error -show_entries format=duration -of csv=p=0 <path>` via `exec.CommandContext`, parse the float-seconds stdout. `videoToGIF`: if `probeVideoDuration` result exceeds `maxVideoDurationSec`, return an error without invoking `ffmpeg`; otherwise run `ffmpeg -i <path> -vf fps=<fps> -f gif -` via `exec.CommandContext` and capture stdout.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -run "TestFfmpegAvailable|TestProbeVideoDuration|TestVideoToGIF" -v`
Expected: PASS (or clean SKIP where ffmpeg is absent)

- [ ] **Step 6: Commit**

```bash
git add cmd/syncserver/convert.go cmd/syncserver/convert_test.go cmd/syncserver/testdata/tiny.mp4
git commit -m "feat(syncserver): add video-to-gif conversion via ffmpeg"
```

---

### Task 11: HTTP handlers, concurrency+timeout middleware, and route wiring

**Files:**
- Create: `cmd/syncserver/tools_handlers.go`
- Create: `cmd/syncserver/tools_handlers_test.go`
- Modify: `cmd/syncserver/main.go` — register routes, startup `ffmpeg`/`ffprobe` check, ensure `data/tmp/` exists.

**Interfaces:**
- Consumes: every function produced in Tasks 2–10; `acquireToolSlot`/`releaseToolSlot`, `toolRequestTimeout` (Task 1); `checkGifSize`, `errGifTooBig` (`gifguard.go`).
- Produces: `func withToolLimits(h http.HandlerFunc) http.HandlerFunc` (applies timeout + slot acquisition, returns 429/504 as appropriate); `func decodeAndValidateGIF(r *http.Request, field string) (*gif.GIF, error)`; the 9 `handleTool*` functions, each `func(w http.ResponseWriter, r *http.Request)`.

- [ ] **Step 1: Write failing handler tests (representative subset, covering happy path + each Review Focus item)**

```go
func TestHandleToolResize_HappyPath(t *testing.T) {
    body, ct := multipartGifRequest(t, "file", makeTestGIF(t, 2, 300, 300))
    req := httptest.NewRequest("POST", "/api/tools/resize?width=150&height=150", body)
    req.Header.Set("Content-Type", ct)
    w := httptest.NewRecorder()
    withToolLimits(handleToolResize)(w, req)
    if w.Code != http.StatusOK { t.Fatalf("status: %d body: %s", w.Code, w.Body.String()) }
    if _, err := gif.DecodeAll(w.Body); err != nil { t.Fatalf("not a valid gif: %v", err) }
}

func TestHandleToolResize_RejectsGifBomb(t *testing.T) {
    body, ct := multipartBombRequest(t, "file") // header claims small frames, body lies about frame count per gifguard_test.go's existing fixture pattern
    req := httptest.NewRequest("POST", "/api/tools/resize?width=50&height=50", body)
    req.Header.Set("Content-Type", ct)
    w := httptest.NewRecorder()
    withToolLimits(handleToolResize)(w, req)
    if w.Code != http.StatusBadRequest { t.Fatalf("status: %d want 400", w.Code) }
}

func TestWithToolLimits_RejectsOverConcurrency(t *testing.T) {
    slow := func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(200 * time.Millisecond)
        w.WriteHeader(http.StatusOK)
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
    for i := 0; i < maxConcurrentToolJobs+1; i++ { codes[<-results]++ }
    if codes[http.StatusTooManyRequests] == 0 {
        t.Fatalf("want at least one 429 among: %v", codes)
    }
}

func TestWithToolLimits_TimesOutAndCleansTempFiles(t *testing.T) {
    hang := func(w http.ResponseWriter, r *http.Request) {
        tmp, _ := os.CreateTemp(toolTmpDir, "hang-*")
        defer os.Remove(tmp.Name())
        <-r.Context().Done()
    }
    saved := toolRequestTimeout
    toolRequestTimeout = 20 * time.Millisecond
    defer func() { toolRequestTimeout = saved }()
    w := httptest.NewRecorder()
    withToolLimits(hang)(w, httptest.NewRequest("POST", "/api/tools/resize", nil))
    if w.Code != http.StatusGatewayTimeout { t.Fatalf("status: %d want 504", w.Code) }
    leftover, _ := os.ReadDir(toolTmpDir)
    if len(leftover) != 0 { t.Fatalf("temp files left behind: %v", leftover) }
}
```

(`multipartGifRequest`/`multipartBombRequest` are small test-only helpers added in `tools_handlers_test.go`, reusing `gif.EncodeAll` and the existing bomb-fixture bytes already present in `gifguard_test.go`.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/syncserver/... -run "TestHandleToolResize|TestWithToolLimits" -v`
Expected: FAIL

- [ ] **Step 3: Implement `withToolLimits` and `decodeAndValidateGIF`**

`withToolLimits`: `ctx, cancel := context.WithTimeout(r.Context(), toolRequestTimeout)`; replace `r = r.WithContext(ctx)`; `if err := acquireToolSlot(ctx); err != nil { write 429 if ctx still fresh at call time... }` — concretely: attempt `acquireToolSlot(ctx)`; if it returns `ctx.Err()` because the deadline passed while waiting, respond 504; if the slot pool itself is the reason (first `maxConcurrentToolJobs` requests already hold slots and this one's context expires while queued), still respond 429 for queued-and-rejected vs 504 for ran-too-long — implementer disambiguates via a short dedicated wait context: first try a non-blocking `select` on the slot with a cap of ~1s wrapped in `errPoolFull`; on `errPoolFull` return 429; otherwise proceed and let the outer `toolRequestTimeout` context produce 504 if the handler itself overruns. Always `defer releaseToolSlot()` once acquired, and `defer cancel()`. `decodeAndValidateGIF`: read the named multipart field, call `checkGifSize` then `gif.DecodeAll`.

- [ ] **Step 4: Implement the 9 `handleTool*` functions**

Each: parse query/form params relevant to its tool (per the API table in `docs/BA-gif-toolkit.md` §3), call `decodeAndValidateGIF` (all except `handleToolConvert`, which instead reads video/image(s) per its `kind` field), call the corresponding pure function from Tasks 2–10, validate the result's dimensions/frame count against `maxGifSide`/`maxGifFrames`/`maxGifTotalPixels` via `checkGifSize` on the re-encoded bytes before writing the response, `w.Header().Set("Content-Type", "image/gif")` (or `application/zip` for split), write the bytes. On any validation error: 400 with a plain-text message; ffmpeg-missing: 503.

- [ ] **Step 5: Wire routes and startup checks in `main.go`**

Register `POST /api/tools/{convert,resize,crop,optimize,rotate,reverse,speed,split,caption}`, each wrapped in `withToolLimits(...)`, outside any `requireAuth` group. At startup: `if !ffmpegAvailable() { log.Println("warning: ffmpeg/ffprobe not found, /api/tools/convert video path will return 503") }`; `os.MkdirAll(toolTmpDir, 0o755)`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./cmd/syncserver/... -v`
Expected: PASS (all tests in the package, including Tasks 1–10)

- [ ] **Step 7: Commit**

```bash
git add cmd/syncserver/tools_handlers.go cmd/syncserver/tools_handlers_test.go cmd/syncserver/main.go
git commit -m "feat(syncserver): wire gif-toolkit http handlers and routes"
```

---

## Manual Verification (after Task 11, before calling the backend done)

Not a substitute for the tests above — a quick end-to-end sanity check against a *running* server, since `go test` never starts `syncserver` itself:

- [ ] `cd cmd/syncserver && go run .` (ensure `data/` exists first)
- [ ] For each of the 9 endpoints, one `curl -F file=@sample.gif http://localhost:9090/api/tools/<name>?<params> -o out.gif` (or `.zip` for split) with a real small GIF, and confirm the output opens in an image viewer / unzips correctly.
- [ ] One `curl -F file=@sample.mp4 http://localhost:9090/api/tools/convert` with a short real video, confirm a playable GIF comes back.
- [ ] One deliberately-oversized/invalid input per endpoint (e.g. a 2000x2000 resize target, a crop box outside bounds, an empty caption) — confirm 400 with a readable message, not a stack trace or hang.
