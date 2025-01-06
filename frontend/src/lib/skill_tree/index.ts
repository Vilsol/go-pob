import { writable } from 'svelte/store';
import { syncWrap } from '../go/worker';
import type { Group, Sprite, Tree, Node } from './types';
import { Assets, Spritesheet, type SpritesheetData, Texture } from 'pixi.js';

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

export enum SpritesheetType {
  ACTIVE,
  INACTIVE,
  OTHERS
}

export let allInverseSpritesheets: Record<SpritesheetType, Record<string, Spritesheet>>;
export let allExtraImages: Record<string, Texture>;

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

export const urlToCDN = (cdnBase: string, url: string) => {
  const urlPath = new URL(url).pathname;
  return cdnBase + `/tree/assets/` + urlPath.substring(urlPath.lastIndexOf('/') + 1);
};

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

  skillTree.set(loadedSkillTree);
  skillTreeVersion.set(version);
};

export const initializeSpritesheets = async (cdnBase: string) => {
  // Type -> URL -> SpritesheetData
  const allInverseSprites: Record<SpritesheetType, Record<string, SpritesheetData>> = {
    [SpritesheetType.ACTIVE]: {},
    [SpritesheetType.INACTIVE]: {},
    [SpritesheetType.OTHERS]: {}
  };

  for (const key of ['keystoneInactive', 'notableInactive', 'normalInactive', 'masteryInactive'] as const) {
    const sprites = loadedSkillTree.sprites[key]?.[zoomLevel];
    if (sprites) {
      Object.keys(sprites?.coords || {}).forEach((c) => {
        inverseSpritesInactive[c] = sprites;

        if (!(sprites.filename in allInverseSprites[SpritesheetType.INACTIVE])) {
          allInverseSprites[SpritesheetType.INACTIVE][sprites.filename] = {
            meta: {
              scale: 1,
              image: urlToCDN(cdnBase, sprites.filename),
              size: {
                w: sprites.w,
                h: sprites.h
              }
            },
            frames: {}
          };
        }

        allInverseSprites[SpritesheetType.INACTIVE][sprites.filename].frames[c] = {
          frame: {
            ...sprites.coords[c]
          }
        };
      });
    }
  }

  for (const key of ['keystoneActive', 'notableActive', 'normalActive', 'masteryActiveSelected'] as const) {
    const sprites = loadedSkillTree.sprites[key]?.[zoomLevel];
    if (sprites) {
      Object.keys(sprites.coords).forEach((c) => {
        inverseSpritesActive[c] = sprites;

        if (!(sprites.filename in allInverseSprites[SpritesheetType.ACTIVE])) {
          allInverseSprites[SpritesheetType.ACTIVE][sprites.filename] = {
            meta: {
              scale: 1,
              image: urlToCDN(cdnBase, sprites.filename),
              size: {
                w: sprites.w,
                h: sprites.h
              }
            },
            frames: {}
          };
        }

        allInverseSprites[SpritesheetType.ACTIVE][sprites.filename].frames[c] = {
          frame: {
            ...sprites.coords[c]
          }
        };
      });
    }
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
    let sprites = loadedSkillTree.sprites[key]?.[zoomLevel];
    if (!sprites) {
      sprites = loadedSkillTree.sprites[key]?.[Object.keys(loadedSkillTree.sprites[key])[0]];
    }

    if (sprites) {
      Object.keys(sprites?.coords || {}).forEach((c) => {
        inverseSpritesOther[c] = sprites;

        if (!(sprites.filename in allInverseSprites[SpritesheetType.OTHERS])) {
          allInverseSprites[SpritesheetType.OTHERS][sprites.filename] = {
            meta: {
              scale: 1,
              image: urlToCDN(cdnBase, sprites.filename),
              size: {
                w: sprites.w,
                h: sprites.h
              }
            },
            frames: {}
          };
        }

        allInverseSprites[SpritesheetType.OTHERS][sprites.filename].frames[c] = {
          frame: {
            ...sprites.coords[c]
          }
        };
      });
    }
  }

  const urlToTexture = await Assets.load(
    Object.values(allInverseSprites)
      .flatMap((x) => Object.keys(x))
      .map((url) => urlToCDN(cdnBase, url))
  );

  const parsers: Promise<unknown>[] = [];

  // Type -> Name -> Spritesheet
  allInverseSpritesheets = Object.entries(allInverseSprites).reduce(
    (types, [type, sheet]) => ({
      ...types,
      [type]: Object.entries(sheet).reduce(
        (names, [url, data]) => {
          const spritesheet = new Spritesheet(urlToTexture[urlToCDN(cdnBase, url)] as Texture, data);
          parsers.push(spritesheet.parse());
          return {
            ...names,
            ...Object.fromEntries(Object.keys(spritesheet.data.frames).map((name) => [name, spritesheet]))
          };
        },
        {} as Record<string, Spritesheet>
      )
    }),
    {} as Record<SpritesheetType, Record<string, Spritesheet>>
  );

  await Promise.all(parsers);

  allExtraImages = await Assets.load(
    Object.entries(loadedSkillTree.extraImages).map(([k, v]) => ({
      alias: k,
      src: cdnBase + '/raw/' + v.image
    }))
  );
};

export type Point = {
  x: number;
  y: number;
};

export const scaling = 2.6;
export const toCanvasCoords = (x: number, y: number): Point => ({
  x: x / scaling,
  y: y / scaling
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
export const calculateNodePos = (node: Node): Point => {
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

  return toCanvasCoords(nodePosCache[node.skill].x, nodePosCache[node.skill].y);
};
