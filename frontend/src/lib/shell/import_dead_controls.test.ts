// @vitest-environment jsdom
import { expect, describe, it, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import { Blob as NodeBlob } from 'node:buffer';
import { deflateSync } from 'node:zlib';

afterEach(async () => {
  cleanup();
  (await import('./import_modal.svelte')).importDialog.open = false;
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

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

describe('the import dialog has no enabled-but-dead controls', () => {
  // a PoE1 build is open, so both tabs and every import mode are offered
  const mount = async (tab: 'import' | 'export') => {
    const { currentBuild } = await import('$lib/global');
    currentBuild.set({ ExportParty: () => Promise.resolve(false) } as never);
    (await import('./import_modal.svelte')).openImport(tab);
    const view = render((await import('./AppImportModal.svelte')).default);
    const panel = view.getByRole('tabpanel');
    if (tab === 'export') await vi.waitFor(() => expect(within(panel).getByRole<HTMLTextAreaElement>('textbox', { name: 'Build code' }).value).not.toBe(''));
    return { ...within(panel), container: panel };
  };
  const controls = (root: HTMLElement) => [...root.querySelectorAll<HTMLInputElement | HTMLButtonElement | HTMLSelectElement>('button, select, input')];
  const nameOf = (el: HTMLElement) =>
    el.getAttribute('aria-label') ?? (el.closest('label')?.textContent ?? el.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 40);

  it('renders the design page, one side per tab', async () => {
    const exp = await mount('export');
    expect(exp.getByRole('region', { name: 'Build sharing' })).toBeTruthy();
    cleanup();
    const imp = await mount('import');
    for (const name of ['Build import', 'Character import']) expect(imp.getByRole('region', { name })).toBeTruthy();
  });

  it('enables exactly the controls the app wires', async () => {
    const enabledIn = (root: HTMLElement) =>
      controls(root)
        .filter((c) => !c.disabled)
        .map(nameOf);
    const exp = enabledIn((await mount('export')).container);
    cleanup();
    const enabled = [...exp, ...enabledIn((await mount('import')).container)];
    expect(enabled).toHaveLength(9);
    expect(enabled[0]).toBe('Copy code');
    expect(enabled[1]).toMatch(/^Share via/);
    expect(enabled[2]).toBe('Share');
    expect(enabled[3]).toMatch(/^Export support/);
    expect(enabled[4]).toMatch(/^To import a build/);
    expect(enabled[5]).toMatch(/^Import as a new build/);
    expect(enabled[6]).toMatch(/^Import as a comparison/);
    expect(enabled[7]).toMatch(/^Realm/);
    expect(enabled[8]).toMatch(/^Account name/);
  });

  it('imports a code as the comparison, then opens the Compare page', async () => {
    vi.stubGlobal('Blob', NodeBlob);
    const view = await mount('import');
    calls.length = 0;
    navigations.length = 0;
    const code = deflateSync('<PathOfBuilding/>').toString('base64');
    await fireEvent.input(view.getByRole('textbox', { name: /^To import a build/ }), { target: { value: code } });
    await fireEvent.click(view.getByRole('radio', { name: /Import as a comparison/ }));
    const button = view.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    const box = view.getByRole('textbox', { name: /^To import a build/ });
    await vi.waitFor(() => expect(document.getElementById(box.getAttribute('aria-describedby')!)!.textContent).toBe('Code is valid'));
    await vi.waitFor(() => expect(button.disabled).toBe(false));
    await fireEvent.click(button);
    await vi.waitFor(() => expect(navigations).toEqual(['/compare']));
    expect(calls).toEqual([['LoadCompareCode', code]]);
    vi.unstubAllGlobals();
  });

  it('every disabled control says WHY it is disabled', async () => {
    const exp = await mount('export');
    const exported = controls(exp.container);
    cleanup();
    const view = await mount('import');
    const import_ = view.getByRole('button', { name: 'Import' });
    const offenders = [...exported, ...controls(view.container)]
      .filter((c) => c.disabled && c !== import_)
      .filter((c) => !c.title.trim())
      .map(nameOf);
    expect(offenders).toEqual([]);
  });

  // ImportTab.lua:539-554
  it('shares the code to the chosen site and shows the link, or the error', async () => {
    const posted: [string, string][] = [];
    let answer = new Response('https://pobb.in/abc', { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
    vi.stubGlobal('fetch', (url: string, init: { body: string }) => (posted.push([url, init.body]), Promise.resolve(answer)));
    const view = await mount('export');
    const select = view.getByRole<HTMLSelectElement>('combobox', { name: 'Share via' });
    await fireEvent.change(select, { target: { value: 'pobb.in' } });
    await fireEvent.click(view.getByRole('button', { name: 'Share' }));
    await vi.waitFor(() => expect(view.getByRole<HTMLInputElement>('textbox', { name: 'Share link' }).value).toBe('https://pobb.in/abc'));
    expect(posted).toEqual([['/api/share?site=POBBin', 'eNoLz08BAAKkAU4=']]);
    expect(localStorage.getItem('options:lastExportWebsite')).toBe('pobb.in');
    answer = new Response('pobb.in answered 429', { status: 429, headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
    await fireEvent.click(view.getByRole('button', { name: 'Share' }));
    await vi.waitFor(() => expect(view.getByRole('alert').textContent).toBe('Error creating link: pobb.in answered 429'));
    expect(view.queryByRole('textbox', { name: 'Share link' })).toBeNull();
    vi.unstubAllGlobals();
  });
});
