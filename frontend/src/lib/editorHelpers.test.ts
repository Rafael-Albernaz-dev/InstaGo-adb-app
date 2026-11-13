import { describe, expect, it } from 'vitest';
import {
  alignLayerOnPage,
  duplicatePage,
  movePage,
  removePageAt,
  summarizeVersionChanges
} from './editorHelpers';
import type { DocumentModel, Layer, Page } from './types';

const sampleLayer: Layer = {
  id: 'txt-1',
  name: 'Título',
  kind: 'text',
  x: 10,
  y: 10,
  width: 200,
  height: 50,
  scaleX: 1,
  scaleY: 1,
  angle: 0,
  opacity: 1,
  visible: true,
  locked: false,
  zIndex: 1,
  text: 'Olá'
};

describe('editorHelpers', () => {
  it('aligns layers accurately on workspace dimensions', () => {
    const centered = alignLayerOnPage(sampleLayer, 600, 750, 'center');
    expect(centered.x).toBe(200);

    const middle = alignLayerOnPage(sampleLayer, 600, 750, 'middle');
    expect(middle.y).toBe(350);

    const right = alignLayerOnPage(sampleLayer, 600, 750, 'right', 30);
    expect(right.x).toBe(370);
  });

  it('duplicates, reorders, and removes pages without colliding layer IDs', () => {
    const page1: Page = {
      id: 'p1',
      index: 0,
      title: 'Capa',
      background: '#FFFFFF',
      layers: [sampleLayer],
      annotations: [{ id: 'm1', kind: 'pin', x: 50, y: 50, note: 'ajustar' }]
    };
    const dup = duplicatePage(page1, 'p2');
    expect(dup.id).toBe('p2');
    expect(dup.layers[0].id).not.toBe(page1.layers[0].id);
    expect(dup.annotations).toEqual([]);

    const moved = movePage([page1, { ...dup, index: 1 }], 1, -1);
    expect(moved.map((p) => p.id)).toEqual(['p2', 'p1']);
    expect(moved.map((p) => p.index)).toEqual([0, 1]);

    const trimmed = removePageAt(moved, 0);
    expect(trimmed).toHaveLength(1);
    expect(trimmed[0].id).toBe('p1');
    expect(trimmed[0].index).toBe(0);
  });

  it('summarizes version differences clearly', () => {
    const baseDoc: DocumentModel = {
      id: 'doc-1',
      schemaVersion: 1,
      title: 'Peça A',
      companyId: 'rural-conecta',
      offerRef: '',
      format: 'feed',
      workspaceWidth: 600,
      workspaceHeight: 750,
      exportWidth: 1080,
      exportHeight: 1350,
      exportScale: 1.8,
      revision: 1,
      status: 'draft',
      caption: 'Legenda 1',
      pages: [{ id: 'p1', index: 0, title: 'P1', background: '#FFFFFF', layers: [sampleLayer] }],
      createdAt: '',
      updatedAt: ''
    };
    const nextDoc: DocumentModel = {
      ...baseDoc,
      revision: 2,
      caption: 'Legenda 2',
      pages: [
        baseDoc.pages[0],
        { id: 'p2', index: 1, title: 'P2', background: '#FFFFFF', layers: [] }
      ]
    };
    const summary = summarizeVersionChanges(baseDoc, nextDoc);
    expect(summary).toContain('Quantidade de páginas: 1 → 2');
    expect(summary).toContain('Legenda modificada');
  });
});
