import { hardwareWebGL2 } from './webgl';
import { DISC_FLOATS, SPRITE2_FLOATS, STROKE2_FLOATS, fillScene2, scaleOf, type Scene2, type Scene2State, type View2 } from './scene2';
import { sheetPath2 } from '$lib/skill_tree/tree2_art';

export interface Tree2Renderer {
  readonly kind: 'webgl' | 'canvas';
  setView(view: View2): void;
  setState(state: Scene2State): void;
  destroy(): void;
}

// tree units to clip space: screen = p * u_scale + u_offset, in CSS pixels
const HEAD = `#version 300 es
layout(location=0) in vec2 a_corner;
uniform float u_scale;
uniform vec2 u_offset;
uniform vec2 u_size;
vec4 clipOf(vec2 p) {
  vec2 s = (p * u_scale + u_offset) / u_size * 2.0 - 1.0;
  return vec4(s.x, -s.y, 0.0, 1.0);
}`;

const DISC_VS = `${HEAD}
layout(location=1) in vec4 a_disc;
layout(location=2) in vec3 a_fill;
layout(location=3) in vec4 a_ring;
out vec2 v_local;
flat out float v_r;
flat out float v_ringW;
flat out vec3 v_fill;
flat out vec4 v_ring;
void main() {
  // a disc never shrinks below 1.5 CSS pixels, so the zoomed-out tree still shows its nodes
  float r = max(a_disc.z, 1.5 / u_scale);
  float pad = r + 1.0 / u_scale;
  v_local = (a_corner * 2.0 - 1.0) * pad;
  v_r = r;
  v_ringW = a_disc.w * r / a_disc.z;
  v_fill = a_fill;
  v_ring = a_ring;
  gl_Position = clipOf(a_disc.xy + v_local);
}`;

const DISC_FS = `#version 300 es
precision highp float;
in vec2 v_local;
flat in float v_r;
flat in float v_ringW;
flat in vec3 v_fill;
flat in vec4 v_ring;
uniform float u_scale;
out vec4 o;
void main() {
  float d = length(v_local);
  float aa = 1.0 / u_scale;
  float outer = clamp((v_r - d) / aa + 0.5, 0.0, 1.0);
  if (outer <= 0.0) discard;
  float ring = clamp((d - (v_r - v_ringW)) / aa + 0.5, 0.0, 1.0);
  vec3 c = mix(v_fill, v_ring.rgb, ring);
  float a = outer * v_ring.a;
  o = vec4(c * a, a);
}`;

const STROKE_VS = `${HEAD}
layout(location=1) in vec4 a_g;
layout(location=2) in vec3 a_m;
layout(location=3) in vec4 a_colour;
out vec2 v_p;
out float v_off;
flat out vec4 v_g;
flat out vec3 v_m;
flat out vec4 v_colour;
void main() {
  float hw = max(a_m.z * 0.5, 0.75 / u_scale) + 1.0 / u_scale;
  vec2 p;
  if (a_m.y < 0.5) {
    vec2 d = a_g.zw - a_g.xy;
    float len = length(d);
    vec2 n = len > 0.0 ? vec2(-d.y, d.x) / len : vec2(0.0, 1.0);
    float side = a_corner.y * 2.0 - 1.0;
    p = a_g.xy + d * a_corner.x + n * side * hw;
    v_off = side * hw;
  } else {
    float r = a_g.z + hw;
    p = a_g.xy + mix(vec2(-r), vec2(r), a_corner);
    v_off = 0.0;
  }
  v_p = p;
  v_g = a_g;
  v_m = a_m;
  v_colour = a_colour;
  gl_Position = clipOf(p);
}`;

