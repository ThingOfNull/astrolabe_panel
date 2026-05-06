<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getRpc } from '@/api/jsonrpc';
import Canvas from '@/canvas/Canvas.vue';
import { DESIGN_GRID_HEIGHT, DESIGN_GRID_WIDTH } from '@/canvas/types';
import type { Widget } from '@/canvas/types';
import { useViewportMode } from '@/composables/useViewportMode';
import { useBoardStore } from '@/stores/board';
import { useProbeStore } from '@/stores/probe';
import { useWidgetStore } from '@/stores/widgets';
import WidgetRenderer from '@/widgets/WidgetRenderer.vue';
import { Icon } from '@iconify/vue';

const { t } = useI18n();
const router = useRouter();
const boardStore = useBoardStore();
const widgetStore = useWidgetStore();
const probeStore = useProbeStore();

const searchFocusKey = ref(0);

let unsubscribe: (() => void) | null = null;
let widgetTimer: ReturnType<typeof setInterval> | null = null;
let probeTimer: ReturnType<typeof setInterval> | null = null;

const basePx = computed(() => boardStore.board?.grid_base_unit ?? 10);
const { mode, scale } = useViewportMode(() => basePx.value);

// Scaled canvas size fills canvas-shell box (avoid footer overlap / overflow clip).
const canvasVisualWidth = computed(() => DESIGN_GRID_WIDTH * basePx.value * scale.value);
const canvasVisualHeight = computed(() => DESIGN_GRID_HEIGHT * basePx.value * scale.value);

// Compact mode: sort widgets by (y, x) into a vertical feed.
const compactWidgets = computed<Widget[]>(() =>
  [...widgetStore.widgets].sort((a, b) => a.y - b.y || a.x - b.x),
);

onMounted(() => {
  const rpc = getRpc();
  unsubscribe = rpc.onStatus((s) => {
    if (s === 'connected') {
      void refresh();
    }
  });
  widgetTimer = setInterval(() => void widgetStore.fetchAll(), 30_000);
  probeTimer = setInterval(() => void probeStore.fetchAll(), 5_000);
  document.addEventListener('keydown', onGlobalKey);
});

onUnmounted(() => {
  unsubscribe?.();
  if (widgetTimer) clearInterval(widgetTimer);
  if (probeTimer) clearInterval(probeTimer);
  document.removeEventListener('keydown', onGlobalKey);
});

async function refresh(): Promise<void> {
  await Promise.all([boardStore.fetchBoard(), widgetStore.fetchAll(), probeStore.fetchAll()]);
}

function onGlobalKey(e: KeyboardEvent): void {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault();
    searchFocusKey.value += 1;
  }
}
</script>

<template>
  <main class="relative min-h-screen w-full overflow-auto p-6">
    <header class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">
          {{ boardStore.board?.name ?? t('app.name') }}
        </h1>
      </div>
      <button
        type="button"
        :title="t('home.openSettings')"
        class="astro-btn-icon border border-[color:var(--astro-glass-border)] p-2.5 text-[color:var(--astro-text-secondary)] outline-none ring-offset-2 ring-offset-[color:var(--astro-bg-base)] hover:border-[color:var(--astro-accent)]/40 hover:bg-white/[0.06] hover:text-[color:var(--astro-text-primary)] hover:shadow-md focus-visible:ring-2 focus-visible:ring-[color:var(--astro-accent)]"
        @click="router.push({ name: 'settings' })"
      >
        <Icon
          icon="mdi:cog"
          width="22"
          height="22"
          aria-hidden="true"
        />
        <span class="sr-only">{{ t('home.openSettings') }}</span>
      </button>
    </header>

    <section
      v-if="widgetStore.widgets.length === 0 && !widgetStore.loading"
      class="astro-glass mt-12 mx-auto max-w-md p-8 text-center"
    >
      <p class="text-sm text-[color:var(--astro-text-secondary)]">
        {{ t('home.empty') }}
      </p>
      <button
        type="button"
        class="mt-4 rounded-md border border-[color:var(--astro-glass-border)] px-4 py-2 text-sm hover:bg-white/5"
        @click="router.push({ name: 'settings' })"
      >
        {{ t('home.openSettings') }}
      </button>
    </section>

    <!-- Compact: vertical feed -->
    <section
      v-else-if="mode === 'compact'"
      class="flex flex-col gap-3 select-none"
    >
      <div
        v-for="w in compactWidgets"
        :key="w.id"
        class="astro-glass min-h-[120px] w-full p-1 transition duration-200 motion-reduce:transition-none hover:border-[color:var(--astro-accent)]/35 hover:shadow-lg"
      >
        <WidgetRenderer
          :widget="w"
          :interactive="true"
          :search-focus-key="searchFocusKey"
        />
      </div>
    </section>

    <!-- Desktop / tablet: scaled canvas, centered, view-only -->
    <div
      v-else
      class="canvas-shell mx-auto select-none"
      :style="{
        width: `${canvasVisualWidth}px`,
        height: `${canvasVisualHeight}px`,
      }"
    >
      <Canvas
        mode="view"
        :widgets="widgetStore.widgets"
        :base-px="basePx"
        :scale="scale"
      >
        <template #widget="{ widget }">
          <WidgetRenderer
            :widget="widget"
            :interactive="true"
            :search-focus-key="searchFocusKey"
          />
        </template>
      </Canvas>
    </div>

    <footer
      class="pointer-events-none absolute bottom-4 left-0 right-0 z-[2] flex justify-center"
    >
      <a
        href="https://github.com/ThingOfNull/astrolabe_panel"
        class="pointer-events-auto text-[10px] tracking-wide text-[color:var(--astro-text-secondary)] opacity-60 underline decoration-[color:color-mix(in_srgb,var(--astro-text-secondary)_35%,transparent)] underline-offset-2 transition hover:opacity-100 hover:text-[color:var(--astro-text-primary)] hover:decoration-[color:var(--astro-accent)] focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[color:var(--astro-accent)]"
        target="_blank"
        rel="noopener noreferrer"
      >
        Powered by Astrolabe
      </a>
    </footer>
  </main>
</template>

<style scoped>
.canvas-shell {
  /* Scaled design size; mx-auto centers canvas in viewport like settings preview. */
  position: relative;
  overflow: hidden;
}
</style>
