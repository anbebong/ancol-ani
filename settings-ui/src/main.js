import './style.css';
import '@theme/pixel.css';
import { createApp } from 'vue';
import App from './App.vue';
import { createAppRouter } from './router.js';
import { lang, t } from './i18n.js';
import { consumeCallbackToken, setExpiredHandler } from './account.js';
import { toast } from './toast.js';

document.documentElement.lang = lang.value;

// The server portal sends the browser back here as /?token=... after the
// person signs in (account linking). Consume it and rewrite the URL
// *before* creating the router (see createAppRouter), landing on the
// library on success.
consumeCallbackToken().then((result) => {
    if (result) {
        history.replaceState(null, '', location.pathname + (result === 'linked' ? '#/library' : '#/account'));
    }

    const router = createAppRouter();
    setExpiredHandler(() => {
        toast(t('toastExpired'), 'error');
        if (router.currentRoute.value.meta.needsLink) router.push({ name: 'account' });
    });

    createApp(App).use(router).mount('#app');
    if (result === 'linked') toast(t('toastLinked'), 'success');
});
