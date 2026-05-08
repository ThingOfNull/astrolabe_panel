<script setup lang="ts">
// Canvas shell: grid + frames, palette drop, marquee; events bubble to SettingsPage.
//
// Selection stays in the parent so keyboard shortcuts match mouse interaction.

import { computed, ref } from 'vue';

import CanvasGrid from './CanvasGrid.vue';
import SmartGuides from './SmartGuides.vue';
import WidgetFrame from './WidgetFrame.vue';
import type { CanvasMode, Rect, Widget } from './types';
import { DESIGN_GRID_HEIGHT, DESIGN_GRID_WIDTH } from './types';
import { findFreeSpot, pxToMultiplier, rectsOverlap } from './geometry';

const props = withDefaults(
  defineProps<{
    widgets: Widget[];
    basePx: number;
    mode: CanvasMode;
    scale?: number;
    selectedIds?: number[];
  }>(),
  { scale: 1, selectedIds: () => [] as number[] },
);

const emit = defineEmits<{
  (
    e: 'select',
    id: number | null,
    modifiers: { ctrl: boolean; shift: boolean; meta: boolean },
  ): void;
  (e: 'update', id: number, rect: Rect): void;
  (e: 'updateMany', updates: { id: number; rect: Rect }[]): void;
  (e: 'edit', id: number): void;
  (e: 'delete', id: number): void;
  (e: 'drop', payload: { type: string; rect: Rect }): void;
  (e: 'marquee', ids: number[], modifiers: { ctrl: boolean; meta: boolean }): void;
}>();

const containerRef = ref<HTMLDivElement | null>(null);

const canvasWidthPx = computed(() => DESIGN_GRID_WIDTH * props.basePx);
const canvasHeightPx = computed(() => DESIGN_GRID_HEIGHT * props.basePx);

const selectedSet = computed(() => new Set(props.selectedIds));
function isSelected(id: number): boolean {
  return selectedSet.value.has(id);
}

function othersFor(id: number): Rect[] {
  // During multi-drag selected widgets ignore each other as obstacles.
  return props.widgets
    .filter((w) => w.id !== id && !selectedSet.value.has(w.id))
    .map((w) => ({ x: w.x, y: w.y, w: w.w, h: w.h }));
}

function onWidgetUpdate(id: number, rect: Rect): void {
  emit('update', id, rect);
}

// ---- Multi-widget drag ----
// Leader emits grid-snapped deltas; Canvas mirrors transform on other selected frames.
// Commit runs one collision pass and emits updateMany for batchUpdate.

const groupOffset = ref<{ dx: number; dy: number }>({ dx: 0, dy: 0 });
let groupDragLeaderId: number | null = null;
const leaderId = ref<number | null>(null);

// Live rect (in grid units) of the dragging widget, used to render snap
// guides. Re-computed on every onMultiMove tick from the leader widget plus
// the current pixel offset.
const guideActiveRect = computed<Rect | null>(() => {
  const id = leaderId.value;
  if (id === null) return null;
  const w = props.widgets.find((x) => x.id === id);
  if (!w) return null;
  const dx = Math.round(groupOffset.value.dx / props.basePx);
  const dy = Math.round(groupOffset.value.dy / props.basePx);
  return { x: w.x + dx, y: w.y + dy, w: w.w, h: w.h };
});

const guideOtherRects = computed<Rect[]>(() => {
  const id = leaderId.value;
  if (id === null) return [];
  return props.widgets
    .filter((w) => w.id !== id)
    .map((w) => ({ x: w.x, y: w.y, w: w.w, h: w.h }));
});

function applyGroupTransformToFollowers(): void {
  if (!containerRef.value) return;
  // Write the group offset as CSS variables on each selected follower; the
  // WidgetFrame style consumes `--wf-dx / --wf-dy` to translate. This avoids
  // direct el.style.transform mutation (which bypassed Vue reactivity and
  // broke a11y inspection of drag state).
  const followers = containerRef.value.querySelectorAll<HTMLElement>('[data-widget-id]');
  followers.forEach((el) => {
    const id = Number(el.dataset.widgetId);
    if (!Number.isFinite(id) || id === groupDragLeaderId || !selectedSet.value.has(id)) return;
    el.style.setProperty('--wf-dx', `${groupOffset.value.dx}px`);
    el.style.setProperty('--wf-dy', `${groupOffset.value.dy}px`);
  });
}

