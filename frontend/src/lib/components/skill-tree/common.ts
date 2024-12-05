import type { Point } from '$lib/skill_tree';
import type { Coord, Sprite } from '$lib/skill_tree/types';

const spriteCache: Record<string, HTMLImageElement> = {};
const cropCache: Record<string, HTMLCanvasElement> = {};

const drawScaling = 2.6;

export const drawSprite = (
  context: CanvasRenderingContext2D,
  path: string,
  pos: Point,
  source: Record<string, Sprite>,
  scaling: number,
  cdnBase: string,
  mirror = false,
  cropCircle = false,
  active = false
) => {
  const sprite = source[path];
  if (!sprite) {
    return;
  }

  const spriteSheetUrl = sprite.filename;
  if (!(spriteSheetUrl in spriteCache)) {
    const urlPath = new URL(spriteSheetUrl).pathname;
    const base = urlPath.substring(urlPath.lastIndexOf('/') + 1);
    const cdnTreeBase = cdnBase + `/tree/assets/`;
    const finalUrl = cdnTreeBase + base;

    spriteCache[spriteSheetUrl] = new Image();
    spriteCache[spriteSheetUrl].src = finalUrl;
  }

  const self: Coord = sprite.coords[path];

  const newWidth = (self.w / scaling) * drawScaling;
  const newHeight = (self.h / scaling) * drawScaling;

  const topLeftX = pos.x - newWidth / 2;
  const topLeftY = pos.y - newHeight / 2;

  let finalY = topLeftY;
  if (mirror) {
    finalY = topLeftY - newHeight / 2;
  }

  if (cropCircle && spriteCache[spriteSheetUrl].complete) {
    const cacheKey = spriteSheetUrl + ':' + path + ';' + (active ? 'active' : '');
    if (!(cacheKey in cropCache)) {
      const tempCanvas = document.createElement('canvas');
      const tempCtx = tempCanvas.getContext('2d')!;
      tempCanvas.width = self.w;
      tempCanvas.height = self.h;

      tempCtx.save();

      tempCtx.beginPath();
      tempCtx.arc(self.w / 2, self.h / 2, self.w / 2, 0, Math.PI * 2, true);
      tempCtx.closePath();
      tempCtx.clip();

      if (!active) {
        tempCtx.filter = 'brightness(50%) opacity(50%)';
      }

      tempCtx.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, 0, 0, self.w, self.h);

      cropCache[cacheKey] = tempCanvas;
    }

    context.drawImage(cropCache[cacheKey], 0, 0, self.w, self.h, topLeftX, finalY, newWidth, newHeight);
  } else {
    context.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, topLeftX, finalY, newWidth, newHeight);
  }

  if (mirror) {
    context.save();

    context.translate(topLeftX, topLeftY);
    context.rotate(Math.PI);

    context.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, -newWidth, -(newHeight / 2), newWidth, -newHeight);

    context.restore();
  }
};

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
