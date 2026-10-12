import { describe, expect, it } from 'vitest';
import { allocationCosts, calculateAllocationPath, extendTrace, withClusterPaths } from './paths';

const paths = { 1: -1, 2: 1, 3: 2, 10: -1, 11: 10 };

describe('calculateAllocationPath', () => {
  it('a node next to the allocation costs one point: the allocated anchor is not part of the path', () => {
    expect(calculateAllocationPath(paths, 2)).toEqual([2]);
  });

  it('a longer path lists every node to allocate and stops before the anchor', () => {
    expect(calculateAllocationPath(paths, 3)).toEqual([3, 2]);
  });

  it('a node next to the class start costs one point', () => {
    expect(calculateAllocationPath(paths, 11)).toEqual([11]);
  });

  it('an allocated node is its own path', () => {
    expect(calculateAllocationPath(paths, 1)).toEqual([1]);
  });

  it('a disconnected node has no path', () => {
    expect(calculateAllocationPath(paths, 99)).toBeUndefined();
  });
});

describe('allocationCosts', () => {
  it('gives every node the length calculateAllocationPath gives it', () => {
    const costs = allocationCosts(paths);
    for (const id of [...Object.keys(paths).map(Number), 99]) {
      expect(costs.get(id), `node ${id}`).toBe(calculateAllocationPath(paths, id)?.length);
    }
  });

  it('agrees on a long random forest of paths', () => {
    const big: Record<number, number> = {};
    let seed = 7;
    const rnd = () => (seed = (seed * 1103515245 + 12345) % 2 ** 31) / 2 ** 31;
    for (let id = 0; id < 3000; id++) big[id] = id < 10 ? -1 : Math.floor(rnd() * id);
    const costs = allocationCosts(big);
    for (let id = 0; id < 3000; id++) expect(costs.get(id), `node ${id}`).toBe(calculateAllocationPath(big, id)?.length);
  });
});

// PassiveTreeView.lua:327-358
describe('extendTrace (Shift path tracing)', () => {
  const links = new Set(['1-2', '2-3', '3-4', '3-5', '4-9', '5-6']);
  const linked = (a: number, b: number) => links.has(`${Math.min(a, b)}-${Math.max(a, b)}`);
  const mastery = (id: number) => id === 9;
  it('starts with the hovered node’s path, the allocated end first', () => {
    expect(extendTrace([], 3, [3, 2], linked, mastery)).toEqual([2, 3]);
  });
  it('adds a node linked to the last one, moves one already traced to the end, ignores the rest', () => {
    expect(extendTrace([2, 3], 5, [5, 3, 2], linked, mastery)).toEqual([2, 3, 5]);
    expect(extendTrace([2, 3, 4], 3, [3, 2], linked, mastery)).toEqual([2, 4, 3]);
    expect(extendTrace([2, 3], 6, [6, 5, 3, 2], linked, mastery)).toEqual([2, 3]);
    expect(extendTrace([2, 3], 3, [3, 2], linked, mastery)).toEqual([2, 3]);
  });
  it('stops after a mastery and ignores a node with no path', () => {
    expect(extendTrace([3, 4, 9], 4, [4], linked, mastery)).toEqual([3, 9, 4]);
    expect(extendTrace([2, 9], 4, [4, 3], linked, mastery)).toEqual([2, 9]);
    expect(extendTrace([2, 3], 4, undefined, linked, mastery)).toEqual([2, 3]);
  });
});

describe('withClusterPaths', () => {
  const link = (id: number, linked: number[], type = 'Normal') => ({ id, linked, type }) as const;
  const cluster = [link(3, [100]), link(100, [101]), link(101, [102, 103]), link(103, [], 'Mastery'), link(102, [104]), link(104, [])];

  it('paths a node inside a cluster jewel through the subgraph to the tree', () => {
    const p = withClusterPaths(paths, cluster);
    expect(calculateAllocationPath(p, 102)).toEqual([102, 101, 100, 3, 2]);
    expect(allocationCosts(p).get(104)).toBe(6);
  });

  it('starts from an allocated cluster node', () => {
    const p = withClusterPaths({ ...paths, 3: -1, 100: -1 }, cluster);
    expect(calculateAllocationPath(p, 102)).toEqual([102, 101]);
  });

  it('never paths through or to a mastery', () => {
    expect(withClusterPaths(paths, cluster)[103]).toBeUndefined();
  });
});
