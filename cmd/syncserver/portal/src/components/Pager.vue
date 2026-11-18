<script setup>
import { computed } from 'vue';
import { PAGE_SIZE } from '../api.js';

const props = defineProps({
    page: { type: Number, required: true },
    total: { type: Number, required: true },
});
const emit = defineEmits(['change']);
const totalPages = computed(() => Math.max(1, Math.ceil(props.total / PAGE_SIZE)));
</script>

<template>
  <div v-if="totalPages > 1" class="flex items-center justify-center gap-3">
    <button type="button" class="btn btn-ghost btn-xs" :disabled="page <= 1" @click="emit('change', page - 1)">‹ Trước</button>
    <span class="text-xs text-base-content/60">Trang {{ page }}/{{ totalPages }}</span>
    <button type="button" class="btn btn-ghost btn-xs" :disabled="page >= totalPages" @click="emit('change', page + 1)">Sau ›</button>
  </div>
</template>
