<script setup>
import { ref, onMounted, watch } from 'vue';
import { api, errMsg, fmtDate, tagList, forgetImage, PAGE_SIZE } from '../api.js';
import { confirmDialog } from '../modal.js';
import Icon from '../components/Icon.vue';
import Pager from '../components/Pager.vue';
import GifImage from '../components/GifImage.vue';
import PageHeader from '../components/PageHeader.vue';

const q = ref('');
const items = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
const error = ref('');

let seq = 0;
async function load(p = page.value) {
    const mySeq = ++seq;
    error.value = '';
    const params = new URLSearchParams({ page: p, pageSize: PAGE_SIZE });
    if (q.value.trim()) params.set('q', q.value.trim());
    try {
        const res = await api('/public?' + params.toString());
        const data = await res.json();
        if (mySeq !== seq) return;
        if (!data.items.length && p > 1) return load(p - 1);
        items.value = data.items;
        total.value = data.total;
        page.value = p;
    } catch (err) {
        if (mySeq === seq) error.value = errMsg(err);
    } finally {
        if (mySeq === seq) loading.value = false;
    }
}

let debounce;
watch(q, () => {
    clearTimeout(debounce);
    debounce = setTimeout(() => load(1), 250);
});
onMounted(() => load(1));

async function unpublish(e) {
    if (!(await confirmDialog(`Gỡ "${e.name}" của ${e.username} khỏi thư viện public?`, { confirmText: 'Gỡ' }))) return;
    try {
        await api('/admin/public/' + e.slug, { method: 'DELETE' });
        forgetImage('public:' + e.slug);
        await load();
    } catch (err) {
        error.value = errMsg(err);
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="shield" title="Quản trị GIF public" :subtitle="`${total} GIF đang công khai trên toàn hệ thống`"/>

    <label class="input input-bordered input-sm flex items-center gap-2">
      <Icon name="search" class="opacity-50"/>
      <input v-model="q" type="text" placeholder="Tìm theo tên hoặc tag..." class="grow bg-transparent"/>
    </label>

    <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!items.length" class="empty-state">
      <Icon name="globe" :size="40" class="opacity-40"/>
      <p>Không có GIF public nào.</p>
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-4">
      <div v-for="e in items" :key="e.slug" class="card bg-base-100 gif-card">
        <div class="card-body p-3 gap-2">
          <GifImage :cache-key="'public:' + e.slug" :path="'/public/' + e.slug" size="w-full h-32"/>
          <div class="min-w-0">
            <p class="font-bold text-sm truncate">{{ e.name }}</p>
            <p v-if="e.tags" class="text-xs text-secondary truncate">{{ tagList(e.tags) }}</p>
            <p class="text-xs text-base-content/60 truncate">bởi {{ e.username }} · {{ fmtDate(e.updatedAt) }}</p>
          </div>
          <button class="btn btn-error btn-xs gap-1" @click="unpublish(e)"><Icon name="trash-2" :size="12"/> Gỡ public</button>
        </div>
      </div>
    </div>

    <Pager :page="page" :total="total" @change="load"/>
  </div>
</template>
