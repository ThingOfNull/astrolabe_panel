<script setup lang="ts">
/**
 * StatusPill: tiny info capsule used in toolbars and status bars.
 * Variants map to the three-accent token system so selection / warning /
 * info states stay semantically distinct.
 */
type Tone = 'info' | 'accent' | 'warn' | 'danger' | 'neutral';

withDefaults(
  defineProps<{
    tone?: Tone;
    icon?: string; // iconify name, optional
  }>(),
  { tone: 'neutral' },
);
</script>

<template>
  <span
    :data-tone="tone"
    class="astro-pill inline-flex items-center gap-1 rounded-[var(--radius-pill)] border px-2 py-0.5 text-[length:var(--fs-xs)] font-medium"
  >
    <slot />
  </span>
</template>

<style scoped>
.astro-pill {
  border-color: color-mix(in srgb, var(--astro-text-secondary) 25%, transparent);
  color: var(--astro-text-secondary);
  background: color-mix(in srgb, var(--astro-text-secondary) 10%, transparent);
}

.astro-pill[data-tone='accent'] {
  color: var(--astro-accent-brand);
  border-color: color-mix(in srgb, var(--astro-accent-brand) 40%, transparent);
  background: color-mix(in srgb, var(--astro-accent-brand) 12%, transparent);
}

.astro-pill[data-tone='info'] {
  color: var(--astro-text-primary);
  border-color: color-mix(in srgb, var(--astro-text-primary) 30%, transparent);
  background: color-mix(in srgb, var(--astro-text-primary) 6%, transparent);
}

.astro-pill[data-tone='warn'] {
  color: #fbbf24;
  border-color: color-mix(in srgb, #fbbf24 40%, transparent);
  background: color-mix(in srgb, #fbbf24 12%, transparent);
}

.astro-pill[data-tone='danger'] {
  color: var(--astro-status-err);
  border-color: color-mix(in srgb, var(--astro-status-err) 45%, transparent);
  background: color-mix(in srgb, var(--astro-status-err) 14%, transparent);
}
</style>
