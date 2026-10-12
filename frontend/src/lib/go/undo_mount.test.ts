// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  window.addEventListener('unhandledrejection', (e) => pageErrors.push(String(e.reason)));
  process.on('unhandledRejection', (r) => pageErrors.push(String(r)));
});
afterEach(async () => {
  await new Promise((r) => setTimeout(r, 0));
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker } = await import('../../../harness/component/go_runtime');
const { currentBuild, buildRevision } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';
const LARGEST = '0001-necromancer-unknown.xml';

let clock = 1_800_000_000_000;
const later = (ms = 10_000) => (clock += ms);

const settle = async () => {
  await vi.waitFor(() => expect((workerInstance as unknown as { ticksInFlight: number }).ticksInFlight).toBe(0));
  await workerInstance.GetUndoHistoryWhenRecorded();
};

const serialize = () => new TextDecoder().decode(types.builds.SerializeBuild(workerInstance.currentBuild).unwrap());
const level = () => workerInstance.GetBuildInfo()!.level;
const enemyLevel = () => workerInstance.GetAllConfigOptions().enemyLevel;
const actions = () => workerInstance.GetUndoHistory().steps.map((s) => s.action);

const importBuild = async (file: string) => {
  const xml = await readFile(resolve(process.cwd(), '../testdata/corpus-3.29', file), 'utf8');
  await workerInstance.ImportCode(types.pob.CompressEncode(xml).unwrap());
  await settle();
};

let calc: MockInstance<() => Promise<undefined>>;

beforeAll(async () => {
  await bootWorker(currentBuild);
  calc = vi.spyOn(workerInstance as unknown as { computeOutputs: () => Promise<undefined> }, 'computeOutputs').mockResolvedValue(undefined);
  vi.spyOn(Date, 'now').mockImplementation(() => clock);
}, 120_000);

describe('the worker records a step per write and restores it', () => {
  it('an imported build starts with nothing to undo', async () => {
    await importBuild(BUILD);
    expect(workerInstance.GetUndoHistory()).toEqual({ steps: [], undone: 0, max: 50 });
  }, 60_000);

  it('two writes on two pages undo in order, redo in order, and a new write drops the redo', async () => {
    await importBuild(BUILD);
    const level0 = level();
    const enemy0 = enemyLevel();
    const xml0 = serialize();

    later();
    workerInstance.SetLevel(level0 + 1);
    await settle();
    const xml1 = serialize();
    later();
    workerInstance.SetConfigOption('enemyLevel', 70);
    await settle();
    expect(actions()).toEqual(['change a config option', 'change level']);
    expect(workerInstance.GetUndoHistory().steps.map((s) => s.page)).toEqual(['Config', 'Tree']);

    const object = workerInstance.currentBuild;
    let view = workerInstance.Undo();
    await settle();
    expect(view.undone).toBe(1);
    expect(enemyLevel()).toBe(enemy0);
    expect(level()).toBe(level0 + 1);
    expect(workerInstance.currentBuild).toBe(object);
    expect(serialize()).toBe(xml1);

    view = workerInstance.Undo();
    await settle();
    expect(view.undone).toBe(2);
    expect(level()).toBe(level0);
    expect(serialize()).toBe(xml0);
    expect(workerInstance.Undo().undone, 'undo past the first step').toBe(2);

    view = workerInstance.Redo();
    await settle();
    expect(view.undone).toBe(1);
    expect(level()).toBe(level0 + 1);
    expect(enemyLevel()).toBe(enemy0);

    later();
    workerInstance.SetConfigOption('enemyLevel', 60);
    await settle();
    expect(workerInstance.GetUndoHistory().undone).toBe(0);
    expect(actions()).toEqual(['change a config option', 'change level']);
    expect(workerInstance.Redo().undone).toBe(0);
    expect(enemyLevel()).toBe(60);
  }, 60_000);

  it('a restore publishes the build again, so every page reads it again', async () => {
    await importBuild(BUILD);
    later();
    workerInstance.SetLevel(level() + 1);
    await settle();
    const revision = get(buildRevision);
    const proxy = get(currentBuild);
    workerInstance.Undo();
    await settle();
    expect(get(buildRevision)).toBe(revision + 1);
    expect(get(currentBuild)).not.toBe(proxy);
  }, 60_000);

  it('a burst of writes from one box is one step', async () => {
    await importBuild(BUILD);
    const level0 = level();
    later();
    for (const n of [1, 2, 3]) {
      later(800);
      workerInstance.SetLevel(level0 + n);
      await settle();
    }
    expect(actions()).toEqual(['change level']);
    workerInstance.Undo();
    await settle();
    expect(level()).toBe(level0);
  }, 60_000);

  // PassiveSpecListControl.lua:95-111
  it('deleting the current passive tree is one step, and undo restores the tree and which one is current', async () => {
    await importBuild('0280-elementalist-fireball.xml');
    const trees = async () => {
      const { active, specs } = (await workerInstance.GetTreeSpecs())!;
      return { active, titles: specs.map((t) => t.Title) };
    };
    later();
    await workerInstance.SetActiveTreeSpec(3);
    await settle();
    const xml0 = serialize();
    const before = await trees();
    expect(before.titles.length).toBeGreaterThan(2);
    later();
    await workerInstance.DeleteTreeSpec(3);
    await settle();
    expect(actions()[0]).toBe('delete a passive tree');
    const after = await trees();
    expect(after.active).toBe(2);
    expect(after.titles).toEqual(before.titles.filter((_, i) => i !== 2));
    workerInstance.Undo();
    await settle();
    expect(await trees()).toEqual(before);
    expect(serialize()).toBe(xml0);
  }, 60_000);

  it('a change no Tick announced (a notes edit) is undone first, under its own name', async () => {
    await importBuild(BUILD);
    const notes0 = workerInstance.currentBuild!.Notes;
    later();
    workerInstance.SetLevel(level() + 1);
    await settle();
    const level1 = level();
    later();
    workerInstance.currentBuild!.SetNotes('typed after the level change');
    const view = workerInstance.Undo();
    await settle();
    expect(view.steps.map((s) => s.action)).toEqual(['edit notes', 'change level']);
    expect(view.undone).toBe(1);
    expect(workerInstance.currentBuild!.Notes).toBe(notes0);
    expect(level()).toBe(level1);
  }, 60_000);

  it('a Tick that is not a user write records nothing', async () => {
    await importBuild(BUILD);
    later();
    workerInstance.currentBuild!.SetNotes('changed without a write Tick');
    await workerInstance.Tick('setCalcTabElements');
    await workerInstance.Tick('Undo');
    await new Promise((r) => setTimeout(r, 300));
    expect(actions()).toEqual([]);
    later();
    await workerInstance.Tick('SetLevel');
    await vi.waitFor(() => expect(actions(), 'the control: a write Tick records the same change').toEqual(['change level']));
  }, 60_000);

  it('a write made while the opened build is still calculating does not leave the build without a history', async () => {
    const xml = await readFile(resolve(process.cwd(), '../testdata/corpus-3.29', BUILD), 'utf8');
    calc.mockImplementationOnce(() => new Promise((r) => setTimeout(() => r(undefined), 100)));
    await workerInstance.ImportCode(types.pob.CompressEncode(xml).unwrap());
    expect((workerInstance as unknown as { ticksInFlight: number }).ticksInFlight, 'the open Tick already ended').toBe(1);
    workerInstance.SetLevel(level() + 1);
    await settle();
    const level1 = level();
    later();
    workerInstance.SetLevel(level1 + 1);
    await settle();
    expect(actions()).toEqual(['change level']);
    workerInstance.Undo();
    await settle();
    expect(level()).toBe(level1);
  }, 60_000);

  it('opening another build starts a new history; closing it leaves nothing to undo', async () => {
    await importBuild(BUILD);
    later();
    workerInstance.SetLevel(level() + 1);
    await settle();
    expect(actions()).toHaveLength(1);
    await importBuild(BUILD);
    expect(actions()).toEqual([]);
    workerInstance.ClearBuild();
    expect(workerInstance.Undo()).toEqual({ steps: [], undone: 0, max: 50 });
  }, 60_000);
});

describe('the top bar shows the history and drives it', () => {
  const tickEnded = () => (document.documentElement.dataset.calcSeq = String(Number(document.documentElement.dataset.calcSeq ?? 0) + 1));

  it('names the step it undoes, undoes on click and Ctrl+Z, redoes on Ctrl+Y', async () => {
    await importBuild(BUILD);
    const level0 = level();
    const enemy0 = enemyLevel();
    const TopBar = (await import('$lib/shell/AppTopBar.svelte')).default;
    const view = render(TopBar);
    expect((view.getByRole('button', { name: 'Nothing to undo' }) as HTMLButtonElement).disabled).toBe(true);

    later();
    workerInstance.SetLevel(level0 + 1);
    await settle();
    later();
    workerInstance.SetConfigOption('enemyLevel', 70);
    await settle();
    tickEnded();

    const undo = await view.findByRole('button', { name: 'Undo: change a config option (Ctrl+Z)' });
    await fireEvent.click(undo);
    await view.findByRole('button', { name: 'Undo: change level (Ctrl+Z)' });
    expect(enemyLevel()).toBe(enemy0);

    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await view.findByRole('button', { name: 'Nothing to undo' });
    expect(level()).toBe(level0);

    await fireEvent.keyDown(window, { key: 'y', ctrlKey: true });
    await view.findByRole('button', { name: 'Undo: change level (Ctrl+Z)' });
    expect(level()).toBe(level0 + 1);
    expect(view.getByRole('button', { name: 'Redo: change a config option (Ctrl+Y / Ctrl+Shift+Z)' })).toBeDefined();
  }, 60_000);

  // Main.lua:209-211, 230-241
  it('opens About from the build view and labels the version', async () => {
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function () {
        this.open = true;
      };
      HTMLDialogElement.prototype.close = function () {
        this.open = false;
        this.dispatchEvent(new Event('close'));
      };
    }
    await importBuild(BUILD);
    const { devMode } = await import('$lib/global');
    const TopBar = (await import('$lib/shell/AppTopBar.svelte')).default;
    const view = render(TopBar);
    const about = () => view.getByRole('button', { name: 'About go-pob' });
    await vi.waitFor(() => expect(about().getAttribute('title')).toMatch(/^About go-pob · Version: \S+$/));
    devMode.set(true);
    await vi.waitFor(() => expect(about().getAttribute('title')).toMatch(/ · Dev Mode$/));
    devMode.set(false);
    const help = await readFile(resolve(__dirname, '../../../static/help.txt'), 'utf8');
    const changelog = await readFile(resolve(__dirname, '../../../static/changelog.txt'), 'utf8');
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => Promise.resolve(new Response(String(url).endsWith('/changelog.txt') ? changelog : help)))
    );
    try {
      await fireEvent.click(view.getByRole('button', { name: 'About go-pob' }));
      const opened = await view.findByRole('dialog', { name: 'About go-pob' });
      // Main.lua:1434-1437
      await fireEvent.click(await within(opened).findByRole('tab', { name: 'Version history' }));
      expect(await within(opened).findByRole('heading', { name: /^Version 2\.67\.2 \(2026\/08\/07\)$/ })).toBeTruthy();
      await fireEvent.click(view.getByRole('button', { name: 'Close' }));
      // Build.lua:311-315
      await fireEvent.change(view.getByRole('combobox', { name: 'Loadout' }), { target: { value: 'Help >>' } });
      const helpDialog = await view.findByRole('dialog', { name: 'About go-pob' });
      await vi.waitFor(() => expect(helpDialog.querySelector('[data-help-section="Loadouts"]')).not.toBeNull());
      expect(helpDialog.textContent).toContain('Loadouts can be selected from the dropdown');
    } finally {
      vi.unstubAllGlobals();
    }
  }, 60_000);

  it('a history entry goes back to that step', async () => {
    await importBuild(BUILD);
    const level0 = level();
    const TopBar = (await import('$lib/shell/AppTopBar.svelte')).default;
    const view = render(TopBar);
    for (const n of [1, 2, 3]) {
      later();
      workerInstance.SetLevel(level0 + n);
      await settle();
    }
    tickEnded();
    await view.findByRole('button', { name: 'Undo: change level (Ctrl+Z)' });
    await fireEvent.click(view.getByRole('button', { name: /^Undo history/ }));
    await vi.waitFor(() => expect(view.getAllByRole('menuitem')).toHaveLength(3));
    const items = view.getAllByRole('menuitem');
    await fireEvent.click(items[2]);
    await vi.waitFor(() => expect(level()).toBe(level0 + 1));
    expect(workerInstance.GetUndoHistory().undone).toBe(2);
  }, 60_000);
});

