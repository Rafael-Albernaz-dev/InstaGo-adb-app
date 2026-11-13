import type { AgentAnnotation, DocumentModel, Layer, Page } from './types';

export type HorizontalAlign = 'left' | 'center' | 'right';
export type VerticalAlign = 'top' | 'middle' | 'bottom';
export type LayerAlignment = HorizontalAlign | VerticalAlign;

export function createShortId(prefix = 'id'): string {
  const rand = Math.random().toString(36).slice(2, 8);
  const ts = Date.now().toString(36).slice(-4);
  return `${prefix}-${ts}${rand}`;
}

export function alignLayerOnPage(
  layer: Layer,
  workspaceWidth: number,
  workspaceHeight: number,
  alignment: LayerAlignment,
  margin = 32
): Layer {
  const effectiveW = (layer.width || 0) * (layer.scaleX || 1);
  const effectiveH = (layer.height || 0) * (layer.scaleY || 1);
  const next = { ...layer };

  switch (alignment) {
    case 'left':
      next.x = margin;
      break;
    case 'center':
      next.x = Math.round(((workspaceWidth - effectiveW) / 2) * 100) / 100;
      break;
    case 'right':
      next.x = Math.round((workspaceWidth - effectiveW - margin) * 100) / 100;
      break;
    case 'top':
      next.y = margin;
      break;
    case 'middle':
      next.y = Math.round(((workspaceHeight - effectiveH) / 2) * 100) / 100;
      break;
    case 'bottom':
      next.y = Math.round((workspaceHeight - effectiveH - margin) * 100) / 100;
      break;
  }
  return next;
}

export function reindexPages(pages: Page[]): Page[] {
  return pages.map((p, idx) => ({
    ...p,
    index: idx
  }));
}

export function createBlankPage(index: number, pageId = createShortId('page')): Page {
  return {
    id: pageId,
    index,
    title: `Página ${String(index + 1).padStart(2, '0')}`,
    background: '#F6F7EE',
    layers: [],
    annotations: []
  };
}

export function duplicatePage(page: Page, pageId = createShortId('page')): Page {
  const clonedLayers: Layer[] = page.layers.map((l, idx) => ({
    ...l,
    id: createShortId(`lyr-${idx + 1}`)
  }));
  return {
    id: pageId,
    index: page.index + 1,
    title: `${page.title} (cópia)`,
    background: page.background,
    layers: clonedLayers,
    annotations: []
  };
}

export function movePage(pages: Page[], index: number, delta: -1 | 1): Page[] {
  const target = index + delta;
  if (index < 0 || index >= pages.length || target < 0 || target >= pages.length) {
    return pages;
  }
  const copy = [...pages];
  const [item] = copy.splice(index, 1);
  copy.splice(target, 0, item);
  return reindexPages(copy);
}

export function removePageAt(pages: Page[], index: number): Page[] {
  if (pages.length <= 1 || index < 0 || index >= pages.length) {
    return pages;
  }
  const copy = pages.filter((_, idx) => idx !== index);
  return reindexPages(copy);
}

export function createAnnotation(
  kind: 'pin' | 'rect',
  x: number,
  y: number,
  width = 0,
  height = 0,
  layerId = ''
): AgentAnnotation {
  return {
    id: createShortId('mark'),
    kind,
    x: Math.max(0, Math.round(x * 10) / 10),
    y: Math.max(0, Math.round(y * 10) / 10),
    width: kind === 'rect' ? Math.max(16, Math.round(width * 10) / 10) : undefined,
    height: kind === 'rect' ? Math.max(16, Math.round(height * 10) / 10) : undefined,
    layerId: layerId || undefined,
    note: '',
    color: '#D9531E',
    resolved: false,
    createdAt: new Date().toISOString()
  };
}

export function formatDocStatusLabel(status: string): string {
  switch (status) {
    case 'approved':
      return 'Aprovada';
    case 'in_review':
      return 'Em revisão';
    default:
      return 'Rascunho';
  }
}

export function summarizeVersionChanges(docA: DocumentModel, docB: DocumentModel): string[] {
  const notes: string[] = [];
  if (docA.title !== docB.title) {
    notes.push(`Título alterado: "${docA.title}" → "${docB.title}"`);
  }
  if ((docA.pages?.length ?? 0) !== (docB.pages?.length ?? 0)) {
    notes.push(`Quantidade de páginas: ${docA.pages.length} → ${docB.pages.length}`);
  }
  const sharedPages = Math.min(docA.pages.length, docB.pages.length);
  for (let i = 0; i < sharedPages; i++) {
    const pa = docA.pages[i];
    const pb = docB.pages[i];
    if (pa.background !== pb.background) {
      notes.push(`Página ${i + 1}: cor de fundo ${pa.background} → ${pb.background}`);
    }
    if (pa.layers.length !== pb.layers.length) {
      notes.push(`Página ${i + 1}: camadas ${pa.layers.length} → ${pb.layers.length}`);
    } else {
      const changed = pa.layers.filter((la, idx) => JSON.stringify(la) !== JSON.stringify(pb.layers[idx]));
      if (changed.length > 0) {
        notes.push(`Página ${i + 1}: ${changed.length} elemento(s) modificado(s)`);
      }
    }
  }
  if ((docA.caption ?? '') !== (docB.caption ?? '')) {
    notes.push('Legenda modificada');
  }
  if ((docA.ctaDestination ?? '') !== (docB.ctaDestination ?? '') || (docA.ctaText ?? '') !== (docB.ctaText ?? '')) {
    notes.push('Chamada ou link CTA modificado');
  }
  if ((docA.offerId ?? '') !== (docB.offerId ?? '')) {
    notes.push(`Oferta vinculada: "${docA.offerId || 'nenhuma'}" → "${docB.offerId || 'nenhuma'}"`);
  }
  if (notes.length === 0) {
    notes.push('Nenhuma diferença estrutural detectada entre essas duas versões.');
  }
  return notes;
}
