import { describe, expect, it } from 'vitest';
import { identityOf } from './identity';

describe('identityOf', () => {
  it('is the same for one object, different for an equal one, 0 for none', () => {
    const a = { Output: { Life: 1 } };
    expect(identityOf(a)).toBe(identityOf(a));
    expect(identityOf({ Output: { Life: 1 } })).not.toBe(identityOf(a));
    expect(identityOf(undefined)).toBe(0);
    expect(identityOf(null)).toBe(0);
    expect(identityOf(a)).toBeGreaterThan(0);
  });
});