// a_g.w is the arc's start angle; v_m.x its sweep, v_m.y the kind, v_m.z the width in tree units
const STROKE_FS = `#version 300 es
precision highp float;
in vec2 v_p;
in float v_off;
flat in vec4 v_g;
flat in vec3 v_m;
flat in vec4 v_colour;
uniform float u_scale;
out vec4 o;
const float TAU = 6.28318530718;
void main() {
  float hw = max(v_m.z * 0.5, 0.75 / u_scale);
  float d;
  if (v_m.y < 0.5) {
    d = abs(v_off);
  } else {
    vec2 r = v_p - v_g.xy;
    d = abs(length(r) - v_g.z);
    if (mod(atan(r.y, r.x) - v_g.w, TAU) > v_m.x) discard;
  }
  float cov = clamp((hw - d) * u_scale + 0.5, 0.0, 1.0);
  if (cov <= 0.0) discard;
  float a = cov * v_colour.a;
  o = vec4(v_colour.rgb * a, a);
}`;

const MAX_SHEETS = 16;

const SPRITE_VS = `${HEAD}
layout(location=1) in vec2 a_o;
layout(location=2) in vec4 a_uv;
layout(location=3) in vec4 a_src;
layout(location=4) in float a_sheet;
layout(location=5) in vec4 a_tint;
layout(location=6) in vec3 a_shape;
uniform vec2 u_texSize[${MAX_SHEETS}];
out vec2 v_corner;
flat out vec4 v_src;
flat out int v_sheet;
flat out vec4 v_tint;
flat out vec3 v_shape;
void main() {
  int s = int(a_sheet + 0.5);
  // an arc's quad is its kite (0,0), (p,0), (1,1), (0,p), so the clipped corners cost no fragments
  float p = a_shape.x;
  vec2 corner = a_corner.x == a_corner.y ? a_corner : a_corner * p;
  v_corner = corner;
  v_src = vec4(a_src.xy, a_src.zw) / u_texSize[s].xyxy;
  v_sheet = s;
  v_tint = a_tint;
  v_shape = a_shape;
  gl_Position = clipOf(a_o + corner.x * a_uv.xy + corner.y * a_uv.zw);
}`;

// v_shape.x is an orbit arc's kite clip (PassiveTree.lua:746-786), 1 for none; v_shape.y the times the image repeats
// along u, as a line connector's does (PassiveTree.lua:733-741); v_shape.z the radius about the arc's centre corner
// inside which the sheet holds the smaller orbits' arcs
const SPRITE_FS = `#version 300 es
precision highp float;
in vec2 v_corner;
flat in vec4 v_src;
flat in int v_sheet;
flat in vec4 v_tint;
flat in vec3 v_shape;
uniform sampler2D u_tex[${MAX_SHEETS}];
out vec4 o;
vec4 sheet(int s, vec2 uv, vec2 dx, vec2 dy) {
  ${Array.from({ length: MAX_SHEETS }, (_, i) => `if (s == ${i}) return textureGrad(u_tex[${i}], uv, dx, dy);`).join('\n  ')}
  return vec4(0.0);
}
void main() {
  float p = v_shape.x;
  if (v_corner.y > p + (1.0 - p) * v_corner.x || v_corner.x > p + (1.0 - p) * v_corner.y) discard;
  if (length(vec2(1.0) - v_corner) < v_shape.z) discard;
  vec2 t = vec2(v_corner.x * v_shape.y, v_corner.y);
  vec2 local = vec2(v_shape.y == 1.0 ? t.x : fract(t.x), t.y);
  vec2 uv = v_src.xy + local * v_src.zw;
  vec2 grad = t * v_src.zw;
  // SetDrawColor multiplies the image, alpha included; the sheets are premultiplied
  o = sheet(v_sheet, uv, dFdx(grad), dFdy(grad)) * vec4(v_tint.rgb * v_tint.a, v_tint.a);
}`;

// the sprite instance: origin x, y; u edge; v edge; source x, y, w, h; sheet; tint r, g, b, a; clip; repeat
const SPRITE_ATTRS: [number, number, number][] = [
  [1, 2, 0],
  [2, 4, 2],
  [3, 4, 6],
  [4, 1, 10],
  [5, 4, 11],
  [6, 3, 15]
];

