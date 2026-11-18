/** @type {import('tailwindcss').Config} */
// Theme + daisyUI setup is shared with the server portal
// (cmd/syncserver/portal), see ui/tailwind.shared.cjs.
module.exports = require('../ui/tailwind.shared.cjs')(
    require('daisyui'),
    ['./index.html', './src/**/*.{js,vue}'],
);
