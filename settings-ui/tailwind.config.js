/** @type {import('tailwindcss').Config} */
// Theme + daisyUI setup is shared with the server portal
// (cmd/syncserver/portal), see theme/tailwind.shared.cjs.
module.exports = require('../theme/tailwind.shared.cjs')(
    require('daisyui'),
    ['./index.html', './src/**/*.{js,vue}'],
);
