import vue from '@vitejs/plugin-vue';
import { sharedUi } from '../../../theme/vite.shared.js';

const shared = sharedUi(__dirname);

// Proxy API calls to a locally running syncserver during `npm run dev`
// so the portal can be developed with hot reload (go run ./cmd/syncserver
// listens on :9090 by default).
const API_PREFIXES = ['/auth', '/me', '/public', '/admin'];

export default {
    base: './',
    plugins: [vue()],
    resolve: shared.resolve,
    server: {
        ...shared.server,
        proxy: Object.fromEntries(API_PREFIXES.map((p) => [p, 'http://localhost:9090'])),
    },
    build: {
        outDir: 'dist',
    },
};
