<script setup>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { session, issueToken, signIn, loadMe, signOut } from '../session.js';
import AppLogo from '@ui/components/AppLogo.vue';

// #/sso?t=<token>&to=<path> - opened by the desktop app's settings page
// ("Mở portal"). Trades the app's token for this portal's own session, so
// someone who already linked the app doesn't have to log in again here.
// The token only lives in this URL until router.replace() overwrites the
// history entry below.
const route = useRoute();
const router = useRouter();
const failed = ref(false);

function safeTarget(to) {
    return typeof to === 'string' && to.startsWith('/') && !to.startsWith('//') && !to.startsWith('/sso') ? to : '/library';
}

onMounted(async () => {
    const t = typeof route.query.t === 'string' ? route.query.t : '';
    const target = safeTarget(route.query.to);
    if (!t) { router.replace(session.token ? target : { name: 'login' }); return; }
    try {
        const token = await issueToken(t);
        if (session.token) await signOut(); // replace any older portal session
        signIn(token, '');
        await loadMe({ force: true });
        router.replace(target);
    } catch (_) {
        failed.value = true;
        router.replace({ name: 'login', query: target !== '/library' ? { redirect: target } : {} });
    }
});
</script>

<template>
  <div class="auth-screen">
    <div class="flex flex-col items-center gap-4 text-base-content/70">
      <AppLogo :size="48"/>
      <span v-if="!failed" class="loading loading-dots loading-md text-primary"></span>
      <p class="text-sm">Đang đăng nhập bằng tài khoản của app...</p>
    </div>
  </div>
</template>
