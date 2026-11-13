export class HistoryStack<T> {
  private past: T[] = [];
  private future: T[] = [];
  private current: T;
  private readonly limit: number;

  constructor(initial: T, limit = 50) {
    this.current = structuredClone(initial);
    this.limit = limit;
  }

  get state(): T {
    return structuredClone(this.current);
  }

  get canUndo(): boolean {
    return this.past.length > 0;
  }

  get canRedo(): boolean {
    return this.future.length > 0;
  }

  push(next: T): void {
    const serializedCurrent = JSON.stringify(this.current);
    const serializedNext = JSON.stringify(next);
    if (serializedCurrent === serializedNext) {
      return;
    }
    this.past.push(this.current);
    if (this.past.length > this.limit) {
      this.past.shift();
    }
    this.current = structuredClone(next);
    this.future = [];
  }

  undo(): T | null {
    const prev = this.past.pop();
    if (!prev) {
      return null;
    }
    this.future.push(this.current);
    this.current = prev;
    return this.state;
  }

  redo(): T | null {
    const next = this.future.pop();
    if (!next) {
      return null;
    }
    this.past.push(this.current);
    this.current = next;
    return this.state;
  }

  reset(initial: T): void {
    this.past = [];
    this.future = [];
    this.current = structuredClone(initial);
  }
}
