// @vitest-environment jsdom
// ItemSlotControl.lua:132-149

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import SocketPreview from './SocketPreview.svelte';
import type { Tree } from '$lib/skill_tree/types';
import type { Tree2, Tree2Node } from '$lib/components/skill-tree/scene2';

afterEach(cleanup);

const tree = {
  constants: { orbitRadii: [0, 82], skillsPerOrbit: [1, 6], classes: {}, characterAttributes: {}, PSSCentreInnerRadius: 0 },
  groups: { 1: { x: 0, y: 0, orbits: [0, 1], nodes: ['10', '11', '12'] }, 2: { x: 5000, y: 0, orbits: [0], nodes: ['13'] } },
  nodes: {
    10: { skill: 10, group: 1, orbit: 0, orbitIndex: 0, isJewelSocket: true, out: ['11', '13'] },
    11: { skill: 11, group: 1, orbit: 1, orbitIndex: 0, out: ['12'] },
    12: { skill: 12, group: 1, orbit: 1, orbitIndex: 3, isNotable: true },
    13: { skill: 13, group: 2, orbit: 0, orbitIndex: 0 }
  },
  min_x: -100,
  min_y: -100
} as unknown as Tree;

describe('SocketPreview', () => {
  it('draws the nodes near the socket, lit when allocated, outlined when in a radius', () => {
    const { container, getByRole } = render(SocketPreview, {
      tree,
      node: 10,
      radii: [{ inner: 0, outer: 800, col: '^xFF0000', nodes: [12] }],
      alloc: new Set([11]),
      label: 'Passive tree around socket #1'
    });
    expect(getByRole('img', { name: 'Passive tree around socket #1' })).toBeTruthy();
    const ids = [...container.querySelectorAll('circle[data-node]')].map((c) => c.getAttribute('data-node'));
    expect(ids).toEqual(['10', '11', '12']);
    expect(container.querySelector('circle[data-node="11"]')?.getAttribute('data-alloc')).toBe('true');
    expect(container.querySelector('circle[data-node="12"]')?.getAttribute('stroke')).toBe('#FF0000');
  });

  it('connects the drawn nodes as the tree does: an arc along a shared orbit, else a line', () => {
    const { container } = render(SocketPreview, { tree, node: 10, radii: [], alloc: new Set([10, 11]), label: 'socket' });
    const edges = [...container.querySelectorAll<SVGPathElement>('path[data-edge]')];
    expect(edges.map((e) => e.dataset.edge)).toEqual(['10-11', '11-12']);
    expect(edges[0].getAttribute('d')).not.toContain('A');
    expect(edges[1].getAttribute('d')).toContain('A');
    expect(edges[0].dataset.alloc).toBe('true');
    expect(edges[1].dataset.alloc).toBeUndefined();
  });

  // ItemSlotHelper.lua:10-41 (PoB-PoE2): a PoE2 tree, its connections an arc on an orbit or a line
  it('draws a PoE2 tree around the socket, connected as its connectors run', () => {
    const n = (id: number, x: number, y: number, more: Partial<Tree2Node>): Tree2Node => ({
      id,
      x,
      y,
      angle: 0,
      group: 1,
      orbit: 0,
      type: 'Normal',
      name: `n${id}`,
      stats: null,
      connections: null,
      ...more
    });
    const tree2: Tree2 = {
      nodes: [
        n(10, 0, 0, { type: 'Socket', containJewelSocket: true, connections: [{ id: 11, orbit: 0 }] }),
        n(11, 300, 0, { group: 2, connections: [{ id: 12, orbit: 1 }] }),
        n(12, 300, 300, { group: 2, type: 'Notable' }),
        n(13, 900000, 0, {})
      ],
      groups: [
        { id: 1, x: 0, y: 0 },
        { id: 2, x: 300, y: 150 }
      ],
      orbitRadii: [0, 250],
      minX: -40000,
      minY: -40000,
      maxX: 40000,
      maxY: 40000
    };
    const { container } = render(SocketPreview, { tree: tree2, node: 10, radii: [], alloc: new Set([10, 11]), label: 'socket' });
    expect([...container.querySelectorAll('circle[data-node]')].map((c) => c.getAttribute('data-node'))).toEqual(['10', '11', '12']);
    expect(container.querySelector('circle[data-node="11"]')?.getAttribute('data-alloc')).toBe('true');
    const edges = [...container.querySelectorAll<SVGPathElement>('path[data-edge]')];
    expect(edges.map((e) => e.dataset.edge)).toEqual(['10-11', '11-12']);
    expect(edges[0].getAttribute('d')).not.toContain('A');
    expect(edges[1].getAttribute('d')).toContain('A');
    expect(edges[0].dataset.alloc).toBe('true');
    expect(edges[1].dataset.alloc).toBeUndefined();
  });

  it('floats as a tooltip beside the slot it belongs to', () => {
    const anchor = document.createElement('div');
    document.body.append(anchor);
    const { getByRole } = render(SocketPreview, { props: { tree, node: 10, radii: [], alloc: new Set<number>(), label: 'socket', anchor } });
    const panel = getByRole('img', { name: 'socket' }).closest('[data-socket-preview]');
    expect(panel?.classList.contains('tooltip')).toBe(true);
    expect(panel?.classList.contains('fixed')).toBe(true);
    anchor.remove();
  });
});
