import { SPRITE_FLOATS, STROKE_FLOATS, STYLE_COLOURS, STYLE_WIDTHS, classArt, fillScene, unpackTint, type Scene, type SceneState } from './scene';
import type { TreeRenderer, View } from './renderer';

type Sheet = { img: HTMLImageElement; ready: boolean; requested: boolean; bmp?: ImageBitmap };

export class CanvasTreeRenderer implements TreeRenderer {
  readonly kind = 'canvas';
  private ctx: CanvasRenderingContext2D;
  private sheets: Sheet[];
  private art = new Map<string, Sheet>();
  private crops = new Map<string, HTMLCanvasElement>();
  private view: View = { offsetX: 0, offsetY: 0, scaling: 10, width: 1, height: 1, dpr: 1 };
  private state: SceneState = { active: new Set(), hoverPath: new Set(), hovered: undefined, className: undefined, ascendancy: undefined };
  private stateDirty = true;
  private paths: { style: number; path: Path2D }[] = [];
  private pathsScaling = NaN;
  private frame = 0;
  private artPath: string | undefined;
  private artImg: HTMLImageElement | undefined;

  constructor(
    private canvas: HTMLCanvasElement,
    private scene: Scene,
    private cdnBase: string,
    private afterDraw: (ms: number) => void
  ) {
    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('Canvas 2D is not available');
    this.ctx = ctx;
    this.sheets = scene.sheets.map(() => ({ img: new Image(), ready: false, requested: false }));
  }

  setView(view: View) {
    this.view = view;
    this.request();
  }

  setState(state: SceneState) {
    this.state = state;
    this.stateDirty = true;
    this.request();
  }

  complete() {
    const art = classArt(this.scene.tree, this.state.className);
    return this.sheets.every((s) => !s.requested || s.ready) && (!art || !!this.art.get(art.image)?.ready);
  }

  drawListArt() {
    return this.artPath;
  }

  destroy() {
    cancelAnimationFrame(this.frame);
    this.frame = -1;
  }

  private request() {
    if (this.frame === 0) this.frame = requestAnimationFrame(() => this.draw());
  }

  private draw() {
    this.frame = 0;
    const t0 = performance.now();
    const { ctx, view, scene } = this;
    const w = Math.round(view.width * view.dpr);
    const h = Math.round(view.height * view.dpr);
    if (this.canvas.width !== w || this.canvas.height !== h) {
      this.canvas.width = w;
      this.canvas.height = h;
    }
    if (this.stateDirty) this.fill();
    if (this.pathsScaling !== view.scaling) this.buildPaths();

    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.fillStyle = '#080c11';
    ctx.fillRect(0, 0, w, h);

    const k = view.dpr / view.scaling;
    const bx = Math.abs(scene.tree.min_x) + view.offsetX;
    const by = Math.abs(scene.tree.min_y) + view.offsetY;
    const T: [number, number, number, number, number, number] = [k, 0, 0, k, k * bx, k * by];
    const vx0 = -bx;
    const vy0 = -by;
    const vx1 = vx0 + view.width * view.scaling;
    const vy1 = vy0 + view.height * view.scaling;

    const sprites = (arr: Float32Array, count: number) => {
      ctx.setTransform(...T);
      for (let i = 0; i < count; i++) {
        const o = i * SPRITE_FLOATS;
        const [ox, oy, ux, uy, qx, qy] = [arr[o], arr[o + 1], arr[o + 2], arr[o + 3], arr[o + 4], arr[o + 5]];
        const xs0 = Math.min(ox, ox + ux, ox + qx, ox + ux + qx);
        const xs1 = Math.max(ox, ox + ux, ox + qx, ox + ux + qx);
        const ys0 = Math.min(oy, oy + uy, oy + qy, oy + uy + qy);
        const ys1 = Math.max(oy, oy + uy, oy + qy, oy + uy + qy);
        if (xs1 < vx0 || xs0 > vx1 || ys1 < vy0 || ys0 > vy1) continue;
        const sx = arr[o + 6];
        const sy = arr[o + 7];
        const sw = arr[o + 8];
        const sh = arr[o + 9];
        const tint = arr[o + 12];
        const src = tint > 0 ? this.tinted(arr[o + 10], arr[o + 11], sx, sy, sw, sh, tint) : this.source(arr[o + 10], arr[o + 11], sx, sy, sw, sh);
        const cropped = arr[o + 11] > 0 || tint > 0;
        if (uy === 0 && qx === 0 && ux > 0 && qy > 0) {
          if (cropped) ctx.drawImage(src, 0, 0, sw, sh, ox, oy, ux, qy);
          else ctx.drawImage(src, sx, sy, sw, sh, ox, oy, ux, qy);
        } else {
          ctx.transform(ux / sw, uy / sw, qx / sh, qy / sh, ox, oy);
          if (cropped) ctx.drawImage(src, 0, 0, sw, sh, 0, 0, sw, sh);
          else ctx.drawImage(src, sx, sy, sw, sh, 0, 0, sw, sh);
          ctx.setTransform(...T);
        }
      }
    };

    sprites(scene.background, scene.counts.background);
    ctx.setTransform(...T);
    for (const { style, path } of this.paths) {
      ctx.strokeStyle = STYLE_COLOURS[style];
      ctx.lineWidth = STYLE_WIDTHS[style];
      ctx.stroke(path);
    }
    sprites(scene.sprites, scene.counts.sprites);
    this.afterDraw(performance.now() - t0);
  }

