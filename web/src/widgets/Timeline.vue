<script setup lang="ts">
/**
 * Timeline: SVG event strip rendered from an EntityList payload.
 *
 * Each entity's `extra.ts` (unix seconds) places a marker on a horizontal
 * track; status drives the marker color. Falls back to insertion order when
 * timestamps are missing.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import type { EntityListItem, EntityListPayload } from '@/api/types';
import type { Widget } from '@/canvas/types';
import { useEChartsTheme } from '@/composables/useEChartsTheme';
import { useMetricStore } from '@/stores/metric';

interface TimelineConfig {
  title?: string;
  /** Trim labels longer than this; default 16. */
  label_max_chars?: number;
}

const props = defineProps<{ widget: Widget }>();

const theme = useEChartsTheme();
const metricStore = useMetricStore();
const cfg = computed<TimelineConfig>(() => (props.widget.config ?? {}) as TimelineConfig);
const payload = computed(() => metricStore.payloadOf(props.widget.id));
const errorMsg = computed(() => metricStore.errorOf(props.widget.id));

const wrapper = ref<HTMLDivElement | null>(null);
const widthPx = ref(0);
const heightPx = ref(0);
let observer: ResizeObserver | null = null;
onMounted(() => {
  if (!wrapper.value) return;
  observer = new ResizeObserver(([entry]) => {
    widthPx.value = Math.max(0, entry.contentRect.width);
    heightPx.value = Math.max(0, entry.contentRect.height);
  });
  observer.observe(wrapper.value);
});
onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});

interface Marker {
  x: number;
  label: string;
  status: EntityListItem['status'];
  ts?: number;
}

const markers = computed<Marker[]>(() => {
  const ent = payload.value?.entity_list as EntityListPayload | undefined;
  const items = ent?.items ?? [];
  if (items.length === 0 || widthPx.value === 0) return [];

  const tsList = items
    .map((it, idx) => readTs(it) ?? idx)
    .filter((v): v is number => Number.isFinite(v));
  const tMin = Math.min(...tsList);
  const tMax = Math.max(...tsList);
  const range = tMax === tMin ? 1 : tMax - tMin;
  const inner = widthPx.value - 24;

  const labelMax = cfg.value.label_max_chars ?? 16;
  return items.map((it, idx) => {
    const ts = readTs(it) ?? idx;
    const x = 12 + ((ts - tMin) / range) * inner;
    const label = it.label.length > labelMax ? it.label.slice(0, labelMax - 1) + '…' : it.label;
    return { x, label, status: it.status, ts };
  });
});

function readTs(item: EntityListItem): number | undefined {
  const v = item.extra?.ts;
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  return undefined;
}

function statusColor(s: EntityListItem['status']): string {
  switch (s) {
    case 'ok':
      return theme.value.statusOk;
    case 'down':
      return theme.value.statusErr;
    case 'warn':
      return '#fbbf24';
    default:
      return theme.value.statusUnknown;
  }
}

const trackY = computed(() => Math.round(heightPx.value / 2));
</script>

<template>
  <div
    ref="wrapper"
    class="relative h-full w-full p-2"
  >
    <p
      v-if="cfg.title"
      class="text-[length:var(--fs-xs)] text-[color:var(--astro-text-secondary)]"
    >
      {{ cfg.title }}
    </p>
    <svg
      :width="widthPx"
      :height="heightPx"
      class="absolute inset-0"
      role="img"
      aria-hidden="true"
    >
      <line
        :x1="12"
        :y1="trackY"
        :x2="widthPx - 12"
        :y2="trackY"
        :stroke="theme.splitLine"
        stroke-width="1"
      />
      <g
        v-for="(m, i) in markers"
        :key="i"
      >
        <circle
          :cx="m.x"
          :cy="trackY"
          r="5"
          :fill="statusColor(m.status)"
          :stroke="theme.surface"
          stroke-width="1.5"
        />
        <text
          :x="m.x"
          :y="trackY + (i % 2 === 0 ? -10 : 16)"
          text-anchor="middle"
          :fill="theme.textSecondary"
          font-size="9"
        >
          {{ m.label }}
        </text>
      </g>
    </svg>
    <div
      v-if="errorMsg"
      class="absolute inset-x-2 bottom-1 truncate rounded bg-red-700/40 px-2 py-1 text-[length:var(--fs-xs)] text-red-100"
      :title="errorMsg"
    >
      {{ errorMsg }}
    </div>
  </div>
</template>
