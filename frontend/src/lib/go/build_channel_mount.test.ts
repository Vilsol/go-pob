// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';

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

const BUILD_A = '0001-necromancer-unknown.xml';
const BUILD_B = '0002-pathfinder-unknown.xml';

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD_A);
}, 120_000);

describe('the harness is not vacuous', () => {
  it('booted the real Go module and published a build into the store', () => {
    expect(workerInstance.booted).toBe(true);
    expect(typeof workerInstance.currentBuild?.SetNotes).toBe('function');
    expect(get(currentBuild)).toBeDefined();
  });
});

const watch = () => {
  const values: unknown[] = [];
  const stop = currentBuild.subscribe((v) => values.push(v));
  return { values, stop };
};

describe('an in-place build change re-notifies without minting a proxy', () => {
  it('SetNotes notifies subscribers', () => {
    const { values, stop } = watch();
    const before = values.length;

    workerInstance.SetNotes('notes via the worker');

    stop();
    expect(values.length, 'touch() did not re-notify — every page would stop repainting').toBe(before + 1);
  });

  it('SetNotes does NOT change the store value identity', () => {
    const { values, stop } = watch();
    const first = values[0];

    workerInstance.SetNotes('notes again');
    workerInstance.SetNotes('and again');

    stop();
    expect(values.length).toBe(3);
    expect(values[1], 'a second proxy was minted for the same build').toBe(first);
    expect(values[2]).toBe(first);
  });
});

describe('a genuinely different build DOES get a new proxy', () => {
  it('loading a second build replaces the store value AND the page follows', async () => {
    workerInstance.currentBuild!.SetNotes('this is build A');

    const Page = (await import('../../routes/notes/+page.svelte')).default;
    const { findByRole } = render(Page);
    const notesText = async () => (await findByRole('textbox', { name: 'Notes' })).textContent;
    await vi.waitFor(async () => expect(await notesText()).toBe('this is build A'));

    const wasA = get(currentBuild);

    await loadCorpusBuild(BUILD_B);
    const notesOfB = `notes belonging to ${BUILD_B}`;
    workerInstance.currentBuild!.SetNotes(notesOfB);

    expect(get(currentBuild)).not.toBe(wasA);

    await vi.waitFor(async () => expect(await notesText()).toBe(notesOfB));
  });

  it('a value captured before the swap is still readable after it', async () => {
    const held = get(currentBuild);
    expect(held).toBeDefined();

    await loadCorpusBuild(BUILD_A);
    expect(get(currentBuild)).not.toBe(held);

    await expect(Promise.resolve(held!.Build.ClassName)).resolves.toBeTruthy();
  });
});
