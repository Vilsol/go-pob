import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { DIFF_LABELS, LineTable, NOT_A_CHANGE, OPEN_SOURCES, STEP_LABELS, UndoHistory, sameSnapshot, unrecordedLabel } from './undo';

const L = { action: 'a', page: 'p' };

const setup = (max = 50) => {
  const h = new UndoHistory(new LineTable(), max);
  const snap = (name: string) => h.table.encode(`<PathOfBuilding>\n${name}\n</PathOfBuilding>`);
  const name = (s: Uint32Array | undefined) => (s ? h.table.decode(s).split('\n')[1] : undefined);
  h.reset(snap('A'));
  return { h, snap, name };
};

describe('LineTable', () => {
  it('decodes exactly what it encoded, empty and trailing lines included', () => {
    const t = new LineTable();
    for (const xml of ['', '\n', 'a\n\nb\n', '<x>\r\n</x>', 'only']) {
      expect(t.decode(t.encode(xml))).toBe(xml);
    }
  });

  it('stores a line once however many snapshots carry it', () => {
    const t = new LineTable();
    t.encode('a\nb\nc');
    t.encode('a\nb\nd');
    expect(t.size).toBe(4);
  });

  it('compacts away lines no live snapshot uses, keeping the live ones exact', () => {
    const t = new LineTable();
    const keep = t.encode('keep\nme');
    for (let i = 0; i < 5000; i++) {
      t.encode(`dead ${i}`);
    }
    const [after] = t.compact([keep]);
    expect(t.size).toBe(2);
    expect(t.decode(after)).toBe('keep\nme');
  });
});

describe('UndoHistory', () => {
  it('starts with nothing to undo or redo', () => {
    const { h } = setup();
    expect(h.view()).toEqual({ steps: [], undone: 0, max: 50 });
    expect(h.undo()).toBeUndefined();
    expect(h.redo()).toBeUndefined();
  });

  it('undoes in order, redoes in order, and walks back to the base', () => {
    const { h, snap, name } = setup();
    h.record('s1', { action: 'one', page: 'Tree' }, snap('B'), 0);
    h.record('s2', { action: 'two', page: 'Config' }, snap('C'), 10_000);
    expect(h.view().steps.map((s) => s.action)).toEqual(['two', 'one']);
    expect(name(h.undo())).toBe('B');
    expect(h.view().undone).toBe(1);
    expect(name(h.undo())).toBe('A');
    expect(h.undo()).toBeUndefined();
    expect(name(h.redo())).toBe('B');
    expect(name(h.redo())).toBe('C');
    expect(h.redo()).toBeUndefined();
  });

  it('a change after an undo drops the undone steps (UndoHandler.lua:31-33)', () => {
    const { h, snap, name } = setup();
    h.record('s', L, snap('B'), 0);
    h.record('s', L, snap('C'), 10_000);
    h.undo();
    expect(h.record('s', { action: 'new', page: '' }, snap('D'), 20_000)).toBe(true);
    expect(h.view().undone).toBe(0);
    expect(h.view().steps.map((s) => s.action)).toEqual(['new', 'a']);
    expect(h.redo()).toBeUndefined();
    expect(name(h.undo())).toBe('B');
  });

  it('a write that changes nothing records nothing', () => {
    const { h, snap } = setup();
    expect(h.record('s', L, snap('A'), 0)).toBe(false);
    expect(h.view().steps).toEqual([]);
  });

  it('keeps the newest `max` steps; the oldest kept step is as far back as undo goes', () => {
    const { h, snap, name } = setup(3);
    for (const [i, n] of ['B', 'C', 'D', 'E', 'F'].entries()) {
      h.record('s', L, snap(n), i * 10_000);
    }
    expect(h.view().steps).toHaveLength(3);
    expect(name(h.undo())).toBe('E');
    expect(name(h.undo())).toBe('D');
    expect(name(h.undo())).toBe('C');
    expect(h.undo()).toBeUndefined();
  });

  it('joins a burst from one source into one step (typing in a number box)', () => {
    const { h, snap, name } = setup();
    h.record('SetLevel', { action: 'level', page: 'Tree' }, snap('B'), 0);
    h.record('SetLevel', { action: 'level', page: 'Tree' }, snap('C'), 900);
    h.record('SetLevel', { action: 'level', page: 'Tree' }, snap('D'), 1800);
    expect(h.view().steps).toHaveLength(1);
    expect(name(h.undo())).toBe('A');
    expect(name(h.redo())).toBe('D');
  });

  it('does not join the same source once the burst pauses', () => {
    const { h, snap } = setup();
    h.record('SetLevel', L, snap('B'), 0);
    h.record('SetLevel', L, snap('C'), 1000);
    expect(h.view().steps).toHaveLength(2);
  });

  it('joins a different source only within the short window, keeping the first label', () => {
    const { h, snap } = setup();
    h.record('SetClass', { action: 'change class', page: 'Tree' }, snap('B'), 0);
    h.record('SetAscendancy', { action: 'change ascendancy', page: 'Tree' }, snap('C'), 250);
    expect(h.view().steps.map((s) => s.action)).toEqual(['change class']);
    h.record('SetAscendancy', { action: 'change ascendancy', page: 'Tree' }, snap('D'), 600);
    expect(h.view().steps.map((s) => s.action)).toEqual(['change ascendancy', 'change class']);
  });

  it('never joins across an undo, nor when told not to', () => {
    const { h, snap } = setup();
    h.record('s', L, snap('B'), 0);
    h.undo();
    h.redo();
    h.record('s', L, snap('C'), 1);
    expect(h.view().steps).toHaveLength(2);
    h.record('t', L, snap('D'), 2, false);
    expect(h.view().steps).toHaveLength(3);
  });

  it('goes to any step, and to the base', () => {
    const { h, snap, name } = setup();
    for (const [i, n] of ['B', 'C', 'D'].entries()) {
      h.record('s', L, snap(n), i * 10_000);
    }
    expect(name(h.goTo(2))).toBe('B');
    expect(h.view().undone).toBe(2);
    expect(name(h.goTo(3))).toBe('A');
    expect(name(h.goTo(0))).toBe('D');
    expect(h.goTo(4)).toBeUndefined();
    expect(h.goTo(-1)).toBeUndefined();
  });

  it('keeps every state exact through a compaction', () => {
    const { h, snap, name } = setup(2);
    for (let i = 0; i < 6000; i++) {
      h.record('s', L, snap(`S${i}`), i * 10_000);
    }
    expect(h.table.size).toBeLessThan(6000);
    expect(name(h.current())).toBe('S5999');
    expect(name(h.undo())).toBe('S5998');
    expect(name(h.undo())).toBe('S5997');
  });

  it('clear makes nothing undoable until the next reset', () => {
    const { h, snap } = setup();
    h.record('s', L, snap('B'), 0);
    h.clear();
    expect(h.open).toBe(false);
    expect(h.record('s', L, snap('C'), 10_000)).toBe(false);
    expect(h.undo()).toBeUndefined();
  });
});

