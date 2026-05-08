<script setup lang="ts">
/**
 * WidgetTile: pure-render widget chrome used by the read-only HomePage.
 *
 * Mirrors the visual treatment of WidgetFrame (glass / solid variants, focus
 * ring, hover lift) but skips the entire interact.js drag/resize wiring —
 * which the home page never needs and which paid a per-mount cost on every
 * widget. Keeps WidgetFrame focused on edit-mode behavior.
 */

import { computed } from 'vue';

import type { Widget } from './types';
import { parseWidgetAppearance } from '@/widgets/widgetAppearance';

const props = withDefaults(
  defineProps<{
    widget: Widget;
    basePx: number;
    /**
     * When `floating`, the tile is positioned absolutely at the widget's
     * grid coordinates (canvas mode). When `flow`, layout is owned by the
     * parent (compact stack mode) and the tile fills its container.
     */
    layout?: 'floating' | 'flow';
  }>(),
  { layout: 'floating' },
);

const appearance = computed(() => parseWidgetAppearance(props.widget.config));

const chromeStyle = computed((): Record<string, string> => {
  const ap = appearance.value;
  if (ap.variant === 'solid' && ap.solid_color && ap.solid_color.trim() !== '') {
    return {
      background: ap.solid_color,
      border: '1px solid var(--astro-glass-border)',
      borderRadius: 'var(--radius-md)',
      boxShadow: 'var(--elev-1)',
    };
  }
  // Glass: rely on tokens.css surface-1 + elev-1 so light theme stays legible.
  const blurPx = Math.max(1, Math.min(64, ap.blur_px ?? 14));
  return {
    background: 'var(--astro-surface-1, var(--astro-glass-bg))',
    backdropFilter: `blur(${blurPx}px) saturate(160%)`,
    WebkitBackdropFilter: `blur(${blurPx}px) saturate(160%)`,
    border: '1px solid var(--astro-glass-border)',
    borderRadius: 'var(--radius-md)',
    boxShadow: 'var(--elev-1)',
  };
});

const positionStyle = computed((): Record<string, string> => {
  if (props.layout === 'flow') {
    return { position: 'relative', width: '100%', height: '100%' };
  }
  return {
    position: 'absolute',
    left: `${props.widget.x * props.basePx}px`,
    top: `${props.widget.y * props.basePx}px`,
    width: `${props.widget.w * props.basePx}px`,
    height: `${props.widget.h * props.basePx}px`,
    zIndex: String(props.widget.z_index ?? 0),
  };
});
</script>

<template>
  <div
    class="widget-tile transition-shadow duration-200 ease-out hover:shadow-[var(--elev-2)] motion-reduce:transition-none"
    :style="{ ...positionStyle, ...chromeStyle }"
    :data-widget-id="widget.id"
  >
    <div class="relative h-full w-full overflow-hidden rounded-[inherit]">
      <slot />
    </div>
  </div>
</template>
