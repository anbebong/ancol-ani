/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{js,vue}'],
  theme: { extend: {} },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        // Matches the desktop app's local settings page (web/src/style.css)
        // so the portal and the client feel like one product instead of
        // two unrelated UIs.
        pixel: {
          'color-scheme': 'dark',
          primary: '#5c3eeb',
          'primary-content': '#ffffff',
          secondary: '#ff6bd6',
          'secondary-content': '#1a0a14',
          accent: '#5c3eeb',
          'accent-content': '#ffffff',
          neutral: '#35335a',
          'base-100': '#14142b',
          'base-200': '#0e0e22',
          'base-300': '#0a0a14',
          'base-content': '#e8e8f0',
          info: '#5c3eeb',
          success: '#2fbf71',
          warning: '#e8a53d',
          error: '#ef4458',

          '--rounded-box': '0.5rem',
          '--rounded-btn': '0.5rem',
          '--tab-radius': '0.5rem',
        },
      },
    ],
  },
};
