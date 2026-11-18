<script setup>
import logo from '../assets/logo.png';

// The app's main icon (same artwork as ancol.ico / the Windows exe icon).
// It's dark-on-transparent, so it sits on a light pixel tile to stay
// visible on the portal's dark theme.
defineProps({
    size: { type: Number, default: 36 },
});
</script>

<template>
  <!-- padding in px, not %: a % padding resolves against the *parent's*
       width and can swallow the whole tile -->
  <span class="app-logo" :style="{ width: size + 'px', height: size + 'px', padding: Math.round(size * 0.12) + 'px' }">
    <img :src="logo" alt="Ancol-ani"/>
  </span>
</template>

<style scoped>
.app-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-sizing: border-box;
  background: #f1eefc;
  border: 2px solid #35335a;
  border-radius: 0.4rem;
  box-shadow: 2px 2px 0 #000;
}
.app-logo img { width: 100%; height: 100%; max-width: none; object-fit: contain; }
</style>
