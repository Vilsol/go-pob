// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import CharacterImport from './CharacterImport.svelte';

afterEach(cleanup);

const realms = [{ id: 'PC', label: 'PC', realmCode: 'pc' }] as never;

describe('Character import › account history', () => {
  it('hides the history and its remove button while it is empty', () => {
    const { queryByRole } = render(CharacterImport, { props: { realms, history: [] } });
    expect(queryByRole('combobox', { name: 'Account history' })).toBeNull();
    expect(queryByRole('button', { name: 'Remove account from history' })).toBeNull();
  });

  it('shows the history once an account was imported', () => {
    const { getByRole } = render(CharacterImport, { props: { realms, history: ['Name#1234'] } });
    const select = getByRole('combobox', { name: 'Account history' });
    expect([...select.querySelectorAll('option')].map((o) => o.textContent)).toContain('Name#1234');
    expect(getByRole('button', { name: 'Remove account from history' })).toBeTruthy();
  });
});
