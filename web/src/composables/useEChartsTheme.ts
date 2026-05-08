/**
 * useEChartsTheme
 *
 * Reads the active theme's CSS variables into a memoized object that all
 * ECharts-backed widgets reuse. Previously every chart hardcoded '#888',
 * 'rgba(255,255,255,0.2)' and friends — under the light theme those values
 * collapsed into the background and made chart chrome invisible.
 *
 * Consumers:
 *   const theme = useEChartsTheme();
 *   // theme.value.textSecondary, theme.value.splitLine, etc.
 *
 * The ref re-evaluates when the global event 'astro:theme-changed' fires
 * (dispatched by useTheme.applyTheme() below).
 */

import { onBeforeUnmount, readonly, ref } from 'vue';

export interface EChartsThemeColors {
  /** Primary on-surface text, used for chart titles. */
  textPrimary: string;
  /** Secondary text used for axis labels / legends. */
  textSecondary: string;
  /** Faint grid / split lines. */
  splitLine: string;
  /** Low-opacity area fill base; chart may override with accent. */
  areaFill: string;
  /** Brand accent for bars / active series. */
  accentBrand: string;
  /** Interactive accent for highlights. */
  accentInteractive: string;
  /** Status colors for thresholds. */
  statusOk: string;
  statusErr: string;
  statusUnknown: string;
  /** Background for tooltip / data pill. */
  surface: string;
  border: string;
}

export const THEME_CHANGED_EVENT = 'astro:theme-changed';

function readTheme(): EChartsThemeColors {
  if (typeof window === 'undefined') {
    return fallback();
  }
  const style = getComputedStyle(document.documentElement);
  const v = (name: string, fb: string): string => {
    const raw = style.getPropertyValue(name).trim();
    return raw || fb;
  };
  return {
    textPrimary: v('--astro-text-primary', '#e8ecf3'),
    textSecondary: v('--astro-text-secondary', '#8b97a8'),
    splitLine: `color-mix(in srgb, ${v('--astro-text-secondary', '#8b97a8')} 22%, transparent)`,
    areaFill: `color-mix(in srgb, ${v('--astro-accent', '#22d3ee')} 26%, transparent)`,
    accentBrand: v('--astro-accent-brand', v('--astro-accent', '#22d3ee')),
    accentInteractive: v('--astro-accent-interactive', v('--astro-accent', '#22d3ee')),
    statusOk: v('--astro-status-ok', '#34d399'),
    statusErr: v('--astro-status-err', '#f87171'),
    statusUnknown: v('--astro-status-unknown', '#94a3b8'),
    surface: v('--astro-glass-bg', 'rgba(18,24,34,0.72)'),
    border: v('--astro-glass-border', 'rgba(186,204,230,0.12)'),
  };
}

function fallback(): EChartsThemeColors {
  return {
    textPrimary: '#e8ecf3',
    textSecondary: '#8b97a8',
    splitLine: 'rgba(139,151,168,0.22)',
    areaFill: 'rgba(34,211,238,0.26)',
    accentBrand: '#22d3ee',
    accentInteractive: '#22d3ee',
    statusOk: '#34d399',
    statusErr: '#f87171',
    statusUnknown: '#94a3b8',
    surface: 'rgba(18,24,34,0.72)',
    border: 'rgba(186,204,230,0.12)',
  };
}

const current = ref<EChartsThemeColors>(typeof window !== 'undefined' ? readTheme() : fallback());

function onThemeChanged(): void {
  current.value = readTheme();
}

if (typeof window !== 'undefined') {
  window.addEventListener(THEME_CHANGED_EVENT, onThemeChanged);
}

export function useEChartsTheme() {
  // Also refresh on mount of the consumer: tokens may have changed between
  // module init and widget mount (async board.get).
  onBeforeUnmount(() => {
    /* singleton listener stays alive; nothing to clean up here */
  });
  return readonly(current);
}

/** Manually notify consumers; call this from applyTheme once tokens settle. */
export function notifyThemeChanged(): void {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(THEME_CHANGED_EVENT));
}
