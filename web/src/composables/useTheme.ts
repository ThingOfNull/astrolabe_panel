// Theme helper: project board.theme + board.theme_custom + board.wallpaper onto :root.
//
//   - dark / light: use CSS variables from main.css;
//   - custom: use theme_custom JSON as CSS variable overrides;
//   - custom_image: use wallpaper plus auto/manual Aero glass tint.

export type ThemeKind = 'dark' | 'light' | 'custom' | 'custom_image';
export type GlassTintMode = 'auto' | 'manual';

export interface CustomThemeVars {
  bg_base?: string;
  glass_bg?: string;
  glass_border?: string;
  glass_glow?: string;
  glass_highlight?: string;
  glass_shadow?: string;
  text_primary?: string;
  text_secondary?: string;
  status_ok?: string;
  status_err?: string;
  status_unknown?: string;
  accent?: string;
  glass_tint_mode?: GlassTintMode;
  glass_tint?: string;
}

type ThemeCSSKey = Exclude<keyof CustomThemeVars, 'glass_tint_mode' | 'glass_tint'>;

const VAR_MAP: Record<ThemeCSSKey, string> = {
  bg_base: '--astro-bg-base',
  glass_bg: '--astro-glass-bg',
  glass_border: '--astro-glass-border',
  glass_glow: '--astro-glass-glow',
  glass_highlight: '--astro-glass-highlight',
  glass_shadow: '--astro-glass-shadow',
  text_primary: '--astro-text-primary',
  text_secondary: '--astro-text-secondary',
  status_ok: '--astro-status-ok',
  status_err: '--astro-status-err',
  status_unknown: '--astro-status-unknown',
  accent: '--astro-accent',
};

interface ExtractedTint {
  glassBg: string;
  border: string;
  glow: string;
  highlight: string;
}

interface RGB {
  r: number;
  g: number;
  b: number;
}

const tintCache = new Map<string, ExtractedTint | null>();
let themeApplySeq = 0;

export function applyTheme(
  kind: string,
  options: { customJSON?: string | null; wallpaper?: string | null } = {},
): void {
  const seq = ++themeApplySeq;
  const root = document.documentElement;
  root.removeAttribute('data-wallpaper');
  root.style.backgroundImage = '';
  root.style.backgroundSize = '';
  root.style.backgroundPosition = '';
  root.style.backgroundRepeat = '';
  root.style.backgroundAttachment = '';

  for (const cssVar of Object.values(VAR_MAP)) {
    root.style.removeProperty(cssVar);
  }

  const parsed = parseCustomJSON(options.customJSON);

  if (kind === 'custom_image') {
    root.setAttribute('data-theme', 'dark');
    const wp = (options.wallpaper ?? '').trim();
    const resolvedURL = resolveWallpaperURL(wp);
    if (wp !== '') {
      root.setAttribute('data-wallpaper', '1');
      root.style.backgroundImage = `url(${JSON.stringify(resolvedURL)})`;
      root.style.backgroundSize = 'cover';
      root.style.backgroundPosition = 'center';
      root.style.backgroundRepeat = 'no-repeat';
      root.style.backgroundAttachment = 'fixed';
    }
    applyCustomVariables(root, parsed);
    applyCustomImageGlass(root, parsed, resolvedURL, seq);
    return;
  }

  const safeKind: ThemeKind = kind === 'light' || kind === 'custom' ? kind : 'dark';
  root.setAttribute('data-theme', safeKind);

  if (safeKind !== 'custom') {
    return;
  }

  applyCustomVariables(root, parsed);
}

function applyCustomVariables(root: HTMLElement, parsed: CustomThemeVars): void {
  for (const key of Object.keys(VAR_MAP) as ThemeCSSKey[]) {
    const v = parsed[key];
    if (typeof v === 'string' && v.trim() !== '') {
      root.style.setProperty(VAR_MAP[key], v);
    }
  }
}

function applyCustomImageGlass(
  root: HTMLElement,
  parsed: CustomThemeVars,
  wallpaperURL: string,
  seq: number,
): void {
  const mode = parsed.glass_tint_mode === 'manual' ? 'manual' : 'auto';
  const manualTint = typeof parsed.glass_tint === 'string' ? parsed.glass_tint.trim() : '';
  const manualGlow = typeof parsed.glass_glow === 'string' ? parsed.glass_glow.trim() : '';

  if (mode === 'manual' && manualTint !== '') {
    setGlassTint(root, tintFromColor(manualTint, manualGlow));
    return;
  }
  if (wallpaperURL === '') {
    return;
  }

  void extractWallpaperTint(wallpaperURL).then((tint) => {
    if (seq !== themeApplySeq || tint === null) return;
    const withManualGlow = manualGlow !== '' ? { ...tint, glow: colorWithAlpha(manualGlow, 0.28) } : tint;
    setGlassTint(root, withManualGlow);
  });
}

function setGlassTint(root: HTMLElement, tint: ExtractedTint): void {
  root.style.setProperty('--astro-glass-bg', tint.glassBg);
  root.style.setProperty('--astro-glass-border', tint.border);
  root.style.setProperty('--astro-glass-glow', tint.glow);
  root.style.setProperty('--astro-glass-highlight', tint.highlight);
}

