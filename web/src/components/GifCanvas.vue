<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue';
import { playGif } from '../gif-player.js';

// Draws a GIF on a canvas with our own decoder (see gif-player.js:
// Chromium's native <img> GIF animation can stall permanently).
// `load` returns a Promise<ArrayBuffer>; changing `cacheKey` restarts it.
const props = defineProps({
    load: { type: Function, required: true },
    cacheKey: { type: String, required: true },
    size: { type: String, default: 'w-24 h-24' },
});

const canvas = ref(null);
const failed = ref(false);
let generation = 0;
let alive = true;

function start() {
    if (!canvas.value) return;
    const mine = ++generation;
    failed.value = false;
    // On failure show the error label and hand playGif a promise that never
    // settles, so it simply stops (it has no rejection handling of its own).
    const load = () => props.load().catch(() => {
        if (mine === generation) failed.value = true;
        return new Promise(() => {});
    });
    playGif(canvas.value, load, () => alive && mine === generation);
}

onMounted(start);
watch(() => props.cacheKey, start);
onBeforeUnmount(() => { alive = false; });
</script>

<template>
  <div :class="[size, 'gif-frame relative']">
    <canvas ref="canvas" class="max-w-full max-h-full object-contain"></canvas>
    <span v-if="failed" class="absolute text-[10px] text-error">lỗi tải ảnh</span>
  </div>
</template>
