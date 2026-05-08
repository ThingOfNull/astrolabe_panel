/**
 * Fetches widget schema from the backend (single source of truth).
 *
 * The Go store declares `AcceptedShapesByType`; this module hydrates the
 * frontend's compatible fallback table on boot, so adding a new widget type
 * on the server is visible to the frontend without shipping a code change.
 * The TypeScript fallback stays authoritative only during the brief window
 * between app start and response.
 */

import type { Shape } from './types';
import { ACCEPTED_SHAPES } from '@/widgets/types';

export interface WidgetSchema {
  types: string[];
  accepted_shapes: Record<string, Shape[]>;
  icon_types: string[];
}

let cached: WidgetSchema | null = null;

export function getWidgetSchemaSync(): WidgetSchema | null {
  return cached;
}

/**
 * Load once and cache in-memory. Overwrites ACCEPTED_SHAPES in place so every
 * module that imported it before hydration sees the latest mapping via the
 * same object reference.
 */
export async function loadWidgetSchema(): Promise<WidgetSchema | null> {
  if (cached) return cached;
  try {
    const url = `${window.location.origin}/api/schema/widgets`;
    const res = await fetch(url, { credentials: 'same-origin' });
    if (!res.ok) {
      console.warn('[schema] non-200 status', res.status);
      return null;
    }
    const payload = (await res.json()) as WidgetSchema;
    cached = payload;
    if (payload.accepted_shapes) {
      // Mutate the shared object rather than re-assign: existing importers
      // (e.g. widget/types.ts consumers) already hold a reference.
      for (const key of Object.keys(ACCEPTED_SHAPES)) {
        delete ACCEPTED_SHAPES[key];
      }
      Object.assign(ACCEPTED_SHAPES, payload.accepted_shapes);
    }
    return payload;
  } catch (err) {
    console.warn('[schema] load failed; keeping static fallback', err);
    return null;
  }
}
