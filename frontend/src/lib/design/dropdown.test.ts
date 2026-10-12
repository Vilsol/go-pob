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
});
