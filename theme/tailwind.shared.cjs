// Shared Tailwind + daisyUI setup for the server portal
// (cmd/syncserver/portal) and the client's local settings page (web/).
//
// Each project's tailwind.config.js calls this with its own `daisyui`
// module (require('daisyui') must resolve from the project's
// node_modules - there is none next to this file) and its own content
// globs; the shared ui/ components are always scanned too.
const path = require('path');

const pixelTheme = {
    // Dark pixel-art palette, same everywhere in the product.
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
};

module.exports = function sharedTailwindConfig(daisyui, projectContent) {
    return {
        content: [...projectContent, path.join(__dirname, '**/*.{js,vue}').replace(/\\/g, '/')],
        theme: { extend: {} },
        plugins: [daisyui],
        daisyui: { themes: [{ pixel: pixelTheme }] },
    };
};