  private tinted(sheet: number, variant: number, sx: number, sy: number, sw: number, sh: number, tint: number): CanvasImageSource {
    const key = `${sheet}:${sx},${sy},${sw},${sh}:${variant}:${tint}`;
    let c = this.crops.get(key);
    if (!c) {
      const src = this.source(sheet, variant, sx, sy, sw, sh);
      const [x, y] = variant === 0 ? [sx, sy] : [0, 0];
      const rgb = unpackTint(tint) ?? [1, 1, 1];
      c = document.createElement('canvas');
      c.width = sw;
      c.height = sh;
      const t = c.getContext('2d')!;
      t.drawImage(src, x, y, sw, sh, 0, 0, sw, sh);
      t.globalCompositeOperation = 'multiply';
      t.fillStyle = `rgb(${rgb.map((v) => Math.round(v * 255)).join(',')})`;
      t.fillRect(0, 0, sw, sh);
      t.globalCompositeOperation = 'destination-in';
      t.drawImage(src, x, y, sw, sh, 0, 0, sw, sh);
      this.crops.set(key, c);
    }
    return c;
  }

  private source(sheet: number, variant: number, sx: number, sy: number, sw: number, sh: number): CanvasImageSource {
    const img = sheet >= this.sheets.length ? this.artImg! : this.sheets[sheet].img;
    if (variant === 0) return (sheet < this.sheets.length && this.sheets[sheet].bmp) || img;
    const key = `${sheet}:${sx},${sy},${sw},${sh}:${variant}`;
    let c = this.crops.get(key);
    if (!c) {
      c = document.createElement('canvas');
      c.width = sw;
      c.height = sh;
      const t = c.getContext('2d')!;
      t.beginPath();
      t.arc(sw / 2, sh / 2, sw / 2, 0, Math.PI * 2, true);
      t.closePath();
      t.clip();
      if (variant === 2) t.filter = 'brightness(50%) opacity(50%)';
      t.drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh);
      this.crops.set(key, c);
    }
    return c;
  }

  private fill() {
    const { scene } = this;
    this.stateDirty = false;
    const art = classArt(scene.tree, this.state.className);
    let artInfo: { sheet: number; w: number; h: number } | undefined;
    this.artPath = undefined;
    if (art) {
      let s = this.art.get(art.image);
      if (!s) {
        s = { img: new Image(), ready: false, requested: true };
        this.art.set(art.image, s);
        this.load(s, `${this.cdnBase}/raw/${art.image}`);
      }
      if (s.ready) {
        this.artImg = s.img;
        artInfo = { sheet: this.sheets.length, w: s.img.naturalWidth, h: s.img.naturalHeight };
        this.artPath = `raw/${art.image}`;
      }
    }
    fillScene(scene, this.state, (i) => this.sheetReady(i), artInfo);
    this.pathsScaling = NaN;
  }

  private buildPaths() {
    const { scene, view } = this;
    this.pathsScaling = view.scaling;
    this.paths = [];
    const st = scene.strokes;
    let cur: { style: number; path: Path2D } | undefined;
    for (let i = 0; i < scene.counts.strokes; i++) {
      const o = i * STROKE_FLOATS;
      const style = st[o + 5];
      if (!cur || cur.style !== style) {
        cur = { style, path: new Path2D() };
        this.paths.push(cur);
      }
      if (st[o + 6] === 0) {
        cur.path.moveTo(st[o], st[o + 1]);
        cur.path.lineTo(st[o + 2], st[o + 3]);
      } else {
        const r = st[o + 2] + view.scaling;
        const a0 = st[o + 3];
        cur.path.moveTo(st[o] + r * Math.cos(a0), st[o + 1] + r * Math.sin(a0));
        cur.path.arc(st[o], st[o + 1], r, a0, a0 + st[o + 4]);
      }
    }
  }

  private sheetReady(i: number) {
    const s = this.sheets[i];
    if (!s.requested) {
      s.requested = true;
      this.load(s, `${this.cdnBase}/${this.scene.sheets[i]}`);
    }
    return s.ready;
  }

  private load(s: Sheet, url: string) {
    s.img.onload = async () => {
      if (this.frame === -1) return;
      s.bmp = await createImageBitmap(s.img);
      s.ready = true;
      this.stateDirty = true;
      this.request();
    };
    s.img.src = url;
  }
}
