// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { float, pressedOutside, raise } from './floating';

const W = 1000;
const H = 800;
Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, get: () => W });
Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, get: () => H });

const box = (x: number, y: number, w: number, h: number) => {
  const el = document.body.appendChild(document.createElement('div'));
  el.getBoundingClientRect = () => ({ x, y, left: x, top: y, right: x + w, bottom: y + h, width: w, height: h, toJSON: () => ({}) });
  Object.defineProperty(el, 'offsetWidth', { get: () => w });
  Object.defineProperty(el, 'offsetHeight', { get: () => h });
  return el;
};
const settle = () => new Promise((r) => setTimeout(r, 0));

afterEach(() => {
  document.body.innerHTML = '';
});

describe('float', () => {
  it('opens below the anchor when there is room', async () => {
    const anchor = box(100, 100, 80, 30);
    const panel = box(0, 0, 200, 150);
    const stop = float(panel, anchor, { gap: 4 });
    await settle();
    expect([panel.style.left, panel.style.top]).toEqual(['100px', '134px']);
    stop();
  });

  it('flips above when the room below is short, and writes --x/--y in vars mode', async () => {
    const anchor = box(100, 700, 80, 30);
    const panel = box(0, 0, 200, 150);
    const stop = float(panel, anchor, { gap: 4, vars: true });
    await settle();
    expect([panel.style.getPropertyValue('--x'), panel.style.getPropertyValue('--y')]).toEqual(['100px', '546px']);
    expect(panel.style.left).toBe('');
    stop();
  });

  it('keeps the panel inside the viewport padding', async () => {
    const anchor = box(950, 100, 40, 30);
    const panel = box(0, 0, 200, 150);
    const stop = float(panel, anchor);
    await settle();
    expect(panel.style.left).toBe(`${W - 8 - 200}px`);
    stop();
  });

  it('beside an anchor, falls back to the left side', async () => {
    const anchor = box(900, 100, 40, 30);
    const tip = box(0, 0, 200, 150);
    const stop = float(tip, anchor, { placement: 'right-start', fallback: ['left-start'], gap: 10, vars: true });
    await settle();
    expect(tip.style.getPropertyValue('--x')).toBe(`${900 - 10 - 200}px`);
    stop();
  });

  it('fit caps the height to the larger side', async () => {
    const anchor = box(100, 300, 80, 30);
    const panel = box(0, 0, 200, 600);
    const stop = float(panel, anchor, { gap: 4, fit: true });
    await settle();
    // below: 800 - 330 - 4 - 8 = 458, above: 300 - 4 - 8 = 288
    expect(panel.style.maxHeight).toBe('458px');
    stop();
  });

  it('places at a point', async () => {
    const panel = box(0, 0, 100, 100);
    const stop = float(panel, { x: 300, y: 200 });
    await settle();
    expect([panel.style.left, panel.style.top]).toEqual(['300px', '200px']);
    stop();
  });
});

describe('raise', () => {
  it('makes a manual popover and shows it where the Popover API exists', () => {
    const el = document.body.appendChild(document.createElement('div'));
    const show = vi.fn();
    el.showPopover = show;
    raise(el);
    expect(el.getAttribute('popover')).toBe('manual');
    expect(show).toHaveBeenCalledOnce();
  });

  it('leaves the element alone without it, or when showing throws', () => {
    const plain = document.body.appendChild(document.createElement('div'));
    raise(plain);
    expect(plain.hasAttribute('popover')).toBe(false);
    const broken = document.body.appendChild(document.createElement('div'));
    broken.showPopover = () => {
      throw new Error('not supported');
    };
    raise(broken);
    expect(broken.hasAttribute('popover')).toBe(false);
  });
});

describe('pressedOutside', () => {
  it('is true only when the target is in none of the roots', () => {
    const a = document.body.appendChild(document.createElement('div'));
    const inner = a.appendChild(document.createElement('span'));
    const b = document.body.appendChild(document.createElement('div'));
    const ev = (target: EventTarget) => ({ target }) as unknown as Event;
    expect(pressedOutside(ev(inner), a)).toBe(false);
    expect(pressedOutside(ev(b), a)).toBe(true);
    expect(pressedOutside(ev(b), a, b)).toBe(false);
    expect(pressedOutside(ev(b), undefined)).toBe(true);
  });
});
