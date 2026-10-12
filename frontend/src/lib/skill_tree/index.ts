import { writable } from 'svelte/store';
import { syncWrap } from '../go/worker';
import type { Group, Sprite, Tree, Node } from './types';
import { ascendancyOffsets, removeLegacyAlternateAscendancies } from './geometry';

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
/* PassiveTreeView.lua:867-870 */
export const masterySprites: Record<string, Sprite> = {};
/* PassiveTreeView.lua:874-877 */
export const tattooSprites: Record<string, Sprite> = {};
export const bloodlineSprites: Record<string, Record<string, Sprite>> = {};

// src/Classes/PassiveTree.lua:308-347
const bloodlineSpriteTypes: Record<string, string> = {
  Trialmaster: 'trialmasterBloodline',
  Oshabi: 'oshabiBloodline',
  Olroth: 'olrothBloodline',
  Lycia: 'lyciaBloodline',
  KingInTheMists: 'kingInTheMistsBloodline',
  Farrul: 'farrulBloodline',
  Delirious: 'deliriousBloodline',
  Catarina: 'catarinaBloodline',
  Breachlord: 'breachlordBloodline',
  Aul: 'aulBloodline',
  Azmeri: 'azmeriBloodline'
};
const bloodlineAssetNames = [
  'AscendancyButton',
  'AscendancyButtonHighlight',
  'AscendancyButtonPressed',
  'AscendancyFrameLargeNormal',
  'AscendancyFrameLargeCanAllocate',
  'AscendancyFrameLargeAllocated',
  'AscendancyFrameSmallNormal',
  'AscendancyFrameSmallCanAllocate',
  'AscendancyFrameSmallAllocated',
  'AscendancyMiddle'
];
// src/Classes/PassiveTree.lua:349-364
const legacyClasses = new Set(['ClassesPrimalist', 'ClassesWarlock', 'ClassesWarden']);

const clear = (r: Record<string | number, unknown>) => Object.keys(r).forEach((k) => delete r[k]);

let zoomLevel = 0.3835;

export const ascendancyGroupPositionOffsets: Record<string, { x: number; y: number }> = {};

let loading: string | undefined;

export const loadSkillTree = async (version: string) => {
  if (!syncWrap || loading === version) {
    return;
  }
  loading = version;

  const treeData = await syncWrap.GetTree(version).catch((e: unknown) => {
    loading = undefined;
    throw e;
  });
  if (loading !== version) {
    return;
  }
  loadedSkillTree = JSON.parse(treeData) as Tree;
  console.log('Loaded skill tree', loadedSkillTree);
  removeLegacyAlternateAscendancies(loadedSkillTree);

  drawnGroups.clear();
  drawnNodes.clear();
  ascendancyStartGroups.clear();
  for (const r of [
    ascendancyGroups,
    classStartGroups,
    classStartNodes,
    inverseSpritesInactive,
    inverseSpritesActive,
    inverseSpritesOther,
    masterySprites,
    tattooSprites,
    bloodlineSprites
  ]) {
    clear(r);
  }
  clear(ascendancyGroupPositionOffsets);
  Object.assign(ascendancyGroupPositionOffsets, ascendancyOffsets(loadedSkillTree));

  zoomLevel = 0.3835;
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
          if (!target?.skill) {
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
    'ascendancyBackground',
    'ascendancy',
    'startNode',
    'groupBackground',
    'frame',
    'jewel',
    'line',
    'jewelRadius',
    // src/Classes/PassiveTreeView.lua:855-859
    'masteryActiveEffect'
  ] as const) {
    let sprite = loadedSkillTree.sprites[key]?.[zoomLevel];
    if (!sprite) {
      sprite = loadedSkillTree.sprites[key]?.[Object.keys(loadedSkillTree.sprites[key])[0]];
    }
    Object.keys(sprite?.coords || {}).forEach((c) => sprite && (inverseSpritesOther[c] = sprite));
  }

  for (const key of [
    'tattooActiveEffect',
    'normalActive',
    'notableActive',
    'keystoneActive',
    'normalInactive',
    'notableInactive',
    'keystoneInactive'
  ] as const) {
    const sprites = loadedSkillTree.sprites[key];
    const sprite = sprites?.[zoomLevel] ?? sprites?.[Object.keys(sprites ?? {})[0]];
    if (sprite) tattooSprites[key] = sprite;
  }

  {
    const sprites = loadedSkillTree.sprites.mastery;
    const sprite = sprites?.[zoomLevel] ?? sprites?.[Object.keys(sprites ?? {})[0]];
    Object.keys(sprite?.coords || {}).forEach((c) => sprite && (masterySprites[c] = sprite));
  }

  for (const [ascendancyName, spriteType] of Object.entries(bloodlineSpriteTypes)) {
    const sprite = loadedSkillTree.sprites[spriteType]?.[zoomLevel];
    for (const name of bloodlineAssetNames) {
      if (sprite?.coords[name]) (bloodlineSprites[ascendancyName] ??= {})[name] = sprite;
    }
  }

  for (const sprites of Object.values(loadedSkillTree.sprites)) {
    const sprite = sprites[zoomLevel];
    for (const name of Object.keys(sprite?.coords ?? {})) {
      if (name.startsWith('Classes') && !legacyClasses.has(name) && !inverseSpritesOther[name]) inverseSpritesOther[name] = sprite;
    }
  }

  skillTree.set(loadedSkillTree);
  skillTreeVersion.set(version);
};

export type { Point } from './geometry';
