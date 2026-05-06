<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import type { Widget } from '@/canvas/types';

import ClockFlipDigit from './ClockFlipDigit.vue';
import { defaultClockWidgetConfig, type ClockWidgetConfig } from './types';

const props = defineProps<{
  widget: Widget;
}>();

const DEFAULT_W = 22;
const DEFAULT_H = 10;
const scale = computed(() => {
  const rw = props.widget.w / DEFAULT_W;
  const rh = props.widget.h / DEFAULT_H;
  return Math.max(0.75, Math.min(rw, rh));
});

const cfg = computed<ClockWidgetConfig>(() => {
  const c = (props.widget.config ?? {}) as Partial<ClockWidgetConfig>;
  const base = defaultClockWidgetConfig();
  const variant = c.variant === 'flip' ? 'flip' : 'digital';
  return {
    ...base,
    ...c,
    variant,
    show_seconds: c.show_seconds !== false,
    show_date: c.show_date !== false,
    use_24h: c.use_24h !== false,
    timezone: typeof c.timezone === 'string' ? c.timezone : '',
  };
});

const now = ref(new Date());
let timer: number | null = null;

function tick(): void {
  now.value = new Date();
}

onMounted(() => {
  tick();
  timer = window.setInterval(tick, 1000);
});

onBeforeUnmount(() => {
  if (timer !== null) window.clearInterval(timer);
});

watch(
  () => [cfg.value.timezone, cfg.value.use_24h, cfg.value.show_seconds],
  () => tick(),
);

interface TimeParts {
  h: string;
  m: string;
  s: string;
  ampms: '' | 'AM' | 'PM';
  dateStr: string;
}

function formatParts(date: Date, c: ClockWidgetConfig): TimeParts {
  const tzRaw = (c.timezone ?? '').trim();
  const tz = tzRaw !== '' ? tzRaw : undefined;
  const use24 = c.use_24h !== false;
  const showSec = c.show_seconds !== false;

  const tryOnce = (
    tzOpt: string | undefined,
  ): Omit<TimeParts, 'dateStr'> & { dateLine: string } => {
    const hourCycle = use24 ? 'h23' : 'h12';
    const fmt = new Intl.DateTimeFormat('en-GB', {
      hour: '2-digit',
      minute: '2-digit',
      ...(showSec ? { second: '2-digit' as const } : {}),
      hourCycle,
      ...(tzOpt !== undefined ? { timeZone: tzOpt } : {}),
    });
    const partsIn = fmt.formatToParts(date);
    const v = (t: Intl.DateTimeFormatPartTypes) =>
      partsIn.find((p) => p.type === t)?.value ?? '';
    let h = v('hour');
    const m = v('minute').padStart(2, '0');
    let s = showSec ? v('second').padStart(2, '0') : '';
    if (!/^\d{1,2}$/.test(h)) {
      h = '00';
    }
    h = h.padStart(2, '0');

    let ampms: TimeParts['ampms'] = '';
    if (!use24) {
      const ap = v('dayPeriod').toUpperCase();
      ampms = ap === 'AM' || ap === 'PM' ? ap : '';
    }

    let dateLine = '--';
    if (c.show_date !== false) {
      try {
        const df = new Intl.DateTimeFormat('zh-CN', {
          weekday: 'short',
          year: 'numeric',
          month: 'numeric',
          day: 'numeric',
          ...(tzOpt !== undefined ? { timeZone: tzOpt } : {}),
        });
        dateLine = df.format(date);
      } catch {
        dateLine = '--';
      }
    }

    return { h, m, s, ampms, dateLine };
  };

  try {
    const r = tryOnce(tz);
    return {
      h: r.h,
      m: r.m,
      s: r.s,
      ampms: r.ampms,
      dateStr: c.show_date === false ? '' : r.dateLine,
    };
  } catch {
    const r = tryOnce(undefined);
    return {
      h: r.h,
      m: r.m,
      s: r.s,
      ampms: r.ampms,
      dateStr: c.show_date === false ? '' : r.dateLine,
    };
  }
}

const parts = computed(() => formatParts(now.value, cfg.value));

const digitalMain = computed(() => {
  const p = parts.value;
  const sec = cfg.value.show_seconds !== false ? `:${p.s}` : '';
  return `${p.h}:${p.m}${sec}`;
});

