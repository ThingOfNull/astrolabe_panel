<script setup lang="ts">
import { computed } from 'vue';

import type { Widget } from '@/canvas/types';

import { defaultDividerWidgetConfig, type DividerWidgetConfig } from './types';

const props = defineProps<{
  widget: Widget;
}>();

const cfg = computed<DividerWidgetConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<DividerWidgetConfig>;
  const base = defaultDividerWidgetConfig();
  const orientation =
    c.orientation === 'vertical' || c.orientation === 'horizontal'
      ? c.orientation
      : base.orientation;
  return {
    ...base,
    ...c,
    orientation,
  };
});

// Default grid footprint 28x2.
const DEFAULT_W = 28;
const DEFAULT_H = 2;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(1, Math.min(rw, rh));
});
const s = computed(() => scale.value);

const minLen = computed(() => `${Math.round(32 * s.value)}px`);

const lineCss = computed((): Record<string, string> => {
  const thickness = cfg.value.thickness != null && cfg.value.thickness > 0 ? cfg.value.thickness : 2;
  const color =
    cfg.value.color && cfg.value.color.trim() !== ''
      ? cfg.value.color
      : 'var(--astro-glass-border)';
  const style = cfg.value.line_style ?? 'solid';
  const t = String(thickness);
  if (cfg.value.orientation === 'vertical') {
    if (style === 'solid') {
      return {
        width: `${t}px`,
        height: '100%',
        minHeight: minLen.value,
        backgroundColor: color,
      };
    }
    return {
      width: '0',
      height: '100%',
      minHeight: minLen.value,
      borderRightWidth: `${t}px`,
      borderRightStyle: style,
      borderRightColor: color,
    };
  }
  if (style === 'solid') {
    return {
      height: `${t}px`,
      width: '100%',
      minWidth: minLen.value,
      backgroundColor: color,
    };
  }
  return {
    height: '0',
    width: '100%',
    minWidth: minLen.value,
    borderTopWidth: `${t}px`,
    borderTopStyle: style,
    borderTopColor: color,
  };
});

const padPx = computed(() => `${Math.round(8 * s.value)}px`);
</script>

<template>
  <div
    class="flex h-full w-full items-center justify-center box-border"
    :style="{ padding: padPx }"
  >
    <div
      role="separator"
      class="shrink-0"
      :aria-orientation="cfg.orientation === 'vertical' ? 'vertical' : 'horizontal'"
      :style="lineCss"
    />
  </div>
</template>
