import image from 'lucide-static/icons/image.svg?raw';
import images from 'lucide-static/icons/images.svg?raw';
import globe from 'lucide-static/icons/globe.svg?raw';
import logOut from 'lucide-static/icons/log-out.svg?raw';
import logIn from 'lucide-static/icons/log-in.svg?raw';
import upload from 'lucide-static/icons/upload.svg?raw';
import download from 'lucide-static/icons/download.svg?raw';
import share2 from 'lucide-static/icons/share-2.svg?raw';
import eye from 'lucide-static/icons/eye.svg?raw';
import eyeOff from 'lucide-static/icons/eye-off.svg?raw';
import crown from 'lucide-static/icons/crown.svg?raw';
import users from 'lucide-static/icons/users.svg?raw';
import user from 'lucide-static/icons/user.svg?raw';
import userPlus from 'lucide-static/icons/user-plus.svg?raw';
import check from 'lucide-static/icons/check.svg?raw';
import minus from 'lucide-static/icons/minus.svg?raw';
import key from 'lucide-static/icons/key.svg?raw';
import lock from 'lucide-static/icons/lock.svg?raw';
import trash2 from 'lucide-static/icons/trash-2.svg?raw';
import history from 'lucide-static/icons/history.svg?raw';
import compass from 'lucide-static/icons/compass.svg?raw';
import shield from 'lucide-static/icons/shield.svg?raw';
import search from 'lucide-static/icons/search.svg?raw';
import type from 'lucide-static/icons/type.svg?raw';
import link from 'lucide-static/icons/link.svg?raw';
import unlink from 'lucide-static/icons/unlink.svg?raw';
import rotateCcw from 'lucide-static/icons/rotate-ccw.svg?raw';
import monitor from 'lucide-static/icons/monitor.svg?raw';
import languages from 'lucide-static/icons/languages.svg?raw';
import externalLink from 'lucide-static/icons/external-link.svg?raw';
import server from 'lucide-static/icons/server.svg?raw';
import sparkles from 'lucide-static/icons/sparkles.svg?raw';
import tag from 'lucide-static/icons/tag.svg?raw';
import square from 'lucide-static/icons/square.svg?raw';

// Raw lucide SVG markup by name, rendered through <Icon name="..."/>.
// Shared by the server portal and the client's local settings page.
// Lucide icons use stroke="currentColor", so color follows the text.
// To add one: import it from lucide-static/icons/<name>.svg?raw below
// and list it here (browse names at https://lucide.dev/icons).
export const ICONS = {
    image, images, globe, 'log-out': logOut, 'log-in': logIn, upload, download,
    'share-2': share2, eye, 'eye-off': eyeOff, crown, users, user, 'user-plus': userPlus,
    check, minus, key, lock, 'trash-2': trash2, history, compass, shield, search,
    type, link, unlink, 'rotate-ccw': rotateCcw, monitor, languages,
    'external-link': externalLink, server, sparkles, tag, square,
};