const tzHint = computed(() => {
  const t = (cfg.value.timezone ?? '').trim();
  return t !== '' ? t : '本地';
});

const flipDigitHeight = computed(() => Math.round(44 * scale.value));
const sepSize = computed(() => `${Math.round(28 * scale.value)}px`);

const flipSlots = computed(() => {
  const p = parts.value;
  const showSec = cfg.value.show_seconds !== false;
  const slots: (
    | { kind: 'd'; ch: string }
    | { kind: 'sep' }
    | { kind: 'ampm'; text: string }
  )[] = [];
  for (const ch of p.h) {
    slots.push({ kind: 'd', ch });
  }
  slots.push({ kind: 'sep' });
  for (const ch of p.m) {
    slots.push({ kind: 'd', ch });
  }
  if (showSec && p.s) {
    slots.push({ kind: 'sep' });
    for (const ch of p.s) {
      slots.push({ kind: 'd', ch });
    }
  }
  if (!cfg.value.use_24h && p.ampms) {
    slots.push({ kind: 'ampm', text: p.ampms });
  }
  return slots;
});

const digitFont = computed(() => `${Math.round(42 * scale.value)}px`);

const accentGlow = computed(
  (): Record<string, string> => ({
    textShadow:
      '0 0 24px color-mix(in srgb, var(--astro-accent) 45%, transparent), 0 1px 0 rgba(0,0,0,0.35)',
  }),
);
</script>

<template>
  <div
    class="relative flex h-full min-h-0 min-w-0 flex-col overflow-hidden rounded-[inherit] text-[color:var(--astro-text-primary)]"
    :style="{ padding: `${Math.round(12 * scale)}px ${Math.round(14 * scale)}px` }"
  >
    <div
      class="pointer-events-none absolute left-1/2 top-[12%] h-[42%] w-[72%] -translate-x-1/2 rounded-full opacity-[0.12]"
      style="background: radial-gradient(circle at 50% 30%, var(--astro-accent), transparent 72%)"
    />

    <div
      v-if="cfg.show_date !== false && parts.dateStr"
      class="relative z-[1] mb-1 text-center font-medium text-[color:var(--astro-text-secondary)]"
      :style="{ fontSize: `${Math.round(12 * scale)}px` }"
    >
      {{ parts.dateStr }}
    </div>

    <div
      v-if="cfg.variant === 'digital'"
      class="relative z-[1] flex flex-1 flex-col items-center justify-center gap-1"
    >
      <div
        class="font-mono tabular-nums tracking-tight text-[color:var(--astro-text-primary)]"
        :style="{ fontSize: digitFont, ...accentGlow }"
      >
        {{ digitalMain }}
      </div>
      <div
        v-if="!cfg.use_24h && parts.ampms"
        class="text-[color:var(--astro-accent)]"
        :style="{ fontSize: `${Math.round(14 * scale)}px`, letterSpacing: '0.2em' }"
      >
        {{ parts.ampms }}
      </div>
      <div
        class="text-[color:var(--astro-text-secondary)] opacity-85"
        :style="{ fontSize: `${Math.round(10 * scale)}px` }"
      >
        {{ tzHint }}
      </div>
    </div>

    <div
      v-else
      class="relative z-[1] flex flex-1 flex-col items-center justify-center gap-2"
    >
      <div class="flex flex-wrap items-center justify-center gap-[0.08em]">
        <template
          v-for="(slot, i) in flipSlots"
          :key="i"
        >
          <ClockFlipDigit
            v-if="slot.kind === 'd'"
            :digit="slot.ch"
            :height-px="flipDigitHeight"
          />
          <span
            v-else-if="slot.kind === 'sep'"
            class="mx-px font-light tabular-nums text-[color:var(--astro-text-primary)] animate-pulse"
            :style="{ fontSize: sepSize, paddingBottom: '0.06em', lineHeight: '1' }"
          >
            :
          </span>
          <span
            v-else
            class="ml-2 font-semibold tracking-wider text-[color:var(--astro-accent)]"
            :style="{ fontSize: `${Math.round(flipDigitHeight * 0.36)}px` }"
          >
            {{ slot.text }}
          </span>
        </template>
      </div>
      <div
        class="text-[color:var(--astro-text-secondary)] opacity-85"
        :style="{ fontSize: `${Math.round(10 * scale)}px` }"
      >
        {{ tzHint }}
      </div>
    </div>
  </div>
</template>
