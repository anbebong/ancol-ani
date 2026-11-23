// CLI: node render-svg.js <svg-path> <out-png-path> <size>
// Rasterizes an SVG to a square PNG of <size>x<size> via @resvg/resvg-js
// (true vector rendering, not upscaling - see scripts/gen-icons.ps1, which
// shells out to this for every size it needs instead of resizing a raster).
const fs = require('fs');
const { Resvg } = require('@resvg/resvg-js');

const [, , svgPath, outPath, sizeStr] = process.argv;
if (!svgPath || !outPath || !sizeStr) {
    console.error('usage: node render-svg.js <svg-path> <out-png-path> <size>');
    process.exit(1);
}
const size = parseInt(sizeStr, 10);
const svg = fs.readFileSync(svgPath, 'utf8');
const resvg = new Resvg(svg, { fitTo: { mode: 'width', value: size } });
const png = resvg.render().asPng();
fs.writeFileSync(outPath, png);
