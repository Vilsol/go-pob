// @vitest-environment jsdom
// Main.lua:212-229

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render } from '@testing-library/svelte';
import BuildListPage from './BuildListPage.svelte';

afterEach(cleanup);

const base = { builds: [], folders: [], classes: [], gameVersion: '3.29', folder: '', open: {}, onfolder: () => {}, ontoggle: () => {}, onmove: () => {} };

describe('the build list’s update button', () => {
  it('checks, says so while checking, and offers the update once one waits', async () => {
    const calls: string[] = [];
    const update = { running: false, ready: false, oncheck: () => calls.push('check'), onapply: () => calls.push('apply') };
    const { getByRole, rerender } = render(BuildListPage, { ...base, update });
    await fireEvent.click(getByRole('button', { name: 'Check for Update' }));
    await rerender({ ...base, update: { ...update, running: true } });
    expect((getByRole('button', { name: 'Checking...' }) as HTMLButtonElement).disabled).toBe(true);
    await rerender({ ...base, update: { ...update, ready: true } });
    await fireEvent.click(getByRole('button', { name: 'Update Ready' }));
    expect(calls).toEqual(['check', 'apply']);
  });

  it('has neither in dev mode', () => {
    const { queryByRole } = render(BuildListPage, base);
    expect(queryByRole('button', { name: /Check for Update|Update Ready/ })).toBeNull();
  });
});
