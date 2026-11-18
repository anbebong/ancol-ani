/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './gallery.html', './src/**/*.{js,html}'],
  theme: { extend: {} },
  plugins: [require('daisyui')],
  daisyui: {
    themes: ['light'],
  },
};
