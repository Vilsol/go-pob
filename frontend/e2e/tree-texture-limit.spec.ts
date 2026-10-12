import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { installTreeProbe, waitLastFrame } from './support/tree_probe';
import * as tree from './support/tree_model';

const LIMIT = 2048;

test('the WebGL renderer fits every texture under a small MAX_TEXTURE_SIZE', async ({ page }) => {
  await installTreeProbe(page);
  await page.addInitScript((limit) => {
    const P = WebGL2RenderingContext.prototype;
    const uploads: { w: number; h: number; refused: boolean }[] = [];
    (globalThis as unknown as { __uploads: typeof uploads }).__uploads = uploads;
    const getParameter = Object.getOwnPropertyDescriptor(P, 'getParameter')!.value as (this: WebGL2RenderingContext, p: number) => unknown;
    P.getParameter = function (this: WebGL2RenderingContext, p: number): unknown {
      return p === this.MAX_TEXTURE_SIZE ? limit : getParameter.call(this, p);
    };
    const texImage2D = Object.getOwnPropertyDescriptor(P, 'texImage2D')!.value as (this: WebGL2RenderingContext, ...a: unknown[]) => void;
    P.texImage2D = function (this: WebGL2RenderingContext, ...a: unknown[]) {
      const src = a[a.length - 1] as { naturalWidth?: number; naturalHeight?: number; width?: number; height?: number } | null;
      const sized = a.length > 6 && typeof a[3] === 'number' && typeof a[4] === 'number';
      const w = sized ? (a[3] as number) : (src?.naturalWidth ?? src?.width ?? 0);
      const h = sized ? (a[4] as number) : (src?.naturalHeight ?? src?.height ?? 0);
      const refused = w > limit || h > limit;
      uploads.push({ w, h, refused });
      if (!refused) texImage2D.apply(this, a);
    } as typeof P.texImage2D;
    (globalThis as unknown as { __treeProbe: { renderer: string } }).__treeProbe.renderer = 'webgl';
  }, LIMIT);
  const app = await App.open(page, 'new-build');
  await tree.prepareTree();
  await app.openTree(tree.treeVersion());
  await waitLastFrame(page, '(f) => f.renderer === "webgl" && f.complete', 60_000);
  const uploads = await page.evaluate(() => (globalThis as unknown as { __uploads: { w: number; h: number; refused: boolean }[] }).__uploads);
  expect(
    uploads.filter((u) => u.refused),
    'a texture larger than MAX_TEXTURE_SIZE was uploaded'
  ).toEqual([]);
  expect(uploads.some((u) => u.h === LIMIT || u.w === LIMIT)).toBe(true);
  app.assertNoErrors();
});
