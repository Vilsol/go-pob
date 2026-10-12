// @vitest-environment jsdom
// Build.lua:1160-1162

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import ImportExportPage from './ImportExportPage.svelte';

afterEach(cleanup);

describe('import code focus', () => {
  it('focuses the code field only when asked', async () => {
    const shown = render(ImportExportPage, { props: { code: '', realms: ['PC'] } });
    const input = shown.getByRole('textbox', { name: /To import a build/ });
    expect(document.activeElement).not.toBe(input);
    cleanup();
    const asked = render(ImportExportPage, { props: { code: '', realms: ['PC'], focusCode: true } });
    await new Promise((r) => setTimeout(r, 0));
    expect(document.activeElement).toBe(asked.getByRole('textbox', { name: /To import a build/ }));
  });
});
