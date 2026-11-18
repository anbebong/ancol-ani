import './style.css';
import { getToken, setToken, clearToken, setUsername, clearUsername, api, showError } from './api.js';
import { renderIcons, iconHTML } from './icons.js';

const authBox = document.getElementById('auth-box');
const accountBox = document.getElementById('account-box');
const adminSection = document.getElementById('admin-section');
const authError = document.getElementById('auth-error');
const accountError = document.getElementById('account-error');
const adminError = document.getElementById('admin-error');
const whoami = document.getElementById('whoami');

renderIcons();

// If this tab already has a session (logged in earlier, separately from
// whatever just opened this ?callback= link), hand that existing token
// back immediately instead of waiting for a fresh login that will never
// happen - refreshUI() below would otherwise just show the dashboard and
// silently ignore the callback.
const callbackParam = new URLSearchParams(location.search).get('callback');
if (callbackParam && getToken()) {
    location.href = callbackParam + (callbackParam.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(getToken());
} else if (callbackParam) {
    document.getElementById('callback-note').classList.remove('hidden');
}

let authMode = 'login';
document.getElementById('tab-login').addEventListener('click', () => setAuthMode('login'));
document.getElementById('tab-register').addEventListener('click', () => setAuthMode('register'));

function setAuthMode(mode) {
    authMode = mode;
    document.getElementById('tab-login').classList.toggle('tab-active', mode === 'login');
    document.getElementById('tab-register').classList.toggle('tab-active', mode === 'register');
    document.getElementById('auth-submit-btn').textContent = mode === 'login' ? 'Đăng nhập' : 'Tạo tài khoản';
    authError.textContent = '';
}

async function refreshUI() {
    const loggedIn = !!getToken();
    authBox.classList.toggle('hidden', loggedIn);
    accountBox.classList.toggle('hidden', !loggedIn);
    if (!loggedIn) {
        whoami.innerHTML = '';
        return;
    }
    await loadMe();
    loadGifLibrary();
    loadHistory();
    await tryShowAdmin();
}

function renderWhoami(username, isAdmin) {
    whoami.innerHTML = isAdmin
        ? `<span class="w-3.5 h-3.5 [&>svg]:w-3.5 [&>svg]:h-3.5">${iconHTML('crown')}</span> ${username}`
        : username;
}

async function loadMe() {
    try {
        const res = await api('/me');
        const me = await res.json();
        lastKnownQuota = me.gifQuota;
        document.getElementById('gif-quota').textContent = me.gifQuota;
        checkQuota(Number(document.getElementById('gif-count').textContent));
        renderWhoami(me.username, me.isAdmin);
    } catch (err) {
        showError(accountError, err);
    }
}

function fmtDate(s) {
    return new Date(s.replace(' ', 'T') + 'Z').toLocaleString();
}

function iconCell(name) {
    return `<span class="w-4 h-4 inline-block align-middle [&>svg]:w-4 [&>svg]:h-4">${iconHTML(name)}</span>`;
}

// ---------- GIF library (up to the user's quota, each slot shareable
// independently; exactly one slot is "active" = synced to devices) ----------

async function loadGifLibrary() {
    try {
        const res = await api('/me/gifs');
        const gifs = await res.json();
        renderGifLibrary(gifs);
    } catch (err) {
        showError(accountError, err);
    }
}

function renderGifLibrary(gifs) {
    const grid = document.getElementById('gif-grid');
    const empty = document.getElementById('gif-empty');
    grid.innerHTML = '';

    if (!gifs.length) {
        empty.classList.remove('hidden');
    } else {
        empty.classList.add('hidden');
        for (const g of gifs) {
            grid.appendChild(renderGifCard(g));
        }
    }

    document.getElementById('gif-count').textContent = gifs.length;
    // quota is only known precisely via /admin/users for other users;
    // for the logged-in user's own quota, infer "full" by trying upload
    // and reacting to 409, but show a live count either way.
    checkQuota(gifs.length);
}

let lastKnownQuota = null;
function checkQuota(count) {
    const full = lastKnownQuota !== null && count >= lastKnownQuota;
    document.getElementById('upload-row').classList.toggle('hidden', full);
    document.getElementById('quota-full-note').classList.toggle('hidden', !full);
}

function renderGifCard(g) {
    const card = document.createElement('div');
    card.className = 'card bg-base-200 p-3 gap-2 items-center text-center';
    card.innerHTML = `
        <div class="w-20 h-20 rounded-lg bg-base-100 border-2 border-base-300 flex items-center justify-center overflow-hidden">
            <img class="object-contain max-w-full max-h-full"/>
        </div>
        <span class="badge badge-sm ${g.isPublic ? 'badge-success' : 'badge-outline'}">${g.isPublic ? 'công khai' : 'riêng tư'}</span>

        ${g.isPublic ? `
            <div class="w-full text-left">
                <p class="text-xs font-semibold truncate">${g.name}</p>
                ${g.tags ? `<p class="text-[10px] text-base-content/60 truncate">${g.tags.split(',').map((t) => '#' + t).join(' ')}</p>` : ''}
            </div>
            <button class="btn btn-xs btn-error btn-outline w-full unshare-btn">Gỡ chia sẻ</button>
        ` : `
            <input type="text" placeholder="Tên GIF (bắt buộc để chia sẻ)" class="input input-bordered input-xs w-full share-name-input"/>
            <input type="text" placeholder="Tag, phân tách bằng dấu phẩy" class="input input-bordered input-xs w-full share-tags-input"/>
            <button class="btn btn-xs btn-secondary w-full share-btn">Chia sẻ</button>
        `}
        <button class="btn btn-xs btn-ghost w-full delete-btn">Xóa</button>
        ${g.isPublic ? `<p class="text-[10px] break-all text-base-content/60">${location.origin}/public/${g.slug}</p>` : ''}
    `;

    api(`/me/gifs/${g.id}/image`)
        .then((res) => res.blob())
        .then((blob) => { card.querySelector('img').src = URL.createObjectURL(blob); })
        .catch(() => {});

    card.querySelector('.share-btn')?.addEventListener('click', () => {
        const name = card.querySelector('.share-name-input').value;
        const tags = card.querySelector('.share-tags-input').value;
        shareGif(g.id, name, tags);
    });
    card.querySelector('.unshare-btn')?.addEventListener('click', () => unshareGif(g.id));
    card.querySelector('.delete-btn').addEventListener('click', () => deleteGif(g.id));

    return card;
}

async function shareGif(id, name, tags) {
    accountError.textContent = '';
    if (!name.trim()) {
        accountError.textContent = 'Phải đặt tên cho GIF trước khi chia sẻ';
        return;
    }
    try {
        await api(`/me/gifs/${id}/share`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, tags }),
        });
        loadGifLibrary();
        loadHistory();
    } catch (err) {
        showError(accountError, err);
    }
}

