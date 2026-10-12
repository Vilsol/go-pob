// @vitest-environment jsdom
// PassiveTreeView.lua:1158-1204

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render } from '@testing-library/svelte';
import JewelRing from './JewelRing.svelte';
import type { Sprite } from '$lib/skill_tree/types';

afterEach(cleanup);

const sheet: Sprite = {
  filename: 'https://example/jewel-radius.png',
  w: 1000,
  h: 500,
  coords: { KaruiJewelCircle1: { x: 100, y: 0, w: 200, h: 200 }, KaruiJewelCircle2: { x: 300, y: 0, w: 200, h: 200 } }
};
const layers = (c: HTMLElement) =>
  [...c.querySelectorAll<HTMLElement>('[data-ring-layer]')].map((l) => ({
    name: l.dataset.ringLayer,
    size: l.style.width,
    turn: `${l.style.getPropertyValue('--turn-period')} ${l.style.getPropertyValue('--turn-dir')}`
  }));

describe('the jewel radius ring art', () => {
  it("draws PoB's shaded outer and inner rings for a jewel without its own art", () => {
    const { container } = render(JewelRing, { nodeId: 1, x: 100, y: 100, outer: 50, inner: 20, sheet });
    expect(layers(container)).toEqual([
      { name: 'ShadedOuterRing', size: '100px', turn: '299.2s reverse' },
      { name: 'ShadedOuterRingFlipped', size: '100px', turn: '299.2s normal' },
      { name: 'ShadedInnerRing', size: '40px', turn: '299.2s reverse' },
      { name: 'ShadedInnerRingFlipped', size: '40px', turn: '299.2s normal' }
    ]);
  });

  it("draws a timeless jewel's two circles from the jewel-radius sheet", () => {
    const { container } = render(JewelRing, { nodeId: 1, x: 100, y: 100, outer: 50, inner: 20, art: 'Lethal Pride', sheet });
    expect(layers(container)).toEqual([
      { name: 'KaruiJewelCircle1', size: '100px', turn: '299.2s reverse' },
      { name: 'KaruiJewelCircle2', size: '100px', turn: '299.2s normal' }
    ]);
    const first = container.querySelector<HTMLElement>('[data-ring-layer="KaruiJewelCircle1"]')!;
    expect(first.style.backgroundSize).toBe('500px 250px');
    expect(first.style.backgroundPosition).toBe('-50px 0px');
  });

  it("turns Impossible Escape's rings as PoB does on its keystone", () => {
    const { container } = render(JewelRing, { nodeId: 1, x: 100, y: 100, outer: 50, inner: 15, keystone: true, sheet });
    expect(layers(container).map((l) => l.turn)).toEqual(['261.8s reverse', '209.4s normal', '174.5s reverse', '209.4s normal']);
  });
});
