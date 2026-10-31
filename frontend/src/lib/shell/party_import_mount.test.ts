// @vitest-environment jsdom
// PartyTab.lua:195-202

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { Blob as NodeBlob } from 'node:buffer';

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
const { ROOT_MISSING, unreadableBuild } = await import('$lib/design/share');
const types = await import('$lib/types');

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
}, 120_000);

const importParty = async (xml: string) => {
  vi.stubGlobal('Blob', NodeBlob);
  const Party = (await import('./AppParty.svelte')).default;
  const page = render(Party);
  const box = within(await page.findByRole('region', { name: 'Party import' }, { timeout: 10_000 }));
  await fireEvent.input(box.getByRole('textbox', { name: /Enter a build code/ }), { target: { value: types.pob.CompressEncode(xml).unwrap() } });
  const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
  await vi.waitFor(() => expect(go.disabled).toBe(false));
  await fireEvent.click(go);
  return (await box.findByRole('alert', undefined, { timeout: 10_000 })).textContent;
};

describe('Party import', () => {
  it('says a code whose XML does not parse holds no readable build', async () => {
    expect(await importParty('<PathOfBuilding')).toBe(unreadableBuild('XML syntax error on line 1: unexpected EOF'));
  });

  it('says XML that is not a Path of Building build holds no readable build', async () => {
    expect(await importParty('<Party/>')).toBe(unreadableBuild(ROOT_MISSING));
  });
});
