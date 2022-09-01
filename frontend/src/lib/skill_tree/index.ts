import { writable } from 'svelte/store';
import { syncWrap } from '../go/worker';
import type { Group, Sprite, Tree, Node } from './types';

export const skillTree = writable<Tree | undefined>(undefined);
export const skillTreeVersion = writable<string | undefined>(undefined);

let loadedSkillTree: Tree;

export const drawnGroups: Record<number, Group> = {};
export const drawnNodes: Record<number, Node> = {};

export const ascendancyGroups: Record<number, string> = {};
export const ascendancyStartGroups = new Set<number>();
export const classStartGroups: Record<number, number> = {};

export const inverseSpritesInactive: Record<string, Sprite> = {};
export const inverseSpritesActive: Record<string, Sprite> = {};
export const inverseSpritesOther: Record<string, Sprite> = {};

let zoomLevel = 0.3835;

export const loadSkillTree = async (version: string) => {
  if (!syncWrap) {
    return;
  }

  const treeData = await syncWrap.GetTree(version);
  loadedSkillTree = JSON.parse(treeData);
  console.log('Loaded skill tree', loadedSkillTree);

  if (loadedSkillTree.imageZoomLevels) {
    zoomLevel = loadedSkillTree.imageZoomLevels[loadedSkillTree.imageZoomLevels.length - 1];
  }

  Object.keys(loadedSkillTree.groups).forEach((groupId) => {
    const group = loadedSkillTree.groups[groupId];
    const nGroupId = parseInt(groupId);
    drawnGroups[nGroupId] = group;
    group.nodes?.forEach((nodeId) => {
      const node = loadedSkillTree.nodes[nodeId];
      drawnNodes[parseInt(nodeId)] = node;

      if (node.classStartIndex !== undefined) {
        classStartGroups[nGroupId] = node.classStartIndex;
      }

      if (node.ascendancyName !== undefined) {
        ascendancyGroups[nGroupId] = node.ascendancyName;
      }

      if (node.isAscendancyStart) {
        ascendancyStartGroups.add(nGroupId);
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

export const calculateNodePos = (node: Node, offsetX: number, offsetY: number, scaling: number): Point => {
  if (
    node.group === undefined ||
    node.orbit === undefined ||
    node.orbitIndex === undefined ||
    !loadedSkillTree.groups ||
    !loadedSkillTree.constants.orbitRadii
  ) {
    return { x: 0, y: 0 };
  }

  const targetGroup = loadedSkillTree.groups[node.group];
  const targetAngle = orbitAngleAt(node.orbit, node.orbitIndex);

  const targetGroupPos = toCanvasCoords(targetGroup.x, targetGroup.y, offsetX, offsetY, scaling);
  const targetNodePos = toCanvasCoords(targetGroup.x, targetGroup.y - loadedSkillTree.constants.orbitRadii[node.orbit], offsetX, offsetY, scaling);
  return rotateAroundPoint(targetGroupPos, targetNodePos, targetAngle);
};

export const distance = (p1: Point, p2: Point): number => Math.sqrt(Math.pow(p1.x - p2.x, 2) + Math.pow(p1.y - p2.y, 2));