// the stroke instance: x0, y0, x1, y1 (or cx, cy, r, a0), sweep, kind, width, r, g, b, a
const STROKE_ATTRS: [number, number, number][] = [
  [1, 4, 0],
  [2, 3, 4],
  [3, 4, 7]
];
// the disc instance: x, y, r, ring width, fill rgb, ring rgb, alpha
const DISC_ATTRS: [number, number, number][] = [
  [1, 4, 0],
  [2, 3, 4],
  [3, 4, 7]
];

type Tex = { tex: WebGLTexture; ready: boolean; w: number; h: number };

export class GlTree2Renderer implements Tree2Renderer {
  readonly kind = 'webgl';
  private discProg: WebGLProgram;
  private strokeProg: WebGLProgram;
  private spriteProg: WebGLProgram;
  private bufs: { discs: WebGLBuffer; strokes: WebGLBuffer; rings: WebGLBuffer; under: WebGLBuffer; lines: WebGLBuffer; over: WebGLBuffer };
  private vaos: {
    discs: WebGLVertexArrayObject;
    strokes: WebGLVertexArrayObject;
    rings: WebGLVertexArrayObject;
    under: WebGLVertexArrayObject;
    lines: WebGLVertexArrayObject;
    over: WebGLVertexArrayObject;
  };
  private sheets: Tex[] = [];
  private view: View2 = { zoomLevel: 3, zoomX: 0, zoomY: 0, width: 1, height: 1, dpr: 1 };
  private state?: Scene2State;
  private dirty = true;
  private frame = 0;

