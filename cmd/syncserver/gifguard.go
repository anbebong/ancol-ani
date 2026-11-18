package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
)

// GIF "decompression bomb" guard.
//
// An upload is capped at maxGifBytes (5MB) compressed, but image/gif's
// DecodeAll inflates every frame into memory: a small, highly compressible
// file can declare huge frames or thousands of them and expand to gigabytes
// - enough to get the whole Termux process (sync server, Ancha and the
// tunnel) killed on the phone this runs on. inspectGif walks the GIF block
// structure WITHOUT decompressing anything and rejects files whose decoded
// size would be too large, before DecodeAll ever runs.
const (
	maxGifSide        = 1024     // logical screen and every frame, per side
	maxGifFrames      = 500      // image descriptors
	maxGifTotalPixels = 64 << 20 // sum of frame w*h (~64MB as paletted)
)

var errGifTooBig = httpErr{fmt.Sprintf("GIF qua lon: toi da %dx%d, %d frame", maxGifSide, maxGifSide, maxGifFrames)}

type gifStats struct {
	width, height int
	frames        int
	totalPixels   int64
}

var errGifMalformed = errors.New("malformed gif")

func inspectGif(b []byte) (gifStats, error) {
	var st gifStats
	p := 0
	need := func(n int) bool { return p+n <= len(b) }
	skipSubBlocks := func() bool {
		for {
			if !need(1) {
				return false
			}
			size := int(b[p])
			p++
			if size == 0 {
				return true
			}
			if !need(size) {
				return false
			}
			p += size
		}
	}
	colorTableSize := func(packed byte) int {
		if packed&0x80 == 0 {
			return 0
		}
		return 3 * (1 << ((packed & 0x07) + 1))
	}

	if !need(13) || (string(b[0:6]) != "GIF87a" && string(b[0:6]) != "GIF89a") {
		return st, errGifMalformed
	}
	st.width = int(binary.LittleEndian.Uint16(b[6:8]))
	st.height = int(binary.LittleEndian.Uint16(b[8:10]))
	packed := b[10]
	p = 13
	if n := colorTableSize(packed); n > 0 {
		if !need(n) {
			return st, errGifMalformed
		}
		p += n
	}

	for {
		if !need(1) {
			return st, errGifMalformed // no trailer
		}
		block := b[p]
		p++
		switch block {
		case 0x21: // extension: label + sub-blocks
			if !need(1) {
				return st, errGifMalformed
			}
			p++
			if !skipSubBlocks() {
				return st, errGifMalformed
			}
		case 0x2C: // image descriptor
			if !need(9) {
				return st, errGifMalformed
			}
			w := int(binary.LittleEndian.Uint16(b[p+4 : p+6]))
			h := int(binary.LittleEndian.Uint16(b[p+6 : p+8]))
			lpacked := b[p+8]
			p += 9
			if n := colorTableSize(lpacked); n > 0 {
				if !need(n) {
					return st, errGifMalformed
				}
				p += n
			}
			if !need(1) { // LZW minimum code size
				return st, errGifMalformed
			}
			p++
			if !skipSubBlocks() {
				return st, errGifMalformed
			}
			st.frames++
			st.totalPixels += int64(w) * int64(h)
			if w > maxGifSide || h > maxGifSide || st.frames > maxGifFrames || st.totalPixels > maxGifTotalPixels {
				return st, errGifTooBig
			}
		case 0x3B: // trailer
			return st, nil
		default:
			return st, errGifMalformed
		}
	}
}

// checkGifSize returns nil if the GIF is safe to fully decode.
func checkGifSize(b []byte) error {
	st, err := inspectGif(b)
	if err == errGifTooBig {
		return errGifTooBig
	}
	if err != nil {
		return errInvalidGif
	}
	if st.width > maxGifSide || st.height > maxGifSide {
		return errGifTooBig
	}
	if st.frames == 0 {
		return errInvalidGif
	}
	return nil
}

// Even with the size guard, decoding + re-encoding a large GIF takes a lot
// of CPU and RAM on a phone, so only a couple run at once; extra uploads
// wait their turn (or give up when the client disconnects).
var gifDecodeSlots = make(chan struct{}, 2)

func acquireGifSlot(ctx context.Context) error {
	select {
	case gifDecodeSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseGifSlot() { <-gifDecodeSlots }
