package main

import (
	"context"
	"time"
)

// Limits for the GIF toolkit (/api/tools/*). These routes are public, so
// every input is capped before it is read into memory or written to disk.
const (
	maxVideoBytes       int64 = 20 << 20 // upload size for the video path
	maxVideoDurationSec       = 30       // checked with ffprobe before ffmpeg runs
	maxImageBytes       int64 = 20 << 20 // per image
	maxImagesCount            = 100      // images combined into one GIF

	// maxImagesTotalBytes caps the whole multipart body for the
	// image(s)->GIF convert path. maxImagesCount * maxImageBytes would in
	// principle allow a ~2GB body; this is a much tighter, still generous
	// total that a real multi-image batch should never need.
	maxImagesTotalBytes int64 = 200 << 20 // 200MB combined

	// maxConvertBodyBytes is the http.MaxBytesReader cap for
	// /api/tools/convert's whole request body, shared by both its video and
	// image(s) sub-paths (the handler doesn't know which one it is until
	// after parsing). It's the larger of the two per-path totals; the
	// per-file/per-video checks further down still apply on top of it.
	maxConvertBodyBytes int64 = maxImagesTotalBytes

	// toolMultipartMemThreshold is the "maxMemory" ParseMultipartForm uses
	// for every /api/tools/* multipart body: how much of it is kept in
	// memory before the rest spills to on-disk temp files. The real upload
	// size limit is the http.MaxBytesReader cap applied to the body first
	// (maxGifBytes / maxConvertBodyBytes) - this threshold only controls the
	// memory/disk split, not the total allowed size. Spilled temp files are
	// always removed explicitly (see parseToolMultipart in
	// tools_handlers.go), since net/http's automatic multipart-temp-file
	// cleanup only runs for the original *http.Request, not the
	// context-copied one withToolLimits hands to handlers.
	toolMultipartMemThreshold = 1 << 20 // 1MB
)

// toolRequestTimeout is a var so tests can shorten it.
var toolRequestTimeout = 60 * time.Second

// Separate from gifDecodeSlots so the toolkit and the sync/share feature
// do not compete for the same slots.
const maxConcurrentToolJobs = 2

var toolJobSlots = make(chan struct{}, maxConcurrentToolJobs)

func acquireToolSlot(ctx context.Context) error {
	select {
	case toolJobSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseToolSlot() { <-toolJobSlots }
