import { describe, expect, it } from 'vitest';
import { HOLD_MS, gestureDriver, type Gesture } from './tree_gestures';

const setup = () => {
  const log: string[] = [];
  let fire: (() => void) | undefined;
  const d = gestureDriver(
    {
      gesture: (g: Gesture, at) => log.push(`${g.kind} ${g.node} @${at.x},${at.y}`),
      pan: (dx, dy) => log.push(`pan ${dx},${dy}`),
      pinch: (f, mid) => log.push(`pinch ${f} @${mid.x},${mid.y}`)
    },
    (fn) => {
      fire = fn;
      return () => (fire = undefined);
    }
  );
  const ev = (type: string, t: number, x: number, y = 0, pointerId = 1) => ({ type, timeStamp: t, clientX: x, clientY: y, pointerId });
  return { log, d, ev, hold: () => fire?.() };
};

describe('the phone tree gesture driver', () => {
  it('a tap reports its node and where the press started; a second tap on it is a double tap', () => {
    const { log, d, ev } = setup();
    d.down(ev('pointerdown', 0, 10, 20), 7);
    d.up(ev('pointerup', 50, 11, 20));
    d.down(ev('pointerdown', 150, 10, 20), 7);
    d.up(ev('pointerup', 200, 10, 20));
    expect(log).toEqual(['tap 7 @10,20', 'double 7 @10,20']);
  });

  it('a hold fires from the timer while the finger is still down, and its release is not a tap', () => {
    const { log, d, ev, hold } = setup();
    d.down(ev('pointerdown', 0, 5, 5), 3);
    hold();
    d.up(ev('pointerup', HOLD_MS + 100, 5, 5));
    expect(log).toEqual(['hold 3 @5,5']);
  });

  it('one finger past the slop pans and taps nothing; the timer is cancelled', () => {
    const { log, d, ev, hold } = setup();
    d.down(ev('pointerdown', 0, 0), 3);
    d.move(ev('pointermove', 10, 20));
    d.move(ev('pointermove', 20, 25));
    hold();
    d.up(ev('pointerup', 30, 25));
    expect(log).toEqual(['pan 20,0', 'pan 5,0']);
  });

  it('two fingers pinch by the ratio of their distances about the midpoint', () => {
    const { log, d, ev } = setup();
    d.down(ev('pointerdown', 0, 0, 0, 1));
    d.down(ev('pointerdown', 0, 100, 0, 2));
    d.move(ev('pointermove', 10, 200, 0, 2));
    d.up(ev('pointerup', 20, 200, 0, 2));
    d.up(ev('pointerup', 20, 0, 0, 1));
    expect(log).toEqual(['pinch 2 @100,0']);
  });

  it('a release of a pointer that never pressed the canvas leaves a press on it alone', () => {
    const { log, d, ev } = setup();
    d.down(ev('pointerdown', 0, 0, 0, 1), 3);
    d.up(ev('pointerup', 10, 300, 0, 2));
    d.up(ev('pointerup', 20, 0, 0, 1));
    expect(log).toEqual(['tap 3 @0,0']);
  });
});
