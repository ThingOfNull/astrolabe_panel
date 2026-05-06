<script setup lang="ts">
import { Icon } from '@iconify/vue';

import { palette } from '@/widgets/registry';

function onDragStart(e: DragEvent, type: string, w: number, h: number): void {
  if (!e.dataTransfer) return;
  const payload = JSON.stringify({ type, w, h });
  e.dataTransfer.setData('application/x-astrolabe-widget', payload);
  e.dataTransfer.effectAllowed = 'copy';
}
</script>

<template>
  <div class="space-y-3">
    <p class="text-xs text-[color:var(--astro-text-secondary)]">
      把组件拖到左侧画布上即可放置。
    </p>
    <ul class="space-y-2">
      <li
        v-for="entry in palette"
        :key="entry.type"
        class="astro-btn-icon flex flex-1 cursor-grab items-start gap-3 rounded-md border border-[color:var(--astro-glass-border)] p-3 hover:border-[color:var(--astro-accent)]/35 hover:bg-white/5 hover:shadow-md active:cursor-grabbing motion-reduce:hover:shadow-none motion-reduce:hover:border-[color:var(--astro-glass-border)]"
        draggable="true"
        @dragstart="onDragStart($event, entry.type, entry.defaultW, entry.defaultH)"
      >
        <Icon
          :icon="entry.icon"
          width="24"
          height="24"
          class="mt-1"
        />
        <div class="flex-1">
          <div class="text-sm font-medium">
            {{ entry.label }}
          </div>
          <div class="text-xs text-[color:var(--astro-text-secondary)]">
            {{ entry.description }}
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>
