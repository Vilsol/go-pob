// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import SkillGroupEditor from './SkillGroupEditor.svelte';
import type { SocketGroupFixture } from './fixtures';

afterEach(cleanup);

const gem = (name: string, support: boolean) => ({ name, level: 20, quality: 0, enabled: true, support, attr: 'int' as const, count: 1 });
const group: SocketGroupFixture = {
  label: '',
  slot: 'Body Armour',
  enabled: true,
  inFull: true,
  count: 1,
  sockets: 'B-B-G R',
  gems: [gem('Fireball', false), gem('Spell Echo', true), gem('Frostbolt', false)]
};
const catalogue = [
  { name: 'Arc', attr: 'int' as const, support: false, tags: ['grants_active_skill', 'spell'] },
  { name: 'Added Cold Damage', attr: 'dex' as const, support: true, tags: ['support', 'cold'] },
  { name: 'Awakened Spell Echo', attr: 'int' as const, support: true, tags: ['support', 'spell'] }
];
const noop = () => {};
const mount = (extra: Record<string, unknown> = {}) =>
  render(SkillGroupEditor, {
    group,
    offset: 0,
    catalogue,
    deltas: {},
    slots: ['Body Armour'],
    sortOn: false,
    sortStat: '',
    onchange: noop,
    ongem: noop,
    onadd: noop,
    onremove: noop,
    onreorder: noop,
    ...extra
  });

describe('SkillGroupEditor', () => {
  it("draws PoB's Item sockets line", () => {
    mount();
    expect(screen.getByText(/Item sockets:/).textContent).toBe('Item sockets: B=B=G  R');
  });

  it('puts gems in the sockets that are not abyssal', () => {
    const { container } = mount({ group: { ...group, sockets: 'A B-G' } });
    const rows = [...container.querySelectorAll<HTMLElement>('[role="group"][aria-label="Gems"] > div.relative')];
    expect(rows.map((r) => r.querySelector('[aria-label$=" socket"]')?.getAttribute('aria-label'))).toEqual(['Blue socket', 'Green socket', 'White socket']);
  });

  // SkillsTab.lua:1210-1246
  it('a group with no item sockets is all linked, not broken', () => {
    mount({ group: { ...group, slot: '', sockets: '' } });
    expect(screen.getByText(/^Links: /).textContent).toBe('Links: all 3 linked');
  });

  // GemSelectControl.lua:620-627
  it('highlights the gems a hovered gem supports or is supported by', async () => {
    const { container } = mount({ supporting: [[1, 0]] });
    const rows = [...container.querySelectorAll<HTMLElement>('[role="group"][aria-label="Gems"] > div.relative')];
    expect(rows).toHaveLength(3);
    await fireEvent.mouseEnter(rows[1]);
    expect(rows.map((r) => r.dataset.supporting === 'true')).toEqual([true, false, false]);
    await fireEvent.mouseEnter(rows[2]);
    expect(rows.map((r) => r.dataset.supporting === 'true')).toEqual([false, false, false]);
    await fireEvent.mouseEnter(rows[0]);
    expect(rows.map((r) => r.dataset.supporting === 'true')).toEqual([false, true, false]);
  });

  // GemSelectControl.lua:629-646
  it('hovering anywhere on a gem row floats its gem tooltip beside the row', async () => {
    const gemtip = () => Promise.resolve([{ Text: 'Spell Echo tip', Separator: false }]);
    // jsdom lays nothing out: give the positioning a viewport
    Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, value: 1024 });
    Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, value: 768 });
    const { container } = mount({ gemtip });
    const rows = [...container.querySelectorAll<HTMLElement>('[role="group"][aria-label="Gems"] > div.relative')];
    rows[1].getBoundingClientRect = () => ({ left: 100, right: 400, top: 200, bottom: 230, width: 300, height: 30, x: 100, y: 200 }) as DOMRect;
    await fireEvent.mouseEnter(rows[1]);
    const tip = await screen.findByRole('status', { name: 'What-if' });
    await vi.waitFor(() => expect(tip.textContent).toContain('Spell Echo tip'));
    await vi.waitFor(() => expect(tip.closest('[data-floating]')?.getAttribute('style')).toMatch(/--x: 410px.*--y: 200px/));
    await fireEvent.mouseLeave(rows[1]);
    await vi.waitFor(() => expect(screen.queryByRole('status', { name: 'What-if' })).toBeNull());
    for (const k of ['clientWidth', 'clientHeight']) delete (document.documentElement as unknown as Record<string, unknown>)[k];
  });

  // GemSelectControl.lua:651-681, 732-741
  it('S and A keep only support or only active gems', async () => {
    mount();
    const box = screen.getByRole('combobox', { name: 'Add a gem' });
    await fireEvent.focus(box);
    const names = () =>
      within(screen.getByRole('listbox', { name: 'Gem results' }))
        .getAllByRole('option')
        .map((o) => o.textContent ?? '');
    expect(names().length).toBe(3);
    await fireEvent.click(screen.getByRole('button', { name: 'Only show Support gems' }));
    expect(names().every((n) => !n.includes('Arc'))).toBe(true);
    expect(names().length).toBe(2);
    await fireEvent.click(screen.getByRole('button', { name: 'Only show Active gems' }));
    expect(names().length).toBe(1);
    await fireEvent.click(screen.getByRole('button', { name: 'Only show Active gems' }));
    expect(names().length).toBe(3);
  });

  // SkillsTab.lua:294-343
  it('the imbued support selector writes the chosen support, and x clears it', async () => {
    const onimbued = vi.fn();
    mount({ imbued: { value: 'Added Cold Damage', options: ['Added Cold Damage', 'Spell Echo'], enabled: true }, onimbued });
    const select = screen.getByRole<HTMLSelectElement>('combobox', { name: 'Imbued Support' });
    expect(select.value).toBe('Added Cold Damage');
    await fireEvent.change(select, { target: { value: 'Spell Echo' } });
    expect(onimbued).toHaveBeenLastCalledWith('Spell Echo');
    await fireEvent.click(screen.getByRole('button', { name: 'Remove this imbued support.' }));
    expect(onimbued).toHaveBeenLastCalledWith('');
    cleanup();
    mount({ imbued: { value: '', options: ['Spell Echo'], enabled: false }, onimbued });
    expect(screen.getByRole('combobox', { name: 'Imbued Support' })).toHaveProperty('disabled', true);
  });
});
