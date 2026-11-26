/** @type {import('tailwindcss').Config} */
// Theme + daisyUI setup is shared with the client's settings page (web/),
// see theme/tailwind.shared.cjs.
module.exports = require('../../../theme/tailwind.shared.cjs')(
    require('daisyui'),
    ['./index.html', './src/**/*.{js,vue}'],
);
