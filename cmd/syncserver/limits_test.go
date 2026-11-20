package main

import (
	"context"
	"testing"
	"time"
)

func TestToolSlotPool_LimitsConcurrency(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < maxConcurrentToolJobs; i++ {
		if err := acquireToolSlot(ctx); err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
	}
	t.Cleanup(func() {
		for i := 0; i < maxConcurrentToolJobs; i++ {
			releaseToolSlot()
		}
	})
	cctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := acquireToolSlot(cctx); err == nil {
		t.Fatal("expected acquire to block past the limit and time out")
	}
}
