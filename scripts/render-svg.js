// CLI: node render-svg.js <svg-path> <out-png-path> <size> [mode]
// Rasterizes an SVG via @resvg/resvg-js (true vector rendering, not
// upscaling - see scripts/gen-icons.ps1, which shells out to this for
// every size it needs instead of resizing a raster). <size> is the pixel
// value for <mode> ("width", the default, or "height"); the other
// dimension comes from the SVG's own viewBox aspect ratio, so a non-square
// viewBox (e.g. ui/assets/logo-navbar.svg) produces a non-square PNG.
const fs = require('fs');
const { Resvg } = require('@resvg/resvg-js');

const [, , svgPath, outPath, sizeStr, modeArg] = process.argv;
if (!svgPath || !outPath || !sizeStr) {
    console.error('usage: node render-svg.js <svg-path> <out-png-path> <size> [width|height]');
    process.exit(1);
}
const size = parseInt(sizeStr, 10);
const mode = modeArg === 'height' ? 'height' : 'width';
const svg = fs.readFileSync(svgPath, 'utf8');
const resvg = new Resvg(svg, { fitTo: { mode, value: size } });
const png = resvg.render().asPng();
fs.writeFileSync(outPath, png);
