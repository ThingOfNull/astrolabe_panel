<script setup lang="ts">
import { computed } from 'vue';

import type { Widget } from '@/canvas/types';

import WidgetIcon from './WidgetIcon.vue';
import type { SmartLinkConfig, TextStyle } from './types';

const props = defineProps<{
  widget: Widget;
  status: 'ok' | 'down' | 'unknown';
  interactive: boolean;
}>();

const cfg = computed<SmartLinkConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<SmartLinkConfig>;
  const def: SmartLinkConfig = {
    title: c.title ?? '未命名',
    url: c.url ?? '',
    layout: c.layout === 'vertical' ? 'vertical' : 'horizontal',
    show_icon: c.show_icon !== false,
    show_title: c.show_title !== false,
    show_url: c.show_url !== false,
    title_style: { ...(c.title_style ?? {}) },
    url_style: { ...(c.url_style ?? {}) },
    open_in_new_tab: c.open_in_new_tab ?? true,
    probe: c.probe,
  };
  return def;
});

// Default grid footprint 12x8.
const DEFAULT_W = 12;
const DEFAULT_H = 8;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(1, Math.min(rw, rh));
});

const s = computed(() => scale.value);

const statusMeta = computed(() => {
  switch (props.status) {
    case 'ok':
      return { color: 'var(--astro-status-ok)', label: 'Online' };
    case 'down':
      return { color: 'var(--astro-status-err)', label: 'Offline' };
    default:
      return { color: 'var(--astro-status-unknown)', label: 'Unknown' };
  }
});

const titleCss = computed(() => textStyleToCss(cfg.value.title_style, false));
const urlCss = computed(() => textStyleToCss(cfg.value.url_style, true));

function textStyleToCss(st: TextStyle | undefined, isUrl: boolean): Record<string, string> {
  const out: Record<string, string> = {};
  if (!st) {
    out.color = 'var(--astro-text-primary)';
    if (isUrl) out.fontSize = `${Math.round(12 * s.value)}px`;
    return out;
  }
  if (st.font_size) {
    out.fontSize = st.font_size;
  } else if (isUrl) {
    out.fontSize = `${Math.round(12 * s.value)}px`;
  }
  if (st.font_weight) out.fontWeight = String(st.font_weight);
  if (st.color && st.color.trim() !== '') {
    out.color = st.color;
  } else {
    out.color = isUrl ? 'var(--astro-text-secondary)' : 'var(--astro-text-primary)';
  }
  return out;
}

const isVertical = computed(() => cfg.value.layout === 'vertical');

const linkStyle = computed(() => {
  const pad = Math.round(16 * s.value);
  const gap = Math.round(isVertical.value ? 8 : 12) * s.value;
  return {
    padding: `${pad}px`,
    gap: `${Math.round(gap)}px`,
  } as Record<string, string>;
});

const textColStyle = computed(() => {
  const gap = Math.round((isVertical.value ? 8 : 4) * s.value);
  return { gap: `${gap}px` } as Record<string, string>;
});

const dotSize = computed(() => `${Math.round(8 * s.value)}px`);

const iconSize = computed(() => `${Math.round(36 * s.value)}px`);

function onClick(e: MouseEvent): void {
  if (!props.interactive) {
    e.preventDefault();
  }
}
</script>

<template>
  <a
    :href="cfg.url"
    :target="cfg.open_in_new_tab ? '_blank' : '_self'"
    rel="noopener noreferrer"
    class="group flex h-full w-full min-h-0 min-w-0 transition-transform duration-200 hover:brightness-110 hover:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.06)] motion-reduce:transition-none motion-reduce:hover:shadow-none motion-reduce:hover:brightness-100"
    :class="[
      isVertical ? 'flex-col items-center justify-center text-center' : 'flex-row items-center text-left',
      { 'pointer-events-none': !interactive },
    ]"
    :style="linkStyle"
    @click="onClick"
  >
    <WidgetIcon
      v-if="cfg.show_icon"
      class="shrink-0 transition-transform duration-200 group-hover:scale-105 motion-reduce:group-hover:scale-100"
      :type="widget.icon_type"
      :value="widget.icon_value"
      :size="iconSize"
      fallback="mdi:link-variant"
    />
    <div
      class="min-w-0 flex-1 overflow-hidden"
      :class="isVertical ? 'flex-col items-center' : 'flex-col justify-center'"
      :style="textColStyle"
    >
      <div
        v-if="cfg.show_title"
        class="flex items-center gap-2"
        :class="isVertical ? 'justify-center' : ''"
      >
        <span
          class="inline-block shrink-0 rounded-full"
          :style="{ backgroundColor: statusMeta.color, width: dotSize, height: dotSize }"
          :title="statusMeta.label"
        />
        <span
          class="truncate font-medium"
          :title="cfg.title"
          :style="titleCss"
        >{{ cfg.title }}</span>
      </div>
      <div
        v-else-if="cfg.show_url"
        class="flex items-center gap-2"
        :class="isVertical ? 'justify-center' : ''"
      >
        <span
          class="inline-block shrink-0 rounded-full"
          :style="{ backgroundColor: statusMeta.color, width: dotSize, height: dotSize }"
          :title="statusMeta.label"
        />
        <span
          class="block truncate opacity-90"
          :style="urlCss"
        >{{ cfg.url }}</span>
      </div>
      <span
        v-if="cfg.show_title && cfg.show_url"
        class="block truncate opacity-90"
        :style="urlCss"
      >{{ cfg.url }}</span>
    </div>
  </a>
</template>
