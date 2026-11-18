import './style.css';
import { getToken, api } from './api.js';
import { renderIcons } from './icons.js';

const needLogin = document.getElementById('need-login');
const galleryBox = document.getElementById('gallery-box');
const searchQ = document.getElementById('search-q');
const searchUser = document.getElementById('search-user');

const PAGE_SIZE = 10;

// GIF blobs never change once created, so cache them for this page
// session instead of re-fetching/re-decoding on every page change or
// search re-run that happens to include an already-seen slug.
const imageCache = new Map();
function cachedImageFetch(slug) {
    if (imageCache.has(slug)) return Promise.resolve(imageCache.get(slug));
    return api('/public/' + slug)
        .then((res) => res.blob())
        .then((blob) => { imageCache.set(slug, blob); return blob; });
}

renderIcons();

let page = 1;

async function load(p = page) {
    if (!getToken()) {
        needLogin.classList.remove('hidden');
        galleryBox.classList.add('hidden');
        return;
    }

    const params = new URLSearchParams({ page: p, pageSize: PAGE_SIZE });
    if (searchQ.value.trim()) params.set('q', searchQ.value.trim());
    if (searchUser.value.trim()) params.set('user', searchUser.value.trim());

    let items, total;
    try {
        const res = await api('/public?' + params.toString());
        ({ items, total } = await res.json());
    } catch (_) {
        needLogin.classList.remove('hidden');
        galleryBox.classList.add('hidden');
        return;
    }
    page = p;

    const grid = document.getElementById('grid');
    const emptyNote = document.getElementById('empty-note');
    grid.innerHTML = '';

    emptyNote.classList.toggle('hidden', items.length > 0);

    for (const entry of items) {
        const item = document.createElement('div');
        item.className = 'card bg-base-200 p-3 items-center gap-2 text-center';
        item.innerHTML = `
            <img class="w-20 h-20 object-contain rounded-lg bg-base-100"/>
            <div class="flex flex-col w-full">
                <span class="text-xs font-semibold truncate">${entry.name}</span>
                <span class="text-[10px] text-base-content/60 truncate">${entry.tags ? entry.tags.split(',').map((t) => '#' + t).join(' ') : ''}</span>
                <span class="text-xs text-base-content/60">bởi ${entry.username}</span>
                <span class="text-[10px] text-base-content/40">${new Date(entry.updatedAt.replace(' ', 'T') + 'Z').toLocaleString()}</span>
            </div>
        `;
        grid.appendChild(item);
        cachedImageFetch(entry.slug)
            .then((blob) => { item.querySelector('img').src = URL.createObjectURL(blob); })
            .catch(() => {});
    }

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
    document.getElementById('gallery-page-label').textContent = `Trang ${page}/${totalPages}`;
    document.getElementById('gallery-prev').disabled = page <= 1;
    document.getElementById('gallery-next').disabled = page >= totalPages;
}

document.getElementById('gallery-prev').addEventListener('click', () => load(page - 1));
document.getElementById('gallery-next').addEventListener('click', () => load(page + 1));
searchQ.addEventListener('input', () => load(1));
searchUser.addEventListener('input', () => load(1));

load();
