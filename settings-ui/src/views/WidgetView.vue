<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import Icon from '@theme/components/Icon.vue';
import PageHeader from '@theme/components/PageHeader.vue';
import { confirmDialog } from '@theme/modal.js';
import { splitToFit, makeFitsWithinLines, WIDGET_BUBBLE } from '@theme/text-fit.js';
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
// Set only by the user actually editing the textarea (see @input below) -
// NOT by focus: the textarea has autofocus, so document.activeElement
// would be it from the moment the page loads even if nobody touched it,
// which used to block every single poll refresh forever.
const textDirty = ref(false);
async function loadText() {
    try {
        text.value = (await (await fetch('/text')).text()).trim();
        textDirty.value = false;
    } catch (_) {}
}
onMounted(loadText);
// The widget's text can also change on its own now (an imported .wasm
// adapter polling in the background, see the automatic text source
// section below). The Settings window and the widget are two SEPARATE
// Wails processes (main.go spawns "settings" as its own exe instance, so
// it can't read the widget's Go state directly either) - a Wails
// EventsEmit from the widget's process never reaches this window's
// runtime, so there's no live-push channel here. Polling the plain HTTP
// endpoint is the only thing that actually crosses that boundary. Only
// runs while an adapter is active (manual-only mode never changes on its
// own) and skips refreshing while the user has a half-typed draft they
// haven't sent yet, so an adapter update never overwrites it.
let textPoll = null;
onMounted(() => {
    textPoll = setInterval(() => {
        if (adapterEnabled.value && !textDirty.value) loadText();
    }, 3000);
});
onUnmounted(() => clearInterval(textPoll));

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
// Bubble background + border/tail color and background opacity - see
// app.go's defaultBubbleColor. Independent of textColor above, saved
// through the same /text/style endpoint.
const bubbleColor = ref('#14142b');
const bubbleBorderColor = ref('#8b73ff');
const bubbleOpacity = ref(0.92);
// Drives .text-bubble-preview's inline style below - computed so the
// preview updates live while dragging the opacity slider / picking a color,
// same as textColor already does via :style further down.
const bubbleStyle = computed(() => {
    const hex = bubbleColor.value.slice(1);
    const n = parseInt(hex, 16) || 0;
    const rgba = `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${bubbleOpacity.value})`;
    return { background: rgba, borderColor: bubbleBorderColor.value };
});
async function loadTextStyle() {
    try {
        const style = await (await fetch('/text/style')).json();
        textColor.value = style.color || '#ffffff';
        textEffect.value = style.effect || 'none';
        textInterval.value = Number(style.interval) || 4;
        bubbleColor.value = style.bubbleColor || '#14142b';
        bubbleBorderColor.value = style.bubbleBorderColor || '#8b73ff';
        bubbleOpacity.value = Number(style.bubbleOpacity) || 0.92;
    } catch (_) {}
}
onMounted(loadTextStyle);
async function saveTextStyle() {
    const form = new FormData();
    form.append('color', textColor.value);
    form.append('effect', textEffect.value);
    form.append('interval', String(textInterval.value));
    form.append('bubbleColor', bubbleColor.value);
    form.append('bubbleBorderColor', bubbleBorderColor.value);
    form.append('bubbleOpacity', String(bubbleOpacity.value));
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
        textDirty.value = false;
        toast(t('toastTextSent'), 'success');
    } catch (err) {
        toastError(err);
    }
}

