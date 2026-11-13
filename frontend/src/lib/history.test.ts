import { describe, expect, it } from 'vitest';
import { HistoryStack } from './history';

describe('HistoryStack', () => {
  it('supports push, undo, redo, and deduplicates identical states', () => {
    const h = new HistoryStack({ value: 1 }, 10);
    expect(h.canUndo).toBe(false);
    expect(h.canRedo).toBe(false);

    h.push({ value: 1 });
    expect(h.canUndo).toBe(false);

    h.push({ value: 2 });
    h.push({ value: 3 });
    expect(h.canUndo).toBe(true);
    expect(h.state.value).toBe(3);

    expect(h.undo()?.value).toBe(2);
    expect(h.canRedo).toBe(true);
    expect(h.undo()?.value).toBe(1);
    expect(h.canUndo).toBe(false);

    expect(h.redo()?.value).toBe(2);
    h.push({ value: 99 });
    expect(h.canRedo).toBe(false);
    expect(h.state.value).toBe(99);
  });
});
