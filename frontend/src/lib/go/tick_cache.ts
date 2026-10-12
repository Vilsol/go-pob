import type { Outputs, TickWarnings } from '../custom_types';

export type GroupDisplay = { label: string; slotEnabled: boolean };

export type TickCacheEntry = {
  payload: Outputs;
  groups: GroupDisplay[];
  warnings?: Omit<TickWarnings, 'seq'>;
};

export class TickCache {
  private entries = new Map<string, TickCacheEntry>();

  constructor(readonly max = 10) {}

  get size() {
    return this.entries.size;
  }

  get(key: string): TickCacheEntry | undefined {
    const entry = this.entries.get(key);
    if (entry) {
      this.entries.delete(key);
      this.entries.set(key, entry);
    }
    return entry;
  }

  set(key: string, entry: TickCacheEntry) {
    this.entries.delete(key);
    this.entries.set(key, entry);
    while (this.entries.size > this.max) {
      this.entries.delete(this.entries.keys().next().value!);
    }
  }

  clear() {
    this.entries.clear();
  }
}