// ---------- automatic text source (.wasm adapters) ----------
// An adapter is a .wasm file (written by whoever sets it up, e.g. for a
// customer's own notification source) that supplies the widget's text
// automatically instead of someone typing it in the box above - see
// wasmsource.go. This app only ever sees: the adapter's own declared
// config fields (config_schema, used to render the form below) and the
// text it returns; it never knows what the adapter actually connects to.
// Several can be imported at once (one per customer/source), but only one
// is ever ACTIVE+ENABLED - the list below is what the server tracks them
// as; importing never auto-selects one, the user picks explicitly. If the
// active one errors mid-poll, the server falls back to "Thủ công" on its
// own (adapterError below reflects why, until the next successful select).
const adapterList = ref([]); // [{id, name, active}, ...]
const adapterEnabled = ref(false);
const adapterError = ref('');
const activeAdapterId = computed(() => adapterList.value.find((a) => a.active)?.id || '');
// Clicking a box in the list just opens ITS config below (viewingAdapterId)
// - it does NOT switch the widget to it. Only the "Sử dụng" button under
// the form actually calls applyAdapter(), so looking at/editing a
// non-active adapter's config can never accidentally take over the widget.
const viewingAdapterId = ref('');
const viewingAdapter = computed(() => adapterList.value.find((a) => a.id === viewingAdapterId.value) || null);
const adapterFields = ref([]); // [{key, label}, ...] from the viewed adapter's config_schema
// "_interval_sec" is a reserved key an adapter MAY declare in its own
// config_schema if it wants the poll cycle user-configurable - it's not
// something every adapter gets shown regardless, so it's rendered
// specially (numeric input, fixed label) rather than looping it through
// like an ordinary field, and not shown at all unless the schema has it.
const hasIntervalField = computed(() => adapterFields.value.some((f) => f.key === '_interval_sec'));
const nonIntervalFields = computed(() => adapterFields.value.filter((f) => f.key !== '_interval_sec'));
const adapterConfig = ref({}); // current field values, keyed like adapterFields
const adapterFileInput = ref(null);
const adapterBusy = ref(false);
// Self-reported by the active adapter's own status()/log() (wasmsource.go),
// separate from adapterError above (which is only the APP's own trap
// fallback reason) - status is the "ok"/"error" the adapter itself wants
// shown, log is its own activity trail, both purely informational, never
// drive any Thủ công/Tự động switch on their own.
const adapterStatus = ref({ state: 'ok', message: '' });
const adapterLog = ref([]); // [{time, message}, ...], newest last
const showAdapterLog = ref(false);

async function loadAdapterStatus() {
    try {
        const res = await (await fetch('/text-adapter')).json();
        adapterList.value = Array.isArray(res.list) ? res.list : [];
        adapterEnabled.value = !!res.enabled;
        adapterStatus.value = res.status || { state: 'ok', message: '' };
        adapterLog.value = Array.isArray(res.log) ? res.log : [];
        const prevError = adapterError.value;
        adapterError.value = res.error || '';
        // The server just fell back to manual on its own (error appeared,
        // mode flipped off since the last time we checked) - say so once,
        // loudly, instead of leaving the user to notice the switch flipped
        // back by itself.
        if (adapterError.value && adapterError.value !== prevError) {
            toast(t('toastAdapterFellBack', { error: adapterError.value }), 'error');
            loadText();
        }
    } catch (_) {}
}
onMounted(loadAdapterStatus);
// Keep status()/log() fresh while an adapter is actually running - same
// 3s cadence as the text poll above (same reason: separate Wails process,
// no live-push channel, plain HTTP poll is the only thing that crosses it).
let adapterStatusPoll = null;
onMounted(() => {
    adapterStatusPoll = setInterval(() => {
        if (adapterEnabled.value) loadAdapterStatus();
    }, 3000);
});
onUnmounted(() => clearInterval(adapterStatusPoll));

// Opens (or closes, toggling the same box again) an adapter's config panel
// without applying anything - viewing/editing config is always allowed,
// regardless of which one is currently active or which mode is on.
async function viewAdapter(id) {
    if (viewingAdapterId.value === id) {
        viewingAdapterId.value = '';
        return;
    }
    viewingAdapterId.value = id;
    try {
        const schema = await (await fetch('/text-adapter/schema?id=' + encodeURIComponent(id))).json();
        adapterFields.value = Array.isArray(schema) ? schema : [];
        adapterConfig.value = await (await fetch('/text-adapter/config?id=' + encodeURIComponent(id))).json();
    } catch (_) {
        adapterFields.value = [];
        adapterConfig.value = {};
    }
}

