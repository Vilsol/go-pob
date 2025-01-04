import type { Coord, Sprite } from '$lib/skill_tree/types';
import { Texture } from 'three';
import { useTexture } from '@threlte/extras';

export const wrapText = (text: string, context: CanvasRenderingContext2D, width: number): string[] => {
  const result = [];

  let currentWord = '';
  text.split(' ').forEach((word) => {
    if (context.measureText(currentWord + word).width < width) {
      currentWord += ' ' + word;
    } else {
      result.push(currentWord.trim());
      currentWord = word;
    }
  });

  if (currentWord.length > 0) {
    result.push(currentWord.trim());
  }

  return result;
};

const textureCache: Record<string, Record<string, { texture: Texture; sprite: Coord }>> = {};

export const loadSpriteTexture = async (spriteName: string, cdnBase: string, source: Record<string, Sprite>): Promise<{ texture: Texture; sprite: Coord }> => {
  const spriteSheet = source[spriteName];

  if (!(spriteSheet.filename in textureCache)) {
    textureCache[spriteSheet.filename] = {};
  }

  if (!(spriteName in textureCache[spriteSheet.filename])) {
    const urlPath = new URL(spriteSheet.filename).pathname;
    const base = urlPath.substring(urlPath.lastIndexOf('/') + 1);
    const cdnTreeBase = cdnBase + `/tree/assets/`;
    const finalUrl = cdnTreeBase + base;

    const sprite = spriteSheet.coords[spriteName];

    const uvX = sprite.x / spriteSheet.w;
    const uvY = 1 - sprite.y / spriteSheet.h - sprite.h / spriteSheet.h;
    const uvWidth = sprite.w / spriteSheet.w;
    const uvHeight = sprite.h / spriteSheet.h;

    const texture = await useTexture(finalUrl);

    const spriteTexture = texture.clone();

    spriteTexture.repeat.set(uvWidth, uvHeight);
    spriteTexture.offset.set(uvX, uvY);
    spriteTexture.needsUpdate = true;

    textureCache[spriteSheet.filename][spriteName] = {
      texture: spriteTexture,
      sprite
    };
  }

  return textureCache[spriteSheet.filename][spriteName];
};

const assetScale = 100;

export const relativeScale = (sprite: { w: number; h: number }, vec: [x: number, y: number, z: number]): [x: number, y: number, z: number] => [
  (vec[0] * sprite.w) / assetScale,
  (vec[1] * sprite.h) / assetScale,
  vec[2]
];
