import {
  Canvas,
  FabricImage,
  FabricObject,
  Rect,
  Textbox
} from 'fabric';
import type { Layer, Page } from './types';

export interface TaggedObject extends FabricObject {
  instagoLayerId?: string;
}

export function tagLayerObject(obj: FabricObject, layerId: string): TaggedObject {
  const tagged = obj as TaggedObject;
  tagged.instagoLayerId = layerId;
  return tagged;
}

export function getLayerId(obj: FabricObject | null | undefined): string | null {
  if (!obj) {
    return null;
  }
  return (obj as TaggedObject).instagoLayerId ?? null;
}

export function computeCoverCrop(
  naturalW: number,
  naturalH: number,
  targetW: number,
  targetH: number,
  posX = 0.5,
  posY = 0.5
): { cropX: number; cropY: number; width: number; height: number; scaleX: number; scaleY: number } {
  if (naturalW <= 0 || naturalH <= 0 || targetW <= 0 || targetH <= 0) {
    return { cropX: 0, cropY: 0, width: naturalW, height: naturalH, scaleX: 1, scaleY: 1 };
  }
  const targetRatio = targetW / targetH;
  const naturalRatio = naturalW / naturalH;

  let cropW = naturalW;
  let cropH = naturalH;

  if (naturalRatio > targetRatio) {
    cropW = naturalH * targetRatio;
  } else {
    cropH = naturalW / targetRatio;
  }

  const clampedX = Math.max(0, Math.min(1, posX));
  const clampedY = Math.max(0, Math.min(1, posY));
  const cropX = Math.max(0, (naturalW - cropW) * clampedX);
  const cropY = Math.max(0, (naturalH - cropH) * clampedY);

  return {
    cropX,
    cropY,
    width: cropW,
    height: cropH,
    scaleX: targetW / cropW,
    scaleY: targetH / cropH
  };
}

export function computeContainBox(
  naturalW: number,
  naturalH: number,
  boxX: number,
  boxY: number,
  boxW: number,
  boxH: number
): { left: number; top: number; scaleX: number; scaleY: number } {
  if (naturalW <= 0 || naturalH <= 0 || boxW <= 0 || boxH <= 0) {
    return { left: boxX, top: boxY, scaleX: 1, scaleY: 1 };
  }
  const scale = Math.min(boxW / naturalW, boxH / naturalH);
  const renderedW = naturalW * scale;
  const renderedH = naturalH * scale;
  return {
    left: boxX + (boxW - renderedW) / 2,
    top: boxY + (boxH - renderedH) / 2,
    scaleX: scale,
    scaleY: scale
  };
}

function applyLockState(obj: FabricObject, locked: boolean): void {
  obj.set({
    lockMovementX: locked,
    lockMovementY: locked,
    lockScalingX: locked,
    lockScalingY: locked,
    lockRotation: locked,
    hasControls: !locked
  });
}

async function createImageObject(layer: Layer): Promise<FabricObject> {
  if (!layer.assetUrl) {
    throw new Error(`Missing assetUrl for image layer ${layer.id}`);
  }
  const img = await FabricImage.fromURL(layer.assetUrl, { crossOrigin: 'anonymous' });
  const natW = img.width || layer.width;
  const natH = img.height || layer.height;

  if (layer.objectFit === 'contain') {
    const box = computeContainBox(natW, natH, layer.x, layer.y, layer.width, layer.height);
    img.set({
      originX: 'left',
      originY: 'top',
      left: box.left,
      top: box.top,
      scaleX: box.scaleX * (layer.scaleX || 1),
      scaleY: box.scaleY * (layer.scaleY || 1),
      angle: layer.angle || 0,
      opacity: layer.opacity ?? 1,
      visible: layer.visible
    });
  } else {
    const cover = computeCoverCrop(
      natW,
      natH,
      layer.width,
      layer.height,
      layer.positionX ?? 0.5,
      layer.positionY ?? 0.5
    );
    img.set({
      originX: 'left',
      originY: 'top',
      left: layer.x,
      top: layer.y,
      cropX: cover.cropX,
      cropY: cover.cropY,
      width: cover.width,
      height: cover.height,
      scaleX: cover.scaleX * (layer.scaleX || 1),
      scaleY: cover.scaleY * (layer.scaleY || 1),
      angle: layer.angle || 0,
      opacity: layer.opacity ?? 1,
      visible: layer.visible
    });
  }

  applyLockState(img, layer.locked);
  return tagLayerObject(img, layer.id);
}

