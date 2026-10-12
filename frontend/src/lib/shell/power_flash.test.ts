import { afterEach, describe, expect, it, vi } from 'vitest';
import { POWER_FLASH_MS, PowerFlashes } from './power_flash';

afterEach(() => vi.useRealTimers());

describe('the power report flashes (UI_ISSUES Tree 4)', () => {
  it('flashes every node that completed in one poll, each on its own timer', () => {
    vi.useFakeTimers();
    const f = new PowerFlashes();
    f.arrive([1, 2]);
    expect([...f.flashing.keys()]).toEqual([1, 2]);
    vi.advanceTimersByTime(POWER_FLASH_MS / 2);
    f.arrive([1, 2, 3]);
    expect([...f.flashing.keys()]).toEqual([1, 2, 3]);
    vi.advanceTimersByTime(POWER_FLASH_MS / 2);
    expect([...f.flashing.keys()]).toEqual([3]);
    vi.advanceTimersByTime(POWER_FLASH_MS / 2);
    expect(f.flashing.size).toBe(0);
  });

  it('flashes the nodes again after a reset (a new report)', () => {
    vi.useFakeTimers();
    const f = new PowerFlashes();
    f.arrive([1]);
    f.reset();
    expect(f.flashing.size).toBe(0);
    f.arrive([1]);
    expect([...f.flashing.keys()]).toEqual([1]);
  });
});
