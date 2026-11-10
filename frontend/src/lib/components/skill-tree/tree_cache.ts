// The tree's renderer and its canvas, kept while the tree page is away: a canvas keeps its WebGL context, and with it
// every uploaded sprite sheet, while it is out of the document, so a return draws the finished art on its first frame.
// One tree is kept; another tree, game or renderer kind releases it, as does its context being lost.
export type Parked<R, S> = {
  key: string;
  canvas: HTMLCanvasElement;
  renderer: R & { destroy(): void };
  scene: S;
  // the mounted tree's frame callback, which the renderer reaches through the entry
  onDraw: (ms: number) => void;
};

let parked: Parked<unknown, unknown> | undefined;

const ids = new WeakMap<object, number>();
let nextId = 0;
// a number per object, for a key that must tell one loaded tree from another
export const objectId = (o: object) => {
  let id = ids.get(o);
  if (id === undefined) ids.set(o, (id = ++nextId));
  return id;
};

const release = () => {
  const p = parked;
  parked = undefined;
  if (!p) return;
  p.canvas.removeEventListener('webglcontextlost', release);
  p.renderer.destroy();
};

export const unpark = <R, S>(key: string): Parked<R, S> | undefined => {
  if (parked?.key !== key) {
    release();
    return undefined;
  }
  const p = parked as unknown as Parked<R, S>;
  parked = undefined;
  p.canvas.removeEventListener('webglcontextlost', release);
  return p;
};

export const park = <R, S>(p: Parked<R, S>) => {
  release();
  p.canvas.remove();
  p.onDraw = () => {};
  p.canvas.addEventListener('webglcontextlost', release);
  parked = p as unknown as Parked<unknown, unknown>;
};

if (typeof addEventListener === 'function') addEventListener('pagehide', release);
