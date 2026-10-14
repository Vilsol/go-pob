// @vitest-environment jsdom

import { describe, expect, it, afterEach, beforeAll } from 'vitest';
import { render, cleanup, within } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import TreePage from './TreePage.svelte';
import type { TreeFixture, TreeNodeFx } from './fixtures';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const mastery = (picked?: number): TreeNodeFx => ({
  id: 7,
  k: 'm',
  x: 0.5,
  y: 0.5,
  name: 'Life Mastery',
  pts: 2,
  effects: [['+50 to maximum Life'], ['8% increased maximum Life'], ['Regenerate 1% of Life per second']],
  effect: -1,
  picked
});
const tree = (n: TreeNodeFx): TreeFixture => ({
  version: '3_29',
  classes: ['Witch'],
  ascendancies: ['None'],
  specs: ['Main'],
  activeSpec: 0,
  aspect: 1,
  radius: 0.1,
  nodes: [n],
  edges: [],
  tips: {},
  sockets: []
});
const open = (n: TreeNodeFx) =>
  render(TreePage, {
    tree: tree(n),
    className: 'Witch',
    ascendancy: 'None',
    alloc: {},
    mastery: n.id,
    onmastery: () => {},
    canvas: createRawSnippet(() => ({ render: () => '<div></div>' }))
  });

describe('the mastery modal marks the power report’s pick', () => {
  it('marks only the picked effect', () => {
    const view = open(mastery(1));
    const effects = within(view.getByRole('group', { name: 'Mastery effects' })).getAllByRole('button');
    expect(effects.map((b) => b.hasAttribute('data-power-pick'))).toEqual([false, true, false]);
    expect(effects[1].textContent).toContain('power pick');
  });

  it('marks nothing without a shown report', () => {
    const view = open(mastery());
    const effects = within(view.getByRole('group', { name: 'Mastery effects' })).getAllByRole('button');
    expect(effects.some((b) => b.hasAttribute('data-power-pick'))).toBe(false);
    expect(view.getByRole('group', { name: 'Mastery effects' }).textContent).not.toContain('power pick');
  });
});
