import { resolve } from 'path';

export default {
    base: './',
    build: {
        outDir: 'dist',
        rollupOptions: {
            input: {
                main: resolve(__dirname, 'index.html'),
                gallery: resolve(__dirname, 'gallery.html'),
            },
        },
    },
};
