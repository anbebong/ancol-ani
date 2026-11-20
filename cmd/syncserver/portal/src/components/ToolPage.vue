<script setup>
import { ref } from 'vue';
import Icon from '@ui/components/Icon.vue';
import ResultPreview from './ResultPreview.vue';

defineProps({ result: Object, loading: Boolean, error: String, runLabel: { type: String, default: 'Chạy' } });
const emit = defineEmits(['run']);

const file = ref(null);
function onFile(e) { file.value = e.target.files[0] || null; }
function run() { if (file.value) emit('run', file.value); }
</script>

<template>
  <div class="flex flex-col gap-6 max-w-xl">
    <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>
    <slot name="params"/>
    <button type="button" class="btn btn-primary w-full" :disabled="!file || loading" @click="run">
      <Icon name="sparkles"/> {{ runLabel }}
    </button>
    <ResultPreview :result="result" :loading="loading" :error="error"/>
  </div>
</template>
