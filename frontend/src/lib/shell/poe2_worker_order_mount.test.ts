// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, onTestFinished, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import type { Poe2Calc } from '$lib/go/sync_worker2';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

const WASM2 = resolve(process.cwd(), 'static/go-pob2.wasm');

const { worker2 } = await import('$lib/go/sync_worker2');
// a worker call named here answers that many ms late, as a busy worker would; a list gives each call its own delay
const delays: Record<string, number | number[]> = {};
const poe2Worker = vi.fn(() =>
  Promise.resolve(
    new Proxy(worker2, {
      get: (target, prop, receiver) => {
        const v = Reflect.get(target, prop, receiver) as unknown;
        if (typeof v !== 'function') return v;
        return (...args: unknown[]) => {
          const r = (v as (...a: unknown[]) => unknown).apply(target, args);
          const d = delays[String(prop)];
          const ms = Array.isArray(d) ? d.shift() : d;
          return ms ? new Promise((done) => setTimeout(() => done(r), ms)) : r;
        };
      }
    })
  )
);
vi.mock('$lib/go/worker2', () => ({ poe2Worker }));
// the PoE2 worker caches game data through zenfs, which the harness stubs; a Map stands in for the calls it makes
vi.mock('@zenfs/core', () => {
  const files = new Map<string, Uint8Array>();
  return {
    configure: () => Promise.resolve(undefined),
    fs: {
      promises: {
        exists: (path: string) => Promise.resolve(files.has(path)),
        readFile: (path: string) => (files.has(path) ? Promise.resolve(files.get(path)!) : Promise.reject(new Error(`ENOENT: ${path}`))),
        writeFile: (path: string, value: Uint8Array) => {
          files.set(path, new Uint8Array(value));
          return Promise.resolve();
        },
        readdir: (dir: string) => Promise.resolve([...files.keys()].filter((k) => k.startsWith(`${dir}/`)).map((k) => k.slice(dir.length + 1))),
        rm: (path: string) => {
          for (const k of [...files.keys()]) if (k === path || k.startsWith(`${path}/`)) files.delete(k);
          return Promise.resolve();
        }
      }
    }
  };
});
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));

const { poe2Build } = await import('$lib/global');
const AppSkills = (await import('./AppSkills.svelte')).default;

const fetched: string[] = [];
beforeAll(async () => {
  if (!existsSync(WASM2)) throw new Error(`REFUSING: the PoE2 wasm module is missing — run \`mise run build-go2\`\n  missing: ${WASM2}`);
  // Go's fetch passes jsdom's AbortSignal, which node's fetch refuses, and reads a streamed body as
  // node's Uint8Array, which is not jsdom's; a whole-body response avoids both
  const nodeFetch = globalThis.fetch;
  vi.stubGlobal('fetch', async (input: RequestInfo | URL, init?: RequestInit) => {
    fetched.push(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url);
    const res = await nodeFetch(input, { ...init, signal: undefined });
    const body = await res.arrayBuffer();
    return {
      ok: res.ok,
      status: res.status,
      statusText: res.statusText,
      headers: res.headers,
      url: res.url,
      redirected: res.redirected,
      body: undefined,
      arrayBuffer: () => Promise.resolve(body)
    };
  });
  const wasm = await readFile(WASM2);
  // a typed array: jsdom's ArrayBuffer is another realm's, which boot's instanceof check would refuse
  await worker2.boot(new Uint8Array(wasm) as unknown as ArrayBuffer);
}, 120_000);

// calculator2/testdata/skills_edit_pob.jsonl: PoB-PoE2's sidebar values after each edit through its Skills tab
const pob = async (step: string) =>
  (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit_pob.jsonl'), 'utf8'))
    .trim()
    .split('\n')
    .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> })
    .find((r) => r.step === step)!.output;
const skillsEdit = () => readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8');

describe('the PoE2 worker runs one call at a time', () => {
  // first in the file: the minion tables load on the first minion this worker calculates
  it('a call made while a gem sort waits on a table load runs after it, against the build without the sorted gem', async () => {
    await worker2.Open(await skillsEdit());
    await worker2.EditSkills({ op: 'newGroup' });
    await worker2.SetMainSocketGroup(2);
    const gems = await worker2.GemList();
    const id = (name: string) => gems.find((g) => g.name === name)!.id;
    await worker2.EditSkills({ op: 'gem', group: 2, gem: 1, id: id('Fireball') });

    const from = fetched.length;
    // Living Lightning makes a minion, so the sort calculates one and fetches the minion tables part way through
    const sort = worker2.GemSortDPS(2, ['Metadata/Items/Gems/SkillGemLivingLightningSupport'], 'CombinedDPS');
    const groups = worker2.SkillGroups();
    const saved = worker2.Serialize();
    const added = worker2.EditSkills({ op: 'gem', group: 2, gem: 2, id: id('Elemental Focus') });
    const [, g, xml, calc] = await Promise.all([sort, groups, saved, added]);

    expect(
      fetched.slice(from).some((u) => u.includes('/MonsterVarieties.')),
      'the sort did not wait on the minion tables'
    ).toBe(true);
    expect(g.map((x) => x.gems.map((m) => m.name))).toEqual([['Spark'], ['Fireball']]);
    expect(xml).not.toMatch(/LivingLightning/);
    expect(calc.output.TotalDPS).toBe((await pob('add Elemental Focus')).TotalDPS);
  }, 120_000);
});

