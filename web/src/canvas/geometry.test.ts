import { describe, expect, it } from 'vitest';

import { clampToCanvas, findFreeSpot, hasCollision, pxToMultiplier, rectsOverlap, snap } from './geometry';

describe('snap & multiplier conversion', () => {
  it('snaps pixel values to nearest base unit', () => {
    expect(snap(0, 10)).toBe(0);
    expect(snap(11, 10)).toBe(10);
    expect(snap(15, 10)).toBe(20);
    expect(snap(-13, 10)).toBe(-10);
  });

  it('round-trips multiplier <-> pixel', () => {
    expect(pxToMultiplier(43, 10)).toBe(4);
    expect(pxToMultiplier(45, 10)).toBe(5);
  });
});

describe('rect overlap', () => {
  it('detects overlap for intersecting rects', () => {
    expect(rectsOverlap({ x: 0, y: 0, w: 4, h: 2 }, { x: 2, y: 0, w: 4, h: 2 })).toBe(true);
  });

  it('treats edge-touching rects as non-overlap', () => {
    expect(rectsOverlap({ x: 0, y: 0, w: 4, h: 2 }, { x: 4, y: 0, w: 4, h: 2 })).toBe(false);
  });
});

describe('clampToCanvas', () => {
  it('keeps rect inside design canvas', () => {
    const r = clampToCanvas({ x: 199, y: 119, w: 8, h: 4 }, 200, 120);
    expect(r.w).toBeLessThanOrEqual(8);
    expect(r.h).toBeLessThanOrEqual(4);
    expect(r.x + r.w).toBeLessThanOrEqual(200);
    expect(r.y + r.h).toBeLessThanOrEqual(120);
  });
});

describe('findFreeSpot', () => {
  it('returns desired position when no collision', () => {
    const free = findFreeSpot({ x: 0, y: 0, w: 4, h: 2 }, []);
    expect(free).toEqual({ x: 0, y: 0, w: 4, h: 2 });
  });

  it('relocates to nearest free spot when colliding', () => {
    const others = [{ x: 0, y: 0, w: 4, h: 2 }];
    const free = findFreeSpot({ x: 1, y: 0, w: 4, h: 2 }, others);
    expect(hasCollision(free, others)).toBe(false);
  });
});
