<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { confirmDialog } from '@ui/modal.js';
import { splitToFit, makeFitsWithinLines, WIDGET_BUBBLE } from '@ui/text-fit.js';
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
// Seconds each line stays up when the text has several lines (app.go
// accepts 2-30).
const TEXT_INTERVALS = [2, 3, 4, 5, 8, 10, 15, 30];
const textInterval = ref(4);
async function loadTextStyle() {
    try {
        const style = await (await fetch('/text/style')).json();
        textColor.value = style.color || '#ffffff';
        textEffect.value = style.effect || 'none';
        textInterval.value = Number(style.interval) || 4;
    } catch (_) {}
}
onMounted(loadTextStyle);
async function saveTextStyle() {
    const form = new FormData();
    form.append('color', textColor.value);
    form.append('effect', textEffect.value);
    form.append('interval', String(textInterval.value));
    try {
        await fetch('/text/style', { method: 'POST', body: form });
    } catch (err) {
        toastError(err);
    }
}

// Live preview of the bubble (color + effect) right next to the GIF
// preview below, so changing color/effect/text shows the result
// immediately instead of only after it's saved and shown on the widget.
// One message per line; the widget shows them one after another. The
// preview runs the same rotation (same timings as frontend/index.html),
// including splitting a line too long for the bubble into several
// messages (splitToFit, shared with the widget via ui/text-fit.js) so the
// "x/y" counter below matches what the widget will actually rotate
// through.
// Canvas-measured fit check (see ui/text-fit.js) - uses WIDGET_BUBBLE, the
// REAL widget's own box model, not this preview's (bigger, for readability)
// CSS. Otherwise the preview's wider/taller box would let more text fit per
// line than the actual widget does, splitting differently from what the
// user will really see.
const bubbleRef = ref(null);
const measurerReady = ref(true);
const fitsPreviewLine = makeFitsWithinLines(WIDGET_BUBBLE.fontCss, WIDGET_BUBBLE.contentWidthPx, WIDGET_BUBBLE.maxLines);

const previewLines = computed(() => {
    const entered = text.value.split('\n').map((l) => l.trim()).filter(Boolean);
    if (!entered.length) return [t('textPreviewPlaceholder')];
    if (!measurerReady.value) return entered;
    return entered.flatMap((l) => splitToFit(l, fitsPreviewLine));
});
const previewIdx = ref(0);
const previewShown = ref('');
// Bumped on every line change, used as the bubble's :key so Vue recreates
// the element - the cleanest way to restart a CSS animation from its first
// frame (toggling the class alone wouldn't replay it).
const previewAnimKey = ref(0);
let previewTimer = null;

function stopPreview() { clearInterval(previewTimer); clearTimeout(previewTimer); }

function showPreviewLine() {
    stopPreview();
    const lines = previewLines.value;
    if (previewIdx.value >= lines.length) previewIdx.value = 0;
    const value = lines[previewIdx.value];
    const multi = lines.length > 1;
    const next = () => { previewIdx.value = (previewIdx.value + 1) % lines.length; showPreviewLine(); };
    previewAnimKey.value++;

    if (textEffect.value === 'typewriter') {
        let i = 0;
        previewShown.value = '';
        previewTimer = setInterval(() => {
            i++;
            previewShown.value = value.slice(0, i);
            if (i >= value.length) {
                clearInterval(previewTimer);
                previewTimer = setTimeout(multi ? next : showPreviewLine, multi ? textInterval.value * 1000 : 1500);
            }
        }, 40);
        return;
    }
    previewShown.value = value;
    if (multi) previewTimer = setTimeout(next, textInterval.value * 1000);
}

watch([previewLines, textEffect, textInterval], () => {
    previewIdx.value = 0;
    showPreviewLine();
}, { immediate: true });

