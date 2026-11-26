<script setup>
import { ref, onMounted, watch } from 'vue';
import Icon from '@theme/components/Icon.vue';
import Pager from '@theme/components/Pager.vue';
import PageHeader from '@theme/components/PageHeader.vue';
import StarButton from '@theme/components/StarButton.vue';
import { confirmDialog } from '@theme/modal.js';
import GifCanvas from '../components/GifCanvas.vue';
import { t } from '../i18n.js';
import { serverApi, cachedGif, pullToWidget, fmtDate, tagList, PAGE_SIZE } from '../account.js';
import { toast, toastError } from '../toast.js';

const q = ref('');
const user = ref('');
const sort = ref('new'); // 'new' | 'stars'
const items = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);

let seq = 0;
async function load(p = page.value) {
    const mine = ++seq;
    const params = new URLSearchParams({ page: p, pageSize: PAGE_SIZE });
    if (q.value.trim()) params.set('q', q.value.trim());
    if (user.value.trim()) params.set('user', user.value.trim());
    if (sort.value === 'stars') params.set('sort', 'stars');
    try {
        const res = await serverApi('/public?' + params.toString());
        const data = await res.json();
        if (mine !== seq) return; // a newer search already started
        items.value = data.items;
        total.value = data.total;
        page.value = p;
    } catch (err) {
        if (mine === seq && err.message !== '401') toastError(err);
    } finally {
        if (mine === seq) loading.value = false;
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
    try {
        const res = await serverApi(`/public/${e.slug}/star`, { method: e.starred ? 'DELETE' : 'POST' });
        const data = await res.json();
        e.stars = data.stars;
        e.starred = data.starred;
    } catch (err) {
        if (err.message !== '401') toastError(err);
    } finally {
        starBusy.value[e.slug] = false;
    }
}

async function use(e) {
    if (!(await confirmDialog(t('confirmPull'), { confirmText: t('useThis'), cancelText: t('modalCancel') }))) return;
    try {
        await pullToWidget('/public/' + e.slug);
        toast(t('toastPulled'), 'success');
    } catch (err) {
        if (err.message !== '401') toastError(err);
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="compass" :title="t('galleryTitle')" :subtitle="t('gallerySubtitle')"/>

    <div class="flex gap-3 flex-wrap">
      <label class="input input-bordered input-sm flex items-center gap-2 flex-1 min-w-[200px]">
        <Icon name="search" class="opacity-50"/>
        <input v-model="q" type="text" :placeholder="t('searchPlaceholder')" class="grow bg-transparent"/>
      </label>
      <label class="input input-bordered input-sm flex items-center gap-2 flex-1 min-w-[200px]">
        <Icon name="user" class="opacity-50"/>
        <input v-model="user" type="text" :placeholder="t('userFilterPlaceholder')" class="grow bg-transparent"/>
      </label>
      <select v-model="sort" class="select select-bordered select-sm" :title="t('sortLabel')">
        <option value="new">{{ t('sortNew') }}</option>
        <option value="stars">{{ t('sortStars') }}</option>
      </select>
    </div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!items.length" class="empty-state">
      <Icon name="globe" :size="40" class="opacity-40"/>
      <p>{{ q || user ? t('galleryNoMatch') : t('galleryEmpty') }}</p>
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-4">
      <div v-for="e in items" :key="e.slug" class="card bg-base-100 gif-card">
        <div class="card-body p-3 gap-2">
          <div class="relative">
            <GifCanvas :load="() => cachedGif('public:' + e.slug, '/public/' + e.slug)" :cache-key="'public:' + e.slug" size="w-full h-32"/>
            <StarButton class="star-overlay" :starred="e.starred" :count="e.stars || 0" :busy="!!starBusy[e.slug]"
              :title="e.starred ? t('unstarTitle') : t('starTitle')" @toggle="toggleStar(e)"/>
          </div>
          <div class="min-w-0">
            <p class="font-bold text-sm truncate">{{ e.name }}</p>
            <p v-if="e.tags" class="text-xs text-secondary truncate">{{ tagList(e.tags) }}</p>
            <p class="text-xs text-base-content/60 truncate">{{ t('by') }} {{ e.username }}</p>
            <p class="text-[10px] text-base-content/40">{{ fmtDate(e.updatedAt) }}</p>
          </div>
          <button class="btn btn-primary btn-xs gap-1" @click="use(e)"><Icon name="download" :size="12"/> {{ t('useThis') }}</button>
        </div>
      </div>
    </div>

    <Pager :page="page" :total="total" :page-size="PAGE_SIZE" :prev-label="t('pagerPrev')" :next-label="t('pagerNext')"
      :format-label="(p, n) => t('pagerLabel', { page: p, total: n })" @change="load"/>
  </div>
</template>