describe('restores do not grow the Go heap', () => {
  it('ten undo / redo round trips keep the Go heap flat', async () => {
    await importBuild(BUILD);
    const level0 = level();
    for (let n = 1; n <= 10; n++) {
      later();
      workerInstance.SetLevel(level0 + n);
      await settle();
    }
    const heap0 = types.exposition.LiveHeapMB();
    for (let i = 0; i < 10; i++) {
      workerInstance.Undo();
      await settle();
    }
    for (let i = 0; i < 10; i++) {
      workerInstance.Redo();
      await settle();
    }
    expect(level()).toBe(level0 + 10);
    expect(types.exposition.LiveHeapMB() - heap0, 'a restore leaked Go memory').toBeLessThan(1);
  }, 120_000);
});

describe.skipIf(!process.env.UNDO_COST)('cost on the largest corpus build', () => {
  it('records 50 steps and walks them back and forth without growing the Go heap', async () => {
    await importBuild(LARGEST);
    const level0 = level();
    const snapMs: number[] = [];
    for (let n = 1; n <= 50; n++) {
      later();
      workerInstance.SetLevel(((level0 + n) % 100) + 1);
      const start = performance.now();
      await settle();
      snapMs.push(performance.now() - start);
    }
    expect(workerInstance.GetUndoHistory().steps).toHaveLength(50);
    const history = (workerInstance as unknown as { history: { table: { size: number; lines: string[] } } }).history;
    const heap0 = types.exposition.LiveHeapMB();
    const undoMs: number[] = [];
    for (let i = 0; i < 10; i++) {
      const start = performance.now();
      workerInstance.Undo();
      undoMs.push(performance.now() - start);
      await settle();
    }
    for (let i = 0; i < 10; i++) {
      workerInstance.Redo();
      await settle();
    }
    const heap1 = types.exposition.LiveHeapMB();
    const xml = serialize();
    const median = (a: number[]) => [...a].sort((x, y) => x - y)[a.length >> 1];
    console.log(
      JSON.stringify({
        build: LARGEST,
        xmlChars: xml.length,
        lines: xml.split('\n').length,
        tableLines: history.table.size,
        tableChars: history.table.lines.reduce((n, l) => n + l.length, 0),
        idBytesPerStep: xml.split('\n').length * 4,
        writeTickMedianMs: median(snapMs),
        undoMedianMs: median(undoMs),
        heapBeforeMB: heap0,
        heapAfter20RestoresMB: heap1
      })
    );
    expect(level()).toBe(((level0 + 50) % 100) + 1);
    expect(heap1 - heap0, 'a restore leaked Go memory').toBeLessThan(5);
  }, 600_000);
});

