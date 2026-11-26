<script setup>
import { ref } from 'vue';
import Icon from '@theme/components/Icon.vue';
import ResultPreview from './ResultPreview.vue';
import { chainedFile, takeChainedFile } from '../toolChain.js';

defineProps({ result: Object, loading: Boolean, error: String, runLabel: { type: String, default: 'Chạy' } });
const emit = defineEmits(['run']);

// Picks up a GIF handed over from a previous tool's result (see
// ResultPreview.vue's "Tiếp tục xử lý GIF này"), so the user doesn't have
// to download and re-upload it.
const file = ref(chainedFile.value ? takeChainedFile() : null);
const fromChain = ref(!!file.value);
function onFile(e) { file.value = e.target.files[0] || null; fromChain.value = false; }
function run() { if (file.value) emit('run', file.value); }
</script>

<template>
  <div class="flex flex-col gap-6 max-w-xl">
    <div v-if="fromChain" class="alert alert-info text-sm py-2">Đã nạp sẵn GIF vừa tạo từ công cụ trước.</div>
    <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>
    <slot name="params"/>
    <button type="button" class="btn btn-primary w-full" :disabled="!file || loading" @click="run">
      <Icon name="sparkles"/> {{ runLabel }}
    </button>
    <ResultPreview :result="result" :loading="loading" :error="error"/>
  </div>
</template>