  constructor(
    private gl: WebGL2RenderingContext,
    private scene: Scene2,
    private afterDraw: (ms: number) => void
  ) {
    this.discProg = this.program(DISC_VS, DISC_FS);
    this.strokeProg = this.program(STROKE_VS, STROKE_FS);
    this.spriteProg = this.program(SPRITE_VS, SPRITE_FS);
    const corners = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, corners);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]), gl.STATIC_DRAW);
    this.bufs = {
      discs: gl.createBuffer(),
      strokes: gl.createBuffer(),
      rings: gl.createBuffer(),
      under: gl.createBuffer(),
      lines: gl.createBuffer(),
      over: gl.createBuffer()
    };
    this.vaos = {
      discs: this.vao(corners, this.bufs.discs, DISC_FLOATS, DISC_ATTRS),
      strokes: this.vao(corners, this.bufs.strokes, STROKE2_FLOATS, STROKE_ATTRS),
      rings: this.vao(corners, this.bufs.rings, STROKE2_FLOATS, STROKE_ATTRS),
      under: this.vao(corners, this.bufs.under, SPRITE2_FLOATS, SPRITE_ATTRS),
      lines: this.vao(corners, this.bufs.lines, SPRITE2_FLOATS, SPRITE_ATTRS),
      over: this.vao(corners, this.bufs.over, SPRITE2_FLOATS, SPRITE_ATTRS)
    };
    const empty = this.texture(0);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, new Uint8Array(4));
    for (let i = 0; i < MAX_SHEETS; i++) {
      gl.activeTexture(gl.TEXTURE0 + i);
      gl.bindTexture(gl.TEXTURE_2D, empty);
    }
    gl.useProgram(this.spriteProg);
    gl.uniform1iv(
      gl.getUniformLocation(this.spriteProg, 'u_tex'),
      Array.from({ length: MAX_SHEETS }, (_, i) => i)
    );
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
  }

  setView(view: View2) {
    this.view = view;
    this.request();
  }

  setState(state: Scene2State) {
    this.state = state;
    this.dirty = true;
    this.request();
  }

  destroy() {
    cancelAnimationFrame(this.frame);
    this.frame = -1;
    this.gl.getExtension('WEBGL_lose_context')?.loseContext();
  }

  private request() {
    if (this.frame === 0) this.frame = requestAnimationFrame(() => this.draw());
  }

  // a sheet past the units the renderer binds is never ready, so what is on it draws as vectors
  private sheetReady = (i: number) => {
    const art = this.scene.art;
    if (!art || i >= MAX_SHEETS) return false;
    let t = this.sheets[i];
    if (!t) {
      t = this.sheets[i] = { tex: this.texture(i), ready: false, w: 1, h: 1 };
      this.load(t, `${art.base}/${sheetPath2(art.atlas.sheets[i])}`, i);
    }
    return t.ready;
  };

  private load(t: Tex, url: string, unit: number) {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      if (this.frame === -1) return;
      const { gl } = this;
      gl.activeTexture(gl.TEXTURE0 + unit);
      gl.bindTexture(gl.TEXTURE_2D, t.tex);
      gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, true);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
      gl.generateMipmap(gl.TEXTURE_2D);
      [t.w, t.h, t.ready] = [img.naturalWidth, img.naturalHeight, true];
      this.dirty = true;
      this.request();
    };
    img.src = url;
  }

  private texture(unit: number) {
    const { gl } = this;
    const t = gl.createTexture();
    gl.activeTexture(gl.TEXTURE0 + unit);
    gl.bindTexture(gl.TEXTURE_2D, t);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    return t;
  }

  private upload(buf: WebGLBuffer, data: Float32Array) {
    const { gl } = this;
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, data, gl.DYNAMIC_DRAW);
  }

  private draw() {
    this.frame = 0;
    if (!this.state) return;
    const t0 = performance.now();
    const { gl, view, scene } = this;
    const canvas = gl.canvas as HTMLCanvasElement;
    const w = Math.round(view.width * view.dpr);
    const h = Math.round(view.height * view.dpr);
    if (canvas.width !== w || canvas.height !== h) [canvas.width, canvas.height] = [w, h];
    const { counts } = scene;
    if (this.dirty) {
      this.dirty = false;
      fillScene2(scene, this.state, this.sheetReady);
      this.upload(this.bufs.discs, scene.discs.subarray(0, counts.discs * DISC_FLOATS));
      this.upload(this.bufs.strokes, scene.strokes.subarray(0, counts.connectorStrokes * STROKE2_FLOATS));
      this.upload(this.bufs.rings, scene.strokes.subarray(counts.connectorStrokes * STROKE2_FLOATS, counts.strokes * STROKE2_FLOATS));
      this.upload(this.bufs.under, scene.under.subarray(0, counts.under * SPRITE2_FLOATS));
      this.upload(this.bufs.lines, scene.lines.subarray(0, counts.lines * SPRITE2_FLOATS));
      this.upload(this.bufs.over, scene.over.subarray(0, counts.over * SPRITE2_FLOATS));
    }
    gl.viewport(0, 0, w, h);
    gl.clearColor(0x08 / 255, 0x0c / 255, 0x11 / 255, 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    const rings = counts.strokes - counts.connectorStrokes;
    const passes: [WebGLProgram, WebGLVertexArrayObject, number][] = scene.art
      ? [
          [this.spriteProg, this.vaos.under, counts.under],
          [this.spriteProg, this.vaos.lines, counts.lines],
          [this.strokeProg, this.vaos.strokes, counts.connectorStrokes],
          [this.discProg, this.vaos.discs, counts.discs],
          [this.spriteProg, this.vaos.over, counts.over],
          [this.strokeProg, this.vaos.rings, rings]
        ]
      : [
          [this.strokeProg, this.vaos.strokes, counts.connectorStrokes],
          [this.strokeProg, this.vaos.rings, rings],
          [this.discProg, this.vaos.discs, counts.discs]
        ];
    for (const [prog, vao, count] of passes) {
      if (!count) continue;
      gl.useProgram(prog);
      gl.uniform1f(gl.getUniformLocation(prog, 'u_scale'), scaleOf(scene, view));
      gl.uniform2f(gl.getUniformLocation(prog, 'u_offset'), view.zoomX + view.width / 2, view.zoomY + view.height / 2);
      gl.uniform2f(gl.getUniformLocation(prog, 'u_size'), view.width, view.height);
      if (prog === this.spriteProg) {
        gl.uniform2fv(
          gl.getUniformLocation(prog, 'u_texSize[0]'),
          Array.from({ length: MAX_SHEETS }, (_, i) => [this.sheets[i]?.w ?? 1, this.sheets[i]?.h ?? 1]).flat()
        );
      }
      gl.bindVertexArray(vao);
      gl.drawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, count);
    }
    gl.bindVertexArray(null);
    this.afterDraw(performance.now() - t0);
  }

  private vao(corners: WebGLBuffer, inst: WebGLBuffer, stride: number, attrs: [loc: number, size: number, offset: number][]) {
    const { gl } = this;
    const vao = gl.createVertexArray();
    gl.bindVertexArray(vao);
    gl.bindBuffer(gl.ARRAY_BUFFER, corners);
    gl.enableVertexAttribArray(0);
    gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);
    gl.bindBuffer(gl.ARRAY_BUFFER, inst);
    for (const [loc, size, offset] of attrs) {
      gl.enableVertexAttribArray(loc);
      gl.vertexAttribPointer(loc, size, gl.FLOAT, false, stride * 4, offset * 4);
      gl.vertexAttribDivisor(loc, 1);
    }
    gl.bindVertexArray(null);
    return vao;
  }

  private program(vs: string, fs: string) {
    const { gl } = this;
    const p = gl.createProgram();
    for (const [type, src] of [
      [gl.VERTEX_SHADER, vs],
      [gl.FRAGMENT_SHADER, fs]
    ] as const) {
      const s = gl.createShader(type)!;
      gl.shaderSource(s, src);
      gl.compileShader(s);
      if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) throw new Error(`tree shader: ${gl.getShaderInfoLog(s)}`);
      gl.attachShader(p, s);
    }
    gl.linkProgram(p);
    if (!gl.getProgramParameter(p, gl.LINK_STATUS)) throw new Error(`tree shader link: ${gl.getProgramInfoLog(p)}`);
    return p;
  }
}

