import vue from '@vitejs/plugin-vue';
import { sharedUi } from '../ui/vite.shared.js';

const shared = sharedUi(__dirname);

// `npm run dev`: proxy the desktop app's local endpoints to a running
// ancol-ani.exe (it serves this page on :8788) so the page can be
// developed with hot reload against the real widget.
const LOCAL_ENDPOINTS = ['/gif', '/text', '/version', '/changelog', '/position'];

export default {
    base: './',
    plugins: [vue()],
    resolve: shared.resolve,
    server: {
        ...shared.server,
        proxy: Object.fromEntries(LOCAL_ENDPOINTS.map((p) => [p, 'http://localhost:8788'])),
    },
    build: {
        outDir: 'dist',
        assetsDir: 'assets',
    },
};
