/**
 * Weather city rows: sorted by bare pinyin, with initials helper for filter UI.
 */

import { pinyin } from 'pinyin-pro';

import rawJson from '@/data/meizuWeatherCities.json';

export interface MeizuCityRow {
  areaid: number;
  countyname: string;
  /** Toneless lowercase pinyin, no spaces */
  py: string;
  /** First-letter initials of locale name */
  initials: string;
}

type RawCity = { areaid: number; countyname: string };

let indexed: MeizuCityRow[] | null = null;

/**
 * Build lazy cached index sorted by py.
 */
export function getMeizuCityIndex(): MeizuCityRow[] {
  if (indexed) return indexed;
  const raw = rawJson as RawCity[];
  const list: MeizuCityRow[] = [];
  for (const r of raw) {
    const name = String(r.countyname ?? '').trim();
    const id = Number(r.areaid);
    if (!Number.isFinite(id) || id <= 0 || !name) continue;
    const py = pinyin(name, { toneType: 'none', type: 'string' })
      .replace(/\s+/g, '')
      .toLowerCase();
    const initials = pinyin(name, {
      pattern: 'first',
      toneType: 'none',
      type: 'string',
    })
      .replace(/\s+/g, '')
      .toLowerCase();
    list.push({ areaid: id, countyname: name, py, initials });
  }
  list.sort((a, b) => a.py.localeCompare(b.py, 'en'));
  indexed = list;
  return indexed;
}

function headerLetter(row: MeizuCityRow): string {
  const c = row.py.charAt(0);
  if (/[a-z]/i.test(c)) return c.toUpperCase();
  return '#';
}

/**
 * Filter rows by localized name substring, py substring, or initials match.
 */
export function filterMeizuCities(query: string, limit: number): MeizuCityRow[] {
  const all = getMeizuCityIndex();
  const q = query.trim();
  if (!q) return all.slice(0, Math.min(limit, all.length));
  const low = q.toLowerCase();
  const out: MeizuCityRow[] = [];
  for (const row of all) {
    if (
      row.countyname.includes(q) ||
      row.py.includes(low) ||
      row.initials.startsWith(low) ||
      row.initials.includes(low)
    ) {
      out.push(row);
      if (out.length >= limit) break;
    }
  }
  return out;
}

export interface LetterGroup {
  letter: string;
  items: MeizuCityRow[];
}

/** Group contiguous rows by header letter for A/B/C headings */
export function groupRowsByLetter(rows: MeizuCityRow[]): LetterGroup[] {
  if (rows.length === 0) return [];
  const groups: LetterGroup[] = [];
  let letter = headerLetter(rows[0]);
  let items: MeizuCityRow[] = [rows[0]];
  for (let i = 1; i < rows.length; i++) {
    const r = rows[i];
    const L = headerLetter(r);
    if (L !== letter) {
      groups.push({ letter, items });
      letter = L;
      items = [r];
    } else {
      items.push(r);
    }
  }
  groups.push({ letter, items });
  return groups;
}

/** Linear lookup by id (settings-only, low frequency) */
export function findMeizuCityById(id: number): MeizuCityRow | undefined {
  return getMeizuCityIndex().find((r) => r.areaid === id);
}
