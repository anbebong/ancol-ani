import vue from '@vitejs/plugin-vue';
import { sharedUi } from '../theme/vite.shared.js';

const shared = sharedUi(__dirname);

// `npm run dev`: proxy the desktop app's local endpoints to a running
// ancol-ani.exe so the page can be developed with hot reload against the
// real widget. The widget's actual port (app.go's resolveListener scans
// forward from DEFAULT_PORT if it's taken) lives in settings.data,
// encrypted on the Go side - not worth re-implementing that AES-GCM read
// here just for this dev convenience proxy, so this assumes the default
// and only needs manual override (edit DEFAULT_PORT below) on the rare
// occasion that port was actually busy when the widget last started.
const DEFAULT_PORT = 40404;
const LOCAL_ENDPOINTS = ['/gif', '/text', '/version', '/changelog', '/position', '/autostart'];

export default {
    base: './',
    plugins: [vue()],
    resolve: shared.resolve,
    server: {
        ...shared.server,
        proxy: Object.fromEntries(LOCAL_ENDPOINTS.map((p) => [p, 'http://localhost:' + DEFAULT_PORT])),
    },
    build: {
        outDir: 'dist',
        assetsDir: 'assets',
    },
};