function tintFromColor(color: string, glowColor: string): ExtractedTint {
  const fallback = color.trim();
  const glowBase = glowColor.trim() || fallback;
  return {
    glassBg: colorWithAlpha(fallback, 0.46),
    border: colorWithAlpha(fallback, 0.3),
    glow: colorWithAlpha(glowBase, 0.28),
    highlight: 'rgba(255, 255, 255, 0.22)',
  };
}

async function extractWallpaperTint(url: string): Promise<ExtractedTint | null> {
  if (tintCache.has(url)) {
    return tintCache.get(url) ?? null;
  }
  try {
    const img = await loadImage(url);
    const size = 56;
    const canvas = document.createElement('canvas');
    canvas.width = size;
    canvas.height = size;
    const ctx = canvas.getContext('2d', { willReadFrequently: true });
    if (!ctx) {
      tintCache.set(url, null);
      return null;
    }
    ctx.drawImage(img, 0, 0, size, size);
    const data = ctx.getImageData(0, 0, size, size).data;
    const rgb = averageWallpaperColor(data);
    if (!rgb) {
      tintCache.set(url, null);
      return null;
    }
    const softened = softenForGlass(rgb);
    const tint: ExtractedTint = {
      glassBg: `rgba(${softened.r}, ${softened.g}, ${softened.b}, 0.46)`,
      border: `rgba(${Math.min(255, softened.r + 46)}, ${Math.min(255, softened.g + 46)}, ${Math.min(255, softened.b + 46)}, 0.22)`,
      glow: `rgba(${softened.r}, ${softened.g}, ${softened.b}, 0.26)`,
      highlight: 'rgba(255, 255, 255, 0.24)',
    };
    tintCache.set(url, tint);
    return tint;
  } catch {
    tintCache.set(url, null);
    return null;
  }
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('image load failed'));
    img.src = url;
  });
}

function averageWallpaperColor(data: Uint8ClampedArray): RGB | null {
  let r = 0;
  let g = 0;
  let b = 0;
  let weightSum = 0;
  for (let i = 0; i < data.length; i += 4) {
    const a = data[i + 3] / 255;
    if (a < 0.4) continue;
    const cr = data[i];
    const cg = data[i + 1];
    const cb = data[i + 2];
    const luma = (0.2126 * cr + 0.7152 * cg + 0.0722 * cb) / 255;
    if (luma < 0.08 || luma > 0.94) continue;
    const saturationWeight = 0.75 + colorSpread(cr, cg, cb) / 255;
    const weight = a * saturationWeight;
    r += cr * weight;
    g += cg * weight;
    b += cb * weight;
    weightSum += weight;
  }
  if (weightSum <= 0) return null;
  return {
    r: Math.round(r / weightSum),
    g: Math.round(g / weightSum),
    b: Math.round(b / weightSum),
  };
}

function softenForGlass(rgb: RGB): RGB {
  const target = { r: 72, g: 88, b: 108 };
  return {
    r: Math.round(rgb.r * 0.72 + target.r * 0.28),
    g: Math.round(rgb.g * 0.72 + target.g * 0.28),
    b: Math.round(rgb.b * 0.72 + target.b * 0.28),
  };
}

function colorSpread(r: number, g: number, b: number): number {
  return Math.max(r, g, b) - Math.min(r, g, b);
}

function colorWithAlpha(color: string, alpha: number): string {
  const hex = parseHexColor(color);
  if (hex) {
    return `rgba(${hex.r}, ${hex.g}, ${hex.b}, ${alpha})`;
  }
  return `color-mix(in srgb, ${color} ${Math.round(alpha * 100)}%, transparent)`;
}

function parseHexColor(color: string): RGB | null {
  const s = color.trim();
  const short = /^#([0-9a-f]{3})$/i.exec(s);
  if (short) {
    const [r, g, b] = short[1].split('').map((ch) => parseInt(ch + ch, 16));
    return { r, g, b };
  }
  const full = /^#([0-9a-f]{6})$/i.exec(s);
  if (!full) return null;
  const raw = full[1];
  return {
    r: parseInt(raw.slice(0, 2), 16),
    g: parseInt(raw.slice(2, 4), 16),
    b: parseInt(raw.slice(4, 6), 16),
  };
}

function parseCustomJSON(customJSON?: string | null): CustomThemeVars {
  if (!customJSON) return {};
  try {
    return JSON.parse(customJSON) as CustomThemeVars;
  } catch {
    return {};
  }
}

function resolveWallpaperURL(wp: string): string {
  if (wp === '') return '';
  return wp.startsWith('http://') ||
    wp.startsWith('https://') ||
    wp.startsWith('/') ||
    wp.startsWith('data:')
    ? wp
    : `/${wp}`;
}

export function defaultCustomVars(): CustomThemeVars {
  return {
    bg_base: '#101218',
    glass_bg: 'rgba(28, 32, 44, 0.7)',
    glass_border: 'rgba(255, 255, 255, 0.12)',
    glass_glow: '',
    glass_highlight: '',
    glass_shadow: '',
    text_primary: '#e8eaf0',
    text_secondary: '#9aa0a8',
    status_ok: '#39ff14',
    status_err: '#ff3131',
    status_unknown: '#888888',
    accent: '#a78bfa',
    glass_tint_mode: 'auto',
    glass_tint: '',
  };
}
