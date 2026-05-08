<script setup lang="ts">
// Widget chrome: drag/resize/select; optional glass or solid fill.
// Full-bleed children can steal corner hits; edit mode uses pointer-events-none on inner slot.

import interact from 'interactjs';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { WidgetAppearance } from '@/widgets/types';
import { parseWidgetAppearance } from '@/widgets/widgetAppearance';

import type { Rect, Widget } from './types';
import { clampToCanvas, hasCollision, snap } from './geometry';
import { DESIGN_GRID_HEIGHT, DESIGN_GRID_WIDTH } from './types';

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    widget: Widget;
    basePx: number;
    mode: 'view' | 'edit';
    selected: boolean;
    others: Rect[];
    scale?: number;
    multiSelected?: boolean;
    /** Omit to parse appearance from widget.config. */
    appearance?: WidgetAppearance | null;
  }>(),
  { scale: 1, multiSelected: false, appearance: null },
);

const emit = defineEmits<{
  (
    e: 'select',
    id: number,
    modifiers: { ctrl: boolean; shift: boolean; meta: boolean },
  ): void;
  (e: 'update', id: number, rect: Rect, opts: { commit: boolean }): void;
  (e: 'edit', id: number): void;
  (e: 'delete', id: number): void;
  (e: 'multi-move', id: number, deltaPx: { dx: number; dy: number }): void;
  (e: 'multi-commit', id: number): void;
}>();

const resolvedAppearance = computed<WidgetAppearance>(() => {
  if (props.appearance && props.appearance.variant) {
    return props.appearance;
  }
  return parseWidgetAppearance(props.widget.config);
});

const chromeStyle = computed((): Record<string, string> => {
  const ap = resolvedAppearance.value;
  if (ap.variant === 'solid' && ap.solid_color && ap.solid_color.trim() !== '') {
    return {
      background: ap.solid_color,
      border: '1px solid var(--astro-glass-border)',
    } as Record<string, string>;
  }
  if (ap.variant === 'glass') {
    const blur = ap.blur_px != null && ap.blur_px > 0 ? Math.min(ap.blur_px, 64) : 16;
    return {
      background: 'var(--astro-glass-bg)',
      backdropFilter: `blur(${blur}px) saturate(180%)`,
      WebkitBackdropFilter: `blur(${blur}px) saturate(180%)`,
      border: '1px solid var(--astro-glass-border)',
    } as Record<string, string>;
  }
  return {};
});

const root = ref<HTMLElement | null>(null);
const dragX = ref(0);
const dragY = ref(0);
let interactable: ReturnType<typeof interact> | null = null;

let lastBroadcastDx = 0;
let lastBroadcastDy = 0;

/** After group drag the click would clear multi-select; swallow one click. */
let suppressNextSelectClickAfterGroupDrag = false;
let suppressSelectClickTimer: ReturnType<typeof setTimeout> | null = null;

function applyTransform(): void {
  if (!root.value) return;
  root.value.style.transform = `translate(${dragX.value}px, ${dragY.value}px)`;
}

