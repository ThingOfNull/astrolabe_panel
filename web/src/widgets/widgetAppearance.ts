import type { WidgetAppearance } from '@/widgets/types';

/** Shared widget chrome semantics; config.appearance is JSON passthrough on the server. */
export type WidgetAppearanceVariant = WidgetAppearance['variant'];

const defaultGlass = (): WidgetAppearance => ({ variant: 'glass', blur_px: 16 });

/** Parse config.appearance; invalid or missing values default to frosted glass. */
export function parseWidgetAppearance(cfg: unknown): WidgetAppearance {
  if (!cfg || typeof cfg !== 'object') return defaultGlass();
  const raw = cfg as Record<string, unknown>;
  const apRaw = raw.appearance as Record<string, unknown> | undefined;
  if (!apRaw || typeof apRaw !== 'object') {
    return defaultGlass();
  }
  const v = String(apRaw.variant ?? 'glass').toLowerCase();
  const variant: WidgetAppearance['variant'] = v === 'solid' ? 'solid' : 'glass';
  const blur_px = typeof apRaw.blur_px === 'number' ? apRaw.blur_px : Number(apRaw.blur_px);
  const solid_color = typeof apRaw.solid_color === 'string' ? apRaw.solid_color : '';

  const out: WidgetAppearance = { variant };

  if (variant === 'glass' && Number.isFinite(blur_px) && blur_px > 0) {
    out.blur_px = Math.min(blur_px, 64);
  }
  if (variant === 'solid' && solid_color.trim() !== '') {
    out.solid_color = solid_color.trim();
  }
  return out;
}

/** For outer containers without WidgetFrame (compact feed, etc.). */
export function widgetChromeOuterClass(_ap: WidgetAppearance): string[] {
  return ['min-h-0', 'min-w-0', 'overflow-hidden', 'rounded-[inherit]'];
}

export function widgetChromeOuterStyle(ap: WidgetAppearance): Record<string, string> {
  if (ap.variant === 'solid' && ap.solid_color) {
    return {
      background: ap.solid_color,
      border: '1px solid var(--astro-glass-border)',
      boxShadow: 'var(--astro-glass-shadow)',
    };
  }
  if (ap.variant === 'glass') {
    const blur = ap.blur_px ?? 16;
    return {
      background: 'var(--astro-glass-bg)',
      backdropFilter: `blur(${blur}px) saturate(180%)`,
      WebkitBackdropFilter: `blur(${blur}px) saturate(180%)`,
      border: '1px solid var(--astro-glass-border)',
      boxShadow:
        'inset 0 1px 0 var(--astro-glass-highlight), var(--astro-glass-shadow), var(--astro-glass-glow)',
    };
  }
  return {};
}
