/**
 * Translate JSON-RPC errors into human-facing, localized messages.
 *
 * The server sends errors with:
 *   - code: numeric JSON-RPC code (stable alias for dashboards/logs),
 *   - message: ErrorCode enum string ("WIDGET_OVERLAP" etc.),
 *   - data: optional structured context.
 *
 * The frontend keys i18n off `message` (the enum). Unknown codes fall back to
 * the raw message so unrecognized server errors still surface to the user.
 */

import type { ComposerTranslation } from 'vue-i18n';

export interface RpcErrorShape {
  code: number;
  message: string;
  data?: unknown;
}

function isRpcError(v: unknown): v is RpcErrorShape {
  return (
    !!v &&
    typeof v === 'object' &&
    'code' in (v as Record<string, unknown>) &&
    'message' in (v as Record<string, unknown>)
  );
}

/**
 * Resolve a localized error string. Accepts anything a rejected promise may
 * carry (RPC error envelope, native Error, string, object).
 */
export function formatRpcError(err: unknown, t: ComposerTranslation): string {
  if (isRpcError(err)) {
    const key = `errors.${err.message}`;
    const localized = t(key);
    // vue-i18n returns the key itself when missing; keep raw enum + data when
    // no translation exists so developers see which code to add.
    if (localized && localized !== key) return localized;
    if (err.data && typeof err.data === 'string') return `${err.message}: ${err.data}`;
    return err.message;
  }
  if (err instanceof Error) return err.message;
  if (typeof err === 'string') return err;
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message: unknown }).message);
  }
  return String(err);
}
