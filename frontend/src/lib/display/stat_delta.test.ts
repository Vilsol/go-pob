import { describe, expect, it } from 'vitest';
import type { Outputs } from '../custom_types';
import { createEditDelta, isEdit } from './stat_delta';

const out = (Life: number, extra: Record<string, number> = {}, minionLife?: number): Outputs => ({
  Output: { Life, ...extra },
  ...(minionLife !== undefined && { Minion: { Output: { Life: minionLife } } })
});
const settled = (source: string) => ({ source, pending: 0 });
const text = (d?: { section: string; label: string; value: string; better: boolean }[]) =>
  d?.map((l) => `${l.section}:${l.label} ${l.value} ${l.better ? 'better' : 'worse'}`);

describe('the last edit’s stat delta', () => {
  it('shows nothing for the opened build, then the difference an edit makes', () => {
    const t = createEditDelta();
    expect(t(out(1000), settled('OpenBuild'))).toEqual([]);
    expect(text(t(out(1100), settled('AllocateNode')))).toEqual(['Player:Total Life +100 better']);
  });

  it('diffs a burst against the state before it, once nothing is in flight', () => {
    const t = createEditDelta();
    t(out(1000), settled('OpenBuild'));
    expect(t(out(1050), { source: 'SetLevel', pending: 1 })).toBeUndefined();
    expect(text(t(out(900), settled('SetLevel')))).toEqual(['Player:Total Life -100 worse']);
  });

  it('replaces the delta on the next edit, and clears it when that edit moves nothing', () => {
    const t = createEditDelta();
    t(out(1000), settled('OpenBuild'));
    t(out(1100), settled('AllocateNode'));
    expect(text(t(out(1150), settled('AllocateNode')))).toEqual(['Player:Total Life +50 better']);
    expect(t(out(1150), settled('AllocateNode'))).toEqual([]);
  });

  it('counts undo and redo as edits, but not a calc-tab selection', () => {
    expect([isEdit('Undo'), isEdit('Redo'), isEdit('UndoTo'), isEdit('setCalcTabElements'), isEdit('OpenBuild')]).toEqual([true, true, true, false, false]);
    const t = createEditDelta();
    t(out(1000), settled('OpenBuild'));
    t(out(1100), settled('AllocateNode'));
    expect(t(out(1100, { Mana: 5 }), settled('setCalcTabElements'))).toBeUndefined();
    expect(text(t(out(1000, { Mana: 5 }), settled('Undo')))).toEqual(['Player:Total Life -100 worse']);
  });

  it('opening a build clears the delta and never diffs across builds', () => {
    const t = createEditDelta();
    t(out(1000), settled('OpenBuild'));
    t(out(1100), settled('AllocateNode'));
    expect(t(out(5000), settled('ImportCode'))).toEqual([]);
    expect(text(t(out(5001), settled('AllocateNode')))).toEqual(['Player:Total Life +1 better']);
    expect(t(out(9000), { source: 'OpenBuild', pending: 1 })).toEqual([]);
    expect(t(out(9001), settled('AllocateNode'))).toBeUndefined();
    expect(text(t(out(9003), settled('AllocateNode')))).toEqual(['Player:Total Life +2 better']);
  });

  it('keeps minion and player lines apart', () => {
    const t = createEditDelta();
    t(out(1000, {}, 200), settled('OpenBuild'));
    expect(text(t(out(1000, {}, 260), settled('SetGemLevel')))).toEqual(['Minion:Total Life +60 better']);
  });
});
