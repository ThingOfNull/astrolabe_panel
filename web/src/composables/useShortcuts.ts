import { onBeforeUnmount, onMounted } from 'vue';

/**
 * useShortcuts: lightweight key → handler dispatcher.
 *
 * - Skips when focus is inside an input / textarea / contenteditable so text
 *   typing never triggers accidental delete/undo.
 * - Normalizes platform: Ctrl on Win/Linux, Meta on macOS.
 *
 * Binding format (case-insensitive):
 *   "mod+c"    → Ctrl+C or ⌘+C
 *   "shift+a"  → Shift+A
 *   "delete"   → Delete or Backspace
 *   "escape"   → Escape
 */
export type ShortcutHandler = (e: KeyboardEvent) => void;
export type ShortcutMap = Record<string, ShortcutHandler>;

export function useShortcuts(bindings: () => ShortcutMap): void {
  function onKey(e: KeyboardEvent): void {
    if (isTypingTarget(e.target)) return;
    const map = bindings();
    const combo = keyOf(e);
    const handler = map[combo];
    if (handler) {
      handler(e);
    }
  }

  onMounted(() => document.addEventListener('keydown', onKey));
  onBeforeUnmount(() => document.removeEventListener('keydown', onKey));
}

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
  if (target.isContentEditable) return true;
  return false;
}

function keyOf(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey || e.metaKey) parts.push('mod');
  if (e.shiftKey) parts.push('shift');
  if (e.altKey) parts.push('alt');
  // Normalize special keys.
  let k = e.key.toLowerCase();
  if (k === ' ') k = 'space';
  if (k === 'backspace') k = 'delete'; // treat together; users expect both
  parts.push(k);
  return parts.join('+');
}
