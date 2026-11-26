<script setup>
import { computed } from 'vue';

const props = defineProps({
    page: { type: Number, required: true },
    total: { type: Number, required: true },
    pageSize: { type: Number, default: 10 },
    prevLabel: { type: String, default: '‹ Trước' },
    nextLabel: { type: String, default: 'Sau ›' },
    // (page, totalPages) => label; lets the client page translate it.
    formatLabel: { type: Function, default: (p, n) => `Trang ${p}/${n}` },
});
const emit = defineEmits(['change']);
const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)));
</script>

<template>
  <div v-if="totalPages > 1" class="flex items-center justify-center gap-3">
    <button type="button" class="btn btn-ghost btn-xs" :disabled="page <= 1" @click="emit('change', page - 1)">{{ prevLabel }}</button>
    <span class="text-xs text-base-content/60">{{ formatLabel(page, totalPages) }}</span>
    <button type="button" class="btn btn-ghost btn-xs" :disabled="page >= totalPages" @click="emit('change', page + 1)">{{ nextLabel }}</button>
  </div>
</template>
