// Touch gestures on the phone tree canvas: tap = hover, double tap on the same node = allocate,
// hold = the node menu; moving past the slop or a second finger cancels all three (pan / pinch).
export const SLOP_PX = 8;
export const HOLD_MS = 450;
export const DOUBLE_MS = 300;

export type GestureEvent =
  | { type: 'down'; t: number; x: number; y: number; pointer: number; node?: number }
  | { type: 'move' | 'up'; t: number; x: number; y: number; pointer: number }
  | { type: 'cancel'; pointer: number }
  | { type: 'timer'; t: number };

export type Gesture = { kind: 'tap' | 'double' | 'hold'; node?: number };

type Press = { t: number; x: number; y: number; pointer: number; node?: number; held: boolean };

export type GestureState = {
  pointers: number;
  press?: Press;
  // the last completed tap, a double tap's first half
  last?: { t: number; node?: number };
};

export const initialGesture: GestureState = { pointers: 0 };

const far = (p: Press, x: number, y: number) => Math.hypot(x - p.x, y - p.y) > SLOP_PX;

export function stepGesture(s: GestureState, e: GestureEvent): { state: GestureState; gesture?: Gesture } {
  switch (e.type) {
    case 'down': {
      const pointers = s.pointers + 1;
      if (pointers > 1) return { state: { pointers } };
      return { state: { pointers, last: s.last, press: { t: e.t, x: e.x, y: e.y, pointer: e.pointer, node: e.node, held: false } } };
    }
    case 'move':
      if (s.press?.pointer === e.pointer && far(s.press, e.x, e.y)) return { state: { pointers: s.pointers } };
      return { state: s };
    case 'timer':
      if (!s.press || s.press.held || e.t - s.press.t < HOLD_MS) return { state: s };
      return { state: { pointers: s.pointers, press: { ...s.press, held: true } }, gesture: { kind: 'hold', node: s.press.node } };
    case 'cancel':
    case 'up': {
      const pointers = Math.max(0, s.pointers - 1);
      const p = s.press;
      if (!p || p.pointer !== e.pointer) return { state: { pointers, last: s.last } };
      if (e.type === 'cancel' || p.held || far(p, e.x, e.y)) return { state: { pointers } };
      if (e.t - p.t >= HOLD_MS) return { state: { pointers }, gesture: { kind: 'hold', node: p.node } };
      const last = s.last;
      if (last && p.node !== undefined && last.node === p.node && p.t - last.t <= DOUBLE_MS)
        return { state: { pointers }, gesture: { kind: 'double', node: p.node } };
      return { state: { pointers, last: { t: e.t, node: p.node } }, gesture: { kind: 'tap', node: p.node } };
    }
  }
}
