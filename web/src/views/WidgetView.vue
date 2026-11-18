<script setup>
import { ref } from 'vue';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { confirmDialog } from '@ui/modal.js';
import GifCanvas from '../components/GifCanvas.vue';
import { t } from '../i18n.js';
import { account, serverApi } from '../account.js';
import { toast, toastError } from '../toast.js';

// ---------- text on the widget ----------
const text = ref('');
async function sendText() {
    if (!text.value.trim()) return;
    const form = new FormData();
    form.append('text', text.value);
    try {
        await fetch('/text', { method: 'POST', body: form });
        text.value = '';
        toast(t('toastTextSent'), 'success');
    } catch (err) {
        toastError(err);
    }
}

// ---------- GIF on the widget ----------
const previewKey = ref(String(Date.now()));
const loadPreview = () => fetch('/gif?t=' + previewKey.value, { cache: 'no-store' }).then((r) => r.arrayBuffer());
function reloadPreview() { previewKey.value = String(Date.now()); }

const fileInput = ref(null);
const chosen = ref(null);
const dragOver = ref(false);
const busy = ref(false);

function pick(file) {
    if (!file) return;
    if (!/\.gif$/i.test(file.name) && file.type !== 'image/gif') {
        toast(t('errGifOnly'), 'error');
        return;
    }
    chosen.value = file;
}
function onDrop(e) {
    dragOver.value = false;
    pick(e.dataTransfer.files[0]);
}

async function apply() {
    if (!chosen.value) { toast(t('errPickFile'), 'error'); return; }
    if (!(await confirmDialog(t('confirmApply'), { confirmText: t('uploadSubmit'), cancelText: t('modalCancel') }))) return;
    busy.value = true;
    try {
        const form = new FormData();
        form.append('gif', chosen.value);
        const res = await fetch('/gif', { method: 'POST', body: form });
        if (!res.ok) throw new Error(await res.text());
        chosen.value = null;
        if (fileInput.value) fileInput.value.value = '';
        reloadPreview();
        toast(t('toastGifChanged'), 'success');
    } catch (err) {
        toastError(err);
    } finally {
        busy.value = false;
    }
}

async function reset() {
    if (!(await confirmDialog(t('confirmReset'), { confirmText: t('resetSubmit'), cancelText: t('modalCancel') }))) return;
    await fetch('/gif/reset', { method: 'POST' });
    reloadPreview();
    toast(t('toastGifReset'), 'success');
}

// Upload whatever the widget currently shows into the linked account's library.
async function pushCurrent() {
    if (!(await confirmDialog(t('confirmPushUpload'), { confirmText: t('pushCurrentGifBtn'), cancelText: t('modalCancel') }))) return;
    try {
        const local = await fetch('/gif', { cache: 'no-store' });
        const form = new FormData();
        form.append('gif', await local.blob(), 'current.gif');
        await serverApi('/me/gifs', { method: 'POST', body: form });
        toast(t('toastUploaded'), 'success');
    } catch (err) {
        if (/quota|toi da/i.test(err.message)) toast(t('toastQuotaFull'), 'error');
        else if (err.message !== '401') toastError(err);
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="monitor" :title="t('widgetTitle')" :subtitle="t('subtitle')"/>

    <!-- Text -->
    <div class="card bg-base-100">
      <form class="card-body gap-3" @submit.prevent="sendText">
        <h2 class="font-bold flex items-center gap-2"><Icon name="type"/> {{ t('textLabel') }}</h2>
        <p class="text-xs text-base-content/60 -mt-1">{{ t('textHint') }}</p>
        <div class="flex flex-col sm:flex-row gap-2">
          <input v-model="text" type="text" :placeholder="t('textPlaceholder')" class="input input-bordered auth-input flex-1 min-w-0" autofocus/>
          <button type="submit" class="btn btn-primary">{{ t('textSubmit') }}</button>
        </div>
      </form>
    </div>

    <!-- GIF -->
    <div class="card bg-base-100">
      <div class="card-body gap-4">
        <h2 class="font-bold flex items-center gap-2"><Icon name="image"/> {{ t('gifLabel') }}</h2>

        <div class="flex flex-col sm:flex-row gap-5">
          <div class="flex flex-col items-center gap-2 shrink-0">
            <GifCanvas :load="loadPreview" :cache-key="previewKey" size="w-36 h-36"/>
            <span class="field-label">{{ t('gifCurrent') }}</span>
          </div>

          <div class="flex-1 min-w-0 flex flex-col gap-3">
            <div class="dropzone flex-1" :class="{ 'is-over': dragOver }"
              @click="fileInput?.click()" @dragover.prevent="dragOver = true" @dragleave="dragOver = false" @drop.prevent="onDrop">
              <input ref="fileInput" type="file" accept=".gif,image/gif" class="hidden" @change="pick($event.target.files[0])"/>
              <Icon name="upload" :size="28" class="text-primary"/>
              <div class="min-w-0">
                <p class="font-bold truncate">{{ chosen ? chosen.name : t('dropzoneDefault') }}</p>
                <p class="text-xs text-base-content/60">{{ t('dropzoneOr') }}</p>
              </div>
            </div>

            <div class="flex gap-2 flex-wrap">
              <button type="button" class="btn btn-primary flex-1" :disabled="busy || !chosen" @click="apply">
                <span v-if="busy" class="loading loading-spinner loading-sm"></span>
                <Icon v-else name="check"/> {{ t('uploadSubmit') }}
              </button>
              <button type="button" class="btn btn-ghost border-2 border-neutral flex-1 gap-1" @click="reset">
                <Icon name="rotate-ccw"/> {{ t('resetSubmit') }}
              </button>
            </div>
            <button v-if="account.token" type="button" class="btn btn-sm btn-ghost border-2 border-neutral gap-1 self-start" @click="pushCurrent">
              <Icon name="upload"/> {{ t('pushCurrentGifBtn') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