function onMultiMove(id: number, delta: { dx: number; dy: number }): void {
  if (groupDragLeaderId === null) {
    groupDragLeaderId = id;
    leaderId.value = id;
  }
  if (groupDragLeaderId !== id) return; // Ignore non-leader frames
  groupOffset.value = {
    dx: groupOffset.value.dx + delta.dx,
    dy: groupOffset.value.dy + delta.dy,
  };
  applyGroupTransformToFollowers();
}

function onMultiCommit(id: number): void {
  if (groupDragLeaderId !== id) return;
  const dxUnits = Math.round(groupOffset.value.dx / props.basePx);
  const dyUnits = Math.round(groupOffset.value.dy / props.basePx);

  // Validate move for every selected rect (in-bounds, no hit on unselected widgets).
  const others: Rect[] = props.widgets
    .filter((w) => !selectedSet.value.has(w.id))
    .map((w) => ({ x: w.x, y: w.y, w: w.w, h: w.h }));

  const candidates: { id: number; rect: Rect }[] = [];
  let valid = dxUnits !== 0 || dyUnits !== 0;
  for (const w of props.widgets) {
    if (!selectedSet.value.has(w.id)) continue;
    const rect: Rect = { x: w.x + dxUnits, y: w.y + dyUnits, w: w.w, h: w.h };
    if (
      rect.x < 0 ||
      rect.y < 0 ||
      rect.x + rect.w > DESIGN_GRID_WIDTH ||
      rect.y + rect.h > DESIGN_GRID_HEIGHT
    ) {
      valid = false;
      break;
    }
    for (const o of others) {
      if (rectsOverlap(rect, o)) {
        valid = false;
        break;
      }
    }
    if (!valid) break;
    candidates.push({ id: w.id, rect });
  }

  // Always clear follower transforms before emitting.
  if (containerRef.value) {
    containerRef.value
      .querySelectorAll<HTMLElement>('[data-widget-id]')
      .forEach((el) => {
        const wid = Number(el.dataset.widgetId);
        if (selectedSet.value.has(wid)) {
          el.style.removeProperty('--wf-dx');
          el.style.removeProperty('--wf-dy');
        }
      });
  }
  groupOffset.value = { dx: 0, dy: 0 };
  groupDragLeaderId = null;
  leaderId.value = null;

  if (valid && candidates.length > 0) {
    emit('updateMany', candidates);
  }
}

// ---- Background click / marquee ----

interface MarqueeBox {
  startX: number;
  startY: number;
  curX: number;
  curY: number;
  modifiers: { ctrl: boolean; meta: boolean };
}

const marquee = ref<MarqueeBox | null>(null);

function onBackgroundMouseDown(e: MouseEvent): void {
  if (props.mode !== 'edit') return;
  // Left button only on canvas chrome, not on a widget node.
  if (e.button !== 0) return;
  const target = e.target as HTMLElement | null;
  if (target && target.closest('[data-widget-id]')) return;

  const container = containerRef.value;
  if (!container) return;
  const rect = container.getBoundingClientRect();
  const px = (e.clientX - rect.left) / props.scale;
  const py = (e.clientY - rect.top) / props.scale;
  marquee.value = {
    startX: px,
    startY: py,
    curX: px,
    curY: py,
    modifiers: { ctrl: e.ctrlKey, meta: e.metaKey },
  };
  // Defer clear-selection until mouseup (distinguish click vs drag box).
  e.preventDefault();
}

function onBackgroundMouseMove(e: MouseEvent): void {
  if (!marquee.value || !containerRef.value) return;
  const rect = containerRef.value.getBoundingClientRect();
  marquee.value.curX = (e.clientX - rect.left) / props.scale;
  marquee.value.curY = (e.clientY - rect.top) / props.scale;
}

