import tailwindcss from '@tailwindcss/vite';

export default {
    base: './',
    plugins: [tailwindcss()],
    build: {
        outDir: 'dist',
        assetsDir: 'assets',
    },
};
