import { calculateAllocationPath, withClusterPaths } from '../../src/lib/components/skill-tree/paths';
import type { Tree } from '../../src/lib/skill_tree/types';
import { removeLegacyAlternateAscendancies } from '../../src/lib/skill_tree/geometry';
import * as go from './go_model';

type Build = Awaited<ReturnType<typeof go.loadXml>>;

let loaded: { tree: Tree; version: string; m: Awaited<ReturnType<typeof go.model>> } | undefined;

export const prepareTree = async () => {
  if (loaded) return;
  const m = await go.model();
  const version = m.exposition.CalcTreeVersion();
  const raw = await m.exposition.GetRawTree(version);
  if (!raw) {
    throw new Error(`REFUSING: the model has no ${version} tree`);
  }
  const tree = JSON.parse(new TextDecoder().decode(raw)) as Tree;
  removeLegacyAlternateAscendancies(tree);
  loaded = { tree, version, m };
};

const classStartNodes = (t: Tree): Record<number, number[]> => {
  const out: Record<number, number[]> = {};
  const nodes = Object.values(t.groups).flatMap((g) => (g.nodes ?? []).map((id) => t.nodes[id]));
  for (const node of nodes) {
    if (node.classStartIndex === undefined) continue;
    for (const n of [...(node.out || []), ...(node.in || [])]) {
      const target = t.nodes[n];
      if (target?.skill && target.ascendancyName === undefined) {
        (out[node.classStartIndex] ??= []).push(target.skill);
      }
    }
  }
  return out;
};

export const treeVersion = (): string => {
  if (!loaded) throw new Error('treeVersion() before prepareTree()');
  return loaded.version;
};

export const rawTree = (): Tree => {
  if (!loaded) throw new Error('rawTree() before prepareTree()');
  return loaded.tree;
};

export type TreeNode = { id: number; name: string };

export const treeState = (build: Build, cluster: { ID: number; Type: string; Linked?: number[] }[] = []) => {
  if (!loaded) {
    throw new Error('treeState() before prepareTree()');
  }
  const { tree: t, version, m } = loaded;
  const className = go.fresh(build, (b) => b.Build.ClassName);
  const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
  const roots = classStartNodes(t)[t.classes.findIndex((c) => c.name === className)];
  const paths = withClusterPaths(
    m.exposition.CalculateAllocationPaths(version, active, roots) ?? {},
    cluster.map((c) => ({ id: c.ID, type: c.Type, linked: c.Linked ?? [] }))
  );
  const selectable = (id: number) => {
    const n = t.nodes[id];
    return !!n?.name && n.classStartIndex === undefined && !n.isAscendancyStart && !n.isMastery;
  };
  const named = (id: number): TreeNode => ({ id, name: t.nodes[id].name! });
  // PassiveSpec.lua:856-857
  const depends = (id: number) => m.exposition.CalculateDependentNodes(version, active, id, roots) ?? [];

  return {
    active,
    path: (id: number) => calculateAllocationPath(paths, id),
    adjacent: () =>
      Object.entries(paths)
        .map(([id, hop]) => [Number(id), hop] as const)
        .filter(([id, hop]) => hop !== -1 && paths[hop] === -1 && selectable(id))
        .map(([id]) => named(id))
        .sort((a, b) => a.id - b.id),
    allocated: () =>
      active
        .filter(selectable)
        .map(named)
        .sort((a, b) => a.id - b.id),
    allocate: (id: number) => {
      const path = calculateAllocationPath(paths, id);
      if (!path) throw new Error(`node ${id} is not reachable from the allocated tree`);
      build.AllocateNodes(path);
    },
    depends: (id: number) => [id, ...depends(id)],
    deallocate: (id: number) => {
      build.DeallocateNodes([id]);
      const prunable = depends(id);
      if (prunable.length) build.DeallocateNodes(prunable);
    }
  };
};

export const resultName = (n: TreeNode) => `${n.name} #${n.id}`;
