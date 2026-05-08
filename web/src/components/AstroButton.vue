<script setup lang="ts">
/**
 * AstroButton: token-backed button atom.
 *
 * The previous codebase had ~5 ad-hoc button styles; this component collapses
 * them into a small, legible matrix (variant × size) so every page follows
 * the same radius / shadow / typography tokens.
 */
import { computed } from 'vue';

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'icon';
type Size = 'sm' | 'md';

const props = withDefaults(
  defineProps<{
    variant?: Variant;
    size?: Size;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    loading?: boolean;
    title?: string;
  }>(),
  { variant: 'secondary', size: 'md', type: 'button', disabled: false, loading: false },
);

defineEmits<{ click: [event: MouseEvent] }>();

const classes = computed(() => {
  const base = [
    'astro-btn-icon',
    'inline-flex items-center justify-center gap-1.5',
    'font-medium select-none outline-none',
    'focus-visible:ring-2 focus-visible:ring-[color:var(--astro-accent-interactive)]',
    'disabled:cursor-not-allowed disabled:opacity-55',
  ];

  // Size → radius from tokens.
  const sizeMap: Record<Size, string[]> = {
    sm: ['px-2.5 py-1', 'text-[length:var(--fs-xs)]', 'rounded-[var(--radius-sm)]'],
    md: ['px-3.5 py-1.5', 'text-[length:var(--fs-sm)]', 'rounded-[var(--radius-sm)]'],
  };

  const variantMap: Record<Variant, string[]> = {
    primary: [
      'bg-[color:var(--astro-accent-interactive)]',
      'text-[color:var(--astro-on-accent-text)]',
      'hover:brightness-110 active:brightness-95',
      'shadow-[var(--elev-1)]',
    ],
    secondary: [
      'border border-[color:var(--astro-glass-border)]',
      'hover:bg-white/5 hover:border-[color:var(--astro-accent-interactive)]/40',
    ],
    ghost: ['text-[color:var(--astro-text-secondary)]', 'hover:text-[color:var(--astro-text-primary)]', 'hover:bg-white/5'],
    danger: [
      'bg-red-600/70 text-white hover:bg-red-600',
      'shadow-[var(--elev-1)]',
    ],
    icon: [
      'p-2 aspect-square',
      'border border-[color:var(--astro-glass-border)]',
      'hover:bg-white/5 hover:border-[color:var(--astro-accent-interactive)]/40',
    ],
  };

  return [...base, ...sizeMap[props.size], ...variantMap[props.variant]].join(' ');
});
</script>

<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
    :title="title"
    :aria-busy="loading || undefined"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>
