// @vitest-environment jsdom
// Build.lua:849

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import Select from './Select.svelte';

afterEach(cleanup);

const texts = (c: HTMLElement) => [...c.querySelectorAll('option')].map((o) => o.textContent);

describe('Select', () => {
  it('draws every option when values repeat, and follows a changed list', async () => {
    const { container, rerender } = render(Select, { options: ['Loadouts:', 'Default', 'New Loadout'] });
    expect(texts(container)).toEqual(['Loadouts:', 'Default', 'New Loadout']);

    const dup = ['Loadouts:', '[3.28] Default', '[3.28] Default', '[3.28] Default', '[3.28] Default', 'Default', 'New Loadout', 'Sync'];
    await rerender({ options: dup });
    expect(texts(container)).toEqual(dup);

    await rerender({ options: ['Loadouts:', 'Default', 'New Loadout', 'Sync'] });
    expect(texts(container)).toEqual(['Loadouts:', 'Default', 'New Loadout', 'Sync']);
  });
});
