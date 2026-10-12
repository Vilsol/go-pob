import { SvelteMap } from 'svelte/reactivity';

export const POWER_FLASH_MS = 1000;

export class PowerFlashes {
  readonly flashing = new SvelteMap<number, number>();
  #seen = new Set<number>();
  #timers = new Map<number, ReturnType<typeof setTimeout>>();
  #arrival = 0;

  arrive(ids: Iterable<number>) {
    const k = ++this.#arrival;
    for (const id of ids) {
      if (this.#seen.has(id)) continue;
      this.#seen.add(id);
      this.flashing.set(id, k);
      clearTimeout(this.#timers.get(id));
      this.#timers.set(
        id,
        setTimeout(() => {
          this.#timers.delete(id);
          this.flashing.delete(id);
        }, POWER_FLASH_MS)
      );
    }
  }

  reset() {
    for (const t of this.#timers.values()) clearTimeout(t);
    this.#timers.clear();
    this.#seen.clear();
    this.flashing.clear();
  }
}
