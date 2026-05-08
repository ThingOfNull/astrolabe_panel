import { computed, ref, type Ref } from 'vue';

import type { Widget } from '@/canvas/types';

/**
 * In-memory clipboard for widget copy/paste.
 *
 * Deliberately SPA-scoped: persisting to localStorage would leak stale
 * widget ids across page reloads. Paste re-lays rectangles via a provided
 * findFreeSpot() so duplicated widgets land on blank canvas cells.
 */

export type ClipSnapshot = Pick<
  Widget,
  | 'type'
  | 'x'
  | 'y'
  | 'w'
  | 'h'
  | 'z_index'
  | 'icon_type'
  | 'icon_value'
  | 'data_source_id'
  | 'metric_query'
  | 'config'
>;

export function useClipboard() {
  const items: Ref<ClipSnapshot[]> = ref([]);
  const count = computed(() => items.value.length);

  function copy(widgets: Widget[]): void {
    items.value = widgets.map((w) => ({
      type: w.type,
      x: w.x,
      y: w.y,
      w: w.w,
      h: w.h,
      z_index: w.z_index,
      icon_type: w.icon_type,
      icon_value: w.icon_value,
      data_source_id: w.data_source_id,
      // Deep-clone the JSON payloads so subsequent edits to the original
      // widget config don't mutate clipboard snapshots through shared refs.
      metric_query: w.metric_query == null ? null : cloneJson(w.metric_query),
      config: w.config == null ? {} : (cloneJson(w.config) ?? {}),
    }));
  }

  function clear(): void {
    items.value = [];
  }

  return { items, count, copy, clear };
}

function cloneJson<T>(v: T | null | undefined): T | null {
  if (v == null) return null;
  return JSON.parse(JSON.stringify(v)) as T;
}
