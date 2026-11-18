import './style.css';
import { t, getLang, setLang, applyTranslations } from './i18n.js';
import { initSync } from './sync.js';
import { playGif } from './gif-player.js';
import { confirmDialog } from './modal.js';

applyTranslations();
document.getElementById('lang-toggle').addEventListener('click', () => {
    setLang(getLang() === 'vi' ? 'en' : 'vi');
    applyTranslations();
});

fetch('/version').then((r) => r.text()).then((v) => {
    document.getElementById('app-version').textContent = 'v' + v;
}).catch(() => {});

const preview = document.getElementById('preview');
let previewToken = 0;
function reloadPreview() {
    const token = ++previewToken;
    const url = '/gif?t=' + Date.now();
    playGif(preview, () => fetch(url, { cache: 'no-store' }).then((r) => r.arrayBuffer()), () => token === previewToken);
}
reloadPreview();

let toastTimer = null;
function toast(message) {
    const el = document.getElementById('toast');
    el.textContent = message;
    el.classList.remove('opacity-0', 'translate-y-2');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.add('opacity-0', 'translate-y-2'), 2200);
}

document.getElementById('text-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    await fetch('/text', { method: 'POST', body: new FormData(form) });
    form.querySelector('input[name=text]').value = '';
    toast(t('toastTextSent'));
});

const dropzone = document.getElementById('dropzone');
const dropzoneName = document.getElementById('dropzone-name');
const fileInput = dropzone.querySelector('input[type=file]');

function setChosenFile(file) {
    dropzoneName.textContent = file ? file.name : t('dropzoneDefault');
}

['dragover', 'dragenter'].forEach((evt) =>
    dropzone.addEventListener(evt, (e) => {
        e.preventDefault();
        dropzone.classList.add('drag');
    })
);
['dragleave', 'dragend'].forEach((evt) =>
    dropzone.addEventListener(evt, () => dropzone.classList.remove('drag'))
);
dropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzone.classList.remove('drag');
    const file = e.dataTransfer.files[0];
    if (!file) return;
    fileInput.files = e.dataTransfer.files;
    setChosenFile(file);
});
fileInput.addEventListener('change', () => setChosenFile(fileInput.files[0]));

document.getElementById('gif-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    const ok = await confirmDialog(t('confirmApply'), { confirmText: t('uploadSubmit'), cancelText: t('modalCancel') });
    if (!ok) return;

    const res = await fetch('/gif', { method: 'POST', body: new FormData(form) });
    if (!res.ok) {
        toast(await res.text());
        return;
    }
    form.reset();
    setChosenFile(null);
    reloadPreview();
    toast(t('toastGifChanged'));
});

document.getElementById('reset-btn').addEventListener('click', async () => {
    const ok = await confirmDialog(t('confirmReset'), { confirmText: t('resetSubmit'), cancelText: t('modalCancel') });
    if (!ok) return;

    await fetch('/gif/reset', { method: 'POST' });
    reloadPreview();
    toast(t('toastGifReset'));
});

initSync({ toast, reloadPreview });
