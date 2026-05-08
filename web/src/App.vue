<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue';

import { applyTheme } from '@/composables/useTheme';
import { useBoardStore } from '@/stores/board';
import { useDataSourceStore } from '@/stores/datasources';
import { useProbeStore } from '@/stores/probe';
import { useWidgetStore } from '@/stores/widgets';

const boardStore = useBoardStore();
const widgetStore = useWidgetStore();
const probeStore = useProbeStore();
const dsStore = useDataSourceStore();

// Root-level SSE subscriptions: every store merges incoming events into its
// reactive state, so all pages (Home, Settings) observe the same source of
// truth without each mounting their own listeners.
let disposers: Array<() => void> = [];

onMounted(() => {
  // Default to dark CSS tokens until board theme loads (avoid FOUC).
  if (!document.documentElement.dataset.theme) {
    document.documentElement.dataset.theme = 'dark';
  }
  disposers = [
    widgetStore.subscribeEvents(),
    probeStore.subscribeEvents(),
    boardStore.subscribeEvents(),
    dsStore.subscribeEvents(),
  ];
});

onUnmounted(() => {
  for (const d of disposers) d();
  disposers = [];
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
