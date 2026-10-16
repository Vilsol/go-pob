// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { colorCodes } from '$lib/display/colors';
import type { CalcsFixture } from './fixtures';
import CalcsPage from './CalcsPage.svelte';

afterEach(cleanup);

const calcs = (life: string, cooldown: string, regen: string): CalcsFixture => ({
  sections: [
    {
      id: 'Speed',
      title: 'Attack/Cast Rate',
      wide: false,
      col: 1,
      color: colorCodes.OFFENCE,
      applicable: true,
      blocks: [{ label: 'Attack/Cast Rate', rows: [{ label: 'Cooldown', cells: [{ text: cooldown, bd: 'Cooldown' }] }] }]
    },
    {
      id: 'Life',
      title: 'Life',
      wide: false,
      col: 2,
      color: colorCodes.LIFE,
      applicable: true,
      blocks: [
        {
          label: 'Life',
          rows: [
            { label: 'Total', cells: [{ text: life, bd: 'Life' }] },
            { label: 'Regen', cells: [{ text: regen }] }
          ]
        }
      ]
    },
    {
      id: 'DamageTaken',
      title: 'Damage Taken',
      wide: true,
      col: 1,
      color: colorCodes.DEFENCE,
      applicable: true,
      blocks: [
        {
          label: 'Damage Taken',
          rows: [
            { label: '', cells: ['Physical:', '^xB97123Fire:', '^x3F6DB3Cold:', '^xADAA47Lightning:', '^xD02090Chaos:'].map((text) => ({ text })) },
            { label: 'Taken', cells: ['1', '2', '3', '4', '5'].map((text) => ({ text })) }
          ]
        }
      ]
    }
  ],
  breakdowns: {}
});
const mount = (c: CalcsFixture) =>
  render(CalcsPage, { props: { calcs: c, socketGroups: ['Group'], mainSocketGroup: 0, onmaingroup: () => {}, oncontrol: () => {} } });
const section = (r: ReturnType<typeof mount>, name: string) => r.getByRole('region', { name });
const changed = (r: ReturnType<typeof mount>) =>
  Object.fromEntries(
    [...r.container.querySelectorAll('[data-change]')].map((e) => [e.closest('[data-key]')!.textContent!.split(/\d/)[0].trim(), e.getAttribute('data-change')])
  );

describe('Calcs colours', () => {
  it('draws each section box in its CalcSections colour (CalcSections.lua: Life = LIFE, Speed = OFFENCE)', () => {
    const r = mount(calcs('5,123', '1.200s', '10'));
    expect(section(r, 'Life').classList).toContain('tone-life');
    expect(section(r, 'Attack/Cast Rate').classList).toContain('tone-calc-off');
    expect(section(r, 'Damage Taken').classList).toContain('tone-calc-def');
  });

  it('draws each damage-type column head in its type colour, as theme tokens', () => {
    const r = mount(calcs('5,123', '1.200s', '10'));
    const heads = [...section(r, 'Damage Taken').querySelectorAll('[role=columnheader]')].slice(1);
    expect(heads.map((h) => [h.textContent, [...h.classList].find((c) => c.startsWith('tone-'))])).toEqual([
      ['Physical:', 'tone-normal'],
      ['Fire:', 'tone-stat-fire'],
      ['Cold:', 'tone-stat-cold'],
      ['Lightning:', 'tone-stat-light'],
      ['Chaos:', 'tone-stat-chaos']
    ]);
    expect(section(r, 'Damage Taken').querySelectorAll('[role=columnheader] [data-pob-color]')).toHaveLength(0);
  });

  it('tints the values that moved in the last recalculation, by their stat’s direction', async () => {
    const r = mount(calcs('5,123', '1.200s', '10'));
    expect(changed(r), 'nothing is tinted on the first calculation').toEqual({});
    await r.rerender({ calcs: calcs('5,400', '1.500s', '12') });
    expect(changed(r)).toEqual({ Total: 'better', Cooldown: 'worse', Regen: 'changed' });
    expect(r.container.querySelector('[data-change="better"]')!.classList).toContain('tone-better');
    expect(r.container.querySelector('[data-change="worse"]')!.classList).toContain('tone-worse');
    await r.rerender({ calcs: calcs('5,400', '1.500s', '12') });
    expect(changed(r), 'a recalculation that moves nothing clears the tints').toEqual({});
  });
});
