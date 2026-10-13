// @vitest-environment jsdom
import { expect, describe, it, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { Blob as NodeBlob } from 'node:buffer';
import { deflateSync } from 'node:zlib';

afterEach(cleanup);

const calls: [string, string][] = [];
vi.mock('$lib/go/worker', () => ({
  syncWrap: {
    ExportCode: () => Promise.resolve('eNoLz08BAAKkAU4='),
    ImportCode: (code: string) => (calls.push(['ImportCode', code]), Promise.resolve()),
    LoadCompareCode: (code: string) => (calls.push(['LoadCompareCode', code]), Promise.resolve({ name: 'Imported build', seq: 1 }))
  }
}));
const navigations: string[] = [];
vi.mock('$app/navigation', () => ({ goto: (to: string) => (navigations.push(to), Promise.resolve()) }));

describe('import page has no enabled-but-dead controls', () => {
  const mount = async () => {
    const Page = (await import('./+page.svelte')).default;
    const view = render(Page);
    await vi.waitFor(() => expect((view.getByRole('textbox', { name: 'Build code' }) as HTMLTextAreaElement).value).not.toBe(''));
    return view;
  };
  const controls = (root: HTMLElement) => [...root.querySelectorAll<HTMLInputElement | HTMLButtonElement | HTMLSelectElement>('button, select, input')];
  const nameOf = (el: HTMLElement) =>
    el.getAttribute('aria-label') ?? (el.closest('label')?.textContent ?? el.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 40);

  it('renders the design page', async () => {
    const view = await mount();
    for (const name of ['Build sharing', 'Build import', 'Character import']) expect(view.getByRole('region', { name })).toBeTruthy();
  });

  it('enables exactly the controls the app wires', async () => {
    const view = await mount();
    const enabled = controls(view.container)
      .filter((c) => !c.disabled)
      .map(nameOf);
    expect(enabled).toHaveLength(8);
    expect(enabled[0]).toBe('Copy code');
    expect(enabled[1]).toMatch(/^Export support/);
    expect(enabled[2]).toMatch(/^To import a build/);
    expect(enabled[3]).toMatch(/^Import as a new build/);
    expect(enabled[4]).toMatch(/^Import to this build/);
    expect(enabled[5]).toMatch(/^Import as a comparison/);
    expect(enabled[6]).toMatch(/^Realm/);
    expect(enabled[7]).toMatch(/^Account name/);
  });

  it('imports a code as the comparison, then opens the Compare page', async () => {
    vi.stubGlobal('Blob', NodeBlob);
    const view = await mount();
    calls.length = 0;
    navigations.length = 0;
    const code = deflateSync('<PathOfBuilding/>').toString('base64');
    await fireEvent.input(view.getByRole('textbox', { name: /^To import a build/ }), { target: { value: code } });
    await fireEvent.click(view.getByRole('radio', { name: /Import as a comparison/ }));
    const button = view.getByRole('button', { name: 'Import' }) as HTMLButtonElement;
    const box = view.getByRole('textbox', { name: /^To import a build/ });
    await vi.waitFor(() => expect(document.getElementById(box.getAttribute('aria-describedby')!)!.textContent).toBe('Code is valid'));
    await vi.waitFor(() => expect(button.disabled).toBe(false));
    await fireEvent.click(button);
    await vi.waitFor(() => expect(navigations).toEqual(['/compare']));
    expect(calls).toEqual([['LoadCompareCode', code]]);
    vi.unstubAllGlobals();
  });

  it('every disabled control says WHY it is disabled', async () => {
    const view = await mount();
    const import_ = view.getByRole('button', { name: 'Import' });
    const offenders = controls(view.container)
      .filter((c) => c.disabled && c !== import_)
      .filter((c) => !c.title.trim())
      .map(nameOf);
    expect(offenders).toEqual([]);
    expect(view.getByRole('button', { name: 'Share' })).toHaveProperty('disabled', true);
  });
});
