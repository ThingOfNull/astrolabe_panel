import { computed, onBeforeUnmount, ref, type Ref } from 'vue';

/**
 * useStageScale
 *
 * Computes a scale factor that fits a fixed design canvas (widthPx × heightPx)
 * inside a container element measured by ResizeObserver. Replaces the old
 * useViewportMode.ts which hardcoded padding=48 and only listened to window
 * resize — that broke whenever the surrounding layout changed.
 *
 * Consumer contract:
 *   const stageRef = ref<HTMLElement | null>(null);
 *   const { scale, mode } = useStageScale(stageRef, designSize);
 *
 * `mode` is 'compact' whenever the scaled result would be too small to keep
 * widgets legible; callers typically switch to a vertical stack in that case.
 */

export interface DesignSize {
  widthPx: number;
  heightPx: number;
}

export interface StageScale {
  scale: Ref<number>;
  mode: Ref<'canvas' | 'compact'>;
}

export function useStageScale(
  container: Ref<HTMLElement | null>,
  design: () => DesignSize,
  options: { minScale?: number; compactBelow?: number } = {},
): StageScale {
  const minScale = options.minScale ?? 0.25;
  const compactBelow = options.compactBelow ?? 0.42;
  const scale = ref(1);
  const mode = ref<'canvas' | 'compact'>('canvas');

  let observer: ResizeObserver | null = null;

  function recompute(): void {
    const el = container.value;
    if (!el) return;
    const box = el.getBoundingClientRect();
    if (box.width <= 0 || box.height <= 0) return;
    const d = design();
    if (d.widthPx <= 0 || d.heightPx <= 0) return;
    const sx = box.width / d.widthPx;
    const sy = box.height / d.heightPx;
    const raw = Math.min(sx, sy);
    const next = Math.max(minScale, Math.min(raw, 1));
    scale.value = next;
    mode.value = next < compactBelow ? 'compact' : 'canvas';
  }

  // Manual setup/teardown via a computed so the caller can pass a ref that is
  // only populated after mount.
  const dispose = computed(() => {
    const el = container.value;
    if (!el) return () => {};
    observer?.disconnect();
    observer = new ResizeObserver(() => recompute());
    observer.observe(el);
    recompute();
    return () => {
      observer?.disconnect();
      observer = null;
    };
  });

  // Trigger the computed once to start observing as soon as ref resolves.
  // Vue's reactivity evaluates the computed lazily; touch it now and again
  // in onBeforeUnmount to clean up.
  void dispose.value;

  onBeforeUnmount(() => {
    observer?.disconnect();
    observer = null;
  });

  return { scale, mode };
}
