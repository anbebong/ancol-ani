const TOKEN_KEY = 'ancol-ani-token';
const USERNAME_KEY = 'ancol-ani-username';

export function getToken() { return localStorage.getItem(TOKEN_KEY); }
export function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
export function clearToken() { localStorage.removeItem(TOKEN_KEY); }

export function getUsername() { return localStorage.getItem(USERNAME_KEY) || ''; }
export function setUsername(u) { localStorage.setItem(USERNAME_KEY, u); }
export function clearUsername() { localStorage.removeItem(USERNAME_KEY); }

// Called whenever the server rejects the token (expired / revoked), so
// the app can drop the session and send the user back to the login page.
let onUnauthorized = () => {};
export function setUnauthorizedHandler(fn) { onUnauthorized = fn; }

export async function api(path, opts = {}) {
    const headers = opts.headers || {};
    const token = getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    const res = await fetch(path, { ...opts, headers });
    if (res.status === 401) {
        clearToken();
        onUnauthorized();
        throw new Error('Phiên đăng nhập hết hạn, vui lòng đăng nhập lại');
    }
    if (!res.ok) {
        throw new Error((await res.text()) || `Lỗi ${res.status}`);
    }
    return res;
}

export function errMsg(err) {
    return err?.message || String(err);
}

export function fmtDate(s) {
    return new Date(s.replace(' ', 'T') + 'Z').toLocaleString();
}

export function tagList(tags) {
    return tags ? tags.split(',').map((t) => '#' + t.trim()).join(' ') : '';
}

export const PAGE_SIZE = 10;

// GIF blobs are immutable once created (sharing/unsharing only changes
// metadata, never the bytes), so once fetched they're cached in memory
// for the whole session - switching pages/routes back and forth never
// re-downloads an image it already has. Each blob gets one object URL
// that's reused, and dropped when the GIF is deleted.
const imageCache = new Map();
export function cachedImageURL(cacheKey, path) {
    if (imageCache.has(cacheKey)) return imageCache.get(cacheKey);
    const p = api(path)
        .then((res) => res.blob())
        .then((blob) => URL.createObjectURL(blob));
    p.catch(() => imageCache.delete(cacheKey));
    imageCache.set(cacheKey, p);
    return p;
}
export function forgetImage(cacheKey) {
    const p = imageCache.get(cacheKey);
    imageCache.delete(cacheKey);
    p?.then((url) => URL.revokeObjectURL(url)).catch(() => {});
}
