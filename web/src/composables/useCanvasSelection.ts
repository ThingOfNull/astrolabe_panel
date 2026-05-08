import { computed, ref, type Ref } from 'vue';

/**
 * Selection state for the editor canvas.
 *
 * - selectedIds: multi-select, insertion-ordered (stable for keyboard ops)
 * - lastAnchorId: the last click anchor for Shift-range extension
 *
 * Consumers pass `getOrderedWidgetIds()` so range selection matches the
 * canvas's paint/list order (usually y→x).
 */
export function useCanvasSelection(getOrderedIds: () => number[]) {
  const selectedIds: Ref<number[]> = ref([]);
  const lastAnchorId: Ref<number | null> = ref(null);

  const selectedSet = computed(() => new Set(selectedIds.value));
  const count = computed(() => selectedIds.value.length);

  function select(
    id: number | null,
    mods: { ctrl: boolean; shift: boolean; meta: boolean },
  ): void {
    if (id === null) {
      selectedIds.value = [];
      lastAnchorId.value = null;
      return;
    }
    const ctrlOrMeta = mods.ctrl || mods.meta;
    if (mods.shift && lastAnchorId.value !== null && lastAnchorId.value !== id) {
      const ordered = getOrderedIds();
      const from = ordered.indexOf(lastAnchorId.value);
      const to = ordered.indexOf(id);
      if (from >= 0 && to >= 0) {
        const [a, b] = from < to ? [from, to] : [to, from];
        const range = ordered.slice(a, b + 1);
        selectedIds.value = ctrlOrMeta
          ? unique([...selectedIds.value, ...range])
          : range;
        return;
      }
    }
    if (ctrlOrMeta) {
      if (selectedSet.value.has(id)) {
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
      } else {
        selectedIds.value = [...selectedIds.value, id];
      }
    } else {
      selectedIds.value = [id];
    }
    lastAnchorId.value = id;
  }

  function setFromMarquee(
    ids: number[],
    mods: { ctrl: boolean; meta: boolean },
  ): void {
    if (mods.ctrl || mods.meta) {
      selectedIds.value = unique([...selectedIds.value, ...ids]);
    } else {
      selectedIds.value = ids;
    }
    lastAnchorId.value = ids[ids.length - 1] ?? null;
  }

  function clear(): void {
    selectedIds.value = [];
    lastAnchorId.value = null;
  }

  function selectAll(): void {
    selectedIds.value = getOrderedIds();
    lastAnchorId.value = selectedIds.value[selectedIds.value.length - 1] ?? null;
  }

  function has(id: number): boolean {
    return selectedSet.value.has(id);
  }

  return {
    selectedIds,
    lastAnchorId,
    selectedSet,
    count,
    select,
    setFromMarquee,
    clear,
    selectAll,
    has,
  };
}

function unique<T>(xs: T[]): T[] {
  return Array.from(new Set(xs));
}