async function unshareGif(id) {
    accountError.textContent = '';
    try {
        await api(`/me/gifs/${id}/unshare`, { method: 'POST' });
        loadGifLibrary();
        loadHistory();
    } catch (err) {
        showError(accountError, err);
    }
}

async function deleteGif(id) {
    if (!confirm('Xóa GIF này khỏi thư viện?')) return;
    accountError.textContent = '';
    try {
        await api(`/me/gifs/${id}`, { method: 'DELETE' });
        loadGifLibrary();
        loadHistory();
    } catch (err) {
        showError(accountError, err);
    }
}

document.getElementById('upload-btn').addEventListener('click', async () => {
    accountError.textContent = '';
    const file = document.getElementById('gif-file').files[0];
    if (!file) { accountError.textContent = 'Chọn file GIF trước'; return; }
    const form = new FormData();
    form.append('gif', file);
    try {
        await api('/me/gifs', { method: 'POST', body: form });
        document.getElementById('gif-file').value = '';
        loadGifLibrary();
    } catch (err) {
        if (/quota|toi da/i.test(err.message)) {
            lastKnownQuota = Number(document.getElementById('gif-count').textContent);
            checkQuota(lastKnownQuota);
        }
        showError(accountError, err);
    }
});

// ---------- share history (across all of the user's gifs) ----------

async function loadHistory() {
    try {
        const res = await api('/me/gif/history');
        const entries = await res.json();
        const rows = document.getElementById('history-rows');
        const empty = document.getElementById('history-empty');
        rows.innerHTML = '';
        if (!entries.length) {
            empty.classList.remove('hidden');
            return;
        }
        empty.classList.add('hidden');
        for (const h of entries) {
            const tr = document.createElement('tr');
            const active = !h.unsharedAt;
            tr.innerHTML = `
                <td class="text-xs break-all">/public/${h.slug}</td>
                <td class="text-xs">${fmtDate(h.sharedAt)}</td>
                <td class="text-xs">${active ? '—' : fmtDate(h.unsharedAt)}</td>
                <td>${active ? '<span class="badge badge-success badge-sm">đang chia sẻ</span>' : '<span class="badge badge-ghost badge-sm">đã gỡ</span>'}</td>
            `;
            rows.appendChild(tr);
        }
    } catch (err) {
        showError(accountError, err);
    }
}

// ---------- auth ----------