async function pickAdapterFile(file) {
    if (!file) return;
    if (!/\.wasm$/i.test(file.name)) {
        toast(t('errAdapterWasmOnly'), 'error');
        return;
    }
    adapterBusy.value = true;
    try {
        const form = new FormData();
        form.append('wasm', file);
        const res = await fetch('/text-adapter', { method: 'POST', body: form });
        if (!res.ok) throw new Error(await res.text());
        toast(t('toastAdapterImported'), 'success');
        await loadAdapterStatus();
    } catch (err) {
        toastError(err);
    } finally {
        adapterBusy.value = false;
        if (adapterFileInput.value) adapterFileInput.value.value = '';
    }
}

// The ONLY action that actually switches the widget to a different
// adapter (and turns adapter mode on for it) - called by the "Sử dụng"
// button, never by just clicking/viewing a box in the list.
async function applyAdapter(id) {
    const form = new FormData();
    form.append('id', id);
    try {
        const res = await fetch('/text-adapter/select', { method: 'POST', body: form });
        if (!res.ok) throw new Error(await res.text());
        await loadAdapterStatus();
        setTimeout(loadText, 300); // its first poll lands almost immediately
    } catch (err) {
        toastError(err);
    }
}

// Called by the Thủ công/Tự động buttons - setAdapterEnabled(false) does
// NOT remove or deselect the active adapter, it just stops the widget
// from using it for now (the server keeps the file/config/selection).
async function setAdapterEnabled(enabled) {
    const form = new FormData();
    form.append('enabled', String(enabled));
    try {
        const res = await fetch('/text-adapter/enabled', { method: 'POST', body: form });
        if (!res.ok) throw new Error(await res.text());
        adapterEnabled.value = enabled;
        if (enabled) setTimeout(loadText, 300);
        else loadText();
    } catch (err) {
        toastError(err);
    }
}

async function saveAdapterConfig() {
    const id = viewingAdapterId.value;
    if (!id) return;
    try {
        await fetch('/text-adapter/config?id=' + encodeURIComponent(id), { method: 'POST', body: JSON.stringify(adapterConfig.value) });
        toast(t('toastAdapterConfigSaved'), 'success');
        // Only this adapter being both active AND enabled means the live
        // widget is actually running it right now - saving a non-active
        // one's config never affects what's currently showing.
        if (adapterEnabled.value && id === activeAdapterId.value) setTimeout(loadText, 300);
    } catch (err) {
        toastError(err);
    }
}

