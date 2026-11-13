import { describe, expect, it } from 'vitest';
import { computeContainBox, computeCoverCrop } from './canvasMapper';

describe('canvasMapper geometry', () => {
  it('computes centered cover crop for wide banner strips', () => {
    const crop = computeCoverCrop(1200, 800, 600, 115, 0.5, 0.6);
    expect(crop.width).toBe(1200);
    expect(Math.round(crop.height)).toBe(230);
    expect(crop.cropX).toBe(0);
    expect(crop.cropY).toBeGreaterThan(300);
    expect(Math.round(crop.width * crop.scaleX)).toBe(600);
    expect(Math.round(crop.height * crop.scaleY)).toBe(115);
  });

  it('computes uniform contain placement for logos inside a card', () => {
    const box = computeContainBox(400, 100, 49, 42, 150, 40);
    expect(box.scaleX).toBeCloseTo(0.375);
    expect(box.scaleY).toBeCloseTo(0.375);
    expect(box.left).toBeCloseTo(49);
    expect(box.top).toBeCloseTo(43.25);
  });
});