describe('sameSnapshot / unrecordedLabel', () => {
  it('compares by content', () => {
    const t = new LineTable();
    expect(sameSnapshot(t.encode('a\nb'), t.encode('a\nb'))).toBe(true);
    expect(sameSnapshot(t.encode('a\nb'), t.encode('a\nc'))).toBe(false);
    expect(sameSnapshot(t.encode('a'), t.encode('a\n'))).toBe(false);
  });

  it('names a notes-only change, and anything else generically', () => {
    const xml = (notes: string, level: number) =>
      `<PathOfBuilding>\n<Build level="${level}"/>\n<Notes>${notes}</Notes>\n<Items/>\n<RichNotes v="1">${notes}</RichNotes>\n</PathOfBuilding>`;
    expect(unrecordedLabel(xml('a', 1), xml('b', 1))).toEqual({ action: 'Edited the notes', page: 'Notes', kind: 'notes' });
    expect(unrecordedLabel(xml('a', 1), xml('a', 2)).action).toBe('Other changes');
    expect(unrecordedLabel(xml('a', 1), xml('b', 2)).action).toBe('Other changes');
  });
});

describe('STEP_LABELS', () => {
  const files = () => {
    const src = resolve(process.cwd(), 'src');
    return readdirSync(src, { recursive: true, encoding: 'utf8' })
      .filter((f) => /\.(ts|svelte)$/.test(f) && !/\.test\.ts$/.test(f))
      .map((f) => ({ file: f, text: readFileSync(resolve(src, f), 'utf8').replace(/^\s*\/\/.*$/gm, '') }));
  };
  const WRITE = /\b(?:Tick|edit|UITick|partyWrite)\(\s*'([^']+)'(\s*\))?/g;
  const writes = () => files().flatMap(({ file, text }) => [...text.matchAll(WRITE)].map((m) => ({ file, source: m[1], bare: !!m[2] })));

  it('names every step a write can record, so the top bar never shows a source name', () => {
    const sources = new Set(writes().map((w) => w.source));
    expect(sources, 'the scan found the worker’s mutators').toContain('SetLevel');
    expect(sources, 'the scan found the pages’ UITick calls').toContain('socketGroups');
    const unlabelled = [...sources].filter((s) => !STEP_LABELS[s] && !NOT_A_CHANGE.has(s) && !OPEN_SOURCES.has(s));
    expect(unlabelled).toEqual([]);
  });

  it('gives every write a label built from its arguments, or names it from what changed', () => {
    const all = writes();
    expect(all.filter((w) => w.file.endsWith('sync_worker.ts')).length, 'the scan found the worker’s writes').toBeGreaterThan(80);
    const generic = all.filter((w) => w.bare && !DIFF_LABELS[w.source] && !NOT_A_CHANGE.has(w.source) && !OPEN_SOURCES.has(w.source));
    expect(generic.map((w) => `${w.file}: ${w.source}`)).toEqual([]);
  });

  it('records every worker write through the labelled path', () => {
    const worker = files().find((f) => f.file.endsWith('go/sync_worker.ts'))!.text;
    const ticks = [...worker.matchAll(/this\.Tick\(([^)]*)\)/g)].map((m) => m[1]);
    expect(ticks.sort()).toEqual(["'EstimatePlayerProgress'", "'setCalcTabElements'", 'source', 'source', 'source, xml']);
  });
});