async function removeAdapter(id) {
    if (!(await confirmDialog(t('confirmAdapterRemove'), { confirmText: t('adapterRemoveBtn'), cancelText: t('modalCancel') }))) return;
    try {
        await fetch('/text-adapter?id=' + encodeURIComponent(id), { method: 'DELETE' });
        toast(t('toastAdapterRemoved'), 'success');
        if (viewingAdapterId.value === id) viewingAdapterId.value = '';
        await loadAdapterStatus();
        loadText();
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

    <!-- Widget screen position + text + GIF - one card: everything that
         shapes what's actually showing on the widget right now, instead of
         position/text and the GIF living in two separate cards. -->
    <div class="card bg-base-100">
      <form class="card-body gap-4" @submit.prevent="sendText">
        <!-- Row 1: left half = GIF preview + import/reset buttons, right
             half = Vị trí + Khởi động cùng Windows. section-divider: a
             visible rule between the two halves, since both now live in
             one merged card with no card border of their own to tell them
             apart. -->
        <div class="flex flex-col sm:flex-row gap-6 section-row">
          <div class="flex-1 flex flex-col sm:flex-row gap-5 min-w-0">
            <div class="flex flex-col items-center shrink-0">
              <h2 class="font-bold flex items-center gap-2 self-start section-title"><Icon name="image"/> {{ t('gifLabel') }}</h2>
              <div class="flex flex-col items-center mt-1">
                <div class="widget-replica">
                  <div class="text-bubble-slot">
                    <div ref="bubbleRef" class="text-bubble-preview" :class="'fx-' + textEffect" :key="previewAnimKey" :style="{ color: textColor, ...bubbleStyle }">
                      {{ previewShown }}
                    </div>
                  </div>
                  <GifCanvas :load="loadPreview" :cache-key="previewKey" size="w-28 h-28"/>
                </div>
                <span v-if="previewLines.length > 1" class="text-[10px] text-base-content/50 mt-1">
                  {{ t('textLineCounter', { i: previewIdx + 1, n: previewLines.length }) }}
                </span>
                <span class="field-label mt-1">{{ t('gifCurrent') }}</span>
              </div>
            </div>

            <!-- @container: once this column gets narrow (gets squeezed by
                 window resize), every button below drops its label text and
                 falls back to icon-only + native title tooltip instead of
                 overflowing/wrapping - see btn-label spans below. -->
            <div class="flex flex-col gap-2 flex-1 min-w-0 btn-container mt-9">
              <div class="dropzone dropzone-compact" :class="{ 'is-over': dragOver }"
                @click="fileInput?.click()" @dragover.prevent="dragOver = true" @dragleave="dragOver = false" @drop.prevent="onDrop">
                <input ref="fileInput" type="file" accept=".gif,image/gif" class="hidden" @change="pick($event.target.files[0])"/>
                <Icon name="upload" :size="18" class="text-primary"/>
                <div class="min-w-0">
                  <p class="font-bold text-sm truncate">{{ chosen ? chosen.name : t('dropzoneDefault') }}</p>
                  <p class="text-xs text-base-content/60">{{ t('dropzoneOr') }}</p>
                </div>
              </div>

              <div class="flex gap-2 flex-wrap">
                <button type="button" class="btn btn-primary btn-sm" :disabled="busy || !chosen" :title="t('uploadSubmit')" @click="apply">
                  <span v-if="busy" class="loading loading-spinner loading-sm"></span>
                  <Icon v-else name="check"/> <span class="btn-label">{{ t('uploadSubmit') }}</span>
                </button>
                <button v-if="chosen" type="button" class="btn btn-sm btn-ghost border-2 border-neutral gap-2" :disabled="busy" :title="t('cancelPickBtn')" @click="cancelPick">
                  <Icon name="x"/> <span class="btn-label">{{ t('cancelPickBtn') }}</span>
                </button>
                <button v-else type="button" class="btn btn-sm btn-ghost border-2 border-neutral gap-2" :title="t('resetSubmit')" @click="reset">
                  <Icon name="rotate-ccw"/> <span class="btn-label">{{ t('resetSubmit') }}</span>
                </button>
                <button v-if="account.token" type="button" class="btn btn-sm btn-ghost border-2 border-neutral gap-2" :title="t('pushCurrentGifBtn')" @click="pushCurrent">
                  <Icon name="upload"/> <span class="btn-label">{{ t('pushCurrentGifBtn') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="flex-1 flex flex-col gap-2 min-w-0 section-right">
            <h2 class="font-bold flex items-center gap-2 section-title"><Icon name="monitor"/> {{ t('positionLabel') }}</h2>
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

            <label class="flex items-center justify-between gap-3 cursor-pointer mt-3">
              <span class="flex flex-col gap-1">
                <h2 class="font-bold flex items-center gap-2 section-title"><Icon name="power"/> {{ t('autoStartLabel') }}</h2>
                <p class="text-xs text-base-content/60">{{ t('autoStartHint') }}</p>
              </span>
              <input type="checkbox" class="toggle toggle-primary toggle-sm shrink-0" :checked="autoStart" @change="setAutoStart($event.target.checked)" :aria-label="t('autoStartLabel')"/>
            </label>
          </div>
        </div>

        <!-- Row 2: Màu chữ/hiệu ứng (1/3, left) + Chữ hiển thị (2/3,
             right). section-row-top: a horizontal rule above, clearly
             separating this row from row 1 above it (on top of the mt-4
             spacing, which alone read as too subtle to tell the two rows
             apart). -->
        <!-- self-start on both columns below: don't stretch to equal
             height. row2-divider stays unstretched-default so it still
             spans the full row. -->
        <div class="flex flex-col sm:flex-row gap-6 mt-4 section-row section-row-top">
          <div class="flex flex-col gap-2 min-w-0 style-container self-start order-2 sm:order-1" style="flex: 1 1 0%;">
            <h2 class="font-bold flex items-center gap-2 section-title"><Icon name="sparkles"/> {{ t('textStyleLabel') }}</h2>
            <!-- style-row: column (each control full-width) while this
                 column is narrow, row (side by side) once it's wide enough
                 - driven by THIS container's own width, not the viewport,
                 since it's the 1/3 column that actually gets squeezed. -->
            <!-- The 3 color swatches (chữ/nền/viền) + opacity slider stay in
                 one row - each swatch labeled (not just a bare circle) so
                 it's clear which part of the bubble it colors, a tooltip
                 alone wasn't enough. Swatches are small circles now
                 (color-swatch below), not full-size <input type=color>
                 boxes. -->
            <div class="flex items-start gap-2 mt-1">
              <div class="flex flex-col gap-0.5 items-center">
                <span class="text-[10px] text-base-content/60">{{ t('textColorShort') }}</span>
                <input v-model="textColor" type="color" class="color-swatch" :title="t('textColorLabel')" @change="saveTextStyle"/>
              </div>
              <div class="flex flex-col gap-0.5 items-center">
                <span class="text-[10px] text-base-content/60">{{ t('bubbleColorShort') }}</span>
                <input v-model="bubbleColor" type="color" class="color-swatch" :title="t('bubbleColorLabel')" @change="saveTextStyle"/>
              </div>
              <div class="flex flex-col gap-0.5 items-center">
                <span class="text-[10px] text-base-content/60">{{ t('bubbleBorderColorShort') }}</span>
                <input v-model="bubbleBorderColor" type="color" class="color-swatch" :title="t('bubbleBorderColorLabel')" @change="saveTextStyle"/>
              </div>
              <div class="flex flex-col gap-0.5 flex-1">
                <span class="text-[10px] text-base-content/60">{{ t('bubbleOpacityShort') }}</span>
                <input v-model.number="bubbleOpacity" type="range" min="0.2" max="1" step="0.02" class="range range-xs" :title="t('bubbleOpacityLabel')" @change="saveTextStyle"/>
              </div>
            </div>
            <div class="style-row mt-2">
              <div class="flex flex-col gap-0.5 flex-1 min-w-0">
                <span class="text-[10px] text-base-content/60">{{ t('textEffectShort') }}</span>
                <select v-model="textEffect" class="select select-bordered select-sm w-full style-select" :title="t('textStyleLabel')" @change="saveTextStyle">
                  <option v-for="fx in TEXT_EFFECTS" :key="fx.id" :value="fx.id">{{ t(fx.labelKey) }}</option>
                </select>
              </div>
              <div class="flex flex-col gap-0.5 min-w-0">
                <span class="text-[10px] text-base-content/60">{{ t('textIntervalShort') }}</span>
                <select v-model.number="textInterval" class="select select-bordered select-sm style-select" :title="t('textIntervalLabel')" @change="saveTextStyle">
                  <option v-for="n in TEXT_INTERVALS" :key="n" :value="n">{{ t('textIntervalOption', { n }) }}</option>
                </select>
              </div>
            </div>
          </div>

          <!-- Real flex item (stretches full row height), not a ::before
               on a self-start column (would be cut short). -->
          <div class="hidden sm:block row2-divider order-2"></div>

          <div class="flex flex-col gap-2 min-w-0 section-right-plain self-start order-1 sm:order-3" style="flex: 2 1 0%;">
            <h2 class="font-bold flex items-center gap-2 section-title"><Icon name="type"/> {{ t('textLabel') }}</h2>
            <p class="text-xs text-base-content/60">{{ adapterEnabled ? t('textHintDisabledByAdapter') : t('textHint') }}</p>
            <div class="flex gap-2 mt-1 items-stretch btn-container">
              <textarea v-model="text" @input="textDirty = true" :disabled="adapterEnabled" :placeholder="t('textPlaceholder')" rows="1" class="textarea textarea-bordered auth-input text-input-thin flex-1 min-w-0 resize-y" autofocus></textarea>
              <button type="submit" class="btn btn-primary" :disabled="adapterEnabled" :title="t('textSubmit')">
                <Icon name="send" :size="16"/> <span class="btn-label">{{ t('textSubmit') }}</span>
              </button>
            </div>
          </div>
        </div>
      </form>
    </div>

    <!-- Automatic text source (.wasm adapter) -->
    <div class="card bg-base-100">
      <div class="card-body gap-4">
        <h2 class="font-bold flex items-center gap-2 section-title"><Icon name="plug"/> {{ t('adapterLabel') }}</h2>
        <p class="text-xs text-base-content/60">{{ t('adapterHint') }}</p>

        <p v-if="adapterError" class="text-xs text-error">{{ t('adapterFellBackNotice', { error: adapterError }) }}</p>

        <!-- The active adapter's own status()/log() (wasmsource.go) -
             separate from adapterError above: this never fell back to
             Thủ công on its own, it's just what the adapter itself says
             about its current run. Only shown while actually running. -->
        <div v-if="adapterEnabled" class="flex items-center gap-2 flex-wrap">
          <span class="flex items-center gap-1.5 text-xs font-bold px-2 py-1 rounded-md"
            :class="adapterStatus.state === 'error' ? 'bg-error/15 text-error' : 'bg-success/15 text-success'">
            <Icon :name="adapterStatus.state === 'error' ? 'x' : 'check'" :size="12"/>
            {{ adapterStatus.state === 'error' ? t('adapterStatusError') : t('adapterStatusOk') }}
          </span>
          <span v-if="adapterStatus.message" class="text-xs text-base-content/60">{{ adapterStatus.message }}</span>
          <button type="button" class="btn btn-xs btn-ghost border-2 border-neutral gap-1" @click="showAdapterLog = !showAdapterLog">
            <Icon name="history" :size="12"/> {{ showAdapterLog ? t('adapterLogToggleHide') : t('adapterLogToggleShow') }}
          </button>
        </div>
        <div v-if="adapterEnabled && showAdapterLog" class="rounded-md border-2 border-neutral bg-base-200 p-2 max-h-40 overflow-y-auto text-xs font-mono flex flex-col gap-1">
          <p v-if="!adapterLog.length" class="text-base-content/50">{{ t('adapterLogEmpty') }}</p>
          <p v-for="(entry, i) in adapterLog" :key="i" class="text-base-content/80">
            <span class="text-base-content/40">{{ entry.time?.slice(11, 19) }}</span> {{ entry.message }}
          </p>
        </div>

        <!-- Mode toggle + import button on one row: both are "what governs
             the whole block below", so they belong together. -->
        <div class="flex items-center gap-2 flex-wrap btn-container">
          <div class="join">
            <button type="button" class="join-item btn btn-sm" :class="!adapterEnabled ? 'btn-primary' : 'btn-ghost border-2 border-neutral'" :title="t('adapterModeManual')" @click="setAdapterEnabled(false)">
              {{ t('adapterModeManual') }}
            </button>
            <button type="button" class="join-item btn btn-sm" :class="adapterEnabled ? 'btn-primary' : 'btn-ghost border-2 border-neutral'" :disabled="!activeAdapterId" :title="t('adapterModeAuto')" @click="setAdapterEnabled(true)">
              {{ t('adapterModeAuto') }}
            </button>
          </div>
          <button type="button" class="btn btn-sm btn-primary gap-2" :disabled="adapterBusy" :title="t('adapterImportBtn')" @click="adapterFileInput?.click()">
            <span v-if="adapterBusy" class="loading loading-spinner loading-xs"></span>
            <Icon v-else name="upload" :size="16"/> <span class="btn-label">{{ t('adapterImportBtn') }}</span>
          </button>
          <input ref="adapterFileInput" type="file" accept=".wasm" class="hidden" @change="pickAdapterFile($event.target.files[0])"/>
        </div>

        <!-- Which SOURCE is selected - several may be imported, only one
             of them is ever the active one. Boxes dim when mode is
             "Thủ công" (the list doesn't matter right now, nothing here
             is actually running), full color in "Tự động". Clicking a box
             only opens its config below - it never switches the widget on
             its own, that's what "Sử dụng" under the form is for. -->
        <div v-if="adapterList.length" class="flex flex-wrap gap-2" :class="{ 'opacity-40': !adapterEnabled }">
          <div v-for="a in adapterList" :key="a.id"
            class="flex items-center gap-1.5 pl-3 pr-1.5 py-1.5 rounded-md border-2 cursor-pointer"
            :class="viewingAdapterId === a.id ? 'bg-primary/20 border-primary' : a.active ? 'bg-success/10 border-success' : 'bg-base-200 border-neutral'"
            :title="`${a.description}\n${t('adapterAuthorLabel')}: ${a.author} - v${a.version}`"
            @click="viewAdapter(a.id)">
            <Icon v-if="a.active" name="check" :size="16" class="text-success"/>
            <span class="font-bold truncate max-w-[10rem]">{{ a.name }}</span>
            <button type="button" class="btn btn-xs btn-circle btn-error text-white" :title="t('adapterRemoveBtn')" @click.stop="removeAdapter(a.id)">
              <Icon name="x" :size="14"/>
            </button>
          </div>
        </div>
        <p v-else class="text-xs text-base-content/60">{{ t('adapterInactive') }}</p>

        <form v-if="viewingAdapter" class="flex flex-col gap-3 p-3 rounded-md border-2 border-neutral" @submit.prevent="saveAdapterConfig">
          <div class="flex flex-col gap-0.5 pb-2 border-b-2 border-neutral/30">
            <p class="text-sm">{{ viewingAdapter.description }}</p>
            <p class="text-xs text-base-content/60">{{ t('adapterAuthorLabel') }}: {{ viewingAdapter.author }} · v{{ viewingAdapter.version }}</p>
          </div>
          <p v-if="!adapterFields.length" class="text-xs text-base-content/60">{{ t('adapterNoFields') }}</p>
          <div v-for="f in nonIntervalFields" :key="f.key" class="flex flex-col gap-1">
            <label class="text-sm font-bold">{{ f.label || f.key }}</label>
            <input v-model="adapterConfig[f.key]" type="text" class="input input-bordered input-sm"/>
          </div>
          <!-- Only shown if this adapter's own config_schema actually
               declares "_interval_sec" - an adapter that doesn't want this
               configurable (uses its own fixed/default cycle) never shows
               it, instead of a field nobody asked for appearing anyway. -->
          <div v-if="hasIntervalField" class="flex flex-col gap-1">
            <label class="text-sm font-bold">{{ t('adapterIntervalLabel') }}</label>
            <input v-model.number="adapterConfig._interval_sec" type="number" min="2" max="600" class="input input-bordered input-sm w-32"/>
          </div>
          <div class="flex gap-2">
            <button type="submit" class="btn btn-sm btn-ghost border-2 border-neutral" :title="t('adapterSaveConfigBtn')">{{ t('adapterSaveConfigBtn') }}</button>
            <button type="button" class="btn btn-sm btn-primary" :disabled="viewingAdapterId === activeAdapterId && adapterEnabled" :title="t('adapterUseBtn')" @click="applyAdapter(viewingAdapterId)">
              {{ t('adapterUseBtn') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Icon+label buttons inside a .btn-container ancestor - hidden by default
   so the button is icon-only with a native title tooltip, shown again once
   that specific container (not the viewport - Tailwind 3 has no @container
   utility yet, that's Tailwind 4; this is plain CSS container queries,
   supported by the Chromium/WebView2 this app embeds) is wide enough that
   the label actually fits without wrapping/overflowing. 12rem chosen
   empirically for the longest label in this view ("Tải lên cloud") at
   btn-sm size. */
.btn-container { container-type: inline-size; }
.btn-label { display: none; }
@container (min-width: 12rem) {
  .btn-label { display: inline; }
}

/* .style-container/.style-row: the color/effect/interval controls stack as
   a column (each control full width) by default, and only switch to a
   single row once their OWN container is wide enough to hold all three
   WITHOUT shrinking below .style-select's min-width - same container-query
   approach as .btn-container above, not a viewport breakpoint, since this
   is the 1/3-width column that shrinks first. The 21rem threshold is
   56px (color) + 2*0.5rem gap + 2*6rem (min-width below) + a little slack,
   so the row never actually squeezes a select past the width its longest
   option ("Mờ dần hiện ra" / "Mỗi câu 30 giây") needs - that's what was
   cutting/overlapping the select's own text at the old 16rem threshold. */
.style-container { container-type: inline-size; }
.style-row { display: flex; flex-direction: column; gap: 0.5rem; }
.style-select { min-width: 6rem; }
@container (min-width: 21rem) {
  .style-row { flex-direction: row; align-items: center; }
}

/* .color-swatch: a small round color picker, instead of daisyUI's full
   .input box (rectangular, way bigger than needed for 3 of these side by
   side with an opacity slider in the same row). Browsers render
   type="color" with their own chrome around the swatch (a border + padding
   on Chromium) that can't be removed with background/border alone, so it's
   stripped first via -webkit-appearance before this shape is applied. */
.color-swatch {
  -webkit-appearance: none;
  appearance: none;
  width: 1.75rem;
  height: 1.75rem;
  padding: 0;
  border: 1.5px solid rgba(255, 255, 255, 0.25);
  border-radius: 50%;
  cursor: pointer;
  background: none;
}
.color-swatch::-webkit-color-swatch-wrapper { padding: 0; border-radius: 50%; }
.color-swatch::-webkit-color-swatch { border: none; border-radius: 50%; }

/* .text-input-thin: thinner border than .auth-input's shared 2px (theme/
   pixel.css, also used by the login form) - overridden locally here only,
   not in the shared file, since login's input isn't too thick, this
   textarea just reads heavier next to its own small icon-only button. */
.text-input-thin.auth-input { border-width: 1px !important; }
.text-input-thin.auth-input:focus-within { box-shadow: 2px 2px 0 #5c3eeb55 !important; }

/* .dropzone-compact: shrinks the shared .dropzone (pixel.css, also used by
   LibraryView here and in the portal) down for this narrower GIF column -
   overridden locally instead of touching the shared rule, which the other
   two usages still need at full size. */
.dropzone-compact { gap: 0.5rem; padding: 0.75rem; }

/* Each card section's own h2 heading (icon + text together, via
   currentColor) - plain white, not themed. */
.section-title { color: #fff !important; }

/* .section-row/.section-right/.section-row-top: visual separators between
   the config blocks now that they all live inside one merged card with no
   card border of their own - a vertical rule between each row's two
   halves, and a horizontal rule above row 2. Dashed + the same
   purple→pink gradient as .gradient-text (the "Ancol-Ani" brand text),
   not a plain border, which can't take a gradient color on its own - a
   ::before overlay with the gradient as background, cut into dashes via
   mask-image (a repeating-linear-gradient alpha pattern), does both at
   once. Only on sm:flex-row (side by side); stacked on mobile-width has
   no "between" to draw a line through. */
@media (min-width: 640px) {
  .section-row > :not(:first-child).section-right {
    position: relative;
    padding-left: 1rem;
  }
  .section-row > :not(:first-child).section-right::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 1px;
    background: linear-gradient(180deg, #8b73ff, #ff6bd6);
    -webkit-mask-image: repeating-linear-gradient(180deg, #000 0 6px, transparent 6px 12px);
    mask-image: repeating-linear-gradient(180deg, #000 0 6px, transparent 6px 12px);
  }
  .section-right-plain { padding-left: 1rem; }
}
/* row2-divider: real flex item version of the same dashed gradient rule
   above, no padding-left of its own (the gap-6 on the row already spaces
   it out). */
.row2-divider {
  width: 1px;
  background: linear-gradient(180deg, #8b73ff, #ff6bd6);
  -webkit-mask-image: repeating-linear-gradient(180deg, #000 0 6px, transparent 6px 12px);
  mask-image: repeating-linear-gradient(180deg, #000 0 6px, transparent 6px 12px);
}
.section-row-top {
  position: relative;
  padding-top: 1rem;
}
.section-row-top::before {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 1px;
  background: linear-gradient(90deg, #8b73ff, #ff6bd6);
  -webkit-mask-image: repeating-linear-gradient(90deg, #000 0 6px, transparent 6px 12px);
  mask-image: repeating-linear-gradient(90deg, #000 0 6px, transparent 6px 12px);
}

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
  /* background/border-color come from the :style="bubbleStyle" binding
     above (user-customizable) - these are just a fallback before it loads. */
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
<style src="@theme/text-effects.css"></style>
