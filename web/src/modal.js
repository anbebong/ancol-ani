// A small reusable pixel-styled modal, used instead of the browser's
// native confirm()/prompt() (which look completely out of place next to
// the rest of this pixel-themed UI). Lazily builds its DOM on first use.
let overlay;

function ensureModal() {
    if (overlay) return;
    overlay = document.createElement('div');
    overlay.id = 'app-modal-overlay';
    overlay.className = 'fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50 hidden';
    overlay.innerHTML = `
        <div class="pixel-box p-4 w-full max-w-xs flex flex-col gap-3 text-left">
            <p id="app-modal-message" class="text-sm"></p>
            <div id="app-modal-fields" class="flex flex-col gap-2"></div>
            <div class="flex gap-2.5 mt-1">
                <button type="button" id="app-modal-cancel" class="pixel-btn-ghost rounded-lg flex-1 px-3 py-2 text-sm font-semibold"></button>
                <button type="button" id="app-modal-confirm" class="pixel-btn rounded-lg flex-1 px-3 py-2 text-sm font-semibold"></button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

/**
 * Shows the modal and resolves when the person answers.
 * - No `fields`: a plain confirm. Resolves true (confirmed) or false (cancelled).
 * - With `fields` ([{id, label, placeholder, required}]): resolves an
 *   object of {id: value}, or null if cancelled.
 */
export function showModal({ message = '', fields = [], confirmText = 'OK', cancelText = 'Huỷ' }) {
    ensureModal();

    return new Promise((resolve) => {
        const msgEl = document.getElementById('app-modal-message');
        const fieldsEl = document.getElementById('app-modal-fields');
        const cancelBtn = document.getElementById('app-modal-cancel');
        const confirmBtn = document.getElementById('app-modal-confirm');

        msgEl.textContent = message;
        msgEl.classList.toggle('hidden', !message);

        fieldsEl.innerHTML = '';
        for (const f of fields) {
            const wrap = document.createElement('div');
            wrap.className = 'flex flex-col gap-1';
            wrap.innerHTML = `
                <label class="text-xs text-[var(--muted)]">${f.label}</label>
                <input type="text" class="pixel-input rounded-lg px-3 py-2 text-sm" placeholder="${f.placeholder || ''}"/>
            `;
            wrap.querySelector('input').id = 'app-modal-field-' + f.id;
            fieldsEl.appendChild(wrap);
        }

        cancelBtn.textContent = cancelText;
        confirmBtn.textContent = confirmText;

        function close(result) {
            overlay.classList.add('hidden');
            confirmBtn.removeEventListener('click', onConfirm);
            cancelBtn.removeEventListener('click', onCancel);
            overlay.removeEventListener('keydown', onKeydown);
            resolve(result);
        }
        function onConfirm() {
            if (!fields.length) { close(true); return; }
            const result = {};
            for (const f of fields) {
                const input = document.getElementById('app-modal-field-' + f.id);
                const val = input.value.trim();
                if (f.required && !val) { input.focus(); return; }
                result[f.id] = val;
            }
            close(result);
        }
        function onCancel() { close(fields.length ? null : false); }
        function onKeydown(e) {
            if (e.key === 'Escape') onCancel();
            if (e.key === 'Enter' && fields.length <= 1) onConfirm();
        }

        confirmBtn.addEventListener('click', onConfirm);
        cancelBtn.addEventListener('click', onCancel);
        overlay.addEventListener('keydown', onKeydown);

        overlay.classList.remove('hidden');
        (fieldsEl.querySelector('input') || confirmBtn).focus();
    });
}

export function confirmDialog(message, { confirmText = 'Xác nhận', cancelText = 'Hủy' } = {}) {
    return showModal({ message, confirmText, cancelText });
}
