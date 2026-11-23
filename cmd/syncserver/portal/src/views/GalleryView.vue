<script setup>
import { ref, onMounted, watch } from 'vue';
import { api, errMsg, fmtDate, tagList, PAGE_SIZE } from '../api.js';
import Icon from '@ui/components/Icon.vue';
import Pager from '@ui/components/Pager.vue';
import GifImage from '../components/GifImage.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import StarButton from '@ui/components/StarButton.vue';

const q = ref('');
const user = ref('');
const sort = ref('new'); // 'new' | 'stars'
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
    if (user.value.trim()) params.set('user', user.value.trim());
    if (sort.value === 'stars') params.set('sort', 'stars');
    try {
        const res = await api('/public?' + params.toString());
        const data = await res.json();
        if (mySeq !== seq) return; // a newer search already started
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
watch([q, user], () => {
    clearTimeout(debounce);
    debounce = setTimeout(() => load(1), 250);
});

watch(sort, () => load(1));
onMounted(() => load(1));

// GitHub-style star: one per user per GIF, toggled on the card.
const starBusy = ref({});
async function toggleStar(e) {
    if (starBusy.value[e.slug]) return;
    starBusy.value[e.slug] = true;
    error.value = '';
    try {
        const res = await api(`/public/${e.slug}/star`, { method: e.starred ? 'DELETE' : 'POST' });
        const data = await res.json();
        e.stars = data.stars;
        e.starred = data.starred;
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        starBusy.value[e.slug] = false;
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="compass" title="Khám phá" subtitle="GIF mọi người đã chia sẻ công khai."/>

    <div class="flex gap-3 flex-wrap">
      <label class="input input-bordered input-sm flex items-center gap-2 flex-1 min-w-[200px]">
        <Icon name="search" class="opacity-50"/>
        <input v-model="q" type="text" placeholder="Tìm theo tên hoặc tag..." class="grow bg-transparent"/>
      </label>
      <label class="input input-bordered input-sm flex items-center gap-2 flex-1 min-w-[200px]">
        <Icon name="user" class="opacity-50"/>
        <input v-model="user" type="text" placeholder="Lọc theo người chia sẻ..." class="grow bg-transparent"/>
      </label>
      <select v-model="sort" class="select select-bordered select-sm" title="Sắp xếp">
        <option value="new">Mới nhất</option>
        <option value="stars">Nhiều star nhất</option>
      </select>
    </div>

    <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!items.length" class="empty-state">
      <Icon name="globe" :size="40" class="opacity-40"/>
      <p>{{ q || user ? 'Không tìm thấy GIF nào khớp.' : 'Chưa có GIF nào được chia sẻ công khai.' }}</p>
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-4">
      <div v-for="e in items" :key="e.slug" class="card bg-base-100 gif-card">
        <div class="card-body p-3 gap-2">
          <div class="relative">
            <GifImage :cache-key="'public:' + e.slug" :path="'/public/' + e.slug" size="w-full h-32"/>
            <StarButton class="star-overlay" :starred="e.starred" :count="e.stars || 0" :busy="!!starBusy[e.slug]"
              :title="e.starred ? 'Bỏ star' : 'Star GIF này'" @toggle="toggleStar(e)"/>
          </div>
          <div class="min-w-0">
            <p class="font-bold text-sm truncate">{{ e.name }}</p>
            <p v-if="e.tags" class="text-xs text-secondary truncate">{{ tagList(e.tags) }}</p>
            <p class="text-xs text-base-content/60 truncate">bởi {{ e.username }}</p>
            <p class="text-[10px] text-base-content/40">{{ fmtDate(e.updatedAt) }}</p>
          </div>        </div>
      </div>
    </div>

    <Pager :page="page" :total="total" @change="load"/>
  </div>
</template>
