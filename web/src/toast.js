import { reactive } from 'vue';

// One app-wide toast at the bottom of the screen (rendered by App.vue).
export const toastState = reactive({ message: '', kind: 'info', visible: false });

let timer = null;
export function toast(message, kind = 'info') {
    toastState.message = message;
    toastState.kind = kind;
    toastState.visible = true;
    clearTimeout(timer);
    timer = setTimeout(() => { toastState.visible = false; }, kind === 'error' ? 4000 : 2400);
}

export function toastError(err) {
    toast(err?.message || String(err), 'error');
}
