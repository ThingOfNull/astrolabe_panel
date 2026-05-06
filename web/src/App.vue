<script setup lang="ts">
import { onMounted, watch } from 'vue';

import { applyTheme } from '@/composables/useTheme';
import { useBoardStore } from '@/stores/board';

const boardStore = useBoardStore();

onMounted(() => {
  // Default to dark CSS tokens until board theme loads (avoid FOUC).
  if (!document.documentElement.dataset.theme) {
    document.documentElement.dataset.theme = 'dark';
  }
});

watch(
  () =>
    [boardStore.board?.theme, boardStore.board?.theme_custom, boardStore.board?.wallpaper] as const,
  ([theme, custom, wallpaper]) => {
    if (!theme) return;
    applyTheme(theme, { customJSON: custom ?? '', wallpaper: wallpaper ?? '' });
  },
  { immediate: true },
);
</script>

<template>
  <div class="relative z-[1] min-h-screen w-full">
    <router-view />
  </div>
</template>
