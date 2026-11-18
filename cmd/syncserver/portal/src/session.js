import { reactive } from 'vue';
import { api, getToken, setToken, clearToken, setUsername, clearUsername } from './api.js';

// App-wide login state. `me` is fetched once per session from /me and
// reused by the navbar, the router's admin guard and the library page.
export const session = reactive({
    token: getToken(),
    me: null, // { username, isAdmin, gifQuota }
});

let mePromise = null;

export function loadMe({ force = false } = {}) {
    if (!session.token) return Promise.resolve(null);
    if (mePromise && !force) return mePromise;
    mePromise = api('/me')
        .then((res) => res.json())
        .then((me) => { session.me = me; return me; })
        .catch((err) => { mePromise = null; throw err; });
    return mePromise;
}

export function signIn(token, username) {
    setToken(token);
    setUsername(username);
    session.token = token;
    session.me = null;
    mePromise = null;
}

export function signOutLocal() {
    clearToken();
    clearUsername();
    session.token = null;
    session.me = null;
    mePromise = null;
}

// Trades a valid token for a brand-new, independent session token for the
// same account (POST /auth/token). Deliberately not via api(): a 401 here
// is about `bearer`, which may not be this portal's own session.
export async function issueToken(bearer) {
    const res = await fetch('/auth/token', { method: 'POST', headers: { Authorization: 'Bearer ' + bearer } });
    if (!res.ok) {
        const err = new Error((await res.text()) || `HTTP ${res.status}`);
        err.status = res.status;
        throw err;
    }
    return (await res.json()).token;
}

// Account linking from the desktop app (?callback=...): keep the portal's
// own session and send the client its *own* token, so logging out on one
// side never unlinks the other.
export async function sendTokenToClient(callback) {
    const token = await issueToken(session.token);
    location.href = callback + (callback.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(token);
}

export async function signOut() {
    try { await api('/auth/logout', { method: 'POST' }); } catch (_) {}
    signOutLocal();
}
