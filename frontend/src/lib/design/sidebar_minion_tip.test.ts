// @vitest-environment jsdom
// Build.lua:544-551, ItemsTab.lua:4138-4147

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import type { calculator } from '$lib/types';
import Sidebar from './Sidebar.svelte';

afterEach(cleanup);

const controls = (minion: number) =>
  ({
    SkillSet: 0,
    SocketGroup: 0,
    GemIndex: 1,
    Skills: ['Raise Spectre', 'Animate Guardian'],
    SkillEnabled: true,
    MainActiveSkill: 2,
    MinionShown: true,
    MinionEnabled: true,
    Minions: [
      { Label: 'Default Item Set', MinionID: '', ItemSetID: 1, Tooltip: ['^7Weapon 1: ^xFFFF77New Item, Void Sceptre'] },
      { Label: 'AG', MinionID: '', ItemSetID: 2, Tooltip: ['^7Weapon 1: ^xAF6025Dying Breath, Iron Staff', '^7Helmet: ^xAF6025Southbound, Soldier Gloves'] },
      { Label: 'Empty', MinionID: '', ItemSetID: 3 },
      { Label: 'Tip', MinionID: '', ItemSetID: 4, Tooltip: ['^7Boots: ^xAF6025Wake of Destruction, Mesh Boots', '', '^x80A080Tip: You can drag items.'] }
    ],
    Minion: minion
  }) as unknown as calculator.MainSkillControls;

const mount = (minion: number) =>
  render(Sidebar, { props: { sidebar: { sections: [], warnings: [] }, socketGroups: ['Group'], mainSocketGroup: 0, controls: controls(minion) } });

describe('Sidebar minion item-set tooltip', () => {
  it('shows the selected set on hover and hides on leave', async () => {
    const { getByRole, queryByRole } = mount(2);
    const select = getByRole('combobox', { name: 'Minion' });
    expect(queryByRole('tooltip')).toBeNull();
    const wrap = select.closest('[data-minion-select]')!;
    await fireEvent.mouseEnter(wrap);
    const tip = getByRole('tooltip');
    expect(tip.textContent).toBe('Weapon 1: Dying Breath, Iron StaffHelmet: Southbound, Soldier Gloves');
    expect((tip.querySelectorAll('[data-pob-color]')[1] as HTMLElement).style.getPropertyValue('--tone')).toBe('#AF6025');
    expect(select.getAttribute('aria-describedby')).toBe(tip.id);
    await fireEvent.mouseLeave(wrap);
    expect(queryByRole('tooltip')).toBeNull();
  });

  it('shows on focus', async () => {
    const { getByRole } = mount(1);
    await fireEvent.focusIn(getByRole('combobox', { name: 'Minion' }));
    expect(getByRole('tooltip').textContent).toBe('Weapon 1: New Item, Void Sceptre');
  });

  it('draws nothing for a set without items', async () => {
    const { getByRole, queryByRole } = mount(3);
    await fireEvent.focusIn(getByRole('combobox', { name: 'Minion' }));
    expect(queryByRole('tooltip')).toBeNull();
  });

  // Build.lua:548-549
  it('draws the separator and the tip line', async () => {
    const { getByRole } = mount(4);
    await fireEvent.focusIn(getByRole('combobox', { name: 'Minion' }));
    const tip = getByRole('tooltip');
    expect(tip.querySelectorAll('hr')).toHaveLength(1);
    expect(tip.textContent).toBe('Boots: Wake of Destruction, Mesh BootsTip: You can drag items.');
  });
});
