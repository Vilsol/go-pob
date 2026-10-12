export type PrecalculatedAllocationPaths = Record<number, number>;

export function calculateAllocationPath(precalculatedPaths: PrecalculatedAllocationPaths, targetNodeId: number): number[] | undefined {
  const maxDepth = Object.keys(precalculatedPaths).length;
  if (maxDepth === 0) {
    return undefined;
  }

  const path = [targetNodeId];
  for (let i = 0; i <= maxDepth; i++) {
    const nextHop = precalculatedPaths[targetNodeId];
    if (nextHop === undefined) {
      return undefined;
    }
    if (nextHop === -1) {
      return path;
    }
    if (precalculatedPaths[nextHop] === -1) {
      return path;
    }
    path.push(nextHop);
    targetNodeId = nextHop;
  }

  throw new Error(`CalculateAllocationPaths produced a loop: ${JSON.stringify(path)}`);
}

export function allocationCosts(precalculatedPaths: PrecalculatedAllocationPaths): Map<number, number> {
  const costs = new Map<number, number>();
  const stack: number[] = [];
  const keys = Object.keys(precalculatedPaths);
  for (const key of keys) {
    let id = Number(key);
    while (!costs.has(id)) {
      const next = precalculatedPaths[id];
      if (next === undefined) break;
      if (next === -1 || precalculatedPaths[next] === -1) {
        costs.set(id, 1);
        break;
      }
      if (stack.length > keys.length) {
        throw new Error(`CalculateAllocationPaths produced a loop: ${JSON.stringify(stack)}`);
      }
      stack.push(id);
      id = next;
    }
    let cost = costs.get(id);
    while (stack.length) {
      const back = stack.pop()!;
      cost = cost === undefined ? undefined : cost + 1;
      if (cost !== undefined) costs.set(back, cost);
    }
  }
  return costs;
}

/* PassiveSpec.lua:1933-1940, 2007, 2253, PassiveSpec.lua:2051-2064 */
export function withClusterPaths(
  precalculatedPaths: PrecalculatedAllocationPaths,
  cluster: readonly { id: number; type: string; linked: readonly number[] }[]
): PrecalculatedAllocationPaths {
  if (!cluster.length) return precalculatedPaths;
  const masteries = new Set(cluster.filter((c) => c.type === 'Mastery').map((c) => c.id));
  const adjacent = new Map<number, number[]>();
  const link = (a: number, b: number) => adjacent.set(a, [...(adjacent.get(a) ?? []), b]);
  for (const c of cluster) {
    if (masteries.has(c.id)) continue;
    for (const o of c.linked) {
      if (masteries.has(o)) continue;
      link(c.id, o);
      link(o, c.id);
    }
  }
  const out = { ...precalculatedPaths };
  const depth = new Map<number, number>();
  const depthOf = (id: number): number => {
    const known = depth.get(id);
    if (known !== undefined) return known;
    const next = out[id];
    const d = next === -1 ? 0 : depthOf(next) + 1;
    depth.set(id, d);
    return d;
  };
  const frontier = [...adjacent.keys()].filter((id) => out[id] !== undefined).map((id) => ({ id, d: depthOf(id) }));
  while (frontier.length) {
    frontier.sort((a, b) => a.d - b.d || a.id - b.id);
    const { id, d } = frontier.shift()!;
    for (const o of adjacent.get(id) ?? []) {
      if (out[o] !== undefined) continue;
      out[o] = id;
      depth.set(o, d + 1);
      frontier.push({ id: o, d: d + 1 });
    }
  }
  return out;
}

/* PassiveTreeView.lua:327-358 */
export function extendTrace(
  trace: number[],
  hovered: number,
  path: number[] | undefined,
  linked: (a: number, b: number) => boolean,
  isMastery: (id: number) => boolean
): number[] {
  if (!path) return trace;
  if (!trace.length) return [...path].reverse();
  const last = trace[trace.length - 1];
  if (hovered === last || !linked(hovered, last)) return trace;
  const index = trace.indexOf(hovered);
  if (index >= 0) return [...trace.slice(0, index), ...trace.slice(index + 1), hovered];
  if (isMastery(last)) return trace;
  return [...trace, hovered];
}
