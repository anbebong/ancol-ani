<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import Icon from '@ui/components/Icon.vue';
import ResultPreview from '../../components/ResultPreview.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const tab = ref('video'); // 'video' | 'image' | 'images'

const videoFile = ref(null);
const videoFps = ref(10);
const imageFile = ref(null);
const imageFps = ref(5);
const imagesFiles = ref([]);
const imagesFps = ref(5);

const result = ref(null);
const loading = ref(false);
const error = ref('');

function switchTab(t) {
    if (t === tab.value) return;
    tab.value = t;
    videoFile.value = null;
    imageFile.value = null;
    imagesFiles.value = [];
    result.value = null;
    error.value = '';
    // Invalidate any in-flight request from the previous tab: without this,
    // a slow video convert finishing after the user switched to "Ảnh" would
    // still satisfy `mySeq === seq` and write its result/error onto the new
    // tab's state.
    seq++;
    loading.value = false;
}

function onVideoFile(e) { videoFile.value = e.target.files[0] || null; }
function onImageFile(e) { imageFile.value = e.target.files[0] || null; }
function onImagesFiles(e) { imagesFiles.value = Array.from(e.target.files || []); }

function canRun() {
    if (tab.value === 'video') return !!videoFile.value;
    if (tab.value === 'image') return !!imageFile.value;
    return imagesFiles.value.length > 0;
}

// Sequence guard (same pattern as ResizeView.vue/GalleryView.vue): a
// response from a superseded request (ran again, or switched tabs and ran
// again, before the first finished) must not overwrite newer state.
let seq = 0;
async function run() {
    if (!canRun()) return;
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const fd = new FormData();
        let url = '/api/tools/convert';
        if (tab.value === 'video') {
            fd.append('file', videoFile.value);
            url += `?fps=${videoFps.value}`;
        } else if (tab.value === 'image') {
            fd.append('file', imageFile.value);
            url += `?fps=${imageFps.value}`;
        } else {
            for (const f of imagesFiles.value) fd.append('file', f);
            url += `?fps=${imagesFps.value}`;
        }
        const r = await postTool(url, fd);
        if (mySeq === seq) result.value = r;
    } catch (e) {
        if (mySeq === seq) error.value = errMsg(e);
    } finally {
        if (mySeq === seq) loading.value = false;
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <ToolBackLink/>
    <PageHeader icon="images" title="Chuyển đổi thành GIF" subtitle="Từ video, một ảnh, hoặc nhiều ảnh."/>

    <div role="tablist" class="tabs tabs-boxed w-fit">
      <a role="tab" class="tab" :class="{ 'tab-active': tab === 'video' }" @click="switchTab('video')">Video</a>
      <a role="tab" class="tab" :class="{ 'tab-active': tab === 'image' }" @click="switchTab('image')">Ảnh</a>
      <a role="tab" class="tab" :class="{ 'tab-active': tab === 'images' }" @click="switchTab('images')">Nhiều ảnh</a>
    </div>

    <div class="flex flex-col gap-4 max-w-xl">
      <template v-if="tab === 'video'">
        <input type="file" accept="video/*" class="file-input file-input-bordered w-full" @change="onVideoFile"/>
        <label class="form-control">
          <span class="label-text">FPS</span>
          <input v-model.number="videoFps" type="number" min="1" max="30" class="input input-bordered"/>
        </label>
      </template>

      <template v-else-if="tab === 'image'">
        <input type="file" accept="image/*" class="file-input file-input-bordered w-full" @change="onImageFile"/>
        <label class="form-control">
          <span class="label-text">Khung/giây</span>
          <input v-model.number="imageFps" type="number" min="1" max="30" class="input input-bordered"/>
        </label>
      </template>

      <template v-else>
        <input type="file" accept="image/*" multiple class="file-input file-input-bordered w-full" @change="onImagesFiles"/>
        <label class="form-control">
          <span class="label-text">Khung/giây</span>
          <input v-model.number="imagesFps" type="number" min="1" max="30" class="input input-bordered"/>
        </label>
      </template>

      <button type="button" class="btn btn-primary w-full" :disabled="!canRun() || loading" @click="run">
        <Icon name="sparkles"/> Chuyển đổi
      </button>

      <ResultPreview :result="result" :loading="loading" :error="error"/>
    </div>
  </div>
</template>
