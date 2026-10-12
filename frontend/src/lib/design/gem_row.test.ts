// @vitest-environment jsdom
// SkillsTab.lua:1007-1012

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import GemRow from './GemRow.svelte';

afterEach(cleanup);

const base = { n: 1, name: 'Fire', level: 20, quality: 0, tone: 'int' as const, socket: 'int' as const, onlevel: () => {}, onquality: () => {} };

describe('GemRow', () => {
  it("shows the gem's error message in ^1", () => {
    const { container } = render(GemRow, { ...base, error: "Ambiguous gem name 'Fire': matches 'Fireball', 'Firestorm'" });
    const msg = container.querySelector('[data-gem-error]') as HTMLElement;
    expect(msg?.textContent).toBe("Ambiguous gem name 'Fire': matches 'Fireball', 'Firestorm'");
    expect((msg.querySelector('[data-pob-color]') as HTMLElement).style.getPropertyValue('--tone')).toBe('#FF0000');
  });

  it('draws nothing without one', () => {
    const { container } = render(GemRow, base);
    expect(container.querySelector('[data-gem-error]')).toBeNull();
  });
});

describe('GemRow global effects', () => {
  it('draws an "Enable <effect>" box per Vaal granted effect and reports its toggle', () => {
    const calls: [number, boolean][] = [];
    const { getByRole, queryByRole } = render(GemRow, {
      ...base,
      name: 'Vaal Fireball',
      globals: [
        { label: 'Vaal Fireball', on: true },
        { label: 'Fireball', on: false }
      ],
      onglobal: (i: number, on: boolean) => calls.push([i, on])
    });
    const vaal = getByRole('checkbox', { name: 'Enable Vaal Fireball' }) as HTMLInputElement;
    const plain = getByRole('checkbox', { name: 'Enable Fireball' }) as HTMLInputElement;
    expect([vaal.checked, plain.checked]).toEqual([true, false]);
    plain.click();
    expect(calls).toEqual([[1, true]]);
    expect(queryByRole('checkbox', { name: /^Enable $/ })).toBeNull();
  });

  it('skips a slot with no label', () => {
    const { queryAllByRole } = render(GemRow, {
      ...base,
      globals: [
        { label: '', on: false },
        { label: 'Fireball', on: true }
      ],
      onglobal: () => {}
    });
    expect(queryAllByRole('checkbox', { name: /^Enable / })).toHaveLength(1);
  });
});
