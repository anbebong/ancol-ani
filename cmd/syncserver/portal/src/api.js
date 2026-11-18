const TOKEN_KEY = 'ancol-ani-token';
const USERNAME_KEY = 'ancol-ani-username';

export function getToken() { return localStorage.getItem(TOKEN_KEY); }
export function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
export function clearToken() { localStorage.removeItem(TOKEN_KEY); }

export function getUsername() { return localStorage.getItem(USERNAME_KEY) || ''; }
export function setUsername(u) { localStorage.setItem(USERNAME_KEY, u); }
export function clearUsername() { localStorage.removeItem(USERNAME_KEY); }

export async function api(path, opts = {}) {
    const headers = opts.headers || {};
    const token = getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    const res = await fetch(path, { ...opts, headers });
    if (res.status === 401) {
        clearToken();
        throw new Error('Phien dang nhap het han, vui long dang nhap lai');
    }
    if (!res.ok) {
        throw new Error(await res.text());
    }
    return res;
}

export function showError(el, err) {
    el.textContent = err.message || String(err);
}