export async function renderPageToCanvas(canvas: Canvas, page: Page): Promise<void> {
  canvas.clear();
  canvas.backgroundColor = page.background || '#F6F7EE';

  const ordered = [...page.layers].sort((a, b) => a.zIndex - b.zIndex);

  for (const layer of ordered) {
    let obj: FabricObject;
    if (layer.kind === 'rect') {
      obj = new Rect({
        originX: 'left',
        originY: 'top',
        left: layer.x,
        top: layer.y,
        width: layer.width,
        height: layer.height,
        scaleX: layer.scaleX || 1,
        scaleY: layer.scaleY || 1,
        angle: layer.angle || 0,
        opacity: layer.opacity ?? 1,
        visible: layer.visible,
        fill: layer.fill || '#98B432',
        stroke: layer.stroke || undefined,
        strokeWidth: layer.strokeWidth || 0,
        rx: layer.radius || 0,
        ry: layer.radius || 0
      });
      applyLockState(obj, layer.locked);
      tagLayerObject(obj, layer.id);
    } else if (layer.kind === 'text') {
      obj = new Textbox(layer.text ?? '', {
        originX: 'left',
        originY: 'top',
        left: layer.x,
        top: layer.y,
        width: Math.max(24, layer.width || 240),
        scaleX: layer.scaleX || 1,
        scaleY: layer.scaleY || 1,
        angle: layer.angle || 0,
        opacity: layer.opacity ?? 1,
        visible: layer.visible,
        fill: layer.fill || '#374F1E',
        fontFamily: layer.fontFamily || 'Inter',
        fontSize: layer.fontSize || 20,
        fontWeight: layer.fontWeight || '400',
        lineHeight: layer.lineHeight || 1.2,
        charSpacing: layer.letterSpacing || 0,
        textAlign: (layer.textAlign as 'left' | 'center' | 'right') || 'left',
        splitByGrapheme: false
      });
      applyLockState(obj, layer.locked);
      tagLayerObject(obj, layer.id);
    } else {
      try {
        obj = await createImageObject(layer);
      } catch {
        obj = new Rect({
          originX: 'left',
          originY: 'top',
          left: layer.x,
          top: layer.y,
          width: layer.width,
          height: layer.height,
          fill: '#E5EBD6',
          stroke: '#98B432',
          strokeWidth: 2,
          rx: 6,
          ry: 6,
          visible: layer.visible
        });
        applyLockState(obj, layer.locked);
        tagLayerObject(obj, layer.id);
      }
    }
    canvas.add(obj);
  }

  canvas.renderAll();
}

export function syncCanvasToPage(canvas: Canvas, page: Page): Page {
  const byId = new Map<string, Layer>();
  for (const layer of page.layers) {
    byId.set(layer.id, { ...layer });
  }

  const objects = canvas.getObjects();
  const updatedLayers: Layer[] = [];

  objects.forEach((obj, idx) => {
    const id = getLayerId(obj);
    if (!id) {
      return;
    }
    const base = byId.get(id);
    if (!base) {
      return;
    }

    const next: Layer = {
      ...base,
      x: Math.round((obj.left ?? base.x) * 100) / 100,
      y: Math.round((obj.top ?? base.y) * 100) / 100,
      angle: Math.round((obj.angle ?? base.angle) * 100) / 100,
      opacity: obj.opacity ?? base.opacity,
      visible: obj.visible ?? base.visible,
      zIndex: idx + 1
    };

    if (base.kind === 'rect') {
      next.width = Math.round(((obj.width ?? base.width) * (obj.scaleX ?? 1)) * 100) / 100;
      next.height = Math.round(((obj.height ?? base.height) * (obj.scaleY ?? 1)) * 100) / 100;
      next.scaleX = 1;
      next.scaleY = 1;
      if (typeof obj.fill === 'string') {
        next.fill = obj.fill;
      }
    } else if (base.kind === 'text') {
      const tb = obj as Textbox;
      next.text = tb.text ?? base.text;
      next.width = Math.round((tb.width ?? base.width) * 100) / 100;
      next.scaleX = tb.scaleX ?? 1;
      next.scaleY = tb.scaleY ?? 1;
      next.fontFamily = tb.fontFamily ?? base.fontFamily;
      next.fontSize = tb.fontSize ?? base.fontSize;
      next.fontWeight = String(tb.fontWeight ?? base.fontWeight ?? '400');
      if (typeof tb.fill === 'string') {
        next.fill = tb.fill;
      }
    } else if (base.kind === 'image') {
      const renderedW = obj.getScaledWidth();
      const renderedH = obj.getScaledHeight();
      if (base.objectFit !== 'contain' && renderedW > 0 && renderedH > 0) {
        next.width = Math.round(renderedW * 100) / 100;
        next.height = Math.round(renderedH * 100) / 100;
        next.scaleX = 1;
        next.scaleY = 1;
      }
    }

    updatedLayers.push(next);
  });

  return {
    ...page,
    layers: updatedLayers
  };
}

export function exportCanvasDataUrl(
  canvas: Canvas,
  format: 'png' | 'jpg',
  multiplier: number,
  logicalWidth?: number,
  logicalHeight?: number
): string {
  const active = canvas.getActiveObject();
  if (active) {
    canvas.discardActiveObject();
  }
  const prevZoom = canvas.getZoom();
  const prevWidth = canvas.getWidth();
  const prevHeight = canvas.getHeight();
  const prevVpt = [...canvas.viewportTransform] as [number, number, number, number, number, number];
  const targetWidth = logicalWidth && logicalWidth > 0 ? logicalWidth : Math.round(prevWidth / (prevZoom || 1));
  const targetHeight = logicalHeight && logicalHeight > 0 ? logicalHeight : Math.round(prevHeight / (prevZoom || 1));

  try {
    canvas.setDimensions({ width: targetWidth, height: targetHeight });
    canvas.setViewportTransform([1, 0, 0, 1, 0, 0]);
    canvas.setZoom(1);
    canvas.renderAll();

    return canvas.toDataURL({
      format: format === 'jpg' ? 'jpeg' : 'png',
      quality: format === 'jpg' ? 0.92 : 1,
      multiplier,
      enableRetinaScaling: false
    });
  } finally {
    canvas.setDimensions({ width: prevWidth, height: prevHeight });
    canvas.setViewportTransform(prevVpt);
    canvas.setZoom(prevZoom);
    if (active) {
      canvas.setActiveObject(active);
    }
    canvas.renderAll();
  }
}
