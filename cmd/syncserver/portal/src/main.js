import './style.css';
import { createApp } from 'vue';
import App from './App.vue';
import { router } from './router.js';
import { getToken, setUnauthorizedHandler } from './api.js';
import { signOutLocal } from './session.js';

// The desktop app's local settings page links an account by opening
// {server}/?callback=<its url>. If this browser already has a portal
// session, hand that token straight back instead of showing the portal
// (the login page handles the not-yet-logged-in case).
const callback = new URLSearchParams(location.search).get('callback');
if (callback && getToken()) {
    location.href = callback + (callback.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(getToken());
} else {
    setUnauthorizedHandler(() => {
        signOutLocal();
        if (router.currentRoute.value.name !== 'login') router.push({ name: 'login' });
    });
    createApp(App).use(router).mount('#app');
}
