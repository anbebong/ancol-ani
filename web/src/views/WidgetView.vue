<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { confirmDialog } from '@ui/modal.js';
import GifCanvas from '../components/GifCanvas.vue';
import { t } from '../i18n.js';
import { account, serverApi } from '../account.js';
import { toast, toastError } from '../toast.js';

// ---------- start with Windows ----------
const autoStart = ref(false);
async function loadAutoStart() {
    try {
        autoStart.value = (await (await fetch('/autostart')).text()).trim() === 'true';
    } catch (_) {}
}
onMounted(loadAutoStart);
async function setAutoStart(enabled) {
    const form = new FormData();
    form.append('enabled', String(enabled));
    try {
        await fetch('/autostart', { method: 'POST', body: form });
        autoStart.value = enabled;
    } catch (err) {
        toastError(err);
    }
}

// ---------- widget screen corner ----------
const CORNERS = [
    { id: 'tl', labelKey: 'positionTL' },
    { id: 'tc', labelKey: 'positionTC' },
    { id: 'tr', labelKey: 'positionTR' },
    { id: 'bl', labelKey: 'positionBL' },
    { id: 'bc', labelKey: 'positionBC' },
    { id: 'br', labelKey: 'positionBR' },
];
const corner = ref('br');
async function loadCorner() {
    try {
        corner.value = (await (await fetch('/position')).text()).trim() || 'br';
    } catch (_) {}
}
onMounted(loadCorner);
async function setCorner(c) {
    if (c === corner.value) return;
    const form = new FormData();
    form.append('corner', c);
    try {
        await fetch('/position', { method: 'POST', body: form });
        corner.value = c;
    } catch (err) {
        toastError(err);
    }
}

// ---------- text on the widget ----------
// Loaded from what's actually showing on the widget, so the box doubles as
// "current text" display - someone who only wants to change color/effect
// can leave it as-is and just hit Apply, instead of needing to retype it.
const text = ref('');
async function loadText() {
    try {
        text.value = (await (await fetch('/text')).text()).trim();
    } catch (_) {}
}
onMounted(loadText);

// ---------- text display style (color + entrance effect) ----------
const TEXT_EFFECTS = [
    { id: 'none', labelKey: 'textEffectNone' },
    { id: 'fade', labelKey: 'textEffectFade' },
    { id: 'bounce', labelKey: 'textEffectBounce' },
    { id: 'shake', labelKey: 'textEffectShake' },
    { id: 'typewriter', labelKey: 'textEffectTypewriter' },
];
const textColor = ref('#ffffff');
const textEffect = ref('none');
async function loadTextStyle() {
    try {
        const style = await (await fetch('/text/style')).json();
        textColor.value = style.color || '#ffffff';
        textEffect.value = style.effect || 'none';
    } catch (_) {}
}
onMounted(loadTextStyle);
async function saveTextStyle() {
    const form = new FormData();
    form.append('color', textColor.value);
    form.append('effect', textEffect.value);
    try {
        await fetch('/text/style', { method: 'POST', body: form });
    } catch (err) {
        toastError(err);
    }
}

// Live preview of the bubble (color + effect) right next to the GIF
// preview below, so changing color/effect/text shows the result
// immediately instead of only after it's saved and shown on the widget.
const previewText = computed(() => text.value.trim() || t('textPreviewPlaceholder'));
// Bumped whenever the text or effect changes, used as the bubble's :key so
// Vue recreates the element - the cleanest way to restart a CSS animation
// from its first frame (toggling the class alone wouldn't replay it).
const previewAnimKey = ref(0);
const typedPreviewText = ref('');
let typewriterTimer = null;

function runTypewriterPreview() {
    clearInterval(typewriterTimer);
    clearTimeout(typewriterTimer);
    const value = previewText.value;
    typedPreviewText.value = '';
    let i = 0;
    const type = () => {
        i++;
        typedPreviewText.value = value.slice(0, i);
        if (i >= value.length) {
            clearInterval(typewriterTimer);
            typewriterTimer = setTimeout(() => {
                typedPreviewText.value = '';
                i = 0;
                typewriterTimer = setInterval(type, 40);
            }, 1500);
        }
    };
    typewriterTimer = setInterval(type, 40);
}

watch([previewText, textEffect], () => {
    previewAnimKey.value++;
    if (textEffect.value === 'typewriter') runTypewriterPreview();
    else { clearInterval(typewriterTimer); clearTimeout(typewriterTimer); }
}, { immediate: true });

onUnmounted(() => { clearInterval(typewriterTimer); clearTimeout(typewriterTimer); });

async function sendText() {
    if (!text.value.trim()) return;
    const form = new FormData();
    form.append('text', text.value);
    try {
        await fetch('/text', { method: 'POST', body: form });
        toast(t('toastTextSent'), 'success');
    } catch (err) {
        toastError(err);
    }
}

