<script setup lang="ts">
/**
 * Snackbar: bottom-centered transient message.
 *
 * Replaces the in-toolbar error <span class="bg-red-600/30">… pattern so
 * error strings do not eat toolbar real estate. Auto-dismisses and stacks
 * visually at elev-3.
 */
import { computed, onBeforeUnmount, watch } from 'vue';

type Tone = 'error' | 'info' | 'success';

const props = withDefaults(
  defineProps<{
    message: string | null;
    tone?: Tone;
    durationMs?: number;
  }>(),
  { tone: 'error', durationMs: 4500 },
);

const emit = defineEmits<{ dismiss: [] }>();

let timer: ReturnType<typeof setTimeout> | null = null;

watch(
  () => props.message,
  (msg) => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (!msg) return;
    timer = setTimeout(() => emit('dismiss'), props.durationMs);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer);
});

const toneClass = computed(() => {
  switch (props.tone) {
    case 'success':
      return 'border-emerald-500/40 text-emerald-100 bg-emerald-900/40';
    case 'info':
      return 'border-slate-400/40 text-slate-100 bg-slate-900/60';
    case 'error':
    default:
      return 'border-red-500/45 text-red-100 bg-red-900/60';
  }
});
</script>

<template>
  <Transition name="snackbar">
    <div
      v-if="message"
      role="status"
      class="astro-glass fixed bottom-6 left-1/2 z-[60] -translate-x-1/2 max-w-[min(640px,90vw)]"
      :class="['astro-snackbar border px-4 py-2.5 text-[length:var(--fs-sm)]', toneClass]"
      :style="{ boxShadow: 'var(--elev-3)' }"
    >
      {{ message }}
    </div>
  </Transition>
</template>

<style scoped>
.snackbar-enter-active,
.snackbar-leave-active {
  transition:
    opacity 0.22s ease,
    transform 0.22s ease;
}
.snackbar-enter-from,
.snackbar-leave-to {
  opacity: 0;
  transform: translate(-50%, 16px);
}
</style>
