<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { api, errMsg, tagList, forgetImage, PAGE_SIZE } from '../api.js';
import { session, loadMe } from '../session.js';
import { confirmDialog } from '@ui/modal.js';
import Icon from '@ui/components/Icon.vue';
import Pager from '@ui/components/Pager.vue';
import GifImage from '../components/GifImage.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import StarButton from '@ui/components/StarButton.vue';

const items = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
const error = ref('');
const notice = ref('');

// Per-card share form inputs, keyed by GIF id.
const drafts = reactive({});
function draft(id) {
    if (!drafts[id]) drafts[id] = { name: '', tags: '' };
    return drafts[id];
}

const quota = computed(() => session.me?.gifQuota ?? null);
const full = computed(() => quota.value !== null && total.value >= quota.value);

async function load(p = page.value) {
    error.value = '';
    loading.value = true;
    try {
        const res = await api(`/me/gifs?page=${p}&pageSize=${PAGE_SIZE}`);
        const data = await res.json();
        // Deleting the last item on a page leaves it empty: step back.
        if (!data.items.length && p > 1) return load(p - 1);
        items.value = data.items;
        total.value = data.total;
        page.value = p;
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        loading.value = false;
    }
}

onMounted(() => {
    loadMe().catch(() => {});
    load(1);
});

async function run(fn, okMsg) {
    error.value = '';
    notice.value = '';
    try {
        await fn();
        if (okMsg) notice.value = okMsg;
        await load();
    } catch (err) {
        error.value = errMsg(err);
    }
}

function share(g) {
    const d = draft(g.id);
    if (!d.name.trim()) { error.value = 'Phải đặt tên cho GIF trước khi chia sẻ'; return; }
    run(() => api(`/me/gifs/${g.id}/share`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: d.name, tags: d.tags }),
    }), 'Đã chia sẻ công khai');
}

function unshare(g) {
    run(() => api(`/me/gifs/${g.id}/unshare`, { method: 'POST' }), 'Đã gỡ chia sẻ');
}

async function remove(g) {
    if (!(await confirmDialog('Xóa GIF này khỏi thư viện?', { confirmText: 'Xóa' }))) return;
    run(async () => {
        await api(`/me/gifs/${g.id}`, { method: 'DELETE' });
        forgetImage('me:' + g.id);
    }, 'Đã xóa GIF');
}

function publicURL(g) {
    return `${location.origin}/public/${g.slug}`;
}
async function copyLink(g) {
    try {
        await navigator.clipboard.writeText(publicURL(g));
        notice.value = 'Đã copy link';
    } catch (_) {
        error.value = 'Không copy được, hãy copy thủ công: ' + publicURL(g);
    }
}

// ---------- upload ----------
const fileInput = ref(null);
const uploading = ref(false);
const dragOver = ref(false);

function pickFile() { fileInput.value?.click(); }

async function upload(file) {
    if (!file) return;
    if (!/\.gif$/i.test(file.name) && file.type !== 'image/gif') {
        error.value = 'Chỉ nhận file .gif';
        return;
    }
    uploading.value = true;
    const form = new FormData();
    form.append('gif', file);
    await run(() => api('/me/gifs', { method: 'POST', body: form }), 'Đã tải lên ' + file.name);
    uploading.value = false;
    if (fileInput.value) fileInput.value.value = '';
}

function onDrop(e) {
    dragOver.value = false;
    if (full.value) return;
    upload(e.dataTransfer.files[0]);
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="images" title="Thư viện của tôi"
      subtitle="Mỗi GIF có thể bật/tắt chia sẻ công khai riêng. Thiết bị đăng nhập cùng tài khoản sẽ chọn 1 GIF ở đây để đồng bộ.">
      <div class="quota-chip" :class="{ 'is-full': full }">
        <span class="text-2xl font-extrabold">{{ total }}</span>
        <span class="text-base-content/60">/ {{ quota ?? '–' }} GIF</span>
      </div>
    </PageHeader>

    <!-- Upload dropzone -->
    <div v-if="!full" class="dropzone" :class="{ 'is-over': dragOver }"
      @click="pickFile" @dragover.prevent="dragOver = true" @dragleave="dragOver = false" @drop.prevent="onDrop">
      <input ref="fileInput" type="file" accept=".gif,image/gif" class="hidden" @change="upload($event.target.files[0])"/>
      <span v-if="uploading" class="loading loading-spinner"></span>
      <Icon v-else name="upload" :size="28" class="text-primary"/>
      <div>
        <p class="font-bold">{{ uploading ? 'Đang tải lên...' : 'Kéo thả file GIF vào đây' }}</p>
        <p class="text-xs text-base-content/60">hoặc bấm để chọn file từ máy</p>
      </div>
    </div>
    <div v-else class="alert alert-warning text-sm">
      Đã đạt số lượng GIF tối đa ({{ quota }}). Xóa bớt 1 GIF để thêm cái mới.
    </div>

    <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>
    <div v-else-if="notice" class="alert alert-success text-sm py-2">{{ notice }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!items.length" class="empty-state">
      <Icon name="image" :size="40" class="opacity-40"/>
      <p>Chưa có GIF nào. Tải lên GIF đầu tiên ở khung phía trên.</p>
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
      <div v-for="g in items" :key="g.id" class="card bg-base-100 gif-card">
        <div class="card-body p-4 gap-3">
          <div class="relative flex justify-center">
            <GifImage :cache-key="'me:' + g.id" :path="`/me/gifs/${g.id}/image`" size="w-full h-36"/>
            <span class="badge badge-sm absolute top-2 left-2" :class="g.isPublic ? 'badge-success' : 'badge-neutral'">
              {{ g.isPublic ? 'công khai' : 'riêng tư' }}
            </span>
            <StarButton v-if="g.isPublic" readonly class="star-overlay" :count="g.stars || 0" title="Số người đã star GIF này"/>
          </div>

          <template v-if="g.isPublic">
            <div class="min-w-0">
              <p class="font-bold truncate">{{ g.name }}</p>
              <p v-if="g.tags" class="text-xs text-base-content/60 truncate">{{ tagList(g.tags) }}</p>
            </div>
            <button type="button" class="link-box" :title="publicURL(g)" @click="copyLink(g)">
              <Icon name="share-2" :size="12"/>
              <span class="truncate">/public/{{ g.slug }}</span>
            </button>
            <div class="flex gap-2">
              <button class="btn btn-warning btn-sm flex-1" @click="unshare(g)">Gỡ chia sẻ</button>
              <button class="btn btn-error btn-sm btn-square" title="Xóa" @click="remove(g)"><Icon name="trash-2"/></button>
            </div>
          </template>

          <template v-else>
            <input v-model="draft(g.id).name" type="text" placeholder="Tên GIF (bắt buộc để chia sẻ)" class="input input-bordered input-sm w-full"/>
            <input v-model="draft(g.id).tags" type="text" placeholder="Tag, phân tách bằng dấu phẩy" class="input input-bordered input-sm w-full"/>
            <div class="flex gap-2">
              <button class="btn btn-success btn-sm flex-1 gap-1" @click="share(g)"><Icon name="globe"/> Chia sẻ</button>
              <button class="btn btn-error btn-sm btn-square" title="Xóa" @click="remove(g)"><Icon name="trash-2"/></button>
            </div>
          </template>
        </div>
      </div>
    </div>

    <Pager :page="page" :total="total" @change="load"/>
  </div>
</template>
