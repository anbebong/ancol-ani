<script setup>
import Icon from './Icon.vue';

// GitHub-style star toggle for a public GIF: "☆ 12" / "★ 13". The parent
// does the API call (POST/DELETE /public/{slug}/star) and passes the new
// count/state back. readonly = just show the count (e.g. own library).
defineProps({
    starred: { type: Boolean, default: false },
    count: { type: Number, default: 0 },
    busy: { type: Boolean, default: false },
    readonly: { type: Boolean, default: false },
    title: { type: String, default: '' },
});
const emit = defineEmits(['toggle']);
</script>

<template>
  <span v-if="readonly" class="star-btn is-readonly" :title="title">
    <Icon name="star" :size="13" class="star-icon"/> {{ count }}
  </span>
  <button v-else type="button" class="star-btn" :class="{ 'is-on': starred }" :disabled="busy"
    :aria-pressed="starred" :title="title" @click.stop="emit('toggle')">
    <Icon name="star" :size="13" class="star-icon"/> {{ count }}
  </button>
</template>

<style scoped>
.star-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.15rem 0.55rem;
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1.4;
  color: #c9c5e0;
  background: #0e0e22;
  border: 2px solid #35335a;
  border-radius: 999px;
  transition: border-color .15s ease, color .15s ease, transform .1s ease;
}
button.star-btn:hover { border-color: #e8a53d; color: #fff; }
button.star-btn:active:not(:disabled) { transform: scale(0.95); }
button.star-btn:disabled { opacity: .6; }
.star-btn.is-on { color: #e8a53d; border-color: #e8a53d; }
.star-btn.is-on .star-icon :deep(svg) { fill: currentColor; }
.star-btn.is-readonly { cursor: default; }

/* Pinned over the GIF's top-right corner (parent must be position:relative):
   <StarButton class="star-overlay" .../> */
.star-overlay {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  z-index: 1;
  background: rgba(14, 14, 34, 0.88);
  box-shadow: 2px 2px 0 rgba(0, 0, 0, 0.6);
}
</style>
