import { afterEach, describe, expect, it, vi } from 'vitest';
import { POWER_FLASH_MS, PowerFlashes, powerRingSize } from './power_flash';

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

describe('the power report hover ring', () => {
  it('is three quarters of its first size (28px floor, 450 tree units)', () => {
    expect(powerRingSize(3)).toBe(0.75 * (450 / 3));
    expect(powerRingSize(100)).toBe(0.75 * 28);
  });
});
