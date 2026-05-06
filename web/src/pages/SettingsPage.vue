<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import Canvas from '@/canvas/Canvas.vue';
import type { Rect, Widget } from '@/canvas/types';
import { DESIGN_GRID_HEIGHT, DESIGN_GRID_WIDTH } from '@/canvas/types';
import { clampToCanvas, findFreeSpot } from '@/canvas/geometry';
import { useBoardStore, type Board } from '@/stores/board';
import { useDataSourceStore } from '@/stores/datasources';
import { useWidgetStore } from '@/stores/widgets';
import {
  defaultCustomVars,
  type CustomThemeVars,
  type GlassTintMode,
} from '@/composables/useTheme';
import {
  ACCEPTED_SHAPES,
  defaultBarConfig,
  defaultBigNumberConfig,
  defaultDividerWidgetConfig,
  defaultGaugeConfig,
  defaultLineConfig,
  defaultLinkConfig,
  defaultSearchConfig,
  defaultStatusGridConfig,
  defaultTextWidgetConfig,
  defaultWeatherWidgetConfig,
  defaultClockWidgetConfig,
} from '@/widgets/types';
import type { PaletteType } from '@/widgets/registry';
import type { Shape } from '@/api/types';
import { downloadConfigBundleFile, httpImportConfigFile } from '@/api/httpConfig';
import { httpUpload } from '@/api/httpUpload';
import { getRpc } from '@/api/jsonrpc';
import ThemedFileTrigger from '@/components/ThemedFileTrigger.vue';
import { type SupportedLocale, setLocale } from '@/i18n';
import WidgetRenderer from '@/widgets/WidgetRenderer.vue';

import DataSourceTab from './settings/DataSourceTab.vue';
import WidgetEditModal from './settings/WidgetEditModal.vue';
import WidgetPalette from './settings/WidgetPalette.vue';

const { t, locale } = useI18n();
const router = useRouter();
const boardStore = useBoardStore();
const widgetStore = useWidgetStore();
const dsStore = useDataSourceStore();

type Tab = 'palette' | 'datasource' | 'board' | 'system';
const tab = ref<Tab>('palette');
const scale = ref(0.65);

/** Must match server upload.MaxWallpaperBytes. */
const WALLPAPER_UPLOAD_MAX_BYTES = 32 * 1024 * 1024;

// Multi-select ids in insertion order (Shift-range uses last anchor).
const selectedIds = ref<number[]>([]);
const lastAnchorId = ref<number | null>(null);

// In-memory clipboard for this SPA session only; JSON-safe widget fields.
type ClipSnapshot = Pick<
  Widget,
  | 'type'
  | 'x'
  | 'y'
  | 'w'
  | 'h'
  | 'z_index'
  | 'icon_type'
  | 'icon_value'
  | 'data_source_id'
  | 'metric_query'
  | 'config'
>;
const clipboard = ref<ClipSnapshot[]>([]);

const editModalOpen = ref(false);
const editTarget = ref<Widget | null>(null);
const submitting = ref(false);
const errorMsg = ref<string | null>(null);

type BoardThemeDraft = 'dark' | 'light' | 'custom' | 'custom_image';

const boardDraftName = ref('');
const boardDraftGrid = ref(10);
const boardDraftTheme = ref<BoardThemeDraft>('dark');
const boardDraftCustom = ref<CustomThemeVars>(defaultCustomVars());
const boardDraftWallpaper = ref('');
const wallpaperBusy = ref(false);
/** Wallpaper upload outcome shown in the system sidebar (visible when canvas toolbar is cramped). */
const wallpaperFeedback = ref<{ kind: 'ok' | 'err'; text: string } | null>(null);
const exportInFlight = ref(false);
const importSummary = ref<{
  board_updated: boolean;
  data_sources_added: number;
  widgets_added: number;
  widgets_skipped?: number[];
} | null>(null);

const THEME_CUSTOM_KEYS = [
  'bg_base',
  'glass_bg',
  'glass_border',
  'text_primary',
  'text_secondary',
  'status_ok',
  'status_err',
  'status_unknown',
  'accent',
] as const;

const GLASS_TINT_MODES: GlassTintMode[] = ['auto', 'manual'];

const customThemeFields = computed(() =>
  THEME_CUSTOM_KEYS.map((key) => ({
    key,
    label: t(`themeFields.${key}`),
  })),
);

const glassTintMode = computed<GlassTintMode>({
  get: () => (boardDraftCustom.value.glass_tint_mode === 'manual' ? 'manual' : 'auto'),
  set: (value) => {
    boardDraftCustom.value = {
      ...boardDraftCustom.value,
      glass_tint_mode: value,
    };
  },
});

