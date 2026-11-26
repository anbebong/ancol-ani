<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import Icon from '@theme/components/Icon.vue';
import Pager from '@theme/components/Pager.vue';
import PageHeader from '@theme/components/PageHeader.vue';
import StarButton from '@theme/components/StarButton.vue';
import { confirmDialog } from '@theme/modal.js';
import GifCanvas from '../components/GifCanvas.vue';
import { t } from '../i18n.js';
import { account, serverApi, loadMe, cachedGif, forgetGif, pullToWidget, tagList, PAGE_SIZE } from '../account.js';
import { toast, toastError } from '../toast.js';

const items = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);

const drafts = reactive({});
function draft(id) {
    if (!drafts[id]) drafts[id] = { name: '', tags: '' };
    return drafts[id];
}

const quota = computed(() => account.me?.gifQuota ?? null);
const full = computed(() => quota.value !== null && total.value >= quota.value);

function report(err) { if (err.message !== '401') toastError(err); }

async function load(p = page.value) {
    try {
        const res = await serverApi(`/me/gifs?page=${p}&pageSize=${PAGE_SIZE}`);
        const data = await res.json();
        if (!data.items.length && p > 1) return load(p - 1);
        items.value = data.items;
        total.value = data.total;
        page.value = p;
    } catch (err) {
        report(err);
    } finally {
        loading.value = false;
    }
}

onMounted(() => {
    loadMe().catch(() => {});
    load(1);
});

async function run(fn, okMsg) {
    try {
        await fn();
        if (okMsg) toast(okMsg, 'success');
        await load();
    } catch (err) {
        report(err);
    }
}

async function use(g) {
    if (!(await confirmDialog(t('confirmPull'), { confirmText: t('useThis'), cancelText: t('modalCancel') }))) return;
    try {
        await pullToWidget(`/me/gifs/${g.id}/image`);
        toast(t('toastPulled'), 'success');
    } catch (err) {
        report(err);
    }
}

function share(g) {
    const d = draft(g.id);
    if (!d.name.trim()) { toast(t('errShareName'), 'error'); return; }
    run(() => serverApi(`/me/gifs/${g.id}/share`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: d.name, tags: d.tags }),
    }), t('toastShared'));
}

async function unshare(g) {
    if (!(await confirmDialog(t('confirmUnshare'), { confirmText: t('unshareBtn'), cancelText: t('modalCancel') }))) return;
    run(() => serverApi(`/me/gifs/${g.id}/unshare`, { method: 'POST' }), t('toastUnshared'));
}

async function remove(g) {
    if (!(await confirmDialog(t('confirmDelete'), { confirmText: t('deleteBtn'), cancelText: t('modalCancel') }))) return;
    run(async () => {
        await serverApi(`/me/gifs/${g.id}`, { method: 'DELETE' });
        forgetGif('me:' + g.id);
    }, t('toastDeleted'));
}

// ---------- upload a file straight into the server library ----------
const fileInput = ref(null);
const uploading = ref(false);
const dragOver = ref(false);

async function upload(file) {
    if (!file) return;
    if (!/\.gif$/i.test(file.name) && file.type !== 'image/gif') { toast(t('errGifOnly'), 'error'); return; }
    uploading.value = true;
    const form = new FormData();
    form.append('gif', file);
    try {
        await serverApi('/me/gifs', { method: 'POST', body: form });
        toast(t('toastUploaded'), 'success');
        await load();
    } catch (err) {
        if (/quota|toi da/i.test(err.message)) toast(t('toastQuotaFull'), 'error');
        else report(err);
    } finally {
        uploading.value = false;
        if (fileInput.value) fileInput.value.value = '';
    }
}
function onDrop(e) {
    dragOver.value = false;
    if (!full.value) upload(e.dataTransfer.files[0]);
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="images" :title="t('libraryTitle')" :subtitle="t('librarySubtitle')">
      <div class="quota-chip" :class="{ 'is-full': full }">
        <span class="text-2xl font-extrabold">{{ total }}</span>
        <span class="text-base-content/60">/ {{ quota ?? '–' }} GIF</span>
      </div>
    </PageHeader>

    <div v-if="!full" class="dropzone" :class="{ 'is-over': dragOver }"
      @click="fileInput?.click()" @dragover.prevent="dragOver = true" @dragleave="dragOver = false" @drop.prevent="onDrop">
      <input ref="fileInput" type="file" accept=".gif,image/gif" class="hidden" @change="upload($event.target.files[0])"/>
      <span v-if="uploading" class="loading loading-spinner"></span>
      <Icon v-else name="upload" :size="28" class="text-primary"/>
      <div>
        <p class="font-bold">{{ t('libraryDropTitle') }}</p>
        <p class="text-xs text-base-content/60">{{ t('dropzoneOr') }}</p>
      </div>
    </div>
    <div v-else class="alert alert-warning text-sm">{{ t('quotaFull', { quota }) }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!items.length" class="empty-state">
      <Icon name="image" :size="40" class="opacity-40"/>
      <p>{{ t('libraryEmpty') }}</p>
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
      <div v-for="g in items" :key="g.id" class="card bg-base-100 gif-card">
        <div class="card-body p-4 gap-3">
          <div class="relative flex justify-center">
            <GifCanvas :load="() => cachedGif('me:' + g.id, `/me/gifs/${g.id}/image`)" :cache-key="'me:' + g.id" size="w-full h-36"/>
            <span class="badge badge-sm absolute top-2 left-2" :class="g.isPublic ? 'badge-success' : 'badge-neutral'">
              {{ g.isPublic ? t('badgePublic') : t('badgePrivate') }}
            </span>
            <StarButton v-if="g.isPublic" readonly class="star-overlay" :count="g.stars || 0" :title="t('starsReceived')"/>
          </div>

          <button class="btn btn-primary btn-sm gap-1" @click="use(g)"><Icon name="download"/> {{ t('useThis') }}</button>

          <template v-if="g.isPublic">
            <div class="min-w-0">
              <p class="font-bold truncate">{{ g.name }}</p>
              <p v-if="g.tags" class="text-xs text-secondary truncate">{{ tagList(g.tags) }}</p>
            </div>
            <div class="flex gap-2">
              <button class="btn btn-warning btn-sm flex-1" @click="unshare(g)">{{ t('unshareBtn') }}</button>
              <button class="btn btn-error btn-sm btn-square" :title="t('deleteBtn')" @click="remove(g)"><Icon name="trash-2"/></button>
            </div>
          </template>

          <template v-else>
            <input v-model="draft(g.id).name" type="text" :placeholder="t('shareNamePlaceholder')" class="input input-bordered input-sm w-full"/>
            <input v-model="draft(g.id).tags" type="text" :placeholder="t('shareTagsPlaceholder')" class="input input-bordered input-sm w-full"/>
            <div class="flex gap-2">
              <button class="btn btn-success btn-sm flex-1 gap-1" @click="share(g)"><Icon name="globe"/> {{ t('shareBtn') }}</button>
              <button class="btn btn-error btn-sm btn-square" :title="t('deleteBtn')" @click="remove(g)"><Icon name="trash-2"/></button>
            </div>
          </template>
        </div>
      </div>
    </div>

    <Pager :page="page" :total="total" :page-size="PAGE_SIZE" :prev-label="t('pagerPrev')" :next-label="t('pagerNext')"
      :format-label="(p, n) => t('pagerLabel', { page: p, total: n })" @change="load"/>
  </div>
</template>