// ---------- GIF on the widget ----------
const previewKey = ref(String(Date.now()));
// While a file is picked but not applied yet, preview it directly from
// disk (not what the widget is currently showing) so the preview matches
// what "Áp dụng" is actually about to upload.
const loadPreview = () => (chosen.value ? chosen.value.arrayBuffer() : fetch('/gif?t=' + previewKey.value, { cache: 'no-store' }).then((r) => r.arrayBuffer()));
function reloadPreview() { previewKey.value = String(Date.now()); }

const fileInput = ref(null);
const chosen = ref(null);
const dragOver = ref(false);
const busy = ref(false);

function pick(file) {
    if (!file) return;
    if (!/\.gif$/i.test(file.name) && file.type !== 'image/gif') {
        toast(t('errGifOnly'), 'error');
        return;
    }
    chosen.value = file;
    previewKey.value = 'local:' + file.name + ':' + file.lastModified + ':' + file.size;
}
function onDrop(e) {
    dragOver.value = false;
    pick(e.dataTransfer.files[0]);
}

async function apply() {
    if (!chosen.value) { toast(t('errPickFile'), 'error'); return; }
    if (!(await confirmDialog(t('confirmApply'), { confirmText: t('uploadSubmit'), cancelText: t('modalCancel') }))) return;
    busy.value = true;
    try {
        const form = new FormData();
        form.append('gif', chosen.value);
        const res = await fetch('/gif', { method: 'POST', body: form });
        if (!res.ok) throw new Error(await res.text());
        chosen.value = null;
        if (fileInput.value) fileInput.value.value = '';
        reloadPreview();
        toast(t('toastGifChanged'), 'success');
    } catch (err) {
        toastError(err);
    } finally {
        busy.value = false;
    }
}

function cancelPick() {
    chosen.value = null;
    if (fileInput.value) fileInput.value.value = '';
    reloadPreview();
}

async function reset() {
    if (!(await confirmDialog(t('confirmReset'), { confirmText: t('resetSubmit'), cancelText: t('modalCancel') }))) return;
    await fetch('/gif/reset', { method: 'POST' });
    reloadPreview();
    toast(t('toastGifReset'), 'success');
}

