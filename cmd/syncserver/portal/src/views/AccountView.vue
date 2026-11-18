<script setup>
import { ref } from 'vue';
import { api, errMsg } from '../api.js';
import { session } from '../session.js';
import Icon from '../components/Icon.vue';
import PageHeader from '../components/PageHeader.vue';

const current = ref('');
const next = ref('');
const confirm = ref('');
const show = ref(false);
const saving = ref(false);
const error = ref('');
const done = ref(false);

async function submit() {
    error.value = '';
    done.value = false;
    if (!current.value || !next.value) { error.value = 'Nhập đủ mật khẩu hiện tại và mật khẩu mới'; return; }
    if (next.value.length < 6) { error.value = 'Mật khẩu mới tối thiểu 6 ký tự'; return; }
    if (next.value !== confirm.value) { error.value = 'Mật khẩu nhập lại không khớp'; return; }
    if (next.value === current.value) { error.value = 'Mật khẩu mới phải khác mật khẩu hiện tại'; return; }

    saving.value = true;
    try {
        await api('/me/password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ currentPassword: current.value, newPassword: next.value }),
        });
        current.value = next.value = confirm.value = '';
        done.value = true;
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        saving.value = false;
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="user" title="Tài khoản" :subtitle="session.me ? `Đăng nhập với tên ${session.me.username}` : ''"/>

    <div class="card bg-base-100 max-w-lg">
      <form class="card-body gap-4" novalidate @submit.prevent="submit">
        <h2 class="font-bold flex items-center gap-2"><Icon name="key"/> Đổi mật khẩu</h2>

        <label class="flex flex-col gap-1.5">
          <span class="field-label">Mật khẩu hiện tại</span>
          <span class="input input-bordered auth-input flex items-center gap-2">
            <Icon name="lock" class="text-base-content/50"/>
            <input v-model="current" :type="show ? 'text' : 'password'" autocomplete="current-password" class="grow bg-transparent min-w-0"/>
            <button type="button" class="auth-eye" tabindex="-1" :aria-label="show ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'" @click="show = !show">
              <Icon :name="show ? 'eye-off' : 'eye'"/>
            </button>
          </span>
        </label>

        <label class="flex flex-col gap-1.5">
          <span class="field-label">Mật khẩu mới</span>
          <span class="input input-bordered auth-input flex items-center gap-2">
            <Icon name="key" class="text-base-content/50"/>
            <input v-model="next" :type="show ? 'text' : 'password'" autocomplete="new-password" placeholder="tối thiểu 6 ký tự" class="grow bg-transparent min-w-0"/>
          </span>
        </label>

        <label class="flex flex-col gap-1.5">
          <span class="field-label">Nhập lại mật khẩu mới</span>
          <span class="input input-bordered auth-input flex items-center gap-2">
            <Icon name="check" class="text-base-content/50"/>
            <input v-model="confirm" :type="show ? 'text' : 'password'" autocomplete="new-password" class="grow bg-transparent min-w-0"/>
          </span>
        </label>

        <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>
        <div v-if="done" class="alert alert-success text-sm py-2">
          Đã đổi mật khẩu. Các thiết bị khác (kể cả app desktop đã liên kết) đã bị đăng xuất và cần đăng nhập lại bằng mật khẩu mới.
        </div>

        <p class="text-xs text-base-content/50">
          Sau khi đổi, mọi phiên đăng nhập khác của tài khoản này sẽ bị đăng xuất. Phiên hiện tại vẫn giữ nguyên.
        </p>

        <button type="submit" class="btn btn-primary self-start" :disabled="saving">
          <span v-if="saving" class="loading loading-spinner loading-sm"></span>
          Đổi mật khẩu
        </button>
      </form>
    </div>
  </div>
</template>
