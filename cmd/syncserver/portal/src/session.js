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

export async function signOut() {
    try { await api('/auth/logout', { method: 'POST' }); } catch (_) {}
    signOutLocal();
}
