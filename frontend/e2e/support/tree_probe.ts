import type { Page } from '@playwright/test';

import type { TreeFrame } from '../../src/lib/components/skill-tree/probe';

export type RecordedCall =
  | { op: 'img'; src: string; ok: boolean; s: number[]; d: number[]; t: number[]; crop?: { src: string; s: number[]; filter: string } }
  | { op: 'begin' }
  | { op: 'move' | 'line'; p: number[]; t: number[] }
  | { op: 'arc'; a: number[]; t: number[] }
  | { op: 'stroke'; style: string; width: number; t: number[] };

type ProbeState = {
  frames: TreeFrame[];
  pointerAt: number;
  record: boolean;
  calls: RecordedCall[];
  lastCalls: RecordedCall[];
};

const probeInit = (withRecorder: boolean) => {
  const P: ProbeState = { frames: [], pointerAt: 0, record: withRecorder, calls: [], lastCalls: [] };
  (globalThis as unknown as { __treeProbe: unknown }).__treeProbe = {
    state: P,
    frame(f: TreeFrame) {
      P.frames.push(f);
      if (P.record) {
        P.lastCalls = P.calls;
        P.calls = [];
      }
    }
  };
  addEventListener('pointermove', (e) => (P.pointerAt = e.timeStamp), { capture: true });
  if (!withRecorder) return;

  const images = new Set<HTMLImageElement>();
  (globalThis as unknown as { __treeImagesLoaded: () => boolean }).__treeImagesLoaded = () => [...images].every((i) => i.complete);
  const srcDesc = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, 'src')!;
  Object.defineProperty(HTMLImageElement.prototype, 'src', {
    ...srcDesc,
    set(this: HTMLImageElement, v: string) {
      images.add(this);
      srcDesc.set!.call(this, v);
    }
  });

  const proto = CanvasRenderingContext2D.prototype;
  const crops = new WeakMap<object, { src: string; s: number[]; filter: string }>();
  const isTree = (c: CanvasRenderingContext2D) => !!c.canvas.closest?.('[aria-label="Passive skill tree"]');
  const tf = (c: CanvasRenderingContext2D) => {
    const m = c.getTransform();
    return [m.a, m.b, m.c, m.d, m.e, m.f];
  };
  const bitmapSrc = new WeakMap<object, string>();
  const createBitmap = globalThis.createImageBitmap;
  globalThis.createImageBitmap = (async (source: ImageBitmapSource, ...rest: unknown[]) => {
    const bmp = await (createBitmap as (...a: unknown[]) => Promise<ImageBitmap>)(source, ...rest);
    if (source instanceof HTMLImageElement) bitmapSrc.set(bmp, source.src);
    return bmp;
  }) as typeof createImageBitmap;
  const srcOf = (img: unknown): string => (img instanceof HTMLImageElement ? img.src : img instanceof ImageBitmap ? (bitmapSrc.get(img) ?? '') : '');

  const drawImage = Object.getOwnPropertyDescriptor(proto, 'drawImage')!.value as (...a: unknown[]) => void;
  proto.drawImage = function (this: CanvasRenderingContext2D, ...a: unknown[]) {
    const n = a.slice(1) as number[];
    if (a.length === 9) {
      if (isTree(this)) {
        const ok = !(a[0] instanceof HTMLImageElement) || (a[0].complete && a[0].naturalWidth > 0);
        P.calls.push({ op: 'img', src: srcOf(a[0]), ok, s: n.slice(0, 4), d: n.slice(4), t: tf(this), crop: crops.get(a[0] as object) });
      } else if (a[0] instanceof HTMLImageElement) {
        crops.set(this.canvas, { src: srcOf(a[0]), s: n.slice(0, 4), filter: this.filter });
      }
    }
    return drawImage.apply(this, a);
  } as typeof proto.drawImage;

  const paths = new WeakMap<Path2D, ((t: number[]) => RecordedCall)[]>();
  const wrap = (target: object, name: 'beginPath' | 'moveTo' | 'lineTo' | 'arc', rec: (a: number[]) => (t: number[]) => RecordedCall) => {
    const orig = Object.getOwnPropertyDescriptor(target, name)!.value as (...a: unknown[]) => void;
    (target as Record<string, unknown>)[name] = function (this: CanvasRenderingContext2D | Path2D, ...a: number[]) {
      if (this instanceof Path2D) paths.set(this, [...(paths.get(this) ?? []), rec(a)]);
      else if (isTree(this)) P.calls.push(rec(a)(tf(this)));
      return orig.apply(this, a);
    };
  };
  for (const target of [proto, Path2D.prototype]) {
    wrap(target, 'moveTo', (a) => (t) => ({ op: 'move', p: a.slice(0, 2), t }));
    wrap(target, 'lineTo', (a) => (t) => ({ op: 'line', p: a.slice(0, 2), t }));
    wrap(target, 'arc', (a) => (t) => ({ op: 'arc', a: [a[0], a[1], a[2], a[3], a[4], a[5] ? 1 : 0], t }));
  }
  wrap(proto, 'beginPath', () => () => ({ op: 'begin' }));
  const stroke = Object.getOwnPropertyDescriptor(proto, 'stroke')!.value as (...a: unknown[]) => void;
  proto.stroke = function (this: CanvasRenderingContext2D, ...a: unknown[]) {
    if (isTree(this)) {
      const t = tf(this);
      if (a[0] instanceof Path2D) P.calls.push({ op: 'begin' }, ...(paths.get(a[0]) ?? []).map((r) => r(t)));
      P.calls.push({ op: 'stroke', style: typeof this.strokeStyle === 'string' ? this.strokeStyle : 'pattern', width: this.lineWidth, t });
    }
    return stroke.apply(this, a);
  } as typeof proto.stroke;
};

export const installTreeProbe = (page: Page) => page.addInitScript(probeInit, false);
export const installDrawRecorder = (page: Page) => page.addInitScript(probeInit, true);

const state = `(globalThis.__treeProbe && globalThis.__treeProbe.state)`;

export const frameCount = (page: Page) => page.evaluate<number>(`${state}.frames.length`);

export const framesSince = (page: Page, from: number) => page.evaluate<TreeFrame[]>(`${state}.frames.slice(${from})`);

export const lastFrame = (page: Page) => page.evaluate<TreeFrame | undefined>(`${state}.frames.at(-1)`);

export const pointerAt = (page: Page) => page.evaluate<number>(`${state}.pointerAt`);

export const lastCalls = (page: Page) => page.evaluate<RecordedCall[]>(`${state}.lastCalls`);

export const waitFrame = async (page: Page, from: number, pred: string, timeout = 30_000): Promise<TreeFrame> => {
  const h = await page.waitForFunction(`(() => { const p = ${pred}; return ${state}?.frames.slice(${from}).find((f) => p(f)) ?? false; })()`, undefined, {
    timeout
  });
  return (await h.jsonValue()) as TreeFrame;
};

export const waitLastFrame = async (page: Page, pred: string, timeout = 30_000): Promise<TreeFrame> => {
  const h = await page.waitForFunction(`(() => { const p = ${pred}; const f = ${state}?.frames.at(-1); return f && p(f) ? f : false; })()`, undefined, {
    timeout
  });
  return (await h.jsonValue()) as TreeFrame;
};
