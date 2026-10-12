// @vitest-environment jsdom

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/svelte';
import TopBar from './TopBar.svelte';

afterEach(cleanup);

const props = { name: 'Build', tabs: [], current: '', history: [], loadouts: [], issues: [], logs: [], dirty: false, version: '1.2.3', onabout: () => {} };

describe('TopBar version', () => {
  it('shows the version as the About button tooltip only', () => {
    const { container } = render(TopBar, props);
    expect(screen.getByRole('button', { name: 'About go-pob' }).getAttribute('title')).toBe('About go-pob · Version: 1.2.3');
    expect(container.textContent).not.toContain('Version:');
  });

  it('keeps Dev Mode on screen', () => {
    render(TopBar, { ...props, devMode: true });
    expect(screen.getByRole('button', { name: 'About go-pob' }).getAttribute('title')).toBe('About go-pob · Version: 1.2.3 · Dev Mode');
    expect(screen.getByText('Dev Mode')).toBeTruthy();
  });
});
