// Shared Vite settings that let a project import the shared design code
// in theme/ as `@theme/...` (components, icons, modal, CSS, logo).
//
// Files under theme/ have no node_modules of their own, so the bare imports
// they make ('vue', 'lucide-static/...') are pinned to the importing
// project's node_modules - one copy of Vue per bundle, no install step
// for theme/ itself.
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const themeDir = path.dirname(fileURLToPath(import.meta.url));

export function sharedUi(projectDir) {
    return {
        resolve: {
            alias: {
                '@theme': themeDir,
                'lucide-static': path.join(projectDir, 'node_modules/lucide-static'),
            },
            dedupe: ['vue'],
        },
        // dev server: allow serving files from theme/ (outside the project root)
        server: { fs: { allow: [projectDir, themeDir] } },
    };
}
