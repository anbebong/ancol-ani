// Links this device's settings page to a syncserver account and lets the
// user manage their GIF library there directly: upload new ones, share/
// unshare/delete personal GIFs, and pull any GIF (personal or public)
// into the app. The token/account info lives only in this page's
// localStorage (never sent to or stored by the Go app) - pulling a GIF
// into the widget just reuses the existing POST /gif upload endpoint,
// exactly like a manual file upload. There is no server-side "active"
// gif - the user always explicitly picks which one to sync.
import { t } from './i18n.js';
import { playGif } from './gif-player.js';
import { showModal, confirmDialog } from './modal.js';

const TOKEN_KEY = 'ancol-ani-sync-token';
const USERNAME_KEY = 'ancol-ani-sync-username';
const SERVER_KEY = 'ancol-ani-sync-server';
const DEFAULT_SERVER = 'http://localhost:9090';

const PAGE_SIZE = 10;

function getToken() { return localStorage.getItem(TOKEN_KEY); }
function getUsername() { return localStorage.getItem(USERNAME_KEY); }
function getServerUrl() { return localStorage.getItem(SERVER_KEY) || DEFAULT_SERVER; }

export function initSync({ toast, reloadPreview }) {
    const unlinkedBox = document.getElementById('sync-unlinked');
    const linkedBox = document.getElementById('sync-linked');
    const serverInput = document.getElementById('sync-server-url');
    const usernameEl = document.getElementById('sync-username');
    const quotaEl = document.getElementById('sync-quota');
    const pushCurrentBtn = document.getElementById('push-current-gif-btn');
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
        pushCurrentBtn.classList.toggle('hidden', !linked);
        pushCurrentBtn.classList.toggle('flex', linked);
        if (linked) {
            usernameEl.textContent = getUsername() || '?';
            loadQuota();
            loadPersonalLibrary();
            loadPublicLibrary();
        }
    }

    // If the server says the token is no longer good (expired, or the
    // session was revoked via logout somewhere else), drop it here too
    // and fall back to the "not linked" view instead of leaving the UI
    // showing a library that can't actually be loaded anymore.
    async function authedFetch(path, opts = {}) {
        const headers = { ...(opts.headers || {}), Authorization: 'Bearer ' + getToken() };
        const res = await fetch(getServerUrl() + path, { ...opts, headers });
        if (res.status === 401) {
            localStorage.removeItem(TOKEN_KEY);
            localStorage.removeItem(USERNAME_KEY);
            toast(t('toastSyncExpired'));
            render();
        }
        return res;
    }

    // GIF bytes never change once a slot is created (sharing/unsharing
    // only touches metadata), so once fetched + decoded they're kept for
    // this page session - paging back and forth, or any action that
    // reloads the current page, never re-downloads/re-decodes an image
    // it already has.
    const imageCache = new Map();
    function cachedArrayBuffer(cacheKey, path) {
        if (imageCache.has(cacheKey)) return Promise.resolve(imageCache.get(cacheKey));
        return authedFetch(path)
            .then((r) => r.arrayBuffer())
            .then((buf) => { imageCache.set(cacheKey, buf); return buf; });
    }

    function renderPager(prevId, nextId, labelId, state, onChange) {
        const totalPages = Math.max(1, Math.ceil(state.total / PAGE_SIZE));
        document.getElementById(labelId).textContent = `${state.page}/${totalPages}`;
        const prevBtn = document.getElementById(prevId);
        const nextBtn = document.getElementById(nextId);
        prevBtn.disabled = state.page <= 1;
        nextBtn.disabled = state.page >= totalPages;
        prevBtn.onclick = () => onChange(state.page - 1);
        nextBtn.onclick = () => onChange(state.page + 1);
    }

    async function loadQuota() {
        try {
            const res = await authedFetch('/me');
            if (!res.ok) throw new Error(await res.text());
            const me = await res.json();
            quotaEl.textContent = me.gifQuota + ' ' + t('syncQuotaSuffix');
        } catch (_) {
            quotaEl.textContent = '';
        }
    }

    // Bumped every time a grid is cleared/reloaded so an in-flight decode
    // for a tile that's already been thrown away doesn't keep drawing.
    let personalEpoch = 0;
    let publicEpoch = 0;

    let personalPage = 1;
    let publicPage = 1;

    async function loadPersonalLibrary(page = personalPage) {
        personalGrid.innerHTML = '';
        const epoch = ++personalEpoch;
        try {
            const res = await authedFetch(`/me/gifs?page=${page}&pageSize=${PAGE_SIZE}`);
            if (!res.ok) throw new Error(await res.text());
            const { items, total } = await res.json();
            personalPage = page;
            personalEmptyEl.classList.toggle('hidden', items.length > 0);
            for (const g of items) {
                personalGrid.appendChild(renderPersonalTile(g, () => epoch === personalEpoch));
            }
            renderPager('sync-personal-prev', 'sync-personal-next', 'sync-personal-page-label',
                { page: personalPage, total }, (p) => loadPersonalLibrary(p));
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    async function loadPublicLibrary(page = publicPage) {
        publicGrid.innerHTML = '';
        const epoch = ++publicEpoch;
        try {
            const res = await authedFetch(`/public?page=${page}&pageSize=${PAGE_SIZE}`);
            if (!res.ok) throw new Error(await res.text());
            const { items, total } = await res.json();
            publicPage = page;
            publicEmptyEl.classList.toggle('hidden', items.length > 0);
            for (const entry of items) {
                publicGrid.appendChild(renderPublicTile(entry, () => epoch === publicEpoch));
            }
            renderPager('sync-public-prev', 'sync-public-next', 'sync-public-page-label',
                { page: publicPage, total }, (p) => loadPublicLibrary(p));
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    function tileShell() {
        const tile = document.createElement('div');
        tile.className = 'flex flex-col gap-1';
        tile.innerHTML = `
            <div class="w-full aspect-square border-2 border-[var(--border)] bg-[var(--bg)] flex items-center justify-center overflow-hidden">
                <canvas class="max-w-full max-h-full"></canvas>
            </div>
        `;
        return tile;
    }

    function actionBtn(label, extraClass) {
        const b = document.createElement('button');
        b.type = 'button';
        b.textContent = label;
        b.className = 'pixel-btn' + (extraClass ? ' ' + extraClass : '') + ' rounded-lg w-full py-1.5 text-[10px] font-semibold';
        return b;
    }

    const ICONS = {
        use: '<path d="M4 4v5h5M20 20v-5h-5"/><path d="M4.5 9a8 8 0 0113.9-3.2M19.5 15a8 8 0 01-13.9 3.2"/>',
        share: '<circle cx="18" cy="5" r="2.5"/><circle cx="6" cy="12" r="2.5"/><circle cx="18" cy="19" r="2.5"/><path d="M8.2 10.7l7.6-4.4M8.2 13.3l7.6 4.4"/>',
        unshare: '<path d="M3 3l18 18"/><path d="M10.6 5.1A9 9 0 0121 12M3 12a9 9 0 0014.9 6.9"/>',
        delete: '<path d="M4 7h16M10 7V4h4v3M6 7l1 13a2 2 0 002 2h6a2 2 0 002-2l1-13"/>',
    };

    // Icon-only action button with a native tooltip (title) - used in the
    // personal library tiles, which otherwise have no room for 3 full
    // text buttons stacked under a small thumbnail.
    function iconActionBtn(icon, title, extraClass) {
        const b = document.createElement('button');
        b.type = 'button';
        b.title = title;
        b.setAttribute('aria-label', title);
        b.className = 'pixel-btn' + (extraClass ? ' ' + extraClass : '') + ' rounded-lg flex-1 flex items-center justify-center py-1.5';
        b.innerHTML = `<svg viewBox="0 0 24 24" fill="none" class="w-3.5 h-3.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg">${ICONS[icon]}</svg>`;
        return b;
    }

    // Public gallery tile: thumbnail (canvas-based, see gif-player.js -
    // plain <img> GIF animation can stall permanently in Chromium after
    // a while), publisher label, pick button.
    function renderPublicTile(entry, isCurrent) {
        const tile = tileShell();
        const imagePath = '/public/' + entry.slug;
        playGif(tile.querySelector('canvas'), () => cachedArrayBuffer('public:' + entry.slug, imagePath), isCurrent);

        const label = document.createElement('p');
        label.className = 'text-[10px] text-center text-[var(--muted)] truncate';
        label.textContent = entry.name + ' · ' + entry.username;
        tile.appendChild(label);

        const useBtn = actionBtn(t('syncUseThis'));
        useBtn.addEventListener('click', () => pull(imagePath));
        tile.appendChild(useBtn);

        return tile;
    }

    // Personal library tile: thumbnail + pick/share/unshare/delete, same
    // actions the portal offers for a user's own GIFs.
    function renderPersonalTile(g, isCurrent) {
        const tile = tileShell();
        const imagePath = '/me/gifs/' + g.id + '/image';
        playGif(tile.querySelector('canvas'), () => cachedArrayBuffer('me:' + g.id, imagePath), isCurrent);

        const badge = document.createElement('p');
        badge.className = 'text-[10px] text-center truncate ' + (g.isPublic ? 'text-[var(--accent-1)]' : 'text-[var(--muted)]');
        badge.textContent = g.isPublic ? (g.name || 'công khai') : 'riêng tư';
        tile.appendChild(badge);

        const actions = document.createElement('div');
        actions.className = 'flex gap-1';

        const useBtn = iconActionBtn('use', t('syncUseThis'));
        useBtn.addEventListener('click', () => pull(imagePath));
        actions.appendChild(useBtn);

        if (g.isPublic) {
            const unshareBtn = iconActionBtn('unshare', t('syncUnshareBtn'), 'pixel-btn-warning');
            unshareBtn.addEventListener('click', () => unshareGif(g.id));
            actions.appendChild(unshareBtn);
        } else {
            const shareBtn = iconActionBtn('share', t('syncShareBtn'), 'pixel-btn-success');
            shareBtn.addEventListener('click', () => shareGif(g.id));
            actions.appendChild(shareBtn);
        }

        const deleteBtn = iconActionBtn('delete', t('syncDeleteBtn'), 'pixel-btn-danger');
        deleteBtn.addEventListener('click', () => deleteGif(g.id));
        actions.appendChild(deleteBtn);

        tile.appendChild(actions);

        return tile;
    }

    async function pull(imagePath) {
        const ok = await confirmDialog(t('confirmPull'), { confirmText: t('syncUseThis'), cancelText: t('modalCancel') });
        if (!ok) return;
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

    async function shareGif(id) {
        const result = await showModal({
            fields: [
                { id: 'name', label: t('syncSharePromptName'), required: true },
                { id: 'tags', label: t('syncSharePromptTags') },
            ],
            confirmText: t('syncShareBtn'),
            cancelText: t('modalCancel'),
        });
        if (!result) return;
        const { name, tags } = result;
        try {
            const res = await authedFetch('/me/gifs/' + id + '/share', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, tags }),
            });
            if (!res.ok) throw new Error(await res.text());
            toast(t('toastSyncShared'));
            loadPersonalLibrary();
            loadPublicLibrary();
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    async function unshareGif(id) {
        const ok = await confirmDialog(t('confirmUnshare'), { confirmText: t('syncUnshareBtn'), cancelText: t('modalCancel') });
        if (!ok) return;
        try {
            const res = await authedFetch('/me/gifs/' + id + '/unshare', { method: 'POST' });
            if (!res.ok) throw new Error(await res.text());
            toast(t('toastSyncUnshared'));
            loadPersonalLibrary();
            loadPublicLibrary();
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    async function deleteGif(id) {
        const ok = await confirmDialog(t('confirmSyncDelete'), {
            confirmText: t('syncDeleteBtn'),
            cancelText: t('modalCancel'),
        });
        if (!ok) return;
        try {
            const res = await authedFetch('/me/gifs/' + id, { method: 'DELETE' });
            if (!res.ok) throw new Error(await res.text());
            imageCache.delete('me:' + id);
            toast(t('toastSyncDeleted'));
            loadPersonalLibrary();
            loadPublicLibrary();
        } catch (err) {
            toast(err.message || String(err));
        }
    }

    // Uploads whatever GIF the widget currently shows (fetched from the
    // app's own local /gif endpoint) into the linked account's server
    // library - the one upload control, in the "Ảnh GIF" section, that
    // decides both the local widget image and, on demand, the server copy.
    pushCurrentBtn.addEventListener('click', async () => {
        const ok = await confirmDialog(t('confirmPushUpload'), { confirmText: t('pushCurrentGifBtn'), cancelText: t('modalCancel') });
        if (!ok) return;
        try {
            const local = await fetch('/gif', { cache: 'no-store' });
            if (!local.ok) throw new Error(await local.text());
            const blob = await local.blob();

            const form = new FormData();
            form.append('gif', blob, 'current.gif');
            const res = await authedFetch('/me/gifs', { method: 'POST', body: form });
            if (res.status === 409) {
                toast(t('toastSyncUploadQuotaFull'));
                return;
            }
            if (!res.ok) throw new Error(await res.text());
            toast(t('toastSyncUploaded'));
            loadPersonalLibrary();
        } catch (err) {
            toast(err.message || String(err));
        }
    });

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

    document.getElementById('sync-unlink-btn').addEventListener('click', async () => {
        const ok = await confirmDialog(t('confirmUnlink'), { confirmText: t('syncUnlinkBtn'), cancelText: t('modalCancel') });
        if (!ok) return;
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(USERNAME_KEY);
        toast(t('toastSyncUnlinked'));
        render();
    });

    consumeCallbackToken().then(render);
}