function onBackgroundMouseUp(e: MouseEvent): void {
  if (!marquee.value) return;
  const m = marquee.value;
  marquee.value = null;
  const dx = Math.abs(m.curX - m.startX);
  const dy = Math.abs(m.curY - m.startY);
  // Tiny movement => treated as blank click -> clear selection.
  if (dx < 4 && dy < 4) {
    emit('select', null, { ctrl: e.ctrlKey, shift: e.shiftKey, meta: e.metaKey });
    return;
  }
  // Else select every widget intersecting the marquee rectangle.
  const x1 = Math.min(m.startX, m.curX);
  const y1 = Math.min(m.startY, m.curY);
  const x2 = Math.max(m.startX, m.curX);
  const y2 = Math.max(m.startY, m.curY);
  const boxRect: Rect = {
    x: x1 / props.basePx,
    y: y1 / props.basePx,
    w: (x2 - x1) / props.basePx,
    h: (y2 - y1) / props.basePx,
  };
  const hits: number[] = [];
  for (const w of props.widgets) {
    if (rectsOverlap(boxRect, w)) hits.push(w.id);
  }
  emit('marquee', hits, m.modifiers);
}

const marqueeStyle = computed<Record<string, string>>(() => {
  if (!marquee.value) return { display: 'none' } as Record<string, string>;
  const m = marquee.value;
  const x1 = Math.min(m.startX, m.curX);
  const y1 = Math.min(m.startY, m.curY);
  const x2 = Math.max(m.startX, m.curX);
  const y2 = Math.max(m.startY, m.curY);
  return {
    left: `${x1}px`,
    top: `${y1}px`,
    width: `${x2 - x1}px`,
    height: `${y2 - y1}px`,
  } as Record<string, string>;
});

// ---- Palette drop target ----

function onDragOver(e: DragEvent): void {
  if (props.mode !== 'edit') return;
  if (!e.dataTransfer) return;
  if (Array.from(e.dataTransfer.types).includes('application/x-astrolabe-widget')) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  }
}

function onDrop(e: DragEvent): void {
  if (props.mode !== 'edit') return;
  e.preventDefault();
  const raw = e.dataTransfer?.getData('application/x-astrolabe-widget');
  if (!raw) return;
  let payload: { type: string; w?: number; h?: number };
  try {
    payload = JSON.parse(raw);
  } catch {
    return;
  }
  const container = containerRef.value;
  if (!container) return;
  const rect = container.getBoundingClientRect();
  const px = (e.clientX - rect.left) / props.scale;
  const py = (e.clientY - rect.top) / props.scale;
  const desired: Rect = {
    x: pxToMultiplier(px, props.basePx),
    y: pxToMultiplier(py, props.basePx),
    w: payload.w ?? 8,
    h: payload.h ?? 4,
  };
  const free = findFreeSpot(
    desired,
    props.widgets.map((w) => ({ x: w.x, y: w.y, w: w.w, h: w.h })),
  );
  emit('drop', { type: payload.type, rect: free });
}
</script>

<template>
  <div
    ref="containerRef"
    class="canvas-root relative"
    :style="{
      width: `${canvasWidthPx}px`,
      height: `${canvasHeightPx}px`,
      transform: scale === 1 ? undefined : `scale(${scale})`,
      transformOrigin: 'top left',
    }"
    @mousedown="onBackgroundMouseDown"
    @mousemove="onBackgroundMouseMove"
    @mouseup="onBackgroundMouseUp"
    @mouseleave="onBackgroundMouseUp"
    @dragover="onDragOver"
    @drop="onDrop"
  >
    <CanvasGrid
      :base-px="basePx"
      :visible="mode === 'edit'"
    />
    <SmartGuides
      v-if="mode === 'edit'"
      :active="guideActiveRect"
      :others="guideOtherRects"
      :base-px="basePx"
    />
    <WidgetFrame
      v-for="w in widgets"
      :key="w.id"
      :widget="w"
      :base-px="basePx"
      :mode="mode"
      :scale="scale"
      :selected="isSelected(w.id)"
      :multi-selected="isSelected(w.id) && selectedIds.length > 1"
      :others="othersFor(w.id)"
      @select="(id, mods) => emit('select', id, mods)"
      @update="(id, rect) => onWidgetUpdate(id, rect)"
      @edit="emit('edit', $event)"
      @delete="emit('delete', $event)"
      @multi-move="onMultiMove"
      @multi-commit="onMultiCommit"
    >
      <slot
        name="widget"
        :widget="w"
      />
    </WidgetFrame>
    <!-- Marquee overlay -->
    <div
      v-if="marquee"
      class="pointer-events-none absolute z-50 rounded-sm border border-[color:var(--astro-accent)] bg-[color:var(--astro-accent)]/10"
      :style="marqueeStyle"
    />
  </div>
</template>
