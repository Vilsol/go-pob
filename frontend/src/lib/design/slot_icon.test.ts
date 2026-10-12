import { describe, expect, it } from 'vitest';
import { slotIcon } from './slot_icon';

// SkillListControl.lua:234-260
describe('slotIcon', () => {
  it('maps a slot to its icon, a bow / quiver / shield to theirs', () => {
    expect(slotIcon('Helmet', {})).toBe('icon_helmet.png');
    expect(slotIcon('Ring 3', {})).toBe('icon_ring_right.png');
    expect(slotIcon('Weapon 1', { 'Weapon 1': 'Bow' })).toBe('icon_bow.png');
    expect(slotIcon('Weapon 1', { 'Weapon 1': 'Staff' })).toBe('icon_weapon.png');
    expect(slotIcon('Weapon 2', { 'Weapon 2': 'Quiver' })).toBe('icon_quiver.png');
    expect(slotIcon('Weapon 2 Swap', { 'Weapon 2 Swap': 'Shield' })).toBe('icon_shield_swap.png');
  });
  it('shows none where slot_map has no entry', () => {
    expect(slotIcon('Weapon 1 Swap', { 'Weapon 1 Swap': 'Bow' })).toBeUndefined();
    expect(slotIcon('Weapon 2 Swap', { 'Weapon 2 Swap': 'Quiver' })).toBeUndefined();
    expect(slotIcon('', {})).toBeUndefined();
    expect(slotIcon('Flask 1', {})).toBeUndefined();
  });
});
