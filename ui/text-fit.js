// Shared by the widget overlay (frontend/index.html) and the live preview
// in the settings page (web/src/views/WidgetView.vue): splits one line of
// user-entered text into several chunks that each fit the speech bubble,
// instead of letting the overflow get clipped/ellipsised.

// The real widget's #text-bubble box model (frontend/index.html), as a
// single source of truth. The preview bubble in WidgetView.vue is shown
// bigger for readability, but how many characters fit per line must match
// what the ACTUAL widget does - not whatever size the preview happens to
// render at. So both call sites compute the fit check from these numbers
// (never from the preview element's own, larger, CSS), and only the
// preview's *visual* CSS is free to differ.
export const WIDGET_BUBBLE = {
    contentWidthPx: 68 - 4 * 2 - 1.5 * 2, // max-width - padding-x - border-x (border-box)
    fontCss: '700 10px "Segoe UI", system-ui, sans-serif',
    maxLines: 2, // #text-content's -webkit-line-clamp
};

// Counts how many lines `text` would wrap into at `maxWidthPx`, replicating
// CSS `word-break: break-word`: words are kept whole unless a single word
// alone is wider than maxWidthPx, in which case it's broken character by
// character (matching what the browser does with that CSS property).
export function countWrappedLines(ctx, text, maxWidthPx) {
    if (!text) return 0;
    const spaceWidth = ctx.measureText(' ').width;
    let lines = 1;
    let lineWidth = 0;

    const breakWord = (word) => {
        let chunkWidth = 0;
        for (const ch of word) {
            const chWidth = ctx.measureText(ch).width;
            if (chunkWidth > 0 && chunkWidth + chWidth > maxWidthPx) {
                lines++;
                chunkWidth = chWidth;
            } else {
                chunkWidth += chWidth;
            }
        }
        lineWidth = chunkWidth;
    };

    for (const word of text.split(' ')) {
        const wordWidth = ctx.measureText(word).width;
        if (wordWidth > maxWidthPx) {
            // The word alone can't fit any line - start it on a fresh one
            // (unless the current line is still empty) and break it up.
            if (lineWidth > 0) { lines++; lineWidth = 0; }
            breakWord(word);
            continue;
        }
        const added = (lineWidth > 0 ? spaceWidth : 0) + wordWidth;
        if (lineWidth > 0 && lineWidth + added > maxWidthPx) {
            lines++;
            lineWidth = wordWidth;
        } else {
            lineWidth += added;
        }
    }
    return lines;
}

// The pixel width actually available for text inside an element whose
// *max-width* is `maxWidthCss` (a getComputedStyle value, e.g. "68px").
// With `box-sizing: border-box` (as set by a Tailwind preflight reset, or
// explicitly), max-width covers content + padding + border, so those must
// be subtracted to get the content width text actually wraps at; with the
// default `content-box`, max-width already *is* that content width.
// Getting this wrong makes every fit check think there's a few px more
// room than really exists, which is just enough to let one extra word
// sneak onto a line that then overflows and gets clamped with "...".
export function contentWidthOf(style) {
    const maxWidth = parseFloat(style.maxWidth);
    if (style.boxSizing !== 'border-box') return maxWidth;
    const paddingX = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
    const borderX = (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.borderRightWidth) || 0);
    return maxWidth - paddingX - borderX;
}

// Builds a `fits(str)` check backed by Canvas2D.measureText - deterministic
// and independent of live DOM layout, so it can't be thrown off by a hidden
// measuring element's own CSS (padding, scoped-style attributes, animation
// transforms, font-loading timing...). Callers just need the box's resolved
// font and the pixel width/line count it has room for.
export function makeFitsWithinLines(fontCss, maxWidthPx, maxLines) {
    const ctx = document.createElement('canvas').getContext('2d');
    ctx.font = fontCss;
    return (str) => countWrappedLines(ctx, str, maxWidthPx) <= maxLines;
}

// The DOM-agnostic part: binary-searches the longest prefix (by character)
// of `text` that still passes `fits`, backs up to the nearest space so
// words aren't cut mid-way (unless a single word alone is already too
// long - then it's left as-is, for `fits`'s own word-breaking to handle),
// and recurses on what's left.
export function splitToFit(text, fits) {
    text = text.trim();
    if (!text) return [];
    if (fits(text)) return [text];

    let lo = 1, hi = text.length, best = 1;
    while (lo <= hi) {
        const mid = (lo + hi) >> 1;
        if (fits(text.slice(0, mid))) { best = mid; lo = mid + 1; }
        else hi = mid - 1;
    }
    let cut = best;
    const spaceIdx = text.lastIndexOf(' ', cut);
    if (spaceIdx > 0 && spaceIdx > cut - 15) cut = spaceIdx;

    const head = text.slice(0, cut).trim();
    const rest = text.slice(cut).trim();
    return rest ? [head, ...splitToFit(rest, fits)] : [head];
}
