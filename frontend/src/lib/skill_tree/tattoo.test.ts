import { describe, expect, it } from 'vitest';
import { tattooEditKind } from './tattoo';

// PassiveTreeView.lua:1646-1656
describe('tattooEditKind', () => {
  it('offers a tattoo on attribute smalls, +30 notables and keystones, a runegraft on masteries', () => {
    expect(tattooEditKind({ k: 's', name: 'Strength' }, false)).toBe('tattoo');
    expect(tattooEditKind({ k: 'n', name: 'Might', stats: ['+30 to Strength'] }, false)).toBe('tattoo');
    expect(tattooEditKind({ k: 'k', name: 'Iron Reflexes' }, false)).toBe('tattoo');
    expect(tattooEditKind({ k: 'm', name: 'Life Mastery' }, false)).toBe('runegraft');
  });
  it('offers nothing on other nodes unless they carry one', () => {
    expect(tattooEditKind({ k: 's', name: 'Life' }, false)).toBe('');
    expect(tattooEditKind({ k: 'n', name: 'Heart of Oak', stats: ['+20 to Strength'] }, false)).toBe('');
    expect(tattooEditKind({ k: 'j', name: 'Jewel Socket' }, false)).toBe('');
    expect(tattooEditKind({ k: 's', name: 'Life' }, true)).toBe('tattoo');
  });
});
