/**
 * Per-widget responsive strategy.
 *
 * Decides what should happen to a widget when the viewport is too narrow to
 * render the canonical canvas. Three modes:
 *
 *   stack  → keep the widget; render it in the vertical compact feed
 *   shrink → keep the widget; let the chart engine shrink without stacking
 *   hide   → hide the widget (decoration / oversized panels)
 *
 * The strategy is derived from widget.type rather than persisted on the
 * widget so existing boards Just Work; if a future need calls for per-widget
 * overrides the same function reads `widget.config.responsive` first.
 */

import type { Widget } from '@/canvas/types';

export type ResponsiveMode = 'stack' | 'shrink' | 'hide';

const DEFAULT_BY_TYPE: Record<string, ResponsiveMode> = {
  // Decorations: hide on phones.
  divider: 'hide',
  text: 'hide',
  // Mid-density panels render fine in stack mode.
  link: 'stack',
  search: 'stack',
  bignumber: 'stack',
  weather: 'stack',
  clock: 'stack',
  // Heavier visualizations need their full footprint to make sense.
  gauge: 'shrink',
  line: 'shrink',
  bar: 'shrink',
  grid: 'shrink',
  liquid: 'shrink',
  sparkline: 'stack',
  bullet: 'stack',
  progress_ring: 'stack',
  heatmap: 'shrink',
  timeline: 'shrink',
  radial3d: 'shrink',
};

export function responsiveModeOf(w: Widget): ResponsiveMode {
  const cfg = (w.config ?? {}) as { responsive?: ResponsiveMode };
  if (cfg.responsive === 'stack' || cfg.responsive === 'shrink' || cfg.responsive === 'hide') {
    return cfg.responsive;
  }
  return DEFAULT_BY_TYPE[w.type] ?? 'stack';
}