const css = (r: number, g: number, b: number, a: number) => `rgba(${Math.round(r * 255)},${Math.round(g * 255)},${Math.round(b * 255)},${a})`;

// the largest drawn-tree layer the Canvas 2D renderer keeps, in device pixels (about 32 MB)
const LAYER_PIXELS = 8_000_000;
// the most tinted sprite copies it keeps
const TINTED_MAX = 8192;

// the same scene through Canvas 2D, where WebGL2 is missing
export class CanvasTree2Renderer implements Tree2Renderer {
  readonly kind = 'canvas';
  private ctx: CanvasRenderingContext2D;
  private view: View2 = { zoomLevel: 3, zoomX: 0, zoomY: 0, width: 1, height: 1, dpr: 1 };
  private state?: Scene2State;
  private dirty = true;
  private frame = 0;
  private sheets: (HTMLImageElement | undefined)[] = [];
  private ready: boolean[] = [];
  private tinted = new Map<string, HTMLCanvasElement>();
  // the drawn tree at one scale, which a pan only moves; dropped when the scene or the scale changes
  private layer?: { canvas: HTMLCanvasElement; k: number; x: number; y: number };
  private bounds?: [number, number, number, number];
  private spare?: HTMLCanvasElement;

  constructor(
    private canvas: HTMLCanvasElement,
    private scene: Scene2,
    private afterDraw: (ms: number) => void
  ) {
    this.ctx = canvas.getContext('2d')!;
  }

  setView(view: View2) {
    this.view = view;
    this.request();
  }

  setState(state: Scene2State) {
    this.state = state;
    this.dirty = true;
    this.request();
  }