// CalcsTab.lua:314-321, PartyTab.lua:665-700
describe('the Calcs inputs and the party buffers are steps', () => {
  it('a Calculation Mode change and a party import undo and redo', async () => {
    await importBuild(BUILD);
    const mode0 = workerInstance.GetCalcsTabInputs().buffMode;
    later();
    workerInstance.SetCalcsBuffMode(mode0 === 'BUFFED' ? 'COMBAT' : 'BUFFED');
    await settle();
    const exporter = await readFile(resolve(process.cwd(), '../testdata/corpus-3.29', '0160-elementalist-winter-orb-10.xml'), 'utf8');
    later();
    workerInstance.ImportParty(types.pob.CompressEncode(exporter).unwrap(), 'All', false);
    await settle();
    expect(actions().slice(0, 2)).toEqual(['import a party member', 'change the calculation mode']);
    expect(workerInstance.GetUndoHistory().steps[0].page).toBe('Party');
    expect(workerInstance.GetParty()!.buffers.Curse).toContain('Frostbite');

    workerInstance.Undo();
    await settle();
    expect(workerInstance.GetParty()!.buffers.Curse).toBe('');
    workerInstance.Undo();
    await settle();
    expect(workerInstance.GetCalcsTabInputs().buffMode).toBe(mode0);
    workerInstance.Redo();
    workerInstance.Redo();
    await settle();
    expect(workerInstance.GetParty()!.buffers.Curse).toContain('Frostbite');
  }, 60_000);
});
