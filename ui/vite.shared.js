// Shared Vite settings that let a project import the shared design code
// in ui/ as `@ui/...` (components, icons, modal, CSS, logo).
//
// Files under ui/ have no node_modules of their own, so the bare imports
// they make ('vue', 'lucide-static/...') are pinned to the importing
// project's node_modules - one copy of Vue per bundle, no install step
// for ui/ itself.
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const uiDir = path.dirname(fileURLToPath(import.meta.url));

export function sharedUi(projectDir) {
    return {
        resolve: {
            alias: {
                '@ui': uiDir,
                'lucide-static': path.join(projectDir, 'node_modules/lucide-static'),
            },
            dedupe: ['vue'],
        },
        // dev server: allow serving files from ui/ (outside the project root)
        server: { fs: { allow: [projectDir, uiDir] } },
    };
}
