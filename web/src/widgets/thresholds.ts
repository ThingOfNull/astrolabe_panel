// Shared threshold color helpers for data-driven widgets.
//
// `thresholds`: [{ value, color }], ascending by value.
// Use the color of the last row where metric value >= value.

export interface Threshold {
  value: number;
  color: string;
}

// Highest matching threshold color, else fallback.
export function pickColor(
  value: number | null | undefined,
  thresholds: Threshold[] | undefined,
  fallback: string,
): string {
  if (value == null || !Number.isFinite(value)) return fallback;
  if (!thresholds || thresholds.length === 0) return fallback;
  const sorted = [...thresholds].sort((a, b) => a.value - b.value);
  let chosen = fallback;
  for (const t of sorted) {
    if (value >= t.value) {
      chosen = t.color;
    } else {
      break;
    }
  }
  return chosen;
}

// Sort and coerce threshold rows into a clean list.
export function normalizeThresholds(input: unknown): Threshold[] {
  if (!Array.isArray(input)) return [];
  const out: Threshold[] = [];
  for (const item of input) {
    if (!item || typeof item !== 'object') continue;
    const obj = item as Record<string, unknown>;
    const v = Number(obj.value);
    const c = String(obj.color ?? '');
    if (Number.isFinite(v) && c) {
      out.push({ value: v, color: c });
    }
  }
  return out.sort((a, b) => a.value - b.value);
}
