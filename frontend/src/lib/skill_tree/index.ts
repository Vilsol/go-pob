import { writable } from 'svelte/store';
import { syncWrap } from '../go/worker';
import type { Group, Sprite, Tree, Node } from './types';

export const skillTree = writable<Tree | undefined>(undefined);
export const skillTreeVersion = writable<string | undefined>(undefined);

let loadedSkillTree: Tree;

export const drawnGroups: Map<number, Group> = new Map();
export const drawnNodes: Map<number, Node> = new Map();

export const ascendancyGroups: Record<number, string> = {};
export const ascendancyStartGroups = new Set<number>();
export const classStartGroups: Record<number, number> = {};
export const classStartNodes: Record<number, number[]> = {};

export const inverseSpritesInactive: Record<string, Sprite> = {};
export const inverseSpritesActive: Record<string, Sprite> = {};
export const inverseSpritesOther: Record<string, Sprite> = {};

let zoomLevel = 0.3835;

const expectedAscendancyStartingPositions: Record<string, { x: number; y: number }> = {
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
  Ascendant: { x: -7800, y: 7200 }
};

export const ascendancyGroupPositionOffsets: Record<string, { x: number; y: number }> = {};

export const loadSkillTree = async (version: string) => {
  if (!syncWrap) {
    return;
  }

  const treeData = await syncWrap.GetTree(version);
  loadedSkillTree = JSON.parse(treeData) as Tree;
  console.log('Loaded skill tree', loadedSkillTree);

  if (loadedSkillTree.imageZoomLevels) {
    zoomLevel = loadedSkillTree.imageZoomLevels[loadedSkillTree.imageZoomLevels.length - 1];
  }

  Object.keys(loadedSkillTree.groups).forEach((groupId) => {
    const group = loadedSkillTree.groups[groupId];
    const nGroupId = parseInt(groupId);
    drawnGroups.set(nGroupId, group);
    group.nodes?.forEach((nodeId) => {
      const node = loadedSkillTree.nodes[nodeId];
      drawnNodes.set(parseInt(nodeId), node);

      if (node.classStartIndex !== undefined) {
        classStartGroups[nGroupId] = node.classStartIndex;

        for (const n of [...(node.out || []), ...(node.in || [])]) {
          const target = loadedSkillTree.nodes[n];
          if (!target.skill) {
            continue;
          }

          if (target.ascendancyName === undefined) {
            classStartNodes[node.classStartIndex] = [...(classStartNodes[node.classStartIndex] || []), target.skill];
          }
        }
      }

      if (node.ascendancyName !== undefined) {
        ascendancyGroups[nGroupId] = node.ascendancyName;
      }

      if (node.isAscendancyStart) {
        ascendancyStartGroups.add(nGroupId);

        if (node.ascendancyName) {
          ascendancyGroupPositionOffsets[node.ascendancyName] = {
            x: expectedAscendancyStartingPositions[node.ascendancyName].x - group.x,
            y: expectedAscendancyStartingPositions[node.ascendancyName].y - group.y
          };
        }
      }
    });
  });

  for (const key of ['keystoneInactive', 'notableInactive', 'normalInactive', 'masteryInactive'] as const) {
    const sprite = loadedSkillTree.sprites[key]?.[zoomLevel];
    Object.keys(sprite?.coords || {}).forEach((c) => sprite && (inverseSpritesInactive[c] = sprite));
  }

  for (const key of ['keystoneActive', 'notableActive', 'normalActive', 'masteryActiveSelected'] as const) {
    const sprite = loadedSkillTree.sprites[key]?.[zoomLevel];
    Object.keys(sprite.coords).forEach((c) => sprite && (inverseSpritesActive[c] = sprite));
  }

  for (const key of [
    'background',
    'mastery',
    'masteryConnected',
    'ascendancyBackground',
    'ascendancy',
    'startNode',
    'groupBackground',
    'frame',
    'jewel',
    'line',
    'jewelRadius'
  ] as const) {
    let sprite = loadedSkillTree.sprites[key]?.[zoomLevel];
    if (!sprite) {
      sprite = loadedSkillTree.sprites[key]?.[Object.keys(loadedSkillTree.sprites[key])[0]];
    }
    Object.keys(sprite?.coords || {}).forEach((c) => sprite && (inverseSpritesOther[c] = sprite));
  }

  skillTree.set(loadedSkillTree);
  skillTreeVersion.set(version);
};

export type Point = {
  x: number;
  y: number;
};

export const toCanvasCoords = (x: number, y: number, offsetX: number, offsetY: number, scaling: number): Point => ({
  x: (Math.abs(loadedSkillTree.min_x) + x + offsetX) / scaling,
  y: (Math.abs(loadedSkillTree.min_y) + y + offsetY) / scaling
});

export const rotateAroundPoint = (center: Point, target: Point, angle: number): Point => {
  const radians = (Math.PI / 180) * angle;
  const cos = Math.cos(radians);
  const sin = Math.sin(radians);
  const nx = cos * (target.x - center.x) + sin * (target.y - center.y) + center.x;
  const ny = cos * (target.y - center.y) - sin * (target.x - center.x) + center.y;
  return {
    x: nx,
    y: ny
  };
};

export const orbit16Angles = [0, 30, 45, 60, 90, 120, 135, 150, 180, 210, 225, 240, 270, 300, 315, 330];
export const orbit40Angles = [
  0, 10, 20, 30, 40, 45, 50, 60, 70, 80, 90, 100, 110, 120, 130, 135, 140, 150, 160, 170, 180, 190, 200, 210, 220, 225, 230, 240, 250, 260, 270, 280, 290, 300,
  310, 315, 320, 330, 340, 350
];

export const orbitAngleAt = (orbit: number, index: number): number => {
  const nodesInOrbit = loadedSkillTree.constants.skillsPerOrbit?.[orbit];
  if (nodesInOrbit == 16) {
    return orbit16Angles[orbit16Angles.length - index] || 0;
  } else if (nodesInOrbit == 40) {
    return orbit40Angles[orbit40Angles.length - index] || 0;
  } else {
    return 360 - (360 / (nodesInOrbit || 1)) * index;
  }
};

const nodePosCache: Record<number, Point> = {};
export const calculateNodePos = (node: Node, offsetX: number, offsetY: number, scaling: number): Point => {
  if (
    node.group === undefined ||
    node.orbit === undefined ||
    node.orbitIndex === undefined ||
    node.skill === undefined ||
    !loadedSkillTree.groups ||
    !loadedSkillTree.constants.orbitRadii
  ) {
    return { x: 0, y: 0 };
  }

  if (!(node.skill in nodePosCache)) {
    const targetGroup = loadedSkillTree.groups[node.group];

    const posX = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.x) || 0) + targetGroup.x;
    const posY = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.y) || 0) + targetGroup.y;

    const targetAngle = orbitAngleAt(node.orbit, node.orbitIndex);

    nodePosCache[node.skill] = rotateAroundPoint({ x: posX, y: posY }, { x: posX, y: posY - loadedSkillTree.constants.orbitRadii[node.orbit] }, targetAngle);
  }

  return toCanvasCoords(nodePosCache[node.skill].x, nodePosCache[node.skill].y, offsetX, offsetY, scaling);
};

export const distance = (p1: Point, p2: Point): number => Math.sqrt(Math.pow(p1.x - p2.x, 2) + Math.pow(p1.y - p2.y, 2));
