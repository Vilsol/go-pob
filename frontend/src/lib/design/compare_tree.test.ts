// @vitest-environment jsdom
// CompareTab.lua:746-764, 3374-3458, 856-913

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import CompareTree from './CompareTree.svelte';

afterEach(cleanup);

const tree = {
  nodes: [
    { id: 1, name: 'Resolute Technique', kind: 'k', owner: 'a' as const, x: 0.1, y: 0.1 },
    { id: 2, name: 'Iron Reflexes', kind: 'k', owner: 'b' as const, x: 0.5, y: 0.5 },
    { id: 3, name: 'Strength', kind: 's', owner: 'both' as const, x: 0.3, y: 0.3 }
  ],
  edges: [[0, 2]] as [number, number][]
};

const nodesIn = (label: string) => [...screen.getByRole('region', { name: label }).querySelectorAll('[data-kind]')].map((n) => n.getAttribute('title'));

describe('CompareTree', () => {
  it('shows one overlay, then each tree alone side by side', async () => {
    render(CompareTree, { tree, aPoints: 2, bPoints: 2, onnode: () => {} });
    expect(nodesIn('Tree overlay')).toHaveLength(3);
    await fireEvent.click(screen.getByRole('checkbox', { name: 'Overlay comparison' }));
    expect(screen.queryByRole('region', { name: 'Tree overlay' })).toBeNull();
    expect(nodesIn('Tree A')).toEqual(['Resolute Technique (only A)', 'Strength (both)']);
    expect(nodesIn('Tree B')).toEqual(['Iron Reflexes (only B)', 'Strength (both)']);
  });

  it('rings what each tree’s own search matches', async () => {
    render(CompareTree, { tree, aPoints: 2, bPoints: 2, onnode: () => {} });
    await fireEvent.click(screen.getByRole('checkbox', { name: 'Overlay comparison' }));
    await fireEvent.input(screen.getByRole('searchbox', { name: 'Search Tree A' }), { target: { value: 'stren' } });
    const matched = (label: string) => [...screen.getByRole('region', { name: label }).querySelectorAll('[data-match]')].map((n) => n.getAttribute('title'));
    expect(matched('Tree A')).toEqual(['Strength (both)']);
    expect(matched('Tree B')).toEqual([]);
  });
});
