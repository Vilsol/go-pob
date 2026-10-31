// @vitest-environment jsdom
// CompareTab.lua:1462-1525

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';

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
const { NOT_A_CODE, ROOT_MISSING, unreadableBuild } = await import('$lib/design/share');
const types = await import('$lib/types');

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
}, 120_000);

const compare = async (code: string) => {
  const Compare = (await import('./AppCompare.svelte')).default;
  const page = render(Compare);
  const section = within(await page.findByRole('region', { name: 'Choose a build to compare' }, { timeout: 10_000 }));
  await fireEvent.input(section.getByRole('textbox', { name: 'Comparison build code' }), { target: { value: code } });
  await fireEvent.click(section.getByRole('button', { name: 'Compare' }));
  return (await section.findByText(/^Comparison failed:/, undefined, { timeout: 10_000 })).textContent;
};

describe('Compare with a build code', () => {
  it('says a code that does not decode is not a build code', async () => {
    expect(await compare('not-a-code')).toBe(`Comparison failed: ${NOT_A_CODE}`);
    expect(workerInstance.GetCompareInfo().seq).toBe(0);
  });

  it('says a code whose XML does not parse holds no readable build', async () => {
    const code = types.pob.CompressEncode('<PathOfBuilding').unwrap();
    expect(await compare(code)).toBe(`Comparison failed: ${unreadableBuild('XML syntax error on line 1: unexpected EOF')}`);
  });

  // Build.lua:2231-2233
  it('says XML that is not a Path of Building build holds no readable build', async () => {
    const code = types.pob.CompressEncode('<Party><Tree><Spec/></Tree></Party>').unwrap();
    expect(await compare(code)).toBe(`Comparison failed: ${unreadableBuild(ROOT_MISSING)}`);
  });

  it('says a code cut short is not a build code', async () => {
    const code = types.pob.CompressEncode('<PathOfBuilding><Build level="1"/></PathOfBuilding>'.repeat(20)).unwrap();
    expect(await compare(code.slice(0, code.length / 2))).toBe(`Comparison failed: ${NOT_A_CODE}`);
  });
});
