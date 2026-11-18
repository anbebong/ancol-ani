<script setup>
import { ref, watch } from 'vue';
import { cachedImageURL } from '../api.js';

// GIF endpoints need the Authorization header, so <img src> can't point
// at them directly: fetch as a blob (cached for the session) instead.
const props = defineProps({
    cacheKey: { type: String, required: true },
    path: { type: String, required: true },
    size: { type: String, default: 'w-24 h-24' },
});

const src = ref('');
const failed = ref(false);

watch(() => props.cacheKey, (key) => {
    src.value = '';
    failed.value = false;
    cachedImageURL(key, props.path)
        .then((url) => { if (key === props.cacheKey) src.value = url; })
        .catch(() => { failed.value = true; });
}, { immediate: true });
</script>

<template>
  <div :class="[size, 'gif-frame']">
    <img v-if="src" :src="src" alt="" class="object-contain max-w-full max-h-full"/>
    <span v-else-if="failed" class="text-[10px] text-error">lỗi tải ảnh</span>
    <span v-else class="loading loading-dots loading-xs opacity-40"></span>
  </div>
</template>
