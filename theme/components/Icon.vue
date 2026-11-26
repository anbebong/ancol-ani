<script setup>
import { computed } from 'vue';
import { ICONS } from '../icons.js';

const props = defineProps({
    name: { type: String, required: true },
    size: { type: Number, default: 16 },
});
const svg = computed(() => ICONS[props.name] || '');
</script>

<template>
  <span
    class="icon inline-flex shrink-0"
    :style="{ width: size + 'px', height: size + 'px' }"
    aria-hidden="true"
    v-html="svg"
  ></span>
</template>

<style scoped>
.icon :deep(svg) { width: 100%; height: 100%; }
</style>
