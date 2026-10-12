import type { Tree, Node } from './types';

export type Point = { x: number; y: number };

export const expectedAscendancyStartingPositions: Record<string, Point> = {
  Juggernaut: { x: -10400, y: 5200 },
  Berserker: { x: -10400, y: 3700 },
  Chieftain: { x: -10400, y: 2200 },
  Raider: { x: 10200, y: 5200 },
  Deadeye: { x: 10200, y: 2200 },
  Pathfinder: { x: 10200, y: 3700 },
  Occultist: { x: -1500, y: -9850 },
  Elementalist: { x: 0, y: -9850 },
  Necromancer: { x: 1500, y: -9850 },
  Slayer: { x: 1500, y: 9800 },
  Gladiator: { x: -1500, y: 9800 },
  Champion: { x: 0, y: 9800 },
  Inquisitor: { x: -10400, y: -2200 },
  Hierophant: { x: -10400, y: -3700 },
  Guardian: { x: -10400, y: -5200 },
  Assassin: { x: 10200, y: -5200 },
  Trickster: { x: 10200, y: -3700 },
  Saboteur: { x: 10200, y: -2200 },
  Ascendant: { x: -7800, y: 7200 },
  Reliquarian: { x: -7800, y: 8900 },
  Luminary: { x: -7800, y: 10600 },
  Aul: { x: -6750, y: 12000 },
  Breachlord: { x: -5250, y: 12000 },
  Catarina: { x: -3750, y: 12000 },
  Trialmaster: { x: -2250, y: 12000 },
  Delirious: { x: -750, y: 12000 },
  Farrul: { x: 750, y: 12000 },
  Lycia: { x: 2250, y: 12000 },
  KingInTheMists: { x: 3750, y: 12000 },
  Olroth: { x: 5250, y: 12000 },
  Oshabi: { x: 6750, y: 12000 },
  Necromantic: { x: 9750, y: 12000 },
  Abyssal: { x: -750, y: 13600 },
  Brinerot: { x: 750, y: 13600 }
};

// src/Classes/PassiveTree.lua:109-153
const legacyAlternateAscendancyIds = new Set(['Warden', 'Warlock', 'Primalist']);

export const removeLegacyAlternateAscendancies = (tree: Tree) => {
  if (!tree.alternate_ascendancies) return;
  const legacy = new Set(tree.alternate_ascendancies.filter((a) => legacyAlternateAscendancyIds.has(a.id)).map((a) => a.id));
  if (legacy.size === 0) return;
  const groups = new Set<number>();
  for (const [id, node] of Object.entries(tree.nodes)) {
    if (node.ascendancyName && legacy.has(node.ascendancyName)) {
      delete tree.nodes[id];
      if (node.group !== undefined) groups.add(node.group);
    }
  }
  for (const g of groups) delete tree.groups[g];
};

export const ascendancyOffsets = (tree: Tree): Record<string, Point> => {
  const out: Record<string, Point> = {};
  for (const group of Object.values(tree.groups)) {
    for (const id of group.nodes ?? []) {
      const node = tree.nodes[id];
      const e = node.ascendancyName && expectedAscendancyStartingPositions[node.ascendancyName];
      if (node.isAscendancyStart && e) {
        out[node.ascendancyName!] = { x: e.x - group.x, y: e.y - group.y };
      }
    }
  }
  return out;
};

export const rotateAroundPoint = (center: Point, target: Point, angle: number): Point => {
  const radians = (Math.PI / 180) * angle;
  const cos = Math.cos(radians);
  const sin = Math.sin(radians);
  const nx = cos * (target.x - center.x) + sin * (target.y - center.y) + center.x;
  const ny = cos * (target.y - center.y) - sin * (target.x - center.x) + center.y;
  return { x: nx, y: ny };
};

export const orbit16Angles = [0, 30, 45, 60, 90, 120, 135, 150, 180, 210, 225, 240, 270, 300, 315, 330];
export const orbit40Angles = [
  0, 10, 20, 30, 40, 45, 50, 60, 70, 80, 90, 100, 110, 120, 130, 135, 140, 150, 160, 170, 180, 190, 200, 210, 220, 225, 230, 240, 250, 260, 270, 280, 290, 300,
  310, 315, 320, 330, 340, 350
];

export const orbitAngleAt = (tree: Tree, orbit: number, index: number): number => {
  const nodesInOrbit = tree.constants.skillsPerOrbit?.[orbit];
  if (nodesInOrbit == 16) {
    return orbit16Angles[orbit16Angles.length - index] || 0;
  } else if (nodesInOrbit == 40) {
    return orbit40Angles[orbit40Angles.length - index] || 0;
  } else {
    return 360 - (360 / (nodesInOrbit || 1)) * index;
  }
};

export const groupCentre = (tree: Tree, offsets: Record<string, Point>, groupId: number, ascendancyName: string | undefined): Point => {
  const group = tree.groups[groupId];
  return {
    x: ((ascendancyName && offsets[ascendancyName]?.x) || 0) + group.x,
    y: ((ascendancyName && offsets[ascendancyName]?.y) || 0) + group.y
  };
};

export const nodeCentre = (tree: Tree, offsets: Record<string, Point>, node: Node): Point | undefined => {
  if (node.group === undefined || node.orbit === undefined || node.orbitIndex === undefined || node.skill === undefined) {
    return undefined;
  }
  const c = groupCentre(tree, offsets, node.group, node.ascendancyName);
  return rotateAroundPoint(c, { x: c.x, y: c.y - tree.constants.orbitRadii[node.orbit] }, orbitAngleAt(tree, node.orbit, node.orbitIndex));
};

export const toScreen = (tree: Tree, p: Point, offsetX: number, offsetY: number, scaling: number): Point => ({
  x: (Math.abs(tree.min_x) + p.x + offsetX) / scaling,
  y: (Math.abs(tree.min_y) + p.y + offsetY) / scaling
});

/* src/Classes/PassiveSpec.lua:636-643 */
export const secondaryAscendancy = (tree: Tree, id: number): string | undefined => {
  const a = id > 0 ? tree.alternate_ascendancies?.[id - 1] : undefined;
  return a && !legacyAlternateAscendancyIds.has(a.id) ? a.id : undefined;
};

/* PassiveSpec.lua:44 */
export const drawnByPoB = (tree: Tree, node: Node) =>
  !node.isProxy && !(node.group !== undefined && tree.groups[node.group]?.isProxy) && !node.expansionJewel?.parent;
