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
