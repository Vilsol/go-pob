import { expect, within } from 'storybook/test';
import type { BuildFixture } from '../fixtures';

export const phone = { viewport: { value: 'phone', isRotated: false } };
export const phoneTall = { viewport: { value: 'phoneTall', isRotated: false } };

// the delta a tree allocation would show: the first node tip in the fixture that moves a default strip stat
export const treeDelta = (fx: BuildFixture) => {
  const all = Object.values(fx.tree.tips).map((t) => t[0]?.sections.flatMap((s) => s.lines) ?? []);
  return all.find((l) => l.some((x) => ['Total Life', 'Energy Shield', 'Total Mana', 'Full DPS', 'Combined DPS', 'Hit DPS'].includes(x.label))) ?? all[0] ?? [];
};

export const expectPhoneFits = async (canvasElement: HTMLElement, label: string, h = 769) => {
  const c = within(canvasElement);
  await expect(window.innerWidth).toBe(390);
  await expect(window.innerHeight).toBe(h);
  await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(390);
  await expect(document.documentElement.scrollHeight).toBeLessThanOrEqual(h);
  const nav = c.getByRole('navigation', { name: 'Pages' }).getBoundingClientRect();
  await expect(Math.round(nav.bottom)).toBeLessThanOrEqual(h);
  const main = c.getByRole('main', { name: label }).getBoundingClientRect();
  await expect(main.height).toBeGreaterThan(h / 2);
  await expect(Math.round(main.right)).toBeLessThanOrEqual(390);
};
