// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import Dropdown from './Dropdown.svelte';

afterEach(cleanup);

const options = [
  { value: 'a', label: 'Alpha' },
  { value: 'b', label: 'Beta' },
  { value: 'c', label: 'Gamma' }
];

describe('Dropdown', () => {
  it('opens on the chosen entry, reports each hovered entry, and chooses one', async () => {
    const onhover = vi.fn();
    const onchange = vi.fn();
    render(Dropdown, { options, value: 'b', 'aria-label': 'Pick', onhover, onchange });
    const box = screen.getByRole('combobox', { name: 'Pick' });
    expect(box.textContent).toBe('Beta');
    await fireEvent.click(box);
    expect(box.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByRole('option', { selected: true }).textContent?.trim()).toBe('Beta');
    await vi.waitFor(() => expect(onhover).toHaveBeenLastCalledWith(1, expect.any(HTMLElement)));
    await fireEvent.mouseEnter(screen.getByRole('option', { name: 'Gamma' }));
    await vi.waitFor(() => expect(onhover).toHaveBeenLastCalledWith(2, expect.any(HTMLElement)));
    await fireEvent.mouseLeave(screen.getByRole('option', { name: 'Gamma' }));
    expect(onhover).toHaveBeenLastCalledWith(null, null);
    await fireEvent.click(screen.getByRole('option', { name: 'Alpha' }));
    expect(onchange).toHaveBeenCalledWith('a');
    expect(screen.queryByRole('listbox')).toBeNull();
    expect(box.textContent).toBe('Alpha');
  });

  it('moves and chooses with the keys, and Escape closes without a change', async () => {
    const onhover = vi.fn();
    const onchange = vi.fn();
    render(Dropdown, { options, value: 'a', 'aria-label': 'Pick', onhover, onchange });
    const box = screen.getByRole('combobox', { name: 'Pick' });
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    await vi.waitFor(() => expect(onhover).toHaveBeenLastCalledWith(0, expect.any(HTMLElement)));
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    await vi.waitFor(() => expect(box.getAttribute('aria-activedescendant')).toMatch(/-o1$/));
    expect(onhover).toHaveBeenLastCalledWith(1, expect.any(HTMLElement));
    await fireEvent.keyDown(box, { key: 'Escape' });
    expect(screen.queryByRole('listbox')).toBeNull();
    expect(onchange).not.toHaveBeenCalled();
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    await fireEvent.keyDown(box, { key: 'End' });
    await fireEvent.keyDown(box, { key: 'Enter' });
    expect(onchange).toHaveBeenCalledWith('c');
  });

  it('Escape on the open list does not reach its dialog; on the closed box it does', async () => {
    const outer = vi.fn();
    document.body.addEventListener('keydown', outer);
    try {
      render(Dropdown, { options, value: 'a', 'aria-label': 'Pick' });
      const box = screen.getByRole('combobox', { name: 'Pick' });
      await fireEvent.click(box);
      await fireEvent.keyDown(box, { key: 'Escape' });
      expect(screen.queryByRole('listbox')).toBeNull();
      expect(outer).not.toHaveBeenCalled();
      await fireEvent.keyDown(box, { key: 'Escape' });
      expect(outer).toHaveBeenCalledTimes(1);
    } finally {
      document.body.removeEventListener('keydown', outer);
    }
  });

  it('searchable: typing filters the entries by every word, the keys move over what is shown, Escape clears then closes', async () => {
    const onchange = vi.fn();
    const many = [
      { value: '0', label: 'None' },
      { value: '1', label: '+# to maximum Life' },
      { value: '2', label: '#% increased maximum Life' },
      { value: '3', label: '+#% to Fire Resistance' },
      { value: '4', label: '#% increased Life Regeneration rate' }
    ];
    render(Dropdown, { options: many, value: '0', searchable: true, 'aria-label': 'Affix 1', onchange });
    const box = screen.getByRole('combobox', { name: 'Affix 1' });
    await fireEvent.keyDown(box, { key: 'l' });
    await fireEvent.keyDown(box, { key: 'i' });
    await fireEvent.keyDown(box, { key: 'f' });
    await fireEvent.keyDown(box, { key: 'e' });
    await fireEvent.keyDown(box, { key: ' ' });
    await fireEvent.keyDown(box, { key: 'i' });
    await fireEvent.keyDown(box, { key: 'n' });
    await fireEvent.keyDown(box, { key: 'c' });
    expect(screen.getAllByRole('option').map((o) => o.textContent?.trim())).toEqual(['#% increased maximum Life', '#% increased Life Regeneration rate']);
    expect(screen.getByRole('searchbox', { name: 'Filter Affix 1' })).toHaveProperty('value', 'life inc');
    await vi.waitFor(() => expect(box.getAttribute('aria-activedescendant')).toMatch(/-o2$/));
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    await vi.waitFor(() => expect(box.getAttribute('aria-activedescendant')).toMatch(/-o4$/));
    await fireEvent.keyDown(box, { key: 'Backspace' });
    await fireEvent.keyDown(box, { key: 'Backspace' });
    await fireEvent.keyDown(box, { key: 'Backspace' });
    await fireEvent.keyDown(box, { key: 'Backspace' });
    expect(screen.getAllByRole('option')).toHaveLength(3);
    await fireEvent.keyDown(box, { key: 'Escape' });
    expect(screen.getAllByRole('option')).toHaveLength(5);
    await fireEvent.keyDown(box, { key: 'r' });
    await fireEvent.keyDown(box, { key: 'e' });
    await fireEvent.keyDown(box, { key: 's' });
    await fireEvent.keyDown(box, { key: 'Enter' });
    expect(onchange).toHaveBeenCalledWith('3');
    expect(screen.queryByRole('listbox')).toBeNull();
    await fireEvent.click(box);
    expect(screen.getAllByRole('option')).toHaveLength(5);
    await fireEvent.input(screen.getByRole('searchbox', { name: 'Filter Affix 1' }), { target: { value: 'fire' } });
    expect(screen.getAllByRole('option').map((o) => o.textContent?.trim())).toEqual(['+#% to Fire Resistance']);
  });

  it('searchable: an entry the text names exactly is the one Enter picks, wherever it is in the list', async () => {
    const onchange = vi.fn();
    const types = [
      { value: '0', label: 'Helmet: Armour/Energy Shield' },
      { value: '1', label: 'Helmet: Energy Shield' }
    ];
    render(Dropdown, { options: types, value: '0', searchable: true, 'aria-label': 'Type', onchange });
    const box = screen.getByRole('combobox', { name: 'Type' });
    for (const k of 'helmet: energy shield') await fireEvent.keyDown(box, { key: k });
    expect(screen.getAllByRole('option')).toHaveLength(2);
    await fireEvent.keyDown(box, { key: 'Enter' });
    expect(onchange).toHaveBeenCalledWith('1');
  });
});
