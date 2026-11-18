// Link between this device's settings page and a syncserver account.
// The token/account info lives only in this page's localStorage (never
// sent to or stored by the Go app). Pulling a GIF into the widget just
// reuses the app's local POST /gif endpoint, exactly like a manual file
// upload; there is no server-side "active" GIF.
import { reactive } from 'vue';

const TOKEN_KEY = 'ancol-ani-sync-token';
const USERNAME_KEY = 'ancol-ani-sync-username';
const SERVER_KEY = 'ancol-ani-sync-server';
export const DEFAULT_SERVER = 'http://localhost:9090';
export const PAGE_SIZE = 10;

function read(key) { try { return localStorage.getItem(key); } catch (_) { return null; } }
function write(key, v) { try { v == null ? localStorage.removeItem(key) : localStorage.setItem(key, v); } catch (_) {} }

export const account = reactive({
    token: read(TOKEN_KEY),
    username: read(USERNAME_KEY) || '',
    server: read(SERVER_KEY) || DEFAULT_SERVER,
    me: null, // { username, isAdmin, gifQuota } from /me
});

// Called when the server rejects the token (expired, or revoked by a
// logout / password change / admin reset elsewhere).
let onExpired = () => {};
export function setExpiredHandler(fn) { onExpired = fn; }

export function clearLink() {
    write(TOKEN_KEY, null);
    write(USERNAME_KEY, null);
    account.token = null;
    account.username = '';
    account.me = null;
    imageCache.clear();
}

export async function authedFetch(path, opts = {}) {
    const headers = { ...(opts.headers || {}), Authorization: 'Bearer ' + account.token };
    const res = await fetch(account.server + path, { ...opts, headers });
    if (res.status === 401) {
        clearLink();
        onExpired();
        throw new Error('401');
    }
    return res;
}

// authedFetch + throw on any non-2xx with the server's error text.
export async function serverApi(path, opts) {
    const res = await authedFetch(path, opts);
    if (!res.ok) throw new Error((await res.text()) || `HTTP ${res.status}`);
    return res;
}

export async function loadMe() {
    const res = await serverApi('/me');
    account.me = await res.json();
    account.username = account.me.username;
    write(USERNAME_KEY, account.username);
    return account.me;
}

// Sends the browser to the portal's login page; the portal redirects back
// here with ?token=... (see consumeCallbackToken).
export function startLink(server) {
    account.server = (server || '').trim().replace(/\/+$/, '') || DEFAULT_SERVER;
    write(SERVER_KEY, account.server);
    const callback = location.origin + location.pathname;
    location.href = account.server + '/?callback=' + encodeURIComponent(callback);
}

// Returning from the portal's login redirect: /?token=... on this page.
// Returns 'linked', 'failed' or null (nothing to consume).
export async function consumeCallbackToken() {
    const token = new URLSearchParams(location.search).get('token');
    if (!token) return null;
    try {
        const res = await fetch(account.server + '/me', { headers: { Authorization: 'Bearer ' + token } });
        if (!res.ok) throw new Error(await res.text());
        const me = await res.json();
        write(TOKEN_KEY, token);
        account.token = token;
        account.me = me;
        account.username = me.username;
        write(USERNAME_KEY, me.username);
        return 'linked';
    } catch (_) {
        return 'failed';
    }
}

export function unlink() { clearLink(); }

// GIF bytes never change once a slot is created (sharing/unsharing only
// touches metadata), so fetched bytes are kept for the page session -
// paging back and forth never re-downloads an image it already has.
const imageCache = new Map();
export function cachedGif(cacheKey, path) {
    if (!imageCache.has(cacheKey)) {
        const p = serverApi(path).then((r) => r.arrayBuffer());
        p.catch(() => imageCache.delete(cacheKey));
        imageCache.set(cacheKey, p);
    }
    return imageCache.get(cacheKey);
}
export function forgetGif(cacheKey) { imageCache.delete(cacheKey); }

// Copies a server GIF into the widget via the app's local upload endpoint.
export async function pullToWidget(path) {
    const res = await serverApi(path);
    const form = new FormData();
    form.append('gif', await res.blob(), 'synced.gif');
    const up = await fetch('/gif', { method: 'POST', body: form });
    if (!up.ok) throw new Error(await up.text());
}

export function fmtDate(s) {
    return new Date(s.replace(' ', 'T') + 'Z').toLocaleString();
}
export function tagList(tags) {
    return tags ? tags.split(',').map((x) => '#' + x.trim()).join(' ') : '';
}
