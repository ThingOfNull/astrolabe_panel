<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  /** Single digit 0-9 */
  digit: string;
  /** Cell height in px */
  heightPx: number;
}>();

const safe = computed(() => (/^\d$/.test(props.digit) ? props.digit : '0'));

const boxStyle = computed((): Record<string, string> => {
  const h = Math.max(12, props.heightPx);
  const w = Math.round(h * 0.58);
  const fs = Math.round(h * 0.52);
  return {
    width: `${w}px`,
    height: `${h}px`,
    fontSize: `${fs}px`,
  };
});
</script>

<template>
  <div
    class="clock-flip-digit rounded-md border border-[color:var(--astro-glass-border)] bg-gradient-to-b from-[color:var(--astro-glass-bg)] to-[color:var(--astro-bg-base)] shadow-[inset_0_1px_0_rgba(255,255,255,0.06)]"
    :style="boxStyle"
  >
    <div class="digit-3d relative h-full w-full overflow-hidden rounded-[inherit]">
      <Transition
        name="clock-flip"
        mode="out-in"
      >
        <span
          :key="safe"
          class="flex h-full w-full items-center justify-center font-semibold tabular-nums leading-none text-[color:var(--astro-text-primary)]"
        >
          {{ safe }}
        </span>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.digit-3d {
  perspective: 3.2em;
}

.clock-flip-enter-active,
.clock-flip-leave-active {
  transition:
    transform 0.38s cubic-bezier(0.34, 1.15, 0.64, 1),
    opacity 0.28s ease;
}

.clock-flip-enter-from {
  transform: rotateX(-86deg) scale(0.92);
  opacity: 0.25;
}

.clock-flip-leave-to {
  transform: rotateX(86deg) scale(0.92);
  opacity: 0.15;
}
</style>
