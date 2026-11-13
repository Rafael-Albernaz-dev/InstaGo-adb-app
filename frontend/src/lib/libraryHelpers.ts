import type { DatedOffer, LibraryAsset, LibraryFilter } from './types';

export interface ClassificationBadge {
  label: string;
  bg: string;
  fg: string;
  border: string;
}

const BADGES: Record<string, ClassificationBadge> = {
  arte_unica: {
    label: 'Oficial',
    bg: '#e6f2df',
    fg: '#254a18',
    border: '#9cc486'
  },
  capa_auxiliar: {
    label: 'Capa extra',
    bg: '#fef4db',
    fg: '#6b4d08',
    border: '#dfc174'
  },
  cena: {
    label: 'Cena / Foto',
    bg: '#e5eff7',
    fg: '#1b4965',
    border: '#9fc0da'
  },
  marca: {
    label: 'Marca / Logo',
    bg: '#eef4dd',
    fg: '#374f1e',
    border: '#b8cc82'
  },
  capa_editorial: {
    label: 'Capa editorial',
    bg: '#f2ebf8',
    fg: '#4b2b6b',
    border: '#c7b0dc'
  },
  fonte: {
    label: 'Fonte',
    bg: '#ecefe8',
    fg: '#394235',
    border: '#bec7b7'
  },
  historico: {
    label: 'Rascunho antigo',
    bg: '#fbeaea',
    fg: '#7a2520',
    border: '#dfaaaa'
  },
  referencia: {
    label: 'Referência',
    bg: '#edf0ee',
    fg: '#38463c',
    border: '#bdc6c0'
  }
};

export function getClassificationBadge(classification: string): ClassificationBadge {
  return (
    BADGES[classification] ?? {
      label: classification || 'Item',
      bg: '#f1f4ea',
      fg: '#536449',
      border: '#d5ddcc'
    }
  );
}

export function formatBRL(cents: number): string {
  const value = (cents / 100).toFixed(2);
  const [intPart, decPart] = value.split('.');
  const formattedInt = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
  return `R$ ${formattedInt},${decPart}`;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export function filterAssets(assets: LibraryAsset[], filter: LibraryFilter): LibraryAsset[] {
  const q = (filter.query ?? '').trim().toLowerCase();
  const cls = filter.classification ?? 'all';
  const fmt = filter.format ?? 'all';
  return assets.filter((asset) => {
    if (cls !== '' && cls !== 'all' && asset.classification !== cls) return false;
    if (fmt !== '' && fmt !== 'all' && asset.format !== fmt) return false;
    if (!q) return true;
    return (
      asset.title.toLowerCase().includes(q) ||
      asset.fileName.toLowerCase().includes(q) ||
      asset.caption.toLowerCase().includes(q) ||
      asset.pieceId.toLowerCase().includes(q) ||
      asset.subCategory.toLowerCase().includes(q)
    );
  });
}

export function offerCategoryLabel(category: string): string {
  switch (category) {
    case 'internet_fibra':
      return 'Fibra';
    case 'internet_radio':
      return 'Rádio';
    case 'combo':
      return 'Combos';
    case 'ruralcam':
      return 'RuralCam';
    default:
      return category;
  }
}

export function groupOffersByCategory(offers: DatedOffer[]): Array<{ category: string; label: string; items: DatedOffer[] }> {
  const order = ['internet_fibra', 'internet_radio', 'combo', 'ruralcam'];
  const groups = new Map<string, DatedOffer[]>();
  for (const o of offers) {
    const list = groups.get(o.category) ?? [];
    list.push(o);
    groups.set(o.category, list);
  }
  const result: Array<{ category: string; label: string; items: DatedOffer[] }> = [];
  for (const cat of order) {
    const items = groups.get(cat);
    if (items && items.length > 0) {
      result.push({ category: cat, label: offerCategoryLabel(cat), items });
      groups.delete(cat);
    }
  }
  for (const [cat, items] of groups.entries()) {
    result.push({ category: cat, label: offerCategoryLabel(cat), items });
  }
  return result;
}