onUnmounted(stopPreview);

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
              <textarea v-model="text" :placeholder="t('textPlaceholder')" rows="3" class="textarea textarea-bordered auth-input flex-1 min-w-0 resize-y" style="min-height: 3rem;" autofocus></textarea>
              <button type="submit" class="btn btn-primary">{{ t('textSubmit') }}</button>
            </div>

            <h2 class="font-bold flex items-center gap-2 mt-2"><Icon name="sparkles"/> {{ t('textStyleLabel') }}</h2>
            <div class="flex gap-2 items-center flex-wrap">
              <input v-model="textColor" type="color" class="input input-bordered h-10 w-14 p-1" @change="saveTextStyle"/>
              <select v-model="textEffect" class="select select-bordered select-sm flex-1 min-w-0" @change="saveTextStyle">
                <option v-for="fx in TEXT_EFFECTS" :key="fx.id" :value="fx.id">{{ t(fx.labelKey) }}</option>
              </select>
              <select v-model.number="textInterval" class="select select-bordered select-sm" :title="t('textIntervalLabel')" @change="saveTextStyle">
                <option v-for="n in TEXT_INTERVALS" :key="n" :value="n">{{ t('textIntervalOption', { n }) }}</option>
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
              <!-- One single 70x100 (w x h) block, same shape as the real
                   widget window: text panel stacked directly on top of the
                   GIF, no gap, nothing in between - this IS the widget's
                   whole screen, just scaled up for readability. -->
              <div class="widget-replica">
                <div class="text-bubble-slot">
                  <div ref="bubbleRef" class="text-bubble-preview" :class="'fx-' + textEffect" :key="previewAnimKey" :style="{ color: textColor }">
                    {{ previewShown }}
                  </div>
                </div>
                <GifCanvas :load="loadPreview" :cache-key="previewKey" size="w-28 h-28"/>
              </div>
              <span v-if="previewLines.length > 1" class="text-[10px] text-base-content/50 mt-1">
                {{ t('textLineCounter', { i: previewIdx + 1, n: previewLines.length }) }}
              </span>
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
.widget-replica {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.text-bubble-slot {
  /* Real widget window is 70 wide x 100 tall, split 30 (text) / 70 (gif).
     The GIF preview below is a 112px (w-28) square, same as the widget's
     square 70x70 gif area. ONE scale factor k = 112/70 (=1.6) ties this
     slot's height AND every .text-bubble-preview dimension below back to
     the real widget's own numbers (frontend/index.html) - using a
     different k for the bubble than for this slot is what made the two
     look like they didn't share a single consistent layout. */
  flex: none;
  position: relative;
  width: 112px;
  height: calc(112px * 30 / 70); /* = 30 (real text panel height) * k */
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.text-bubble-preview {
  /* Every value here is the real #text-bubble's own value (frontend/
     index.html: max-width 68px, margin-bottom 3px, padding 1px 4px,
     border 1.5px, border-radius 6px, font 10px/1.1) times k = 112/70, the
     same k used for .text-bubble-slot above - so this bubble is a true
     scaled copy of the real one, not an independently chosen size. */
  position: absolute;
  box-sizing: border-box;
  left: 50%;
  bottom: calc(112px * 3 / 70); /* = 3px (real margin-bottom) * k */
  transform: translateX(-50%);
  width: max-content;
  max-width: calc(112px * 68 / 70);
  padding: calc(112px * 1 / 70) calc(112px * 4 / 70);
  background: rgba(20, 20, 43, 0.92);
  border: calc(112px * 1.5 / 70) solid #8b73ff;
  border-radius: calc(112px * 6 / 70);
  font-weight: 700;
  font-size: calc(112px * 10 / 70); /* = 10px (real font-size) * k */
  line-height: 1.1;
  text-align: center;
  word-break: break-word;
  /* Clamp by LINE COUNT (like the real #text-content), not max-height:
     a max-height on this same box would have to subtract the padding/
     border above out of the space left for the 2 lines of text (border-
     box), cutting line 2 off - line-clamp doesn't have that problem. */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
<!-- fx-fade/fx-bounce/fx-shake come from here - shared with the widget
     itself (frontend/index.html) so the two never drift out of sync. -->
<style src="@ui/text-effects.css"></style>
