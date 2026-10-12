// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import CompareConfig from './CompareConfig.svelte';

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

afterEach(cleanup);

const zero = [0, 0, 0];
const config = [
  {
    key: 'conditionStationary',
    section: 'General',
    label: 'Are you always stationary?',
    a: 'No',
    b: 'Yes',
    diff: true,
    impactA: [5, 0, 0],
    impactB: zero,
    kind: 'check' as const,
    aValue: false,
    bValue: true
  },
  {
    key: 'enemyIsBoss',
    section: 'Enemy Stats',
    label: 'Is the enemy a Boss?',
    a: 'Pinnacle',
    b: 'Pinnacle',
    diff: false,
    impactA: zero,
    impactB: zero,
    kind: 'list' as const,
    options: [
      { value: 'None', label: 'No' },
      { value: 'Pinnacle', label: 'Pinnacle' }
    ],
    aValue: 'Pinnacle',
    bValue: 'Pinnacle'
  },
  {
    key: 'enemyLevel',
    section: 'Enemy Stats',
    label: 'Enemy level:',
    a: '83',
    b: '84',
    diff: true,
    impactA: [1, 0, 0],
    impactB: zero,
    kind: 'number' as const,
    aValue: 83,
    bValue: 84
  }
];
const props = {
  config,
  setA: 'Default',
  setB: 'Default',
  metric: 'dps' as const,
  baseA: 100,
  baseB: 100,
  onmetric: () => {},
  onuse: () => {},
  oncopy: () => {}
};

describe('Compare › Config view', () => {
  // CompareTab.lua:2099-2135, 4953-4964
  it('groups the options by section, the differences first and marked', () => {
    const { getByRole } = render(CompareConfig, props);
    const enemy = getByRole('rowgroup', { name: 'Enemy Stats' });
    expect(enemy.textContent).toContain('Enemy Stats  (1 diff)');
    const rows = within(enemy).getAllByRole('row');
    expect(rows.map((r) => r.getAttribute('data-diff'))).toEqual(['true', 'false']);
    expect(within(rows[0]).getAllByRole('cell')[0].textContent).toContain('Enemy level:');
  });

  // CompareTab.lua:1128-1160
  it("edits either build's option through its own control", async () => {
    const set: unknown[] = [];
    const { getByRole } = render(CompareConfig, { ...props, onset: (...a: unknown[]) => set.push(a) });
    await fireEvent.click(getByRole('checkbox', { name: 'Are you always stationary? (B)' }));
    const level = getByRole('textbox', { name: 'Enemy level: (A)' });
    await fireEvent.change(level, { target: { value: '80' } });
    await fireEvent.change(getByRole('combobox', { name: 'Is the enemy a Boss? (B)' }), { target: { value: 'None' } });
    expect(set).toEqual([
      ['b', 'conditionStationary', false],
      ['a', 'enemyLevel', 80],
      ['b', 'enemyIsBoss', 'None']
    ]);
  });

  // CompareTab.lua:4991-4994
  it('says so when the search leaves no option', async () => {
    const { getByRole, getByText } = render(CompareConfig, props);
    await fireEvent.input(getByRole('searchbox', { name: 'Search config options' }), { target: { value: 'zzz' } });
    expect(getByText('No configuration options to display.')).toBeTruthy();
  });
});
