import { describe, expect, it } from 'vitest';
import { DOUBLE_MS, HOLD_MS, SLOP_PX, initialGesture, stepGesture, type Gesture, type GestureEvent } from './tree_gestures';

const run = (events: GestureEvent[]) => {
  let s = initialGesture;
  const out: Gesture[] = [];
  for (const e of events) {
    const r = stepGesture(s, e);
    s = r.state;
    if (r.gesture) out.push(r.gesture);
  }
  return { out, state: s };
};

const down = (t: number, node?: number, x = 100, y = 100, pointer = 1): GestureEvent => ({ type: 'down', t, x, y, pointer, node });
const up = (t: number, x = 100, y = 100, pointer = 1): GestureEvent => ({ type: 'up', t, x, y, pointer });
const move = (t: number, x: number, y = 100, pointer = 1): GestureEvent => ({ type: 'move', t, x, y, pointer });
const tap = (t: number, node?: number) => [down(t, node), up(t + 80)];

describe('phone tree gestures', () => {
  it('a short still press is a tap on its node, or on empty space', () => {
    expect(run(tap(0, 7)).out).toEqual([{ kind: 'tap', node: 7 }]);
    expect(run(tap(0)).out).toEqual([{ kind: 'tap', node: undefined }]);
  });

  it('a press released just under the hold time is a tap; at the hold time it is a hold', () => {
    expect(run([down(0, 7), up(HOLD_MS - 1)]).out).toEqual([{ kind: 'tap', node: 7 }]);
    expect(run([down(0, 7), up(HOLD_MS)]).out).toEqual([{ kind: 'hold', node: 7 }]);
  });

  it('the hold timer fires the menu once, and the release after it adds nothing', () => {
    const r = run([down(0, 7), { type: 'timer', t: HOLD_MS - 1 }, { type: 'timer', t: HOLD_MS }, { type: 'timer', t: HOLD_MS + 50 }, up(HOLD_MS + 200)]);
    expect(r.out).toEqual([{ kind: 'hold', node: 7 }]);
  });

  it('moving exactly the slop is still a tap; one pixel more cancels it', () => {
    expect(run([down(0, 7), move(20, 100 + SLOP_PX), up(40, 100 + SLOP_PX)]).out).toEqual([{ kind: 'tap', node: 7 }]);
    expect(run([down(0, 7), move(20, 100 + SLOP_PX + 1), up(40, 100)]).out).toEqual([]);
    expect(run([down(0, 7), up(40, 100, 100 + SLOP_PX + 1)]).out).toEqual([]);
  });

  it('movement past the slop also cancels a hold', () => {
    expect(run([down(0, 7), move(100, 130), { type: 'timer', t: HOLD_MS }, up(HOLD_MS + 10, 130)]).out).toEqual([]);
  });

  it('two taps on the same node within the window are a double tap', () => {
    expect(run([...tap(0, 7), ...tap(80 + DOUBLE_MS, 7)]).out).toEqual([
      { kind: 'tap', node: 7 },
      { kind: 'double', node: 7 }
    ]);
  });

  it('one millisecond past the window, the second tap is only a tap', () => {
    expect(run([...tap(0, 7), ...tap(80 + DOUBLE_MS + 1, 7)]).out).toEqual([
      { kind: 'tap', node: 7 },
      { kind: 'tap', node: 7 }
    ]);
  });

  it('a double tap matches the node id, not the screen position', () => {
    expect(run([...tap(0, 7), ...tap(150, 8)]).out).toEqual([
      { kind: 'tap', node: 7 },
      { kind: 'tap', node: 8 }
    ]);
    expect(run([...tap(0), ...tap(150)]).out).toEqual([
      { kind: 'tap', node: undefined },
      { kind: 'tap', node: undefined }
    ]);
    expect(run([down(0, 7, 100, 100), up(50, 100, 100), down(150, 7, 300, 300), up(200, 300, 300)]).out.map((g) => g.kind)).toEqual(['tap', 'double']);
  });

  it('a third tap after a double starts over', () => {
    expect(run([...tap(0, 7), ...tap(150, 7), ...tap(300, 7)]).out.map((g) => g.kind)).toEqual(['tap', 'double', 'tap']);
  });

  it('a pan between the two taps breaks the double tap', () => {
    const r = run([...tap(0, 7), down(120), move(140, 160), up(160, 160), ...tap(200, 7)]);
    expect(r.out).toEqual([
      { kind: 'tap', node: 7 },
      { kind: 'tap', node: 7 }
    ]);
  });

  it('a second finger cancels the press, and lifting the fingers adds nothing', () => {
    const r = run([down(0, 7), down(30, undefined, 200, 200, 2), up(60, 100, 100, 1), up(70, 200, 200, 2)]);
    expect(r.out).toEqual([]);
    expect(r.state.pointers).toBe(0);
  });

  it('a pinch between two taps breaks the double tap', () => {
    const r = run([...tap(0, 7), down(100, 7), down(110, undefined, 200, 200, 2), up(150), up(160, 200, 200, 2), ...tap(200, 7)]);
    expect(r.out.map((g) => g.kind)).toEqual(['tap', 'tap']);
  });

  it('a cancelled pointer is neither a tap nor a hold', () => {
    expect(run([down(0, 7), { type: 'cancel', pointer: 1 }]).out).toEqual([]);
    expect(run([down(0, 7), { type: 'cancel', pointer: 1 }]).state.pointers).toBe(0);
  });
});
