import './style.css';
import '@ui/pixel.css';
import { createApp } from 'vue';
import App from './App.vue';
import { router } from './router.js';
import { getToken, setUnauthorizedHandler } from './api.js';
import { signOutLocal, sendTokenToClient } from './session.js';

function mount() {
    setUnauthorizedHandler(() => {
        signOutLocal();
        if (router.currentRoute.value.name !== 'login') router.push({ name: 'login' });
    });
    createApp(App).use(router).mount('#app');
}

// The desktop app's local settings page links an account by opening
// {server}/?callback=<its url>. If this browser already has a portal
// session, give the app a fresh token of its own straight away instead of
// asking to log in again. If that session turns out to be expired, fall
// back to the login page (which handles the callback after login).
const callback = new URLSearchParams(location.search).get('callback');
if (callback && getToken()) {
    sendTokenToClient(callback).catch(() => {
        signOutLocal();
        mount();
    });
} else {
    mount();
}
