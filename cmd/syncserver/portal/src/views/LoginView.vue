<script setup>
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { errMsg } from '../api.js';
import { signIn } from '../session.js';
import Icon from '@ui/components/Icon.vue';
import AppLogo from '@ui/components/AppLogo.vue';

const route = useRoute();
const router = useRouter();

// ?callback= sits in the real query string (before the #), set by the
// desktop app's local settings page when linking an account.
const callback = new URLSearchParams(location.search).get('callback');

const mode = ref('login');
const username = ref('');
const password = ref('');
const confirm = ref('');
const showPassword = ref(false);
const loading = ref(false);
const error = ref('');

const isLogin = computed(() => mode.value === 'login');

function setMode(m) {
    mode.value = m;
    error.value = '';
}

async function submit() {
    error.value = '';
    const u = username.value.trim();
    if (!u || !password.value) { error.value = 'Nhập tên đăng nhập và mật khẩu'; return; }
    if (!isLogin.value) {
        if (password.value.length < 6) { error.value = 'Mật khẩu tối thiểu 6 ký tự'; return; }
        if (password.value !== confirm.value) { error.value = 'Mật khẩu nhập lại không khớp'; return; }
    }

    loading.value = true;
    try {
        const res = await fetch(isLogin.value ? '/auth/login' : '/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username: u, password: password.value }),
        });
        if (!res.ok) throw new Error(await res.text());
        const { token } = await res.json();

        // Linking from the desktop app: hand the token back to it instead
        // of keeping it in this portal.
        if (callback) {
            location.href = callback + (callback.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(token);
            return;
        }

        signIn(token, u);
        const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/library';
        router.replace(redirect);
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        loading.value = false;
    }
}
</script>

<template>
  <div class="auth-screen">
    <div class="w-full max-w-sm flex flex-col items-center">
      <div class="auth-mascot">
        <img src="../assets/mascot.gif" alt="ancol-ani mascot"/>
      </div>

      <div class="card bg-base-100 w-full auth-card">
        <div class="auth-stripe"></div>
        <form class="card-body gap-5" novalidate @submit.prevent="submit">
          <div class="text-center flex flex-col gap-1">
            <h1 class="text-2xl font-extrabold tracking-tight flex items-center justify-center gap-2.5">
              <AppLogo :size="34"/>
              <span class="gradient-text">ANCOL-ANI</span>
            </h1>
            <p class="text-sm text-base-content/60">
              {{ isLogin
                ? 'Chào mừng trở lại! Đăng nhập để quản lý GIF của bạn.'
                : 'Tạo tài khoản để đồng bộ GIF giữa các máy và chia sẻ công khai.' }}
            </p>
          </div>

          <div v-if="callback" class="alert alert-info text-xs py-2">
            <Icon name="key"/>
            <span>Đăng nhập để liên kết tài khoản với app ancol-ani trên máy bạn.</span>
          </div>

          <div role="tablist" class="auth-tabs">
            <button type="button" role="tab" class="auth-tab" :class="{ 'is-active': isLogin }" :aria-selected="isLogin" @click="setMode('login')">
              <Icon name="log-in"/> Đăng nhập
            </button>
            <button type="button" role="tab" class="auth-tab" :class="{ 'is-active': !isLogin }" :aria-selected="!isLogin" @click="setMode('register')">
              <Icon name="user-plus"/> Đăng ký
            </button>
          </div>

          <div class="flex flex-col gap-3">
            <label class="flex flex-col gap-1.5">
              <span class="field-label">Tên đăng nhập</span>
              <span class="input input-bordered auth-input flex items-center gap-2">
                <Icon name="user" class="text-base-content/50"/>
                <input v-model="username" type="text" autocomplete="username" placeholder="vd: ancol" class="grow bg-transparent min-w-0" autofocus/>
              </span>
            </label>

            <label class="flex flex-col gap-1.5">
              <span class="field-label">Mật khẩu</span>
              <span class="input input-bordered auth-input flex items-center gap-2">
                <Icon name="lock" class="text-base-content/50"/>
                <input v-model="password" :type="showPassword ? 'text' : 'password'" :autocomplete="isLogin ? 'current-password' : 'new-password'" placeholder="••••••" class="grow bg-transparent min-w-0"/>
                <button type="button" class="auth-eye" tabindex="-1" :aria-label="showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'" @click="showPassword = !showPassword">
                  <Icon :name="showPassword ? 'eye-off' : 'eye'"/>
                </button>
              </span>
            </label>

            <template v-if="!isLogin">
              <label class="flex flex-col gap-1.5">
                <span class="field-label">Nhập lại mật khẩu</span>
                <span class="input input-bordered auth-input flex items-center gap-2">
                  <Icon name="check" class="text-base-content/50"/>
                  <input v-model="confirm" :type="showPassword ? 'text' : 'password'" autocomplete="new-password" placeholder="••••••" class="grow bg-transparent min-w-0"/>
                </span>
              </label>
              <p class="text-xs text-base-content/50">Mật khẩu tối thiểu 6 ký tự.</p>
            </template>
          </div>

          <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>

          <button type="submit" class="btn btn-primary w-full auth-submit" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            {{ isLogin ? 'Đăng nhập' : 'Tạo tài khoản' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>
