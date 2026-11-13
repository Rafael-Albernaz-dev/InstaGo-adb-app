import { describe, expect, it } from 'vitest';
import { filterAssets, formatBRL, getClassificationBadge, groupOffersByCategory } from './libraryHelpers';
import type { DatedOffer, LibraryAsset } from './types';

function makeAsset(overrides: Partial<LibraryAsset>): LibraryAsset {
  return {
    id: 'a1',
    companyId: 'rural-conecta',
    relPath: 'artes-unicas/timeline/02-combo/02-combo-02.jpg',
    thumbRelPath: '',
    sourceRelPath: '03-banners-prontos/timeline/02-combo/02-combo-02.jpg',
    fileName: '02-combo-02.jpg',
    mimeType: 'image/jpeg',
    mediaKind: 'image',
    classification: 'arte_unica',
    subCategory: 'timeline',
    format: 'feed',
    title: '100 megas na Black Friday',
    caption: 'Combo Completo por R$ 99,99',
    pieceId: '02-combo',
    pageNumber: 2,
    width: 1080,
    height: 1350,
    bytes: 120000,
    sha256: 'abc123',
    isUniqueApproved: true,
    isEditableModel: true,
    linkedDocumentId: 'seed-v5-combo-02',
    metadataJson: '{}',
    importedAt: '2026-11-13T09:00:00Z',
    assetUrl: '/library-assets/artes-unicas/timeline/02-combo/02-combo-02.jpg',
    thumbUrl: '/library-assets/artes-unicas/timeline/02-combo/02-combo-02.jpg',
    ...overrides
  };
}

describe('libraryHelpers', () => {
  it('formats BRL currency from cents', () => {
    expect(formatBRL(9999)).toBe('R$ 99,99');
    expect(formatBRL(24990)).toBe('R$ 249,90');
    expect(formatBRL(125000)).toBe('R$ 1.250,00');
  });

  it('distinguishes classification badges cleanly', () => {
    expect(getClassificationBadge('arte_unica').label).toBe('Oficial');
    expect(getClassificationBadge('capa_auxiliar').label).toBe('Capa extra');
    expect(getClassificationBadge('historico').label).toBe('Rascunho antigo');
  });

  it('filters assets by classification, format, and search text', () => {
    const items: LibraryAsset[] = [
      makeAsset({ id: 'u1', classification: 'arte_unica', format: 'feed', title: 'Combo 100 megas' }),
      makeAsset({ id: 'aux1', classification: 'capa_auxiliar', format: 'feed', title: 'Capa auxiliar' }),
      makeAsset({ id: 'h1', classification: 'historico', format: 'other', title: 'Rascunho antigo' })
    ];

    expect(filterAssets(items, { classification: 'arte_unica' })).toHaveLength(1);
    expect(filterAssets(items, { classification: 'all', query: 'rascunho' })).toHaveLength(1);
    expect(filterAssets(items, { format: 'feed' })).toHaveLength(2);
  });

  it('groups offers by category preserving expected display order', () => {
    const offers: DatedOffer[] = [
      {
        id: 'o2',
        companyId: 'rural-conecta',
        category: 'ruralcam',
        title: 'Somente câmeras',
        promoPriceCents: 3990,
        promoPeriod: 'Mensalidade',
        afterPriceCents: 3990,
        installFeeCents: 25000,
        speedSummary: '',
        servicesSummary: 'Câmeras',
        restrictions: '',
        campaignName: 'BF',
        validUntil: '2026-11-30',
        checkedAt: '2026-11-11',
        sourceRef: 'doc',
        readOnly: true
      },
      {
        id: 'o1',
        companyId: 'rural-conecta',
        category: 'internet_fibra',
        title: 'Fibra 100 megas',
        promoPriceCents: 9999,
        promoPeriod: '3 meses',
        afterPriceCents: 18900,
        installFeeCents: 24990,
        speedSummary: '100 megas',
        servicesSummary: 'Fibra',
        restrictions: '',
        campaignName: 'BF',
        validUntil: '2026-11-30',
        checkedAt: '2026-11-11',
        sourceRef: 'doc',
        readOnly: true
      }
    ];

    const grouped = groupOffersByCategory(offers);
    expect(grouped.map((g) => g.category)).toEqual(['internet_fibra', 'ruralcam']);
  });
});
