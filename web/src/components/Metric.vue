<script setup lang="ts">
/**
 * Metric: unified number typesetting.
 *
 * Replaces the scattered `<span class="astro-mono-num">{{ v.toFixed(1) }}</span>
 * <span>{{ unit }}</span>` pairs with one component that:
 *  - Applies tabular, slashed-zero figures (Inter + JetBrains Mono fallback)
 *  - Optionally thousands-separates the integer portion
 *  - Keeps unit visually subordinate (smaller, secondary color, baseline aligned)
 *  - Exposes a single "size" token so a widget's hero number stays consistent
 *    across BigNumber / Gauge detail / Weather temp / …
 */
import { computed } from 'vue';

type Size = 'sm' | 'md' | 'lg' | 'xl' | 'display';

const props = withDefaults(
  defineProps<{
    value: number | null | undefined;
    unit?: string;
    /** Digits after the decimal. Defaults to whatever value already has. */
    precision?: number;
    /** Thousands separator; default on for integer parts ≥ 4 digits. */
    thousands?: boolean;
    size?: Size;
    /** Override color; defaults to primary text. */
    color?: string;
  }>(),
  { thousands: true, size: 'lg' },
);

const sizePx = computed<string>(() => {
  const map: Record<Size, string> = {
    sm: 'var(--fs-sm)',
    md: 'var(--fs-base)',
    lg: 'var(--fs-xl)',
    xl: 'var(--fs-display)',
    display: 'calc(var(--fs-display) * 1.3)',
  };
  return map[props.size];
});

const formatted = computed(() => {
  const v = props.value;
  if (v == null || !Number.isFinite(v)) return '—';
  const d = props.precision ?? guessPrecision(v);
  // Intl handles thousands grouping consistently across locales.
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: d,
    maximumFractionDigits: d,
    useGrouping: props.thousands,
  }).format(v);
});

function guessPrecision(v: number): number {
  if (Number.isInteger(v)) return 0;
  const s = v.toString();
  const dot = s.indexOf('.');
  return dot === -1 ? 0 : Math.min(2, s.length - dot - 1);
}
</script>

<template>
  <span
    class="astro-metric astro-mono-num inline-flex items-baseline"
    :style="{
      fontSize: sizePx,
      color: color ?? 'var(--astro-text-primary)',
      lineHeight: 'var(--lh-tight)',
    }"
  >
    <span class="astro-metric__value">{{ formatted }}</span>
    <span
      v-if="unit"
      class="astro-metric__unit ml-1 text-[length:var(--fs-sm)] font-medium text-[color:var(--astro-text-secondary)]"
    >
      {{ unit }}
    </span>
  </span>
</template>
