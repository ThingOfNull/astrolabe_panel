<script setup lang="ts">
import { computed } from 'vue';

import type { Widget } from '@/canvas/types';

import { defaultTextWidgetConfig, type TextWidgetConfig } from './types';

const props = defineProps<{
  widget: Widget;
}>();

const cfg = computed<TextWidgetConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<TextWidgetConfig>;
  const base = defaultTextWidgetConfig();
  return {
    ...base,
    ...c,
  };
});

// Default grid footprint 20x8.
const DEFAULT_W = 20;
const DEFAULT_H = 8;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(1, Math.min(rw, rh));
});
const s = computed(() => scale.value);

const blockStyle = computed((): Record<string, string> => {
  const out: Record<string, string> = {
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  };
  const c = cfg.value;
  if (c.font_size) out.fontSize = c.font_size;
  if (c.font_weight) out.fontWeight = String(c.font_weight);
  const col = c.color && c.color.trim() !== '' ? c.color : 'var(--astro-text-primary)';
  out.color = col;
  if (c.text_align) out.textAlign = c.text_align;
  if (c.orientation === 'vertical') {
    out.writingMode = 'vertical-rl';
    out.textOrientation = 'mixed';
  }
  return out as Record<string, string>;
});

const padPx = computed(() => `${Math.round(12 * s.value)}px`);
</script>

<template>
  <div
    class="flex h-full min-h-0 min-w-0 overflow-auto"
    :style="{ padding: padPx }"
  >
    <div
      class="min-h-0 w-full"
      :style="blockStyle"
    >
      {{ cfg.content || '' }}
    </div>
  </div>
</template>