async function doAuth() {
    authError.textContent = '';
    const path = authMode === 'login' ? '/auth/login' : '/auth/register';
    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;
    try {
        const res = await fetch(path, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        if (!res.ok) throw new Error(await res.text());
        const { token } = await res.json();

        // When opened with ?callback=... (the desktop app's local settings
        // page linking its account), hand the token back to that page via
        // redirect instead of keeping it here - this portal never stores
        // tokens on the client's behalf.
        const callback = new URLSearchParams(location.search).get('callback');
        if (callback) {
            location.href = callback + (callback.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(token);
            return;
        }

        setToken(token);
        setUsername(username);
        refreshUI();
    } catch (err) {
        showError(authError, err);
    }
}

document.getElementById('auth-submit-btn').addEventListener('click', doAuth);

document.getElementById('logout-btn').addEventListener('click', async () => {
    try { await api('/auth/logout', { method: 'POST' }); } catch (_) {}
    clearToken();
    clearUsername();
    refreshUI();
});

// ---------- admin ----------

// Thu tai /admin/users: 200 nghia la tai khoan nay co quyen admin -> hien
// them phan quan tri ngay tren trang nay. 403 nghia la user thuong, chi
// an phan do di, khong bao loi gi ca.
async function tryShowAdmin() {
    try {
        await loadUsers();
        await loadPublicAdmin();
        adminSection.classList.remove('hidden');
        adminSection.classList.add('flex');
    } catch (err) {
        adminSection.classList.add('hidden');
        adminSection.classList.remove('flex');
        whoami.innerHTML = '';
    }
}

document.getElementById('admin-public-search').addEventListener('input', () => loadPublicAdmin());

async function loadUsers() {
    const res = await api('/admin/users');
    const users = await res.json();
    const rows = document.getElementById('user-rows');
    rows.innerHTML = '';
    for (const u of users) {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${u.id}</td>
            <td>${u.username}</td>
            <td>${u.isAdmin ? iconCell('crown') : iconCell('minus')}</td>
            <td><input type="number" min="0" max="50" value="${u.gifQuota}" class="input input-bordered input-xs w-16 quota-input"/></td>
            <td>${u.gifCount}</td>
            <td>${u.hasPublic ? iconCell('globe') : iconCell('minus')}</td>
            <td class="text-right"><button class="btn btn-error btn-xs gap-1" data-id="${u.id}">${iconCell('trash-2')}Xóa</button></td>
        `;
        tr.querySelector('.quota-input').addEventListener('change', (e) => setQuota(u.id, e.target.value));
        tr.querySelector('button').addEventListener('click', () => deleteUser(u.id));
        rows.appendChild(tr);
    }
}

async function setQuota(userId, quota) {
    try {
        await api('/admin/users/' + userId + '/quota', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ quota: Number(quota) }),
        });
        await loadUsers();
    } catch (err) {
        showError(adminError, err);
    }
}

async function deleteUser(id) {
    if (!confirm('Xóa user này và toàn bộ GIF của họ?')) return;
    try {
        await api('/admin/users/' + id, { method: 'DELETE' });
        await loadUsers();
        await loadPublicAdmin();
    } catch (err) {
        showError(adminError, err);
    }
}

async function loadPublicAdmin() {
    const q = document.getElementById('admin-public-search')?.value.trim() || '';
    const params = q ? '?q=' + encodeURIComponent(q) : '';
    const res = await api('/public' + params);
    const entries = await res.json();
    const grid = document.getElementById('public-grid');
    grid.innerHTML = '';
    for (const entry of entries) {
        const item = document.createElement('div');
        item.className = 'card bg-base-200 p-3 items-center gap-2 text-center';
        item.innerHTML = `
            <img class="w-20 h-20 object-contain rounded-lg bg-base-100"/>
            <div class="flex flex-col w-full">
                <span class="text-xs font-semibold truncate">${entry.name}</span>
                <span class="text-[10px] text-base-content/60 truncate">${entry.tags ? entry.tags.split(',').map((t) => '#' + t).join(' ') : ''}</span>
                <span class="text-xs text-base-content/60">bởi ${entry.username} · ${fmtDate(entry.updatedAt)}</span>
            </div>
            <button class="btn btn-error btn-xs w-full gap-1" data-slug="${entry.slug}">${iconCell('trash-2')}Gỡ public</button>
        `;
        item.querySelector('button').addEventListener('click', () => unpublish(entry.slug));
        grid.appendChild(item);
        api('/public/' + entry.slug)
            .then((res) => res.blob())
            .then((blob) => { item.querySelector('img').src = URL.createObjectURL(blob); })
            .catch(() => {});
    }
}

async function unpublish(slug) {
    try {
        await api('/admin/public/' + slug, { method: 'DELETE' });
        await loadPublicAdmin();
    } catch (err) {
        showError(adminError, err);
    }
}

refreshUI();
