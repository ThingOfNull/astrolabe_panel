<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  basePx: number;
  visible: boolean;
}>();

const styleVars = computed(() => ({
  backgroundImage: 'radial-gradient(circle, rgba(255,255,255,0.18) 1px, transparent 1px)',
  backgroundSize: `${props.basePx}px ${props.basePx}px`,
  backgroundPosition: '0 0',
}));
</script>

<template>
  <div
    v-if="visible"
    class="pointer-events-none absolute inset-0"
    :style="styleVars"
    aria-hidden="true"
  />
</template>
