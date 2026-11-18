<script setup>
import Icon from './Icon.vue';

defineProps({
    icon: { type: String, required: true },
    title: { type: String, required: true },
    subtitle: { type: String, default: '' },
});
</script>

<template>
  <div class="flex items-start justify-between gap-4 flex-wrap">
    <div class="flex items-center gap-3">
      <div class="page-icon"><Icon :name="icon" :size="20"/></div>
      <div>
        <h1 class="text-xl font-extrabold leading-tight">{{ title }}</h1>
        <p v-if="subtitle" class="text-sm text-base-content/60">{{ subtitle }}</p>
      </div>
    </div>
    <slot/>
  </div>
</template>
