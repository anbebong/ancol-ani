// A small reusable modal matching the portal's pixel/DaisyUI theme, used
// instead of the browser's native confirm() (which looks out of place).
// Same pattern as the desktop app's web/src/modal.js.
let overlay;

function ensureModal() {
    if (overlay) return;
    overlay = document.createElement('div');
    overlay.id = 'app-modal-overlay';
    overlay.className = 'fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50 hidden';
    overlay.innerHTML = `
        <div class="card bg-base-100 w-full max-w-xs">
            <div class="card-body gap-3">
                <p id="app-modal-message" class="text-sm"></p>
                <div class="flex gap-2.5">
                    <button type="button" id="app-modal-cancel" class="btn btn-ghost flex-1"></button>
                    <button type="button" id="app-modal-confirm" class="btn btn-primary flex-1"></button>
                </div>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

export function confirmDialog(message, { confirmText = 'Xác nhận', cancelText = 'Hủy' } = {}) {
    ensureModal();

    return new Promise((resolve) => {
        document.getElementById('app-modal-message').textContent = message;
        const cancelBtn = document.getElementById('app-modal-cancel');
        const confirmBtn = document.getElementById('app-modal-confirm');
        cancelBtn.textContent = cancelText;
        confirmBtn.textContent = confirmText;

        function close(result) {
            overlay.classList.add('hidden');
            confirmBtn.removeEventListener('click', onConfirm);
            cancelBtn.removeEventListener('click', onCancel);
            overlay.removeEventListener('keydown', onKeydown);
            resolve(result);
        }
        function onConfirm() { close(true); }
        function onCancel() { close(false); }
        function onKeydown(e) {
            if (e.key === 'Escape') onCancel();
            if (e.key === 'Enter') onConfirm();
        }

        confirmBtn.addEventListener('click', onConfirm);
        cancelBtn.addEventListener('click', onCancel);
        overlay.addEventListener('keydown', onKeydown);

        overlay.classList.remove('hidden');
        confirmBtn.focus();
    });
}