// Upload whatever the widget currently shows into the linked account's library.
async function pushCurrent() {
    if (!(await confirmDialog(t('confirmPushUpload'), { confirmText: t('pushCurrentGifBtn'), cancelText: t('modalCancel') }))) return;
    try {
        const local = await fetch('/gif', { cache: 'no-store' });
        const form = new FormData();
        form.append('gif', await local.blob(), 'current.gif');
        await serverApi('/me/gifs', { method: 'POST', body: form });
        toast(t('toastUploaded'), 'success');
    } catch (err) {
        if (/quota|toi da/i.test(err.message)) toast(t('toastQuotaFull'), 'error');
        else if (err.message !== '401') toastError(err);
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="monitor" :title="t('widgetTitle')" :subtitle="t('subtitle')"/>

    <!-- Widget screen position + text -->
    <div class="card bg-base-100">
      <form class="card-body gap-4" @submit.prevent="sendText">
        <div class="flex flex-col sm:flex-row gap-6">
          <div class="flex flex-col gap-2 shrink-0">
            <h2 class="font-bold flex items-center gap-2"><Icon name="monitor"/> {{ t('positionLabel') }}</h2>
            <div class="flex flex-col items-center w-48 mt-1 shrink-0">
              <div class="relative aspect-video w-full border-2 border-neutral rounded-md bg-base-200">
                <button v-for="c in CORNERS" :key="c.id" type="button"
                  class="absolute w-6 h-6 rounded-full border-2 transition-colors"
                  :class="[
                    c.id === 'tl' ? 'top-1 left-1' : '',
                    c.id === 'tc' ? 'top-1 left-1/2 -translate-x-1/2' : '',
                    c.id === 'tr' ? 'top-1 right-1' : '',
                    c.id === 'bl' ? 'bottom-1 left-1' : '',
                    c.id === 'bc' ? 'bottom-1 left-1/2 -translate-x-1/2' : '',
                    c.id === 'br' ? 'bottom-1 right-1' : '',
                    corner === c.id ? 'bg-primary border-primary' : 'bg-base-100 border-neutral hover:border-primary',
                  ]"
                  :aria-label="t(c.labelKey)"
                  :title="t(c.labelKey)"
                  @click="setCorner(c.id)">
                </button>
              </div>
              <div class="w-10 h-2 border-x-2 border-b-2 border-neutral rounded-b-sm"></div>
            </div>
          </div>

          <div class="flex flex-col gap-2 flex-1 min-w-0">
            <label class="flex items-center justify-between gap-3 cursor-pointer">
              <span class="flex flex-col gap-1">
                <h2 class="font-bold flex items-center gap-2"><Icon name="power"/> {{ t('autoStartLabel') }}</h2>
                <p class="text-xs text-base-content/60">{{ t('autoStartHint') }}</p>
              </span>
              <input type="checkbox" class="toggle toggle-primary toggle-sm shrink-0" :checked="autoStart" @change="setAutoStart($event.target.checked)" :aria-label="t('autoStartLabel')"/>
            </label>

            <h2 class="font-bold flex items-center gap-2"><Icon name="type"/> {{ t('textLabel') }}</h2>
            <p class="text-xs text-base-content/60">{{ t('textHint') }}</p>
            <div class="flex gap-2 mt-1 items-start">
              <textarea v-model="text" :placeholder="t('textPlaceholder')" rows="1" class="textarea textarea-bordered auth-input flex-1 min-w-0 resize-y" style="min-height: 3rem;" autofocus></textarea>
              <button type="submit" class="btn btn-primary">{{ t('textSubmit') }}</button>
            </div>

            <h2 class="font-bold flex items-center gap-2 mt-2"><Icon name="sparkles"/> {{ t('textStyleLabel') }}</h2>
            <div class="flex gap-2 items-center flex-wrap">
              <input v-model="textColor" type="color" class="input input-bordered h-10 w-14 p-1" @change="saveTextStyle"/>
              <select v-model="textEffect" class="select select-bordered select-sm flex-1 min-w-0" @change="saveTextStyle">
                <option v-for="fx in TEXT_EFFECTS" :key="fx.id" :value="fx.id">{{ t(fx.labelKey) }}</option>
              </select>
            </div>
          </div>
        </div>
      </form>
    </div>

    <!-- GIF -->
    <div class="card bg-base-100">
      <div class="card-body gap-4">
        <h2 class="font-bold flex items-center gap-2"><Icon name="image"/> {{ t('gifLabel') }}</h2>

        <div class="flex flex-col sm:flex-row gap-5">
          <div class="flex flex-col items-center gap-2 shrink-0">
            <div class="flex flex-col items-center">
              <!-- Fixed-height slot so the bubble's own size changing (text
                   length, line wraps, the bounce/shake transforms) never
                   pushes the GIF below it up or down. -->
              <div class="text-bubble-slot">
                <div class="text-bubble-preview" :class="'fx-' + textEffect" :key="previewAnimKey" :style="{ color: textColor }">
                  {{ textEffect === 'typewriter' ? typedPreviewText : previewText }}
                </div>
              </div>
              <GifCanvas :load="loadPreview" :cache-key="previewKey" size="w-36 h-36"/>
            </div>
            <span class="field-label">{{ t('gifCurrent') }}</span>
          </div>

          <div class="flex-1 min-w-0 flex flex-col gap-3">
            <div class="dropzone flex-1" :class="{ 'is-over': dragOver }"
              @click="fileInput?.click()" @dragover.prevent="dragOver = true" @dragleave="dragOver = false" @drop.prevent="onDrop">
              <input ref="fileInput" type="file" accept=".gif,image/gif" class="hidden" @change="pick($event.target.files[0])"/>
              <Icon name="upload" :size="28" class="text-primary"/>
              <div class="min-w-0">
                <p class="font-bold truncate">{{ chosen ? chosen.name : t('dropzoneDefault') }}</p>
                <p class="text-xs text-base-content/60">{{ t('dropzoneOr') }}</p>
              </div>
            </div>

            <div class="flex gap-2 flex-wrap">
              <button type="button" class="btn btn-primary flex-1" :disabled="busy || !chosen" @click="apply">
                <span v-if="busy" class="loading loading-spinner loading-sm"></span>
                <Icon v-else name="check"/> {{ t('uploadSubmit') }}
              </button>
              <button v-if="chosen" type="button" class="btn btn-ghost border-2 border-neutral flex-1 gap-2" :disabled="busy" @click="cancelPick">
                <Icon name="x"/> {{ t('cancelPickBtn') }}
              </button>
              <button v-else type="button" class="btn btn-ghost border-2 border-neutral flex-1 gap-2" @click="reset">
                <Icon name="rotate-ccw"/> {{ t('resetSubmit') }}
              </button>
            </div>
            <button v-if="account.token" type="button" class="btn btn-sm btn-ghost border-2 border-neutral gap-2 self-start" @click="pushCurrent">
              <Icon name="upload"/> {{ t('pushCurrentGifBtn') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Mirrors the widget's own #text-bubble (frontend/index.html) so the
   preview here matches what actually shows on screen, just bigger. */
.text-bubble-slot {
  position: relative;
  width: 100%;
  height: 2.6rem;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.text-bubble-preview {
  position: absolute;
  bottom: 0.4rem;
  max-width: 10rem;
  max-height: 2.2rem;
  overflow: hidden;
  padding: 0.2rem 0.6rem;
  background: rgba(20, 20, 43, 0.92);
  border: 1.5px solid #8b73ff;
  border-radius: 0.5rem;
  font-weight: 700;
  font-size: 0.8rem;
  text-align: center;
  word-break: break-word;
}
</style>
<!-- fx-fade/fx-bounce/fx-shake come from here - shared with the widget
     itself (frontend/index.html) so the two never drift out of sync. -->
<style src="@ui/text-effects.css"></style>