function setupInteract(): void {
  if (!root.value) return;
  if (interactable) {
    interactable.unset();
    interactable = null;
  }
  if (props.mode !== 'edit') return;

  interactable = interact(root.value);
  interactable
    .draggable({
      inertia: false,
      autoScroll: false,
      listeners: {
        start() {
          // Dragging a selected item in a multi-selection: do not emit bare select
          // or parent treats it as single-select and drops the rest.
          if (!(props.selected && props.multiSelected)) {
            emit('select', props.widget.id, { ctrl: false, shift: false, meta: false });
          }
          dragX.value = 0;
          dragY.value = 0;
          lastBroadcastDx = 0;
          lastBroadcastDy = 0;
          applyTransform();
        },
        move(event) {
          const s = props.scale > 0 ? props.scale : 1;
          dragX.value += event.dx / s;
          dragY.value += event.dy / s;
          const snappedDx = snap(dragX.value, props.basePx);
          const snappedDy = snap(dragY.value, props.basePx);
          if (root.value) {
            root.value.style.transform = `translate(${snappedDx}px, ${snappedDy}px)`;
          }
          if (props.multiSelected) {
            const stepDx = snappedDx - lastBroadcastDx;
            const stepDy = snappedDy - lastBroadcastDy;
            if (stepDx !== 0 || stepDy !== 0) {
              lastBroadcastDx = snappedDx;
              lastBroadcastDy = snappedDy;
              emit('multi-move', props.widget.id, { dx: stepDx, dy: stepDy });
            }
          }
        },
        end() {
          if (props.multiSelected) {
            suppressNextSelectClickAfterGroupDrag = true;
            if (suppressSelectClickTimer !== null) {
              clearTimeout(suppressSelectClickTimer);
            }
            suppressSelectClickTimer = setTimeout(() => {
              suppressNextSelectClickAfterGroupDrag = false;
              suppressSelectClickTimer = null;
            }, 500);
            emit('multi-commit', props.widget.id);
            dragX.value = 0;
            dragY.value = 0;
            lastBroadcastDx = 0;
            lastBroadcastDy = 0;
            if (root.value) root.value.style.transform = '';
          } else {
            commitDrag();
          }
        },
      },
    })
    .resizable({
      edges: { left: true, right: true, top: true, bottom: true },
      inertia: false,
      listeners: {
        move(event) {
          if (!root.value) return;
          // event.rect is layout box (corner resize changes both w and h).
          const w = snap(Math.max(props.basePx, event.rect.width), props.basePx);
          const h = snap(Math.max(props.basePx, event.rect.height), props.basePx);
          const s = props.scale > 0 ? props.scale : 1;
          const deltaL = event.deltaRect.left / s;
          const deltaT = event.deltaRect.top / s;
          root.value.style.width = `${w}px`;
          root.value.style.height = `${h}px`;
          dragX.value += deltaL;
          dragY.value += deltaT;
          const snappedDx = snap(dragX.value, props.basePx);
          const snappedDy = snap(dragY.value, props.basePx);
          root.value.style.transform = `translate(${snappedDx}px, ${snappedDy}px)`;
        },
        end() {
          commitResize();
        },
      },
    });
}

function commitDrag(): void {
  if (!root.value) return;
  const baseX = props.widget.x * props.basePx;
  const baseY = props.widget.y * props.basePx;
  const snappedX = snap(baseX + dragX.value, props.basePx);
  const snappedY = snap(baseY + dragY.value, props.basePx);
  const desired: Rect = {
    x: snappedX / props.basePx,
    y: snappedY / props.basePx,
    w: props.widget.w,
    h: props.widget.h,
  };
  finalizeRect(desired);
}

function commitResize(): void {
  if (!root.value) return;
  const baseX = props.widget.x * props.basePx;
  const baseY = props.widget.y * props.basePx;
  const w = Math.max(props.basePx, root.value.offsetWidth);
  const h = Math.max(props.basePx, root.value.offsetHeight);
  const snappedX = snap(baseX + dragX.value, props.basePx);
  const snappedY = snap(baseY + dragY.value, props.basePx);
  const snappedW = Math.max(props.basePx, snap(w, props.basePx));
  const snappedH = Math.max(props.basePx, snap(h, props.basePx));
  const desired: Rect = {
    x: snappedX / props.basePx,
    y: snappedY / props.basePx,
    w: snappedW / props.basePx,
    h: snappedH / props.basePx,
  };
  finalizeRect(desired);
}

function finalizeRect(desired: Rect): void {
  const clamped = clampToCanvas(desired, DESIGN_GRID_WIDTH, DESIGN_GRID_HEIGHT);
  let next = clamped;
  if (hasCollision(clamped, props.others)) {
    next = {
      x: props.widget.x,
      y: props.widget.y,
      w: props.widget.w,
      h: props.widget.h,
    };
  }
  dragX.value = 0;
  dragY.value = 0;
  if (root.value) {
    root.value.style.transform = '';
    root.value.style.width = '';
    root.value.style.height = '';
  }
  emit('update', props.widget.id, next, { commit: true });
}

onMounted(() => {
  setupInteract();
});

onBeforeUnmount(() => {
  if (suppressSelectClickTimer !== null) {
    clearTimeout(suppressSelectClickTimer);
    suppressSelectClickTimer = null;
  }
  if (interactable) {
    interactable.unset();
    interactable = null;
  }
});

watch(
  () => props.mode,
  () => setupInteract(),
);

watch(
  () => props.scale,
  () => setupInteract(),
);

watch(
  () => props.basePx,
  () => setupInteract(),
);

function onClick(e: MouseEvent): void {
  if (props.mode !== 'edit') return;
  if (suppressNextSelectClickAfterGroupDrag) {
    suppressNextSelectClickAfterGroupDrag = false;
    if (suppressSelectClickTimer !== null) {
      clearTimeout(suppressSelectClickTimer);
      suppressSelectClickTimer = null;
    }
    return;
  }
  emit('select', props.widget.id, { ctrl: e.ctrlKey, shift: e.shiftKey, meta: e.metaKey });
}
</script>

