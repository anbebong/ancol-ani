// Links this device's settings page to a syncserver account and lets the
// user pick one GIF - from their own library or from the public gallery -
// to pull into the app. The token/account info lives only in this page's
// localStorage (never sent to or stored by the Go app) - the app just
// receives the resulting image via the existing POST /gif upload
// endpoint, exactly like a manual file upload. There is no server-side
// "active" gif - the user always explicitly picks which one to sync.
import { t } from './i18n.js';
import { playGif } from './gif-player.js';

const TOKEN_KEY = 'ancol-ani-sync-token';
const USERNAME_KEY = 'ancol-ani-sync-username';
const SERVER_KEY = 'ancol-ani-sync-server';
const DEFAULT_SERVER = 'http://localhost:9090';

function getToken() { return localStorage.getItem(TOKEN_KEY); }
function getUsername() { return localStorage.getItem(USERNAME_KEY); }
function getServerUrl() { return localStorage.getItem(SERVER_KEY) || DEFAULT_SERVER; }

export function initSync({ toast, reloadPreview }) {
    const unlinkedBox = document.getElementById('sync-unlinked');
    const linkedBox = document.getElementById('sync-linked');
    const serverInput = document.getElementById('sync-server-url');
    const usernameEl = document.getElementById('sync-username');
    const personalGrid = document.getElementById('sync-library-personal');
    const personalEmptyEl = document.getElementById('sync-library-personal-empty');
    const publicGrid = document.getElementById('sync-library-public');
    const publicEmptyEl = document.getElementById('sync-library-public-empty');

    serverInput.value = getServerUrl();

    function render() {
        const linked = !!getToken();
        unlinkedBox.classList.toggle('hidden', linked);
        linkedBox.classList.toggle('hidden', !linked);
        linkedBox.classList.toggle('flex', linked);
        if (linked) {
            usernameEl.textContent = getUsername() || '?';
            loadPersonalLibrary();
            loadPublicLibrary();
        }
    }

    function authedFetch(path) {
        return fetch(getServerUrl() + path, { headers: { Authorization: 'Bearer ' + getToken() } });
    }

    // Bumped every time a grid is cleared/reloaded so an in-flight decode
    // for a tile that's already been thrown away doesn't keep drawing.
    let personalEpoch = 0;
    let publicEpoch = 0;

    async function loadPersonalLibrary() {
        personalGrid.innerHTML = '';
        const epoch = ++personalEpoch;
        try {
            const res = await authedFetch('/me/gifs');
            if (!res.ok) throw new Error(await res.text());
            const gifs = await res.json();
            personalEmptyEl.classList.toggle('hidden', gifs.length > 0);
            for (const g of gifs) {
                personalGrid.appendChild(renderGifTile('/me/gifs/' + g.id + '/image', () => epoch === personalEpoch));
            }
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    async function loadPublicLibrary() {
        publicGrid.innerHTML = '';
        const epoch = ++publicEpoch;
        try {
            const res = await authedFetch('/public');
            if (!res.ok) throw new Error(await res.text());
            const entries = await res.json();
            publicEmptyEl.classList.toggle('hidden', entries.length > 0);
            for (const entry of entries) {
                publicGrid.appendChild(renderGifTile('/public/' + entry.slug, () => epoch === publicEpoch, entry.name + ' · ' + entry.username));
            }
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    // One grid tile: thumbnail on top (rendered via playGif, same
    // canvas-based decoder the main preview uses - plain <img> GIF
    // animation can stall permanently in Chromium after a while), pick
    // button directly below it.
    function renderGifTile(imagePath, isCurrent, ownerUsername) {
        const tile = document.createElement('div');
        tile.className = 'flex flex-col gap-1.5';
        tile.innerHTML = `
            <div class="w-full aspect-square border-2 border-[var(--border)] bg-[var(--bg)] flex items-center justify-center overflow-hidden">
                <canvas class="max-w-full max-h-full"></canvas>
            </div>
            ${ownerUsername ? `<p class="text-[10px] text-center text-[var(--muted)] truncate">${ownerUsername}</p>` : ''}
            <button type="button" class="pixel-btn rounded-lg w-full py-2 text-[11px] font-semibold"></button>
        `;

        playGif(tile.querySelector('canvas'), () => authedFetch(imagePath).then((r) => r.arrayBuffer()), isCurrent);

        tile.querySelector('button').textContent = t('syncUseThis');
        tile.querySelector('button').addEventListener('click', () => pull(imagePath));

        return tile;
    }

    async function pull(imagePath) {
        try {
            const res = await authedFetch(imagePath);
            if (!res.ok) throw new Error(await res.text());
            const blob = await res.blob();

            const form = new FormData();
            form.append('gif', blob, 'synced.gif');
            const upload = await fetch('/gif', { method: 'POST', body: form });
            if (!upload.ok) throw new Error(await upload.text());

            reloadPreview();
            toast(t('toastSyncPulled'));
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    // Returning from the portal's login redirect: /?token=... on this
    // very page. Pick it up, ask the server who that token belongs to,
    // then drop it from the URL.
    async function consumeCallbackToken() {
        const params = new URLSearchParams(location.search);
        const token = params.get('token');
        if (!token) return;

        history.replaceState(null, '', location.pathname);
        localStorage.setItem(TOKEN_KEY, token);

        try {
            const res = await fetch(getServerUrl() + '/me', {
                headers: { Authorization: 'Bearer ' + token },
            });
            if (!res.ok) throw new Error(await res.text());
            const me = await res.json();
            localStorage.setItem(USERNAME_KEY, me.username);
            toast(t('toastSyncLinked'));
        } catch (err) {
            localStorage.removeItem(TOKEN_KEY);
            toast(err.message || String(err));
        }
    }

    document.getElementById('sync-link-btn').addEventListener('click', () => {
        const server = serverInput.value.trim() || DEFAULT_SERVER;
        localStorage.setItem(SERVER_KEY, server);
        const callback = location.origin + location.pathname;
        location.href = server + '/?callback=' + encodeURIComponent(callback);
    });

    document.getElementById('sync-unlink-btn').addEventListener('click', () => {
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(USERNAME_KEY);
        toast(t('toastSyncUnlinked'));
        render();
    });

    consumeCallbackToken().then(render);
}