const tabOptions = computed(() =>
  ([
    ['palette', 'settings.tabPalette'],
    ['datasource', 'settings.tabDatasource'],
    ['board', 'settings.tabBoard'],
    ['system', 'settings.tabSystem'],
  ] as const).map(([id, msg]) => ({
    id: id as Tab,
    label: t(msg),
  })),
);

function onLocaleChange(e: Event): void {
  const v = (e.target as HTMLSelectElement).value;
  if (v === 'zh-CN' || v === 'en-US') {
    setLocale(v as SupportedLocale);
  }
}

function normalizeBoardTheme(tStr: string): BoardThemeDraft {
  if (tStr === 'light' || tStr === 'custom' || tStr === 'custom_image') {
    return tStr;
  }
  return 'dark';
}

const basePx = computed(() => boardStore.board?.grid_base_unit ?? 10);
const canvasWidthPx = computed(() => DESIGN_GRID_WIDTH * basePx.value);
const canvasHeightPx = computed(() => DESIGN_GRID_HEIGHT * basePx.value);

let unsubscribeRpcStatus: (() => void) | null = null;

async function loadAllFromBackend(): Promise<void> {
  if (!boardStore.board) {
    await boardStore.fetchBoard();
  }
  await widgetStore.fetchAll();
  if (boardStore.board) {
    boardDraftName.value = boardStore.board.name;
    boardDraftGrid.value = boardStore.board.grid_base_unit;
    boardDraftWallpaper.value = boardStore.board.wallpaper ?? '';
    boardDraftTheme.value = normalizeBoardTheme(boardStore.board.theme);
    if (boardStore.board.theme_custom) {
      try {
        boardDraftCustom.value = {
          ...defaultCustomVars(),
          ...(JSON.parse(boardStore.board.theme_custom) as CustomThemeVars),
        };
      } catch {
        boardDraftCustom.value = defaultCustomVars();
      }
    } else {
      boardDraftCustom.value = defaultCustomVars();
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown);
  // Rpc onStatus fires once immediately; reload after each reconnect vs empty canvas race.
  unsubscribeRpcStatus = getRpc().onStatus((s) => {
    if (s === 'connected') {
      void loadAllFromBackend();
    }
  });
});

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown);
  unsubscribeRpcStatus?.();
  unsubscribeRpcStatus = null;
});

async function onExportConfig(): Promise<void> {
  exportInFlight.value = true;
  errorMsg.value = null;
  try {
    await downloadConfigBundleFile();
  } catch (err) {
    errorMsg.value = formatError(err);
  } finally {
    exportInFlight.value = false;
  }
}

async function onImportFile(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  errorMsg.value = null;
  importSummary.value = null;
  try {
    if (!window.confirm(t('settings.importConfirm'))) {
      input.value = '';
      return;
    }
    const summary = await httpImportConfigFile(file);
    importSummary.value = summary ?? null;
    await Promise.all([boardStore.fetchBoard(), widgetStore.fetchAll(), dsStore.fetchAll()]);
  } catch (err) {
    errorMsg.value = formatError(err);
  } finally {
    input.value = '';
  }
}