<template>
  <div
    ref="root"
    :class="[
      'widget-frame',
      'absolute',
      'rounded-[inherit]',
      'border-transparent',
      { 'cursor-move': mode === 'edit', 'wf-selected': selected },
    ]"
    :style="{
      left: `${widget.x * basePx}px`,
      top: `${widget.y * basePx}px`,
      width: `${widget.w * basePx}px`,
      height: `${widget.h * basePx}px`,
      zIndex: widget.z_index,
      // Group-drag offset: Canvas writes --wf-dx/--wf-dy on selected followers
      // and we translate by them. CSS-only path avoids the previous DOM
      // querySelectorAll + el.style.transform side effect, which bypassed Vue
      // reactivity and broke a11y / transitions.
      transform: 'translate(var(--wf-dx, 0px), var(--wf-dy, 0px))',
      ...chromeStyle,
    }"
    :data-widget-id="widget.id"
    @click="onClick"
  >
    <div
      class="relative z-10 h-full min-h-0 min-w-0 overflow-hidden rounded-[inherit]"
      :class="mode === 'edit' ? 'pointer-events-none' : ''"
    >
      <slot />
    </div>
    <div
      v-if="mode === 'edit' && selected"
      class="pointer-events-auto absolute right-2 top-2 z-20 flex gap-1"
    >
      <button
        type="button"
        class="rounded bg-black/40 px-2 py-1 text-xs transition-transform hover:scale-105 active:scale-95 hover:bg-black/60"
        @click.stop="emit('edit', widget.id)"
      >
        {{ t('widget.edit') }}
      </button>
      <button
        type="button"
        class="rounded bg-red-600/70 px-2 py-1 text-xs transition-transform hover:scale-105 active:scale-95 hover:bg-red-600/90"
        @click.stop="emit('delete', widget.id)"
      >
        {{ t('widget.delete') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.widget-frame {
  position: absolute;
  isolation: isolate;
  transition: box-shadow 200ms ease;
  box-shadow:
    inset 0 1px 0 var(--astro-glass-highlight),
    var(--astro-glass-shadow),
    var(--astro-glass-glow);
  will-change: transform;
  border-radius: 12px;
}

.widget-frame::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  pointer-events: none;
  z-index: 1;
  border-top: 1px solid var(--astro-glass-highlight);
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 -1px 0 0 rgba(0, 0, 0, 0.3) inset;
}

.widget-frame:hover {
  box-shadow:
    inset 0 1px 0 var(--astro-glass-highlight),
    0 22px 54px -26px rgba(0, 0, 0, 0.64),
    var(--astro-glass-glow);
}

/* Selection: inset ring so drag/resize does not look oversized vs outline. */
.widget-frame.wf-selected {
  box-shadow:
    inset 0 0 0 2px var(--astro-accent),
    inset 0 1px 0 var(--astro-glass-highlight),
    var(--astro-glass-shadow),
    var(--astro-glass-glow);
}

.widget-frame.wf-selected:hover {
  box-shadow:
    inset 0 0 0 2px var(--astro-accent),
    inset 0 1px 0 var(--astro-glass-highlight),
    0 22px 54px -26px rgba(0, 0, 0, 0.64),
    var(--astro-glass-glow);
}

:root[data-theme='light'] .widget-frame::before {
  border-top: 1px solid rgba(255, 255, 255, 0.7);
  border-left: 1px solid rgba(255, 255, 255, 0.5);
  box-shadow: 0 -1px 0 0 rgba(0, 0, 0, 0.06) inset;
}

:root[data-theme='light'] .widget-frame {
  box-shadow:
    inset 0 1px 0 var(--astro-glass-highlight),
    var(--astro-glass-shadow),
    var(--astro-glass-glow);
}

:root[data-theme='light'] .widget-frame.wf-selected {
  box-shadow:
    inset 0 0 0 2px var(--astro-accent),
    inset 0 1px 0 var(--astro-glass-highlight),
    var(--astro-glass-shadow),
    var(--astro-glass-glow);
}

:root[data-theme='light'] .widget-frame.wf-selected:hover {
  box-shadow:
    inset 0 0 0 2px var(--astro-accent),
    inset 0 1px 0 var(--astro-glass-highlight),
    0 22px 48px -28px rgba(15, 23, 42, 0.34),
    var(--astro-glass-glow);
}

@media (max-width: 767px) {
  .widget-frame {
    backdrop-filter: none !important;
    -webkit-backdrop-filter: none !important;
    background: var(--astro-bg-base) !important;
    border-color: var(--astro-glass-border);
  }
}
</style>
