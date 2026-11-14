// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import SkillsPage from './SkillsPage.svelte';
import Dropdown from './Dropdown.svelte';
import SetManager from './SetManager.svelte';
import { colourLabelSkills } from './fixtures/colour_labels';
import { pobPlain } from './pob_text';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

const CODE = /\^(?:x[0-9A-Fa-f]{6}|\d)/;
const fx = colourLabelSkills();

const mount = (set: number) =>
  render(SkillsPage, {
    sets: fx.sets,
    set,
    selected: 0,
    mainGroup: fx.mainSocketGroup,
    catalogue: [],
    deltas: {},
    slots: ['None'],
    onset: vi.fn(),
    onselect: vi.fn(),
    ongroup: vi.fn(),
    ongem: vi.fn(),
    onaddgem: vi.fn(),
    onremovegem: vi.fn(),
    onnewgroup: vi.fn(),
    onreordergroups: vi.fn()
  });

describe('colour codes in skill set titles and socket group labels', () => {
  it('the fixture carries codes in titles, label-only groups and code-only labels', () => {
    expect(fx.sets.length).toBeGreaterThan(1);
    expect(fx.sets.every((s) => CODE.test(s.title))).toBe(true);
    const groups = fx.sets.flatMap((s) => s.groups);
    expect(groups.some((g) => !g.gems.length && CODE.test(g.label) && /\S/.test(pobPlain(g.label)))).toBe(true);
    expect(groups.some((g) => !g.gems.length && /\S/.test(g.label) && !/\S/.test(pobPlain(g.label)))).toBe(true);
    expect(groups.some((g) => g.gems.length && CODE.test(g.label))).toBe(true);
  });

  it.each(fx.sets.map((s, i) => [s.title, i] as const))('set %s: no raw code reaches the Skills page, and label rows render as PoB', (_, set) => {
    const { container } = mount(set);
    expect(container.textContent).not.toMatch(CODE);
    const list = screen.getByRole('region', { name: 'Socket groups' });
    const groups = fx.sets[set].groups;
    // SkillListControl.lua:79-116
    for (const [i, g] of groups.entries()) {
      const row = list.querySelectorAll<HTMLElement>('button[data-reorder-press]')[i];
      const bare = !g.gems.length && /\S/.test(g.label);
      const blank = bare && !/\S/.test(pobPlain(g.label));
      expect(row.dataset.labelRow, `group ${i} ${g.label}`).toBe(bare ? (blank ? 'blank' : 'label') : undefined);
      if (bare) {
        expect(row.textContent).not.toContain('No gems');
        expect(row.textContent).not.toContain('–');
      }
      if (blank) expect(row.textContent?.replace('★ MAIN', '').trim()).toBe('');
      else expect(row.textContent).toContain(pobPlain(g.label).trim() || '<No active skills>');
      if (bare && !blank && CODE.test(g.label)) expect(row.querySelector('[data-pob-color]')).not.toBeNull();
    }
    const sets = within(screen.getByRole('combobox', { name: 'Skill set' })).getAllByRole('option');
    expect(sets.map((o) => o.textContent)).toEqual(fx.sets.map((s) => pobPlain(s.title)));
  });

  it('a dropdown renders codes as colour, never as text, and searches the plain label', async () => {
    const options = fx.sets.map((s, i) => ({ value: String(i), label: s.title }));
    render(Dropdown, { options, value: '0', 'aria-label': 'Set', searchable: true });
    const box = screen.getByRole('combobox', { name: 'Set' });
    expect(box.textContent).toBe(pobPlain(options[0].label));
    expect(box.querySelector('[data-pob-color]')).not.toBeNull();
    await fireEvent.click(box);
    const list = screen.getByRole('listbox');
    expect(list.textContent).not.toMatch(CODE);
    expect(within(list).getAllByRole('option')[1].getAttribute('title')).toBe(pobPlain(options[1].label));
  });

  it('the set manager lists titles in colour', () => {
    const titles = fx.sets.map((s) => s.title);
    render(SetManager, { open: true, title: 'Manage skill sets', noun: 'skill set', sets: titles, current: 0, onactivate: vi.fn(), onop: vi.fn() });
    const list = screen.getByRole('listbox');
    expect(list.textContent).not.toMatch(CODE);
    expect(within(list).getAllByRole('option')[0].querySelector('[data-pob-color]')).not.toBeNull();
  });
});
