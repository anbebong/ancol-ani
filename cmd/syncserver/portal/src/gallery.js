import './style.css';
import { getToken, api } from './api.js';
import { renderIcons } from './icons.js';

const needLogin = document.getElementById('need-login');
const galleryBox = document.getElementById('gallery-box');
const searchQ = document.getElementById('search-q');
const searchUser = document.getElementById('search-user');

renderIcons();

async function load() {
    if (!getToken()) {
        needLogin.classList.remove('hidden');
        galleryBox.classList.add('hidden');
        return;
    }

    const params = new URLSearchParams();
    if (searchQ.value.trim()) params.set('q', searchQ.value.trim());
    if (searchUser.value.trim()) params.set('user', searchUser.value.trim());
    const qs = params.toString() ? '?' + params.toString() : '';

    let entries;
    try {
        const res = await api('/public' + qs);
        entries = await res.json();
    } catch (_) {
        needLogin.classList.remove('hidden');
        galleryBox.classList.add('hidden');
        return;
    }

    const grid = document.getElementById('grid');
    const emptyNote = document.getElementById('empty-note');
    grid.innerHTML = '';

    if (!entries.length) {
        emptyNote.classList.remove('hidden');
        return;
    }
    emptyNote.classList.add('hidden');

    for (const entry of entries) {
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
        api('/public/' + entry.slug)
            .then((res) => res.blob())
            .then((blob) => { item.querySelector('img').src = URL.createObjectURL(blob); })
            .catch(() => {});
    }
}

searchQ.addEventListener('input', load);
searchUser.addEventListener('input', load);

load();
