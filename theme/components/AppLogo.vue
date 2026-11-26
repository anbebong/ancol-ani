<script setup>
import logo from '../assets/logo.png';

// The app's main icon (same artwork as ancol.ico / the Windows exe icon).
// Its own colors (purple/pink) have enough contrast to sit directly on the
// portal's dark theme, so unlike the favicon/apple-touch-icon (which keep
// a light tile - the browser tab and iOS home screen both need it, see
// docs/ICONS.md) this one is shown with no background at all.
defineProps({
    size: { type: Number, default: 36 },
});
</script>

<template>
  <span class="app-logo" :style="{ height: size + 'px' }">
    <img :src="logo" alt="Ancol-Ani"/>
  </span>
</template>

<style scoped>
/* Height-only: logo.png is a tight crop of the glyph's own (wider-than-tall)
   shape, not a square, so width follows from the image's natural aspect
   ratio instead of being forced square and padded. */
.app-logo {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}
.app-logo img { height: 100%; width: auto; display: block; }
</style>
