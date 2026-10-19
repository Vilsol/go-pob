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

type Point = { x: number; y: number };
export type GestureInput = { pointerId: number; clientX: number; clientY: number; timeStamp: number; type: string };
export type GestureHandlers = {
  // `at` is where the press started, in client coordinates
  gesture: (g: Gesture, at: Point) => void;
  // one finger dragging, in client pixels
  pan: (dx: number, dy: number) => void;
  // two fingers: the change in their distance since the last move, about their midpoint
  pinch: (factor: number, mid: Point) => void;
};

const timer = (fn: () => void, ms: number) => {
  const t = setTimeout(fn, ms);
  return () => clearTimeout(t);
};

// feeds a canvas's pointer events through the classifier: one finger pans, two pinch, and taps, double taps and
// holds come out as gestures; `node` is the node under the press, if any
export function gestureDriver(on: GestureHandlers, hold: (fn: () => void, ms: number) => () => void = timer) {
  const pts = new Map<number, Point>();
  let pinch: number | undefined;
  let g = initialGesture;
  let at: Point = { x: 0, y: 0 };
  let cancelHold = () => {};
  const act = (r: { state: GestureState; gesture?: Gesture }) => {
    g = r.state;
    if (!g.press) cancelHold();
    if (r.gesture) on.gesture(r.gesture, at);
  };
  const spread = () => {
    const [a, b] = [...pts.values()];
    return { d: Math.hypot(a.x - b.x, a.y - b.y), mid: { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 } };
  };
  return {
    down: (e: GestureInput, node?: number) => {
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pts.size === 1) at = { x: e.clientX, y: e.clientY };
      act(stepGesture(g, { type: 'down', t: e.timeStamp, x: e.clientX, y: e.clientY, pointer: e.pointerId, node }));
      if (g.press) cancelHold = hold(() => act(stepGesture(g, { type: 'timer', t: e.timeStamp + HOLD_MS })), HOLD_MS);
      if (pts.size === 2) pinch = spread().d;
    },
    move: (e: GestureInput) => {
      const p = pts.get(e.pointerId);
      if (!p) return;
      act(stepGesture(g, { type: 'move', t: e.timeStamp, x: e.clientX, y: e.clientY, pointer: e.pointerId }));
      const dx = e.clientX - p.x;
      const dy = e.clientY - p.y;
      p.x = e.clientX;
      p.y = e.clientY;
      if (pinch !== undefined && pts.size === 2) {
        const s = spread();
        if (pinch > 0 && s.d > 0) on.pinch(s.d / pinch, s.mid);
        pinch = s.d;
      } else if (pts.size === 1 && !g.press) on.pan(dx, dy);
    },
    up: (e: GestureInput) => {
      if (!pts.delete(e.pointerId)) return;
      if (pts.size < 2) pinch = undefined;
      act(
        stepGesture(
          g,
          e.type === 'pointercancel'
            ? { type: 'cancel', pointer: e.pointerId }
            : { type: 'up', t: e.timeStamp, x: e.clientX, y: e.clientY, pointer: e.pointerId }
        )
      );
    }
  };
}
