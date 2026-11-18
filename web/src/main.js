import './style.css';
import { parseGIF, decompressFrames } from 'gifuct-js';
import { t, getLang, setLang, applyTranslations } from './i18n.js';

applyTranslations();
document.getElementById('lang-toggle').addEventListener('click', () => {
    setLang(getLang() === 'vi' ? 'en' : 'vi');
    applyTranslations();
});

fetch('/version').then((r) => r.text()).then((v) => {
    document.getElementById('app-version').textContent = 'v' + v;
}).catch(() => {});

// Chromium's built-in <img> GIF animation can stall permanently after
// looping a while on some encodes (reproduced even in a plain Chrome tab).
// Decoding frames ourselves with gifuct-js and drawing them on a canvas on
// our own timer avoids that native animation engine entirely.
function playGif(canvas, url, isCurrent) {
    const ctx = canvas.getContext('2d');
    (async () => {
        const buffer = await (await fetch(url, { cache: 'no-store' })).arrayBuffer();
        if (!isCurrent()) return;
        const gif = parseGIF(buffer);
        const frames = decompressFrames(gif, true);
        if (!frames.length) return;

        canvas.width = gif.lsd.width;
        canvas.height = gif.lsd.height;

        const full = ctx.createImageData(gif.lsd.width, gif.lsd.height);
        let previous = null;
        let index = 0;

        function draw() {
            if (!isCurrent()) return;
            const frame = frames[index];
            if (frame.disposalType === 2) {
                full.data.fill(0);
            } else if (frame.disposalType === 3 && previous) {
                full.data.set(previous);
            }

            const { width, height, top, left } = frame.dims;
            const patch = frame.patch;
            for (let y = 0; y < height; y++) {
                for (let x = 0; x < width; x++) {
                    const srcIdx = (y * width + x) * 4;
                    if (patch[srcIdx + 3] === 0) continue;
                    const dstIdx = ((top + y) * gif.lsd.width + (left + x)) * 4;
                    full.data[dstIdx] = patch[srcIdx];
                    full.data[dstIdx + 1] = patch[srcIdx + 1];
                    full.data[dstIdx + 2] = patch[srcIdx + 2];
                    full.data[dstIdx + 3] = patch[srcIdx + 3];
                }
            }

            if (frame.disposalType === 3) previous = full.data.slice();

            ctx.putImageData(full, 0, 0);
            index = (index + 1) % frames.length;
            setTimeout(draw, Math.max(frame.delay, 20));
        }

        draw();
    })();
}

const preview = document.getElementById('preview');
let previewToken = 0;
function reloadPreview() {
    const token = ++previewToken;
    playGif(preview, '/gif?t=' + Date.now(), () => token === previewToken);
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

document.getElementById('reset-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    await fetch('/gif/reset', { method: 'POST' });
    reloadPreview();
    toast(t('toastGifReset'));
});