describe('the PoE2 Skills page', () => {
  const groupButtons = (container: HTMLElement) =>
    [...within(within(container).getByRole('region', { name: 'Socket groups' })).getAllByRole('button')].filter((b) => b.hasAttribute('aria-current'));

  it('a gem added before the page has the last edit back goes after the last gem', async () => {
    poe2Build.set(await worker2.Open(await skillsEdit()));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));
    const settle = async (step: string) => {
      const want = (await pob(step)).TotalDPS;
      await vi.waitFor(() => expect([step, (get(poe2Build) as Poe2Calc | undefined)?.output.TotalDPS]).toEqual([step, want]));
    };
    await fireEvent.click(page.getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(groupButtons(page.container)[1]?.getAttribute('aria-current')).toBe('true'));
    await fireEvent.click(page.getByRole('button', { name: 'Set as main' }));
    await settle('main group 2');
    await vi.waitFor(() => expect(groupButtons(page.container)[1].textContent).toContain('MAIN'));

    const add = async (name: string) => {
      const box = page.getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: name } });
      await fireEvent.keyDown(box, { key: 'Enter' });
    };
    // the page shows the group as it was before Fireball while the group list is on its way
    delays.SkillGroups = 2000;
    try {
      await add('Fireball');
      await settle('add Fireball');
      await add('Elemental Focus');
      await settle('add Elemental Focus');
    } finally {
      delete delays.SkillGroups;
    }
  }, 120_000);

  it('a gem edited before the page has a removal back edits the gem the page shows', async () => {
    poe2Build.set(await worker2.Open(await skillsEdit()));
    await worker2.EditSkills({ op: 'newGroup' });
    const gems = await worker2.GemList();
    const id = (name: string) => gems.find((g) => g.name === name)!.id;
    await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Fireball') });
    poe2Build.set(await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Elemental Focus') }));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await fireEvent.click(groupButtons(page.container)[1]);
    await vi.waitFor(() => page.getByRole('checkbox', { name: 'Elemental Focus enabled' }));

    const { poe2Error } = await import('$lib/go/engine_poe2.svelte');
    delays.SkillGroups = 2000;
    try {
      await fireEvent.click(page.getByRole('button', { name: 'Remove Fireball' }));
      await fireEvent.click(page.getByRole('checkbox', { name: 'Elemental Focus enabled' }));
      await vi.waitFor(async () => expect((await worker2.SkillGroups())[1].gems.map((m) => [m.name, m.enabled])).toEqual([['Elemental Focus', false]]), {
        timeout: 10_000
      });
    } finally {
      delete delays.SkillGroups;
    }
    expect(get(poe2Error)).toBeUndefined();
  }, 120_000);

  it('a group list asked for before a removal does not bring the removed gem back', async () => {
    poe2Build.set(await worker2.Open(await skillsEdit()));
    await worker2.EditSkills({ op: 'newGroup' });
    const gems = await worker2.GemList();
    const id = (name: string) => gems.find((g) => g.name === name)!.id;
    await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Fireball') });
    poe2Build.set(await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Elemental Focus') }));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await fireEvent.click(groupButtons(page.container)[1]);
    await vi.waitFor(() => page.getByRole('checkbox', { name: 'Elemental Focus enabled' }));

    // the list asked for after the first edit answers while the removal's calculation is still on its way
    delays.SkillGroups = 300;
    delays.EditSkills = [0, 2000];
    try {
      const before = get(poe2Build);
      await fireEvent.click(page.getByRole('checkbox', { name: 'Elemental Focus enabled' }));
      await vi.waitFor(() => expect(get(poe2Build)).not.toBe(before));
      await fireEvent.click(page.getByRole('button', { name: 'Remove Fireball' }));
      await new Promise((done) => setTimeout(done, 1000));
      expect(page.queryByRole('button', { name: 'Remove Fireball' })).toBeNull();
    } finally {
      delete delays.SkillGroups;
      delete delays.EditSkills;
    }
  }, 120_000);

  it('a gem edit made while an undo is on its way is not sent to the gem the undo puts in its place', async () => {
    // SkillsTab.lua:594-599 (PoB-PoE2): the tab's own undo, which binds Ctrl+Z only when no whole-build history does
    const { poe2Engine } = await import('$lib/go/engine_poe2.svelte');
    const history = poe2Engine.undo;
    poe2Engine.undo = undefined;
    onTestFinished(() => {
      poe2Engine.undo = history;
    });
    poe2Build.set(await worker2.Open(await skillsEdit()));
    await worker2.EditSkills({ op: 'newGroup' });
    const gems = await worker2.GemList();
    const id = (name: string) => gems.find((g) => g.name === name)!.id;
    await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Fireball') });
    poe2Build.set(await worker2.EditSkills({ op: 'addGem', group: 2, id: id('Elemental Focus') }));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await fireEvent.click(groupButtons(page.container)[1]);
    await fireEvent.click(await vi.waitFor(() => page.getByRole('button', { name: 'Remove Fireball' })));
    const gemsOf = async () => (await worker2.SkillGroups())[1].gems.map((m) => [m.name, m.enabled]);
    await vi.waitFor(async () => expect(await gemsOf()).toEqual([['Elemental Focus', true]]));

    delays.SkillsUndo = 1500;
    try {
      await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
      await fireEvent.click(page.getByRole('checkbox', { name: 'Elemental Focus enabled' }));
      await vi.waitFor(() => page.getByRole('button', { name: 'Remove Fireball' }), { timeout: 10_000 });
    } finally {
      delete delays.SkillsUndo;
    }
    expect(await gemsOf()).toEqual([
      ['Fireball', true],
      ['Elemental Focus', true]
    ]);
  }, 120_000);
});
