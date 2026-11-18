import { parseGIF, decompressFrames } from 'gifuct-js';

// Chromium's built-in <img> GIF animation can stall permanently after
// looping a while on some encodes (reproduced even in a plain Chrome tab).
// Decoding frames ourselves with gifuct-js and drawing them on a canvas on
// our own timer avoids that native animation engine entirely - used for
// every GIF this app displays, not just the main preview.
//
// getBuffer is a () => Promise<ArrayBuffer> so callers can fetch however
// they need to (plain fetch for the local /gif endpoint, an authenticated
// fetch with a Bearer token for a remote syncserver image, ...).
export function playGif(canvas, getBuffer, isCurrent) {
    const ctx = canvas.getContext('2d');
    (async () => {
        const buffer = await getBuffer();
        if (!isCurrent()) return;
        const gif = parseGIF(buffer);
        const frames = decompressFrames(gif, true);
        if (!frames.length) return;

        canvas.width = gif.lsd.width;
        canvas.height = gif.lsd.height;

        const full = ctx.createImageData(gif.lsd.width, gif.lsd.height);
        let previous = null;
        let index = 0;

        function draw() {
            if (!isCurrent()) return;
            const frame = frames[index];
            if (frame.disposalType === 2) {
                full.data.fill(0);
            } else if (frame.disposalType === 3 && previous) {
                full.data.set(previous);
            }

            const { width, height, top, left } = frame.dims;
            const patch = frame.patch;
            for (let y = 0; y < height; y++) {
                for (let x = 0; x < width; x++) {
                    const srcIdx = (y * width + x) * 4;
                    if (patch[srcIdx + 3] === 0) continue;
                    const dstIdx = ((top + y) * gif.lsd.width + (left + x)) * 4;
                    full.data[dstIdx] = patch[srcIdx];
                    full.data[dstIdx + 1] = patch[srcIdx + 1];
                    full.data[dstIdx + 2] = patch[srcIdx + 2];
                    full.data[dstIdx + 3] = patch[srcIdx + 3];
                }
            }

            if (frame.disposalType === 3) previous = full.data.slice();

            ctx.putImageData(full, 0, 0);
            index = (index + 1) % frames.length;
            setTimeout(draw, Math.max(frame.delay, 20));
        }

        draw();
    })();
}