  destroy() {
    cancelAnimationFrame(this.frame);
    this.frame = -1;
  }

  private request() {
    if (this.frame === 0) this.frame = requestAnimationFrame(() => this.draw());
  }

  private sheetReady = (i: number) => {
    const art = this.scene.art;
    if (!art) return false;
    if (!this.sheets[i]) {
      const img = (this.sheets[i] = new Image());
      img.crossOrigin = 'anonymous';
      img.onload = () => {
        if (this.frame === -1) return;
        this.ready[i] = true;
        this.dirty = true;
        this.request();
      };
      img.src = `${art.base}/${sheetPath2(art.atlas.sheets[i])}`;
    }
    return !!this.ready[i];
  };

  // the source rect multiplied by an opaque tint, as SetDrawColor does, with an arc's inner radius cut away
  private source(
    sheet: number,
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    r: number,
    g: number,
    b: number,
    inner = 0
  ): [CanvasImageSource, number, number] {
    const img = this.sheets[sheet]!;
    if (r === 1 && g === 1 && b === 1 && !inner) return [img, sx, sy];
    const key = `${sheet}:${sx},${sy},${sw},${sh}:${r},${g},${b}:${inner}`;
    let c = this.tinted.get(key);
    if (!c) {
      // a heat map run tints every node anew each time its powers arrive
      if (this.tinted.size >= TINTED_MAX) this.tinted.clear();
      c = document.createElement('canvas');
      [c.width, c.height] = [Math.max(1, sw), Math.max(1, sh)];
      const t = c.getContext('2d')!;
      t.drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh);
      t.globalCompositeOperation = 'multiply';
      t.fillStyle = css(r, g, b, 1);
      t.fillRect(0, 0, sw, sh);
      t.globalCompositeOperation = 'destination-in';
      t.drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh);
      if (inner) {
        t.globalCompositeOperation = 'destination-out';
        t.beginPath();
        t.arc(sw, sh, inner * sw, 0, Math.PI * 2);
        t.fill();
      }
      this.tinted.set(key, c);
    }
    return [c, 0, 0];
  }

  private strokes(ctx: CanvasRenderingContext2D, from: number, to: number, s: number) {
    const st = this.scene.strokes;
    for (let i = from; i < to; i++) {
      const o = i * STROKE2_FLOATS;
      ctx.beginPath();
      if (st[o + 5] < 0.5) {
        ctx.moveTo(st[o], st[o + 1]);
        ctx.lineTo(st[o + 2], st[o + 3]);
      } else {
        ctx.arc(st[o], st[o + 1], st[o + 2], st[o + 3], st[o + 3] + st[o + 4]);
      }
      ctx.lineWidth = Math.max(st[o + 6], 1.5 / s);
      ctx.strokeStyle = css(st[o + 7], st[o + 8], st[o + 9], st[o + 10]);
      ctx.stroke();
    }
  }

  private discs(ctx: CanvasRenderingContext2D, s: number) {
    const { scene } = this;
    const d = scene.discs;
    for (let i = 0; i < scene.counts.discs; i++) {
      const o = i * DISC_FLOATS;
      const r = Math.max(d[o + 2], 1.5 / s);
      const ring = (d[o + 3] * r) / d[o + 2];
      ctx.beginPath();
      ctx.arc(d[o], d[o + 1], r - ring / 2, 0, Math.PI * 2);
      ctx.fillStyle = css(d[o + 4], d[o + 5], d[o + 6], d[o + 10]);
      ctx.fill();
      ctx.lineWidth = ring;
      ctx.strokeStyle = css(d[o + 7], d[o + 8], d[o + 9], d[o + 10]);
      ctx.stroke();
    }
  }

  // T is the tree-to-device scale k and offset (tx, ty); an image wholly off the canvas is skipped
  private sprites(ctx: CanvasRenderingContext2D, buf: Float32Array, count: number, k: number, tx: number, ty: number) {
    const [W, H] = [ctx.canvas.width, ctx.canvas.height];
    for (let i = 0; i < count; i++) {
      const o = i * SPRITE2_FLOATS;
      const [a, b, c, d] = [k * buf[o + 2], k * buf[o + 3], k * buf[o + 4], k * buf[o + 5]];
      const [e, f] = [k * buf[o] + tx, k * buf[o + 1] + ty];
      if (
        Math.max(e, e + a, e + c, e + a + c) < 0 ||
        Math.min(e, e + a, e + c, e + a + c) > W ||
        Math.max(f, f + b, f + d, f + b + d) < 0 ||
        Math.min(f, f + b, f + d, f + b + d) > H
      )
        continue;
      const [sw, sh, clip, repeat, inner] = [buf[o + 8], buf[o + 9], buf[o + 15], buf[o + 16], buf[o + 17]];
      const [src, sx, sy] = this.source(buf[o + 10], buf[o + 6], buf[o + 7], sw, sh, buf[o + 11], buf[o + 12], buf[o + 13], inner);
      ctx.setTransform(a, b, c, d, e, f);
      ctx.globalAlpha = buf[o + 14];
      if (clip < 1) {
        /* PassiveTree.lua:782-785: the arc's kite */
        ctx.save();
        ctx.beginPath();
        ctx.moveTo(1, 1);
        ctx.lineTo(0, clip);
        ctx.lineTo(0, 0);
        ctx.lineTo(clip, 0);
        ctx.closePath();
        ctx.clip();
        ctx.drawImage(src, sx, sy, sw, sh, 0, 0, 1, 1);
        ctx.restore();
      } else if (repeat !== 1) {
        for (let j = 0; j < repeat; j++) {
          const part = Math.min(1, repeat - j);
          ctx.drawImage(src, sx, sy, sw * part, sh, j / repeat, 0, part / repeat, 1);
        }
      } else {
        ctx.drawImage(src, sx, sy, sw, sh, 0, 0, 1, 1);
      }
    }
    ctx.globalAlpha = 1;
    ctx.setTransform(k, 0, 0, k, tx, ty);
  }

  // the scene's extent in tree units: every quad's corners, every stroke and every disc
  private extent(): [number, number, number, number] {
    const { scene } = this;
    let [x0, y0, x1, y1] = [Infinity, Infinity, -Infinity, -Infinity];
    const add = (x: number, y: number, r = 0) => {
      x0 = Math.min(x0, x - r);
      y0 = Math.min(y0, y - r);
      x1 = Math.max(x1, x + r);
      y1 = Math.max(y1, y + r);
    };
    for (const layer of ['under', 'lines', 'over'] as const) {
      const buf = scene[layer];
      for (let i = 0; i < scene.counts[layer]; i++) {
        const o = i * SPRITE2_FLOATS;
        const [x, y, ux, uy, vx, vy] = [buf[o], buf[o + 1], buf[o + 2], buf[o + 3], buf[o + 4], buf[o + 5]];
        add(x, y);
        add(x + ux, y + uy);
        add(x + vx, y + vy);
        add(x + ux + vx, y + uy + vy);
      }
    }
    const st = scene.strokes;
    for (let i = 0; i < scene.counts.strokes; i++) {
      const o = i * STROKE2_FLOATS;
      const w = st[o + 6];
      if (st[o + 5] < 0.5) {
        add(st[o], st[o + 1], w);
        add(st[o + 2], st[o + 3], w);
      } else add(st[o], st[o + 1], st[o + 2] + w);
    }
    const d = scene.discs;
    for (let i = 0; i < scene.counts.discs; i++) add(d[i * DISC_FLOATS], d[i * DISC_FLOATS + 1], d[i * DISC_FLOATS + 2]);
    return [x0, y0, x1, y1];
  }

  // the scene's layers at scale k and offset (tx, ty), in device pixels
  private paint(ctx: CanvasRenderingContext2D, s: number, k: number, tx: number, ty: number) {
    const { scene } = this;
    const { counts } = scene;
    ctx.setTransform(k, 0, 0, k, tx, ty);
    if (scene.art) {
      this.sprites(ctx, scene.under, counts.under, k, tx, ty);
      this.sprites(ctx, scene.lines, counts.lines, k, tx, ty);
      this.strokes(ctx, 0, counts.connectorStrokes, s);
      this.discs(ctx, s);
      this.sprites(ctx, scene.over, counts.over, k, tx, ty);
      this.strokes(ctx, counts.connectorStrokes, counts.strokes, s);
    } else {
      this.strokes(ctx, 0, counts.strokes, s);
      this.discs(ctx, s);
    }
  }

  private draw() {
    this.frame = 0;
    if (!this.state) return;
    const t0 = performance.now();
    const { ctx, view, scene, canvas } = this;
    const w = Math.round(view.width * view.dpr);
    const h = Math.round(view.height * view.dpr);
    if (canvas.width !== w || canvas.height !== h) [canvas.width, canvas.height] = [w, h];
    if (this.dirty) {
      this.dirty = false;
      fillScene2(scene, this.state, this.sheetReady);
      this.spare = this.layer?.canvas ?? this.spare;
      this.layer = undefined;
      this.bounds = undefined;
    }
    const s = scaleOf(scene, view);
    ctx.setTransform(view.dpr, 0, 0, view.dpr, 0, 0);
    ctx.fillStyle = '#080c11';
    ctx.fillRect(0, 0, view.width, view.height);
    const [k, tx, ty] = [view.dpr * s, view.dpr * (view.zoomX + view.width / 2), view.dpr * (view.zoomY + view.height / 2)];
    this.bounds ??= this.extent();
    const [x0, y0, x1, y1] = this.bounds;
    const [lw, lh] = [Math.ceil((x1 - x0) * k) + 2, Math.ceil((y1 - y0) * k) + 2];
    // a tree that fits a layer draws once per scene and scale, and a pan moves the layer
    // a test turns the layer off to compare it with the scene drawn straight to the canvas
    const layered = (globalThis as { __tree2Probe?: { layer?: boolean } }).__tree2Probe?.layer !== false;
    if (layered && lw > 0 && lh > 0 && lw * lh <= LAYER_PIXELS) {
      if (this.layer?.k !== k) {
        const c = this.layer?.canvas ?? this.spare ?? document.createElement('canvas');
        this.spare = undefined;
        if (c.width !== lw || c.height !== lh) [c.width, c.height] = [lw, lh];
        const lc = c.getContext('2d')!;
        lc.setTransform(1, 0, 0, 1, 0, 0);
        lc.clearRect(0, 0, lw, lh);
        this.paint(lc, s, k, 1 - x0 * k, 1 - y0 * k);
        this.layer = { canvas: c, k, x: x0 * k - 1, y: y0 * k - 1 };
      }
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.drawImage(this.layer.canvas, Math.round(tx + this.layer.x), Math.round(ty + this.layer.y));
    } else {
      this.spare = this.layer?.canvas ?? this.spare;
      this.layer = undefined;
      this.paint(ctx, s, k, tx, ty);
    }
    this.afterDraw(performance.now() - t0);
  }
}

// a software WebGL (SwiftShader, llvmpipe) pans slower than Canvas 2D, so it takes the canvas, as PoE1's tree does
export const createTree2Renderer = (canvas: HTMLCanvasElement, scene: Scene2, afterDraw: (ms: number) => void, force?: 'webgl' | 'canvas'): Tree2Renderer => {
  if (force === 'webgl' || (force === undefined && hardwareWebGL2())) {
    const gl = canvas.getContext('webgl2', { alpha: false, antialias: false, premultipliedAlpha: true });
    if (gl) return new GlTree2Renderer(gl, scene, afterDraw);
  }
  return new CanvasTree2Renderer(canvas, scene, afterDraw);
};
