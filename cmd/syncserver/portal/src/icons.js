import image from 'lucide-static/icons/image.svg?raw';
import globe from 'lucide-static/icons/globe.svg?raw';
import logOut from 'lucide-static/icons/log-out.svg?raw';
import arrowUp from 'lucide-static/icons/arrow-up.svg?raw';
import refreshCw from 'lucide-static/icons/refresh-cw.svg?raw';
import share2 from 'lucide-static/icons/share-2.svg?raw';
import eyeOff from 'lucide-static/icons/eye-off.svg?raw';
import crown from 'lucide-static/icons/crown.svg?raw';
import users from 'lucide-static/icons/users.svg?raw';
import check from 'lucide-static/icons/check.svg?raw';
import minus from 'lucide-static/icons/minus.svg?raw';
import key from 'lucide-static/icons/key.svg?raw';
import sparkle from 'lucide-static/icons/sparkle.svg?raw';
import trash2 from 'lucide-static/icons/trash-2.svg?raw';

export const ICONS = {
    image, globe, 'log-out': logOut, 'arrow-up': arrowUp, 'refresh-cw': refreshCw,
    'share-2': share2, 'eye-off': eyeOff, crown, users, check, minus, key, sparkle, 'trash-2': trash2,
};

// Replaces every <span data-icon="name"> inside root with the matching
// lucide SVG markup. Sizing/color is controlled by the span's own
// Tailwind classes (w-4 h-4, [&>svg]:w-4 [&>svg]:h-4, text-color...)
// since lucide icons use stroke="currentColor".
export function renderIcons(root = document) {
    root.querySelectorAll('[data-icon]').forEach((el) => {
        const name = el.getAttribute('data-icon');
        if (ICONS[name]) el.innerHTML = ICONS[name];
    });
}

export function iconHTML(name) {
    return ICONS[name] || '';
}
