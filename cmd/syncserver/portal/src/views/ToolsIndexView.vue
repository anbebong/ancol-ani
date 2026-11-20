<script setup>
import PageHeader from '@ui/components/PageHeader.vue';
import Icon from '@ui/components/Icon.vue';

const tools = [
    { to: '/tools/convert', icon: 'images', title: 'Chuyển đổi thành GIF', desc: 'Từ video, một ảnh, hoặc nhiều ảnh.', ready: true },
    { to: '/tools/resize', icon: 'image', title: 'Resize GIF', desc: 'Đổi kích thước GIF theo pixel.', ready: true },
    { to: '/tools/crop', icon: 'square', title: 'Crop GIF', desc: 'Cắt một vùng của GIF.', ready: true },
    { to: '/tools/optimize', icon: 'sparkles', title: 'Optimize GIF', desc: 'Giảm dung lượng GIF.', ready: true },
    { to: '/tools/rotate', icon: 'rotate-ccw', title: 'Rotate GIF', desc: 'Xoay hoặc lật GIF.', ready: true },
    { to: '/tools/reverse', icon: 'history', title: 'Reverse GIF', desc: 'Đảo ngược thứ tự khung hình.', ready: true },
    { to: '/tools/speed', icon: 'wand-2', title: 'Speed GIF', desc: 'Tăng/giảm tốc độ GIF.', ready: true },
    { to: '/tools/split', icon: 'images', title: 'Split GIF', desc: 'Tách GIF thành từng khung hình.', ready: true },
    { to: '/tools/caption', icon: 'type', title: 'Caption GIF', desc: 'Thêm chữ vào GIF.', ready: true },
];
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="wand-2" title="Công cụ GIF" subtitle="Các công cụ xử lý GIF cơ bản."/>

    <div class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
      <RouterLink v-for="t in tools.filter(t => t.ready)" :key="t.title" :to="t.to" class="card bg-base-100 hover:shadow-md transition-shadow">
        <div class="card-body gap-2">
          <Icon :name="t.icon" :size="28" class="text-primary"/>
          <h3 class="card-title text-base">{{ t.title }}</h3>
          <p class="text-sm text-base-content/60">{{ t.desc }}</p>
        </div>
      </RouterLink>
      <div v-for="t in tools.filter(t => !t.ready)" :key="t.title" class="card bg-base-100 opacity-50 cursor-not-allowed relative">
        <span class="badge badge-sm absolute top-3 right-3">Sắp có</span>
        <div class="card-body gap-2">
          <Icon :name="t.icon" :size="28"/>
          <h3 class="card-title text-base">{{ t.title }}</h3>
          <p class="text-sm text-base-content/60">{{ t.desc }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
