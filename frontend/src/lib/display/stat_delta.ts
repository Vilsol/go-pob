import { writable } from 'svelte/store';
import type { Outputs, TickMeta } from '../custom_types';
import { NOT_A_CHANGE, OPEN_SOURCES } from '../go/undo';
import { addStatCompares, type StatCompareLine } from '../skill_tree/stat_compare';

// section matches the Sidebar's: 'Minion' or 'Player'
export type EditDeltaLine = StatCompareLine & { section: string };

const UNDO = new Set(['Undo', 'Redo', 'UndoTo']);
export const isEdit = (source: string) => !OPEN_SOURCES.has(source) && (!NOT_A_CHANGE.has(source) || UNDO.has(source));

export const editDiff = (before: Outputs, after: Outputs): EditDeltaLine[] =>
  addStatCompares(
    after.SkillFlags ?? {},
    { Output: before.Output ?? {}, Minion: before.Minion?.Output },
    { Output: after.Output ?? {}, Minion: after.Minion?.Output }
  ).flatMap((s) => s.lines.map((l) => ({ ...l, section: s.actor ?? 'Player' })));

// the baseline is the last settled output; an edit's delta is shown once its recalculation settles (no tick in
// flight), so a burst of ticks from one edit diffs against the state before the burst. Returns undefined when the
// shown delta stays as it is.
export const createEditDelta = () => {
  let base: Outputs | undefined;
  let edited = false;
  return (out: Outputs | undefined, meta: Pick<TickMeta, 'source' | 'pending'>): EditDeltaLine[] | undefined => {
    const open = OPEN_SOURCES.has(meta.source);
    if (open) {
      base = undefined;
      edited = false;
    } else if (isEdit(meta.source)) {
      edited = true;
    }
    if (meta.pending > 0 || !out) return open ? [] : undefined;
    const before = base;
    base = out;
    if (!edited || !before) return open ? [] : undefined;
    edited = false;
    return editDiff(before, out);
  };
};

export const editDelta = writable<EditDeltaLine[]>([]);
const track = createEditDelta();
export const trackEditDelta = (out: Outputs | undefined, meta: TickMeta) => {
  const d = track(out, meta);
  if (d) editDelta.set(d);
};
