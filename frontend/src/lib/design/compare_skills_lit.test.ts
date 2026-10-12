// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import CompareSkills from './CompareSkills.svelte';
import { pairSkills } from './compare';
import type { SkillsFixture } from './fixtures';

afterEach(cleanup);

const gem = (name: string, support: boolean, imbued = false) => ({
  name,
  level: 20,
  quality: 0,
  enabled: true,
  support,
  attr: 'int' as const,
  count: 1,
  ...(imbued && { imbued })
});
const set = (gems: ReturnType<typeof gem>[]): SkillsFixture => ({
  activeSet: 0,
  sets: [
    {
      title: 's',
      groups: [
        { label: 'Empty', slot: '', enabled: true, inFull: true, count: 1, sockets: '', gems: [] },
        { label: 'Arc', slot: '', enabled: true, inFull: true, count: 1, sockets: '', gems }
      ]
    }
  ]
});

// CompareTab.lua:4139-4157, 4411-4427
describe('Compare › Skills gem highlight', () => {
  it('lights the supported and supporting gems of the hovered one, on each side', async () => {
    const A = set([gem('Arc', false), gem('Spell Echo Support', true), gem('Faster Casting Support', true)]);
    const B = set([gem('Arc', false), gem('Spell Echo Support', true), gem('More Duration Support', true, true)]);
    const { both, onlyA, onlyB } = pairSkills(A, B);
    expect(both[0].ga).toBe(1);
    const asked: string[] = [];
    const { getAllByRole, container } = render(CompareSkills, {
      both,
      onlyA,
      onlyB,
      supportpairs: (side: 'a' | 'b', group: number) => (asked.push(`${side}${group}`), Promise.resolve(side === 'a' ? ([[1, 0]] as [number, number][]) : []))
    });
    const rows = getAllByRole('row').filter((r) => r.querySelector('[role="cell"] [title]'));
    const lit = () => [...container.querySelectorAll('[data-lit]')].map((r) => r.querySelector('[title]')?.getAttribute('title'));
    await fireEvent.mouseEnter(rows.find((r) => r.textContent?.includes('Arc') && !r.textContent.includes('Support'))!);
    await waitFor(() => expect(lit()).toEqual(['Arc', 'Spell Echo Support', 'More Duration Support']));
    expect(asked.sort()).toEqual(['a1', 'b1']);
    await fireEvent.mouseLeave(rows[0]);
    await fireEvent.mouseEnter(rows.find((r) => r.textContent?.includes('Faster Casting'))!);
    await new Promise((r) => setTimeout(r, 10));
    expect(lit(), 'a gem alone lights nothing').toEqual([]);
  });
});
