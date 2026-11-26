<script setup>
import { ref } from 'vue';
import PageHeader from '@theme/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import ToolPage from '../../components/ToolPage.vue';
import Icon from '@theme/components/Icon.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const angle = ref(0);
const flip = ref('none');
const result = ref(null);
const loading = ref(false);
const error = ref('');

let seq = 0;
async function onRun(file) {
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const fd = new FormData();
        fd.append('file', file);
        const r = await postTool(`/api/tools/rotate?angle=${angle.value}&flip=${flip.value}`, fd);
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
    <PageHeader icon="rotate-ccw" title="Rotate GIF" subtitle="Xoay hoặc lật GIF."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Rotate" @run="onRun">
      <template #params>
        <div class="flex flex-col gap-3">
          <div>
            <span class="label-text">Góc xoay</span>
            <div class="join">
              <button type="button" class="btn join-item" :class="{ 'btn-active': angle === 0 }" title="Không xoay" @click="angle = 0">0°</button>
              <button type="button" class="btn join-item" :class="{ 'btn-active': angle === 90 }" title="Xoay 90° cùng chiều kim đồng hồ" @click="angle = 90"><Icon name="rotate-cw" :size="18"/></button>
              <button type="button" class="btn join-item" :class="{ 'btn-active': angle === 180 }" title="Xoay 180°" @click="angle = 180">180°</button>
              <button type="button" class="btn join-item" :class="{ 'btn-active': angle === 270 }" title="Xoay 90° ngược chiều kim đồng hồ" @click="angle = 270"><Icon name="rotate-ccw" :size="18"/></button>
            </div>
          </div>
          <div>
            <span class="label-text">Lật</span>
            <div class="join">
              <button type="button" class="btn join-item" :class="{ 'btn-active': flip === 'none' }" title="Không lật" @click="flip = 'none'">Không</button>
              <button type="button" class="btn join-item" :class="{ 'btn-active': flip === 'h' }" title="Lật ngang" @click="flip = 'h'"><Icon name="flip-horizontal" :size="18"/></button>
              <button type="button" class="btn join-item" :class="{ 'btn-active': flip === 'v' }" title="Lật dọc" @click="flip = 'v'"><Icon name="flip-vertical" :size="18"/></button>
            </div>
          </div>
        </div>
      </template>
    </ToolPage>
  </div>
</template>