async function onBoardSave(): Promise<void> {
  if (!boardStore.board) return;
  errorMsg.value = null;
  try {
    await boardStore.update({
      grid_base_unit: boardDraftGrid.value,
    });
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

async function onSystemSave(): Promise<void> {
  if (!boardStore.board) return;
  errorMsg.value = null;
  if (boardDraftTheme.value === 'custom_image' && boardDraftWallpaper.value.trim() === '') {
    errorMsg.value = t('settings.wallpaperThemeRequiresImage');
    return;
  }
  try {
    const patch: Partial<Board> = {
      name: boardDraftName.value,
      theme: boardDraftTheme.value,
      theme_custom:
        boardDraftTheme.value === 'custom' || boardDraftTheme.value === 'custom_image'
          ? JSON.stringify(boardDraftCustom.value)
          : '',
    };
    if (boardDraftTheme.value === 'custom_image') {
      patch.wallpaper = boardDraftWallpaper.value.trim();
    }
    await boardStore.update(patch);
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

async function onWallpaperFile(ev: Event): Promise<void> {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  wallpaperFeedback.value = null;
  errorMsg.value = null;
  if (file.size > WALLPAPER_UPLOAD_MAX_BYTES) {
    const mb = Math.floor(WALLPAPER_UPLOAD_MAX_BYTES / (1024 * 1024));
    const msg = t('settings.wallpaperTooLarge', { mb });
    wallpaperFeedback.value = { kind: 'err', text: msg };
    errorMsg.value = msg;
    input.value = '';
    return;
  }
  wallpaperBusy.value = true;
  await new Promise<void>((r) => {
    setTimeout(r, 0);
  });
  try {
    const res = await httpUpload('wallpaper', file);
    boardDraftWallpaper.value = res.url;
    wallpaperFeedback.value = {
      kind: 'ok',
      text: t('settings.wallpaperUploadOk'),
    };
  } catch (err) {
    const msg = formatError(err);
    wallpaperFeedback.value = { kind: 'err', text: msg };
    errorMsg.value = msg;
  } finally {
    wallpaperBusy.value = false;
    input.value = '';
  }
}

function backHome(): void {
  void router.push({ name: 'home' });
}

async function onWidgetUpdate(id: number, rect: Rect): Promise<void> {
  errorMsg.value = null;
  try {
    await widgetStore.move(id, rect);
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

async function onWidgetsUpdateMany(
  updates: { id: number; rect: Rect }[],
): Promise<void> {
  if (updates.length === 0) return;
  errorMsg.value = null;
  try {
    await widgetStore.moveMany(updates);
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

// Selection model (common modifier semantics).
// Plain click: replace selection. Ctrl/Meta: toggle id. Shift: range from lastAnchorId in list order.
// id == null clears selection (canvas blank click).
function onCanvasSelect(
  id: number | null,
  mods: { ctrl: boolean; shift: boolean; meta: boolean },
): void {
  if (id == null) {
    selectedIds.value = [];
    lastAnchorId.value = null;
    return;
  }
  const ctrlOrMeta = mods.ctrl || mods.meta;
  if (mods.shift && lastAnchorId.value !== null && lastAnchorId.value !== id) {
    const order = widgetStore.widgets.map((w) => w.id);
    const a = order.indexOf(lastAnchorId.value);
    const b = order.indexOf(id);
    if (a >= 0 && b >= 0) {
      const [lo, hi] = a <= b ? [a, b] : [b, a];
      const range = order.slice(lo, hi + 1);
      const merged = new Set<number>([...selectedIds.value, ...range]);
      selectedIds.value = Array.from(merged);
      return;
    }
  }
  if (ctrlOrMeta) {
    if (selectedIds.value.includes(id)) {
      selectedIds.value = selectedIds.value.filter((x) => x !== id);
    } else {
      selectedIds.value = [...selectedIds.value, id];
    }
    lastAnchorId.value = id;
    return;
  }
  selectedIds.value = [id];
  lastAnchorId.value = id;
}

function onCanvasMarquee(
  ids: number[],
  mods: { ctrl: boolean; meta: boolean },
): void {
  if (mods.ctrl || mods.meta) {
    selectedIds.value = Array.from(new Set([...selectedIds.value, ...ids]));
  } else {
    selectedIds.value = ids;
  }
  lastAnchorId.value = ids[ids.length - 1] ?? null;
}

async function onWidgetDrop({
  type: rawType,
  rect,
}: {
  type: string;
  rect: Rect;
}): Promise<void> {
  errorMsg.value = null;
  submitting.value = true;
  try {
    const type = rawType as PaletteType;
    if (type === 'link') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:link-variant',
        config: defaultLinkConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    if (type === 'search') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:magnify',
        config: defaultSearchConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    if (type === 'text') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:text',
        config: defaultTextWidgetConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    if (type === 'divider') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:minus',
        config: defaultDividerWidgetConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    if (type === 'weather') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:weather-partly-cloudy',
        config: defaultWeatherWidgetConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    if (type === 'clock') {
      const created = await widgetStore.create({
        type,
        ...rect,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:clock-outline',
        config: defaultClockWidgetConfig(),
      } as Partial<Widget>);
      selectedIds.value = [created.id];
      lastAnchorId.value = created.id;
      return;
    }
    // Data widgets need a datasource plus a metric path with the right shape.
    const expectedShape = ACCEPTED_SHAPES[type]?.[0];
    if (!expectedShape) {
      errorMsg.value = `未知的 widget 类型：${type}`;
      return;
    }
    if (dsStore.items.length === 0) {
      await dsStore.fetchAll();
    }
    if (dsStore.items.length === 0) {
      errorMsg.value = '请先在 "数据源" Tab 中添加数据源后再放置数据组件。';
      return;
    }
    // Prefer a datasource whose tree exposes the expected shape first.
    let chosenDsId: number | null = null;
    let chosenLeafPath: string | null = null;
    for (const ds of dsStore.items) {
      try {
        const tree = await dsStore.discover(ds.id);
        const leaf = findFirstShape(tree.roots, expectedShape);
        if (leaf) {
          chosenDsId = ds.id;
          chosenLeafPath = leaf.path;
          break;
        }
      } catch {
        // Skip failing datasource discovers
      }
    }
    if (!chosenDsId || !chosenLeafPath) {
      errorMsg.value = `当前没有数据源能产出 ${expectedShape} 形态。请先在 "数据源" Tab 添加合适的数据源。`;
      return;
    }
    const created = await widgetStore.create({
      type,
      ...rect,
      icon_type: 'ICONIFY',
      icon_value: paletteIcon(type),
      data_source_id: chosenDsId,
      metric_query: {
        path: chosenLeafPath,
        shape: expectedShape,
        ...(expectedShape === 'TimeSeries' ? { window_sec: 1800 } : {}),
      },
      config: defaultConfigFor(type),
    } as Partial<Widget>);
    selectedIds.value = [created.id];
    lastAnchorId.value = created.id;
  } catch (err) {
    errorMsg.value = formatError(err);
  } finally {
    submitting.value = false;
  }
}

function paletteIcon(type: PaletteType): string {
  switch (type) {
    case 'gauge': return 'mdi:gauge';
    case 'bignumber': return 'mdi:numeric';
    case 'line': return 'mdi:chart-line';
    case 'bar': return 'mdi:chart-bar';
    case 'grid': return 'mdi:view-grid';
    case 'search': return 'mdi:magnify';
    case 'text': return 'mdi:text';
    case 'divider': return 'mdi:minus';
    case 'weather': return 'mdi:weather-partly-cloudy';
    case 'clock': return 'mdi:clock-outline';
    default: return 'mdi:link-variant';
  }
}

function defaultConfigFor(type: PaletteType): Record<string, unknown> {
  switch (type) {
    case 'gauge': return defaultGaugeConfig() as unknown as Record<string, unknown>;
    case 'bignumber': return defaultBigNumberConfig() as unknown as Record<string, unknown>;
    case 'line': return defaultLineConfig() as unknown as Record<string, unknown>;
    case 'bar': return defaultBarConfig() as unknown as Record<string, unknown>;
    case 'grid': return defaultStatusGridConfig() as unknown as Record<string, unknown>;
    case 'text': return defaultTextWidgetConfig() as unknown as Record<string, unknown>;
    case 'divider': return defaultDividerWidgetConfig() as unknown as Record<string, unknown>;
    case 'weather':
      return defaultWeatherWidgetConfig() as unknown as Record<string, unknown>;
    case 'clock':
      return defaultClockWidgetConfig() as unknown as Record<string, unknown>;
    default: return {};
  }
}

function findFirstShape(
  nodes: { leaf: boolean; shapes: string[]; path: string; children?: unknown[] }[],
  shape: Shape,
): { path: string } | null {
  for (const n of nodes) {
    if (n.leaf && n.shapes.includes(shape)) return { path: n.path };
    const childResult = findFirstShape(
      (n.children ?? []) as typeof nodes,
      shape,
    );
    if (childResult) return childResult;
  }
  return null;
}

function onWidgetEditClick(id: number): void {
  const w = widgetStore.widgets.find((it) => it.id === id);
  if (!w) return;
  editTarget.value = w;
  editModalOpen.value = true;
}

async function onWidgetDelete(id: number): Promise<void> {
  errorMsg.value = null;
  try {
    await widgetStore.remove(id);
    selectedIds.value = selectedIds.value.filter((x) => x !== id);
    if (lastAnchorId.value === id) lastAnchorId.value = null;
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

async function deleteSelected(): Promise<void> {
  if (selectedIds.value.length === 0) return;
  errorMsg.value = null;
  const ids = [...selectedIds.value];
  try {
    // No batch delete RPC yet; sequential deletes only.
    for (const id of ids) {
      await widgetStore.remove(id);
    }
    selectedIds.value = [];
    lastAnchorId.value = null;
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

// ---- Copy / paste ----

/**
 * Deep clone JSON-serializable fields; strips Vue/Pinia Proxy (structuredClone breaks on Proxy).
 */
function cloneSerializableJson<T>(value: T | null | undefined): T | null {
  if (value == null) {
    return null;
  }
  return JSON.parse(JSON.stringify(value)) as T;
}

function copySelectedToClipboard(): void {
  if (selectedIds.value.length === 0) return;
  const set = new Set(selectedIds.value);
  // Keep list order relative to canvas when copying.
  const snaps: ClipSnapshot[] = widgetStore.widgets
    .filter((w) => set.has(w.id))
    .map((w) => ({
      type: w.type,
      x: w.x,
      y: w.y,
      w: w.w,
      h: w.h,
      z_index: w.z_index,
      icon_type: w.icon_type,
      icon_value: w.icon_value,
      data_source_id: w.data_source_id,
      metric_query: w.metric_query == null ? null : cloneSerializableJson(w.metric_query),
      config: w.config == null ? {} : cloneSerializableJson(w.config) ?? {},
    }));
  clipboard.value = snaps;
}

async function pasteClipboard(): Promise<void> {
  if (clipboard.value.length === 0) return;
  errorMsg.value = null;
  submitting.value = true;
  // Paste: offset (+2,+2), clamp to canvas, findFreeSpot to avoid overlaps.
  const offset = 2;
  const obstacles: Rect[] = widgetStore.widgets.map((w) => ({
    x: w.x,
    y: w.y,
    w: w.w,
    h: w.h,
  }));
  const newIds: number[] = [];
  try {
    for (const snap of clipboard.value) {
      const desired = clampToCanvas(
        { x: snap.x + offset, y: snap.y + offset, w: snap.w, h: snap.h },
        DESIGN_GRID_WIDTH,
        DESIGN_GRID_HEIGHT,
      );
      const free = findFreeSpot(desired, obstacles);
      const created = await widgetStore.create({
        type: snap.type,
        x: free.x,
        y: free.y,
        w: free.w,
        h: free.h,
        z_index: snap.z_index,
        icon_type: snap.icon_type,
        icon_value: snap.icon_value,
        data_source_id: snap.data_source_id,
        metric_query: snap.metric_query as unknown,
        config: snap.config as unknown,
      } as Partial<Widget>);
      newIds.push(created.id);
      obstacles.push({ x: free.x, y: free.y, w: free.w, h: free.h });
    }
    selectedIds.value = newIds;
    lastAnchorId.value = newIds[newIds.length - 1] ?? null;
  } catch (err) {
    errorMsg.value = formatError(err);
  } finally {
    submitting.value = false;
  }
}

// ---- Settings page hotkeys ----
//
// Mounted only on settings. Skip when focus is in form fields or contenteditable.
function onKeydown(e: KeyboardEvent): void {
  const target = e.target as HTMLElement | null;
  if (target) {
    const tag = target.tagName;
    if (
      tag === 'INPUT' ||
      tag === 'TEXTAREA' ||
      tag === 'SELECT' ||
      target.isContentEditable
    ) {
      return;
    }
  }
  // Do not steal keys while edit modal is open.
  if (editModalOpen.value) return;

  const meta = e.ctrlKey || e.metaKey;
  if (e.key === 'Escape') {
    if (selectedIds.value.length > 0) {
      selectedIds.value = [];
      lastAnchorId.value = null;
      e.preventDefault();
    }
    return;
  }
  if (e.key === 'Delete' || e.key === 'Backspace') {
    if (selectedIds.value.length > 0) {
      e.preventDefault();
      void deleteSelected();
    }
    return;
  }
  if (meta && (e.key === 'c' || e.key === 'C')) {
    if (selectedIds.value.length > 0) {
      e.preventDefault();
      copySelectedToClipboard();
    }
    return;
  }
  if (meta && (e.key === 'v' || e.key === 'V')) {
    if (clipboard.value.length > 0) {
      e.preventDefault();
      void pasteClipboard();
    }
    return;
  }
  if (meta && (e.key === 'a' || e.key === 'A')) {
    if (widgetStore.widgets.length > 0) {
      e.preventDefault();
      selectedIds.value = widgetStore.widgets.map((w) => w.id);
      lastAnchorId.value = selectedIds.value[selectedIds.value.length - 1] ?? null;
    }
    return;
  }
}

async function onModalSubmit({ id, patch }: { id: number; patch: Partial<Widget> }): Promise<void> {
  errorMsg.value = null;
  try {
    await widgetStore.update(id, patch);
    editModalOpen.value = false;
  } catch (err) {
    errorMsg.value = formatError(err);
  }
}

function formatError(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <!-- Preview canvas -->
    <section class="relative flex-1 overflow-auto bg-black/20 p-6 select-none">
      <header class="mb-4 flex items-center justify-between">
        <div>
          <h1 class="text-xl font-semibold">
            {{ t('settings.title') }} —— {{ boardStore.board?.name ?? '' }}
          </h1>
          <p class="text-xs text-[color:var(--astro-text-secondary)]">
            {{ t('settings.hint') }}
          </p>
        </div>
        <button
          type="button"
          class="rounded-md border border-[color:var(--astro-glass-border)] px-3 py-1 text-sm hover:bg-white/5"
          @click="backHome"
        >
          {{ t('settings.back') }}
        </button>
      </header>

      <div
        class="astro-glass mb-4 flex flex-wrap items-center justify-center gap-x-3 gap-y-2 px-4 py-3 text-xs text-[color:var(--astro-text-primary)]"
      >
        <span class="text-[color:var(--astro-text-primary)]">{{ t('settings.scale') }}</span>
        <input
          v-model.number="scale"
          type="range"
          min="0.5"
          max="1"
          step="0.05"
        >
        <span class="astro-mono-num w-10 text-[color:var(--astro-text-primary)]">{{ Math.round(scale * 100) }}%</span>
        <span class="ml-2 text-[color:var(--astro-text-primary)]">
          {{ t('settings.selected') }}
          <span class="astro-mono-num font-semibold tabular-nums">{{ selectedIds.length }}</span>
          <span
            v-if="clipboard.length > 0"
            class="ml-2"
          >· {{ t('settings.clipboard') }}
            <span class="astro-mono-num font-semibold tabular-nums">{{
              t('settings.clipboardCount', { n: clipboard.length })
            }}</span></span>
        </span>
        <span
          class="ml-0 hidden max-w-full text-[11px] leading-snug text-[color:var(--astro-text-primary)] md:ml-2 md:inline"
        >
          {{ t('settings.shortcuts') }}
        </span>
        <span
          v-if="errorMsg"
          class="ml-0 rounded bg-red-600/30 px-2 py-1 text-xs text-red-200 md:ml-2"
        >
          {{ errorMsg }}
        </span>
        <span
          v-if="submitting"
          class="text-xs text-[color:var(--astro-text-primary)]"
        >
          {{ t('settings.saving') }}
        </span>
      </div>

      <!-- Outer box sized to scaled canvas; rim via absolute overlay so inner layout stays clean. -->
      <div
        class="canvas-outer relative mx-auto select-none"
        :style="{
          width: `${canvasWidthPx * scale}px`,
          height: `${canvasHeightPx * scale}px`,
        }"
      >
        <div class="pointer-events-none absolute inset-0 rounded-xl shadow-[0_0_0_2px_var(--astro-accent),0_0_24px_rgba(0,0,0,0.5)]" />
        <Canvas
          mode="edit"
          :widgets="widgetStore.widgets"
          :base-px="basePx"
          :scale="scale"
          :selected-ids="selectedIds"
          @select="onCanvasSelect"
          @marquee="onCanvasMarquee"
          @update="onWidgetUpdate"
          @update-many="onWidgetsUpdateMany"
          @edit="onWidgetEditClick"
          @delete="onWidgetDelete"
          @drop="onWidgetDrop"
        >
          <template #widget="{ widget }">
            <WidgetRenderer
              :widget="widget"
              :interactive="false"
            />
          </template>
        </Canvas>
      </div>
    </section>

    <!-- Right sidebar -->
    <aside
      class="settings-sidebar flex w-[360px] flex-col border-l border-[color:var(--astro-glass-border)]"
    >
      <nav
        class="settings-sidebar-nav flex border-b border-[color:var(--astro-glass-border)] text-sm"
      >
        <button
          v-for="opt in tabOptions"
          :key="opt.id"
          type="button"
          class="astro-btn-icon flex-1 px-3 py-2 hover:bg-white/5 hover:shadow-md focus-visible:ring-2 focus-visible:ring-[color:var(--astro-accent)]"
          :class="tab === opt.id ? 'bg-white/10' : ''"
          @click="tab = opt.id"
        >
          {{ opt.label }}
        </button>
      </nav>
      <div class="settings-sidebar-body flex-1 overflow-y-auto p-4">
        <WidgetPalette v-if="tab === 'palette'" />
        <DataSourceTab v-else-if="tab === 'datasource'" />
        <div
          v-else-if="tab === 'board'"
          class="space-y-3 text-sm"
        >
          <p class="text-xs text-[color:var(--astro-text-secondary)]">
            {{ t('settings.boardNote') }}
          </p>
          <label class="block">
            <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{
              t('settings.gridPx')
            }}</span>
            <select
              v-model.number="boardDraftGrid"
              class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
            >
              <option
                v-for="n in [5, 10, 16, 20, 24]"
                :key="n"
                :value="n"
              >{{ n }}</option>
            </select>
          </label>
          <button
            type="button"
            class="rounded-md bg-[color:var(--astro-accent)] px-3 py-1.5 text-xs text-black hover:opacity-90"
            @click="onBoardSave"
          >
            {{ t('settings.saveGrid') }}
          </button>
        </div>
        <div
          v-else-if="tab === 'system'"
          class="space-y-4 text-sm"
        >
          <section class="space-y-3 rounded-md border border-[color:var(--astro-glass-border)] p-3">
            <h3 class="text-xs uppercase tracking-wider text-[color:var(--astro-text-secondary)]">
              {{ t('settings.systemSite') }}
            </h3>
            <label class="block">
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{
                t('settings.siteTitle')
              }}</span>
              <input
                v-model="boardDraftName"
                type="text"
                :disabled="!boardStore.board"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              >
            </label>
            <label class="block">
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{
                t('settings.locale')
              }}</span>
              <select
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
                :value="locale"
                @change="onLocaleChange"
              >
                <option value="zh-CN">
                  {{ t('settings.localeZhCN') }}
                </option>
                <option value="en-US">
                  {{ t('settings.localeEnUS') }}
                </option>
              </select>
            </label>
            <label class="block">
              <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{
                t('settings.boardTheme')
              }}</span>
              <select
                v-model="boardDraftTheme"
                class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2"
              >
                <option value="dark">
                  {{ t('settings.themeDark') }}
                </option>
                <option value="light">
                  {{ t('settings.themeLight') }}
                </option>
                <option value="custom">
                  {{ t('settings.themeCustom') }}
                </option>
                <option value="custom_image">
                  {{ t('settings.themeCustomImage') }}
                </option>
              </select>
            </label>
            <div
              v-if="boardDraftTheme === 'custom'"
              class="space-y-2 rounded border border-[color:var(--astro-glass-border)] p-3"
            >
              <p class="text-xs text-[color:var(--astro-text-secondary)]">
                {{ t('settings.customVarsHint') }}
              </p>
              <div class="grid grid-cols-1 gap-2">
                <label
                  v-for="entry in customThemeFields"
                  :key="entry.key"
                  class="block"
                >
                  <span class="mb-1 block text-[10px] text-[color:var(--astro-text-secondary)]">
                    {{ entry.label }}
                  </span>
                  <div class="flex gap-2">
                    <input
                      v-model="boardDraftCustom[entry.key]"
                      type="color"
                      class="h-8 w-8 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
                    >
                    <input
                      v-model="boardDraftCustom[entry.key]"
                      type="text"
                      class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 font-mono text-xs"
                    >
                  </div>
                </label>
              </div>
            </div>
            <div
              v-if="boardDraftTheme === 'custom_image'"
              class="space-y-2 rounded border border-[color:var(--astro-glass-border)] p-3"
            >
              <p class="text-[11px] text-[color:var(--astro-text-secondary)]">
                {{ t('settings.wallpaperUploadHint') }}
              </p>
              <ThemedFileTrigger
                :label="t('settings.wallpaperPick')"
                :disabled="wallpaperBusy"
                accept="image/jpeg,image/png,image/webp,image/gif,image/svg+xml"
                @change="onWallpaperFile"
              />
              <p
                v-if="wallpaperBusy"
                class="text-xs text-[color:var(--astro-text-primary)]"
              >
                {{ t('settings.wallpaperBusy') }}
              </p>
              <p
                v-if="wallpaperFeedback"
                class="rounded border px-2 py-1.5 text-xs leading-snug"
                :class="
                  wallpaperFeedback.kind === 'ok'
                    ? 'border-emerald-500/35 bg-emerald-500/10 text-emerald-200'
                    : 'border-red-500/35 bg-red-500/10 text-red-200'
                "
              >
                {{ wallpaperFeedback.text }}
              </p>
              <label class="block">
                <span class="mb-1 block text-xs text-[color:var(--astro-text-secondary)]">{{
                  t('settings.wallpaperUrl')
                }}</span>
                <input
                  v-model="boardDraftWallpaper"
                  type="text"
                  class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 font-mono text-xs"
                  placeholder="/uploads/..."
                >
              </label>
              <fieldset class="space-y-3 rounded-md border border-[color:var(--astro-glass-border)] p-3">
                <legend class="px-1 text-[11px] text-[color:var(--astro-text-secondary)]">
                  {{ t('settings.globalGlass') }}
                </legend>
                <label class="block">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">
                    {{ t('settings.glassTintMode') }}
                  </span>
                  <select
                    v-model="glassTintMode"
                    class="w-full rounded-md border border-[color:var(--astro-glass-border)] bg-transparent px-3 py-2 text-xs"
                  >
                    <option
                      v-for="mode in GLASS_TINT_MODES"
                      :key="mode"
                      :value="mode"
                    >
                      {{ t(`settings.glassTintMode_${mode}`) }}
                    </option>
                  </select>
                </label>
                <p
                  v-if="glassTintMode === 'auto'"
                  class="text-[11px] text-[color:var(--astro-text-secondary)]"
                >
                  {{ t('settings.glassTintAutoHint') }}
                </p>
                <label
                  v-if="glassTintMode === 'manual'"
                  class="block"
                >
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">
                    {{ t('settings.glassTint') }}
                  </span>
                  <div class="flex gap-2">
                    <input
                      type="color"
                      class="h-8 w-8 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
                      :value="boardDraftCustom.glass_tint || '#64748b'"
                      @input="
                        boardDraftCustom.glass_tint = ($event.target as HTMLInputElement).value
                      "
                    >
                    <input
                      v-model="boardDraftCustom.glass_tint"
                      type="text"
                      class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 font-mono text-xs"
                      placeholder="#64748b"
                    >
                  </div>
                </label>
                <label class="block">
                  <span class="mb-1 block text-[11px] text-[color:var(--astro-text-secondary)]">
                    {{ t('settings.glassGlow') }}
                  </span>
                  <div class="flex gap-2">
                    <input
                      type="color"
                      class="h-8 w-8 shrink-0 cursor-pointer rounded border border-[color:var(--astro-glass-border)] bg-transparent p-0"
                      :value="boardDraftCustom.glass_glow || boardDraftCustom.glass_tint || '#64748b'"
                      @input="
                        boardDraftCustom.glass_glow = ($event.target as HTMLInputElement).value
                      "
                    >
                    <input
                      v-model="boardDraftCustom.glass_glow"
                      type="text"
                      class="flex-1 rounded border border-[color:var(--astro-glass-border)] bg-transparent px-2 py-1 font-mono text-xs"
                      :placeholder="t('settings.glassGlowPlaceholder')"
                    >
                  </div>
                </label>
              </fieldset>
            </div>
            <button
              type="button"
              class="rounded-md bg-[color:var(--astro-accent)] px-3 py-1.5 text-xs text-black hover:opacity-90"
              @click="onSystemSave"
            >
              {{ t('settings.saveSiteTheme') }}
            </button>
          </section>
          <section class="space-y-2">
            <h3 class="text-xs uppercase tracking-wider text-[color:var(--astro-text-secondary)]">
              {{ t('settings.exportTitle') }}
            </h3>
            <p class="text-xs text-[color:var(--astro-text-secondary)]">
              {{ t('settings.exportDesc') }}
            </p>
            <button
              type="button"
              :disabled="exportInFlight"
              class="rounded-md border border-[color:var(--astro-glass-border)] px-3 py-1.5 text-xs hover:bg-white/5 disabled:opacity-50"
              @click="onExportConfig"
            >
              {{ exportInFlight ? t('settings.exportBusy') : t('settings.exportBtn') }}
            </button>
          </section>
          <section class="space-y-2">
            <h3 class="text-xs uppercase tracking-wider text-[color:var(--astro-text-secondary)]">
              {{ t('settings.importTitle') }}
            </h3>
            <p class="text-xs text-amber-300">
              {{ t('settings.importWarn') }}
            </p>
            <ThemedFileTrigger
              :label="t('settings.importPick')"
              accept="application/json,.json"
              @change="onImportFile"
            />
            <p
              v-if="importSummary"
              class="text-xs text-emerald-400"
            >
              {{
                t('settings.importDone', {
                  ds: importSummary.data_sources_added,
                  widgets: importSummary.widgets_added,
                })
              }}
              <span v-if="importSummary.widgets_skipped?.length">
                {{ t('settings.importSkipped', { n: importSummary.widgets_skipped.length }) }}
              </span>
            </p>
          </section>
        </div>
      </div>
    </aside>

    <WidgetEditModal
      :widget="editTarget"
      :open="editModalOpen"
      @close="editModalOpen = false"
      @submit="onModalSubmit"
    />
  </div>
</template>

<style scoped>
.canvas-outer {
  position: relative;
}
</style>
