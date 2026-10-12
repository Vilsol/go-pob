// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, waitFor, within } from '@testing-library/svelte';
import { readFileSync } from 'node:fs';
import { Blob as NodeBlob } from 'node:buffer';
import { resolve } from 'node:path';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const ROOT = resolve(import.meta.dirname, '../../../..');
const ORACLE = JSON.parse(readFileSync(resolve(ROOT, 'testdata/oracle/party_tab_expected.json'), 'utf8')) as {
  scenarios: { name: string; buffers: Record<string, string> }[];
};
const EXPORTER = '0160-elementalist-winter-orb-10.xml';

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0381-assassin-soulrend-of-reaping-2.xml');
  expect(typeof Blob.prototype.stream, 'jsdom grew Blob.stream; drop this swap').toBe('undefined');
  vi.stubGlobal('Blob', NodeBlob);
}, 120_000);

const go = () => workerInstance.currentBuild!;

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  await view.findByRole('region', { name: 'Party import' });
  return view;
};
const entries = (view: Awaited<ReturnType<typeof mount>>, label: string) =>
  [...view.getByRole('region', { name: label }).querySelectorAll<HTMLElement>('[data-entry]')].map((e) => e.dataset.entry);

describe('the Party page', () => {
  it('shows each buffer as Go summarises it', async () => {
    const view = await mount();
    const want = (types.exposition.GetPartySummaries(go())?.Aura ?? []).map((e) => (e.Group ? `${e.Group}: ${e.Name}` : e.Name));
    expect(want.length).toBeGreaterThan(0);
    await waitFor(() => expect(entries(view, 'Auras')).toEqual(want));
  });

  it('imports a build code into every buffer as PoB does', async () => {
    const view = await mount();
    const imp = within(view.getByRole('region', { name: 'Party import' }));
    const code = types.pob.CompressEncode(readFileSync(resolve(ROOT, 'testdata/corpus-3.29', EXPORTER), 'utf8')).unwrap();
    const field = imp.getByRole('textbox', { name: /Enter a build code/ });
    await fireEvent.input(field, { target: { value: code } });
    const go_ = imp.getByRole('button', { name: 'Import' });
    await waitFor(() => expect((go_ as HTMLButtonElement).disabled).toBe(false));
    await fireEvent.click(go_);
    const want = ORACLE.scenarios.find((s) => s.name === 'replace all')!.buffers;
    await waitFor(() => expect(go().PartyBuffers()).toEqual(want));
    await waitFor(() => expect(entries(view, 'Curses')).toEqual(['Frostbite']));
  });

  it('clears one destination and keeps the options with the build', async () => {
    const view = await mount();
    const imp = within(view.getByRole('region', { name: 'Party import' }));
    await fireEvent.change(imp.getByRole('combobox', { name: 'Destination' }), { target: { value: 'Curse' } });
    await fireEvent.click(imp.getByRole('checkbox', { name: 'Append' }));
    await waitFor(() => expect(go().PartyOptions()).toEqual(['Curse', true, false]));
    await fireEvent.click(imp.getByRole('button', { name: 'Clear' }));
    await waitFor(() => expect(go().PartyBuffers()?.Curse).toBe(''));
    expect(go().PartyBuffers()?.EnemyMods).not.toBe('');
  });

  it('disables the effects, and an edit waits for Rebuild All', async () => {
    const view = await mount();
    const imp = within(view.getByRole('region', { name: 'Party import' }));
    await fireEvent.click(imp.getByRole('checkbox', { name: 'Disable Party Effects' }));
    await waitFor(() => expect(go().PartyEffectsDisabled()).toBe(true));
    await fireEvent.click(imp.getByRole('button', { name: 'Rebuild All' }));
    await waitFor(() => expect(go().PartyEffectsDisabled()).toBe(false));
    await fireEvent.click(imp.getByRole('checkbox', { name: 'Show Advanced Info' }));
    const box = await view.findByRole('textbox', { name: 'Enemy Conditions buffer' });
    await fireEvent.input(box, { target: { value: 'Condition:Blinded' } });
    await waitFor(() => expect(go().PartyBuffers()?.EnemyConditions).toBe('Condition:Blinded'), { timeout: 3000 });
    expect(go().PartyEffectsApplied()).toBe(false);
    await view.findByText(/Rebuild All applies it/);
    await fireEvent.click(imp.getByRole('button', { name: 'Rebuild All' }));
    await waitFor(() => expect(go().PartyEffectsApplied()).toBe(true));
  });

  it('lists each imported build as a member that can be switched off and on, saved with the build', async () => {
    const view = await mount();
    const imp = within(view.getByRole('region', { name: 'Party import' }));
    await fireEvent.change(imp.getByRole('combobox', { name: 'Destination' }), { target: { value: 'All' } });
    const append = imp.getByRole<HTMLInputElement>('checkbox', { name: 'Append' });
    if (append.checked) await fireEvent.click(append);
    await waitFor(() => expect(go().PartyOptions().slice(0, 2)).toEqual(['All', false]));
    const before = go().PartyMembers()?.length ?? 0;
    const code = types.pob.CompressEncode(readFileSync(resolve(ROOT, 'testdata/corpus-3.29', EXPORTER), 'utf8')).unwrap();
    await fireEvent.input(imp.getByRole('textbox', { name: /Enter a build code/ }), { target: { value: code } });
    const button = imp.getByRole('button', { name: 'Import' });
    await waitFor(() => expect((button as HTMLButtonElement).disabled).toBe(false));
    await fireEvent.click(button);
    await waitFor(() => expect(go().PartyMembers()?.length).toBe(before + 1));
    const full = go().PartyBuffers();
    expect(full?.Curse, 'vacuity: the member gives a curse').not.toBe('');
    const panel = () => view.container.querySelector<HTMLElement>(`[data-member="${before}"]`);
    const toggle = () => within(panel()!).getByRole<HTMLInputElement>('checkbox', { name: 'Elementalist active' });
    await waitFor(() => expect(panel()?.dataset.active).toBe('true'));
    expect([...view.container.querySelectorAll<HTMLElement>('[data-member]')].map((e) => e.dataset.active)).toEqual([
      ...Array<string>(before).fill('false'),
      'true'
    ]);
    await fireEvent.click(toggle());
    await waitFor(() => expect(go().PartyBuffers()?.Curse).toBe(''));
    await waitFor(() => expect(panel()?.dataset.active).toBe('false'));
    await fireEvent.click(toggle());
    await waitFor(() => expect(go().PartyBuffers()).toEqual(full));
    await waitFor(() => expect(panel()?.dataset.active).toBe('true'));
    await fireEvent.click(within(panel()!).getByRole('button', { name: 'Remove Elementalist' }));
    await waitFor(() => expect(go().PartyMembers()?.length ?? 0).toBe(before));
  });
});
