import { describe, expect, it, vi } from 'vitest';
import {
  computeContainBox,
  computeCoverCrop,
  updateCanvasBackgroundInPlace,
  updateLayerObjectInPlace
} from './canvasMapper';
import type { Layer } from './types';

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

  it('updates text layer properties and canvas background synchronously in place', () => {
    const setMock = vi.fn();
    const setCoordsMock = vi.fn();
    const initDimensionsMock = vi.fn();
    const requestRenderAllMock = vi.fn();
    const mockObj = {
      instagoLayerId: 'text-1',
      set: setMock,
      setCoords: setCoordsMock,
      initDimensions: initDimensionsMock
    };
    const mockCanvas = {
      backgroundColor: '#FFFFFF',
      getObjects: () => [mockObj],
      requestRenderAll: requestRenderAllMock
    } as unknown as Parameters<typeof updateLayerObjectInPlace>[0];

    const layer: Layer = {
      id: 'text-1',
      name: 'Headline',
      kind: 'text',
      x: 40,
      y: 80,
      width: 300,
      height: 50,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: 1,
      fill: '#374F1E',
      text: 'Novo texto digitado',
      fontFamily: 'Sora',
      fontSize: 32,
      fontWeight: '700',
      lineHeight: 1.15,
      textAlign: 'center'
    };

    const updated = updateLayerObjectInPlace(mockCanvas, layer);
    expect(updated).toBe(true);
    expect(setMock).toHaveBeenCalledWith(
      expect.objectContaining({
        text: 'Novo texto digitado',
        fill: '#374F1E',
        fontSize: 32,
        textAlign: 'center',
        editable: true
      })
    );
    expect(setCoordsMock).toHaveBeenCalled();

    updateCanvasBackgroundInPlace(mockCanvas, '#EBF1DF');
    expect((mockCanvas as unknown as { backgroundColor: string }).backgroundColor).toBe('#EBF1DF');
    expect(requestRenderAllMock).toHaveBeenCalledTimes(2);
  });
});
