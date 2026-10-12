// @vitest-environment jsdom

import { describe, expect, it, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import CompareCalcsHeader from './CompareCalcsHeader.svelte';
import type { CalcsHeaderSide } from './compare';
import type { calculator } from '$lib/types';

afterEach(cleanup);

const controls = (o: Partial<calculator.MainSkillControls>) =>
  ({
    Skills: ['Arc'],
    SkillEnabled: true,
    MainActiveSkill: 1,
    PartShown: false,
    StageShown: false,
    MineShown: false,
    MinionShown: false,
    MinionSkillShown: false,
    ...o
  }) as calculator.MainSkillControls;
const side = (name: string, o: Partial<CalcsHeaderSide>): CalcsHeaderSide => ({
  name,
  groups: ['Arc', 'Orb of Storms'],
  skillNumber: 1,
  buffMode: 'EFFECTIVE',
  info: [{ label: 'Aura/Buff Skills', text: '' }],
  ...o
});

// CompareTab.lua:490-718, 4480-4608
describe('the Compare › Calcs header', () => {
  it("shows each build's controls and writes the side's own", () => {
    const calls: unknown[][] = [];
    const { getByRole } = render(CompareCalcsHeader, {
      sides: [
        side('Mine', { controls: controls({ StageShown: true, StageCount: '3' }), info: [{ label: 'Aura/Buff Skills', text: 'Wrath' }] }),
        side('Theirs', { skillNumber: 2, showMinion: false, controls: controls({ PartShown: true, Parts: ['One', 'Two'], SkillPart: 2 }) })
      ],
      oncontrol: (...a: unknown[]) => calls.push(a)
    });
    const a = within(getByRole('group', { name: 'Mine calcs controls' }));
    const b = within(getByRole('group', { name: 'Theirs calcs controls' }));
    expect(a.getByRole('textbox', { name: 'Stages' })).toHaveProperty('value', '3');
    expect(a.queryByRole('combobox', { name: 'Skill Part' })).toBeNull();
    expect(a.getByLabelText('Aura/Buff Skills').textContent).toBe('Aura/Buff Skills: Wrath');
    expect(b.getByRole('combobox', { name: 'Socket Group' })).toHaveProperty('value', '1');
    expect(b.getByRole('combobox', { name: 'Skill Part' })).toHaveProperty('value', '1');
    expect(a.queryByRole('checkbox', { name: 'Show Minion Stats' })).toBeNull();

    void fireEvent.change(b.getByRole('combobox', { name: 'Socket Group' }), { target: { value: '0' } });
    void fireEvent.change(a.getByRole('textbox', { name: 'Stages' }), { target: { value: '' } });
    void fireEvent.change(b.getByRole('combobox', { name: 'Calc Mode' }), { target: { value: 'COMBAT' } });
    void fireEvent.click(b.getByRole('checkbox', { name: 'Show Minion Stats' }));
    expect(calls).toEqual([
      [1, 'group', 1],
      [0, 'StageCount', undefined],
      [1, 'mode', 'COMBAT'],
      [1, 'showMinion', true]
    ]);
  });

  // CompareTab.lua:4556-4593, CompareCalcsHelpers.lua:355-465
  it("hovering a buff line shows that side's breakdown panel", async () => {
    const asked: unknown[][] = [];
    const rows = [{ Value: '+20 base', Name: 'Life', Source: 'Skill', SourceName: 'Vitality' }];
    const { getByRole, container } = render(CompareCalcsHeader, {
      sides: [
        side('Mine', {
          info: [
            { label: 'Aura/Buff Skills', text: 'Wrath' },
            { label: 'Combat Buffs', text: 'Onslaught' }
          ]
        }),
        side('Theirs', { info: [{ label: 'Aura/Buff Skills', text: 'Vitality' }] })
      ],
      oncontrol: () => {},
      breakdown: (s: 0 | 1, name: string) => (asked.push([s, name]), Promise.resolve(rows))
    });
    const theirs = within(getByRole('group', { name: 'Theirs calcs controls' }));
    await fireEvent.mouseEnter(theirs.getByRole('button', { name: 'Aura/Buff Skills' }));
    const tip = await vi.waitFor(() => {
      const t = container.ownerDocument.querySelector('#compare-skill-breakdown');
      if (!t) throw new Error('no panel');
      return t;
    });
    expect([...tip.querySelectorAll('tr')].map((r) => [...r.children].map((c) => c.textContent))).toEqual([
      ['Value', 'Stat', 'Source', 'Source Name'],
      ['+20 base', 'Life', 'Skill', 'Vitality']
    ]);
    expect(asked).toEqual([[1, 'SkillBuffs']]);
    const mine = within(getByRole('group', { name: 'Mine calcs controls' }));
    expect(mine.queryByRole('button', { name: 'Combat Buffs' })).toBeNull();
  });
});
