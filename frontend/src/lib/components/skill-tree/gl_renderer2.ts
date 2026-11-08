import { hardwareWebGL2 } from './webgl';
import { DISC_FLOATS, STROKE2_FLOATS, fillScene2, scaleOf, type Scene2, type Scene2State, type View2 } from './scene2';

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

export class GlTree2Renderer implements Tree2Renderer {
  readonly kind = 'webgl';
  private discProg: WebGLProgram;
  private strokeProg: WebGLProgram;
  private bufs: { discs: WebGLBuffer; strokes: WebGLBuffer };
  private vaos: { discs: WebGLVertexArrayObject; strokes: WebGLVertexArrayObject };
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
    const corners = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, corners);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]), gl.STATIC_DRAW);
    this.bufs = { discs: gl.createBuffer(), strokes: gl.createBuffer() };
    this.vaos = {
      discs: this.vao(corners, this.bufs.discs, DISC_FLOATS, DISC_ATTRS),
      strokes: this.vao(corners, this.bufs.strokes, STROKE2_FLOATS, STROKE_ATTRS)
    };
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

  private draw() {
    this.frame = 0;
    if (!this.state) return;
    const t0 = performance.now();
    const { gl, view, scene } = this;
    const canvas = gl.canvas as HTMLCanvasElement;
    const w = Math.round(view.width * view.dpr);
    const h = Math.round(view.height * view.dpr);
    if (canvas.width !== w || canvas.height !== h) [canvas.width, canvas.height] = [w, h];
    if (this.dirty) {
      this.dirty = false;
      fillScene2(scene, this.state);
      gl.bindBuffer(gl.ARRAY_BUFFER, this.bufs.discs);
      gl.bufferData(gl.ARRAY_BUFFER, scene.discs.subarray(0, scene.counts.discs * DISC_FLOATS), gl.DYNAMIC_DRAW);
      gl.bindBuffer(gl.ARRAY_BUFFER, this.bufs.strokes);
      gl.bufferData(gl.ARRAY_BUFFER, scene.strokes.subarray(0, scene.counts.strokes * STROKE2_FLOATS), gl.DYNAMIC_DRAW);
    }
    gl.viewport(0, 0, w, h);
    gl.clearColor(0x08 / 255, 0x0c / 255, 0x11 / 255, 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    for (const [prog, vao, count] of [
      [this.strokeProg, this.vaos.strokes, scene.counts.strokes],
      [this.discProg, this.vaos.discs, scene.counts.discs]
    ] as const) {
      gl.useProgram(prog);
      gl.uniform1f(gl.getUniformLocation(prog, 'u_scale'), scaleOf(scene, view));
      gl.uniform2f(gl.getUniformLocation(prog, 'u_offset'), view.zoomX + view.width / 2, view.zoomY + view.height / 2);
      gl.uniform2f(gl.getUniformLocation(prog, 'u_size'), view.width, view.height);
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

// the same scene through Canvas 2D, where WebGL2 is missing
export class CanvasTree2Renderer implements Tree2Renderer {
  readonly kind = 'canvas';
  private ctx: CanvasRenderingContext2D;
  private view: View2 = { zoomLevel: 3, zoomX: 0, zoomY: 0, width: 1, height: 1, dpr: 1 };
  private state?: Scene2State;
  private dirty = true;
  private frame = 0;

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
      fillScene2(scene, this.state);
    }
    const s = scaleOf(scene, view);
    ctx.setTransform(view.dpr, 0, 0, view.dpr, 0, 0);
    ctx.fillStyle = '#080c11';
    ctx.fillRect(0, 0, view.width, view.height);
    ctx.setTransform(view.dpr * s, 0, 0, view.dpr * s, view.dpr * (view.zoomX + view.width / 2), view.dpr * (view.zoomY + view.height / 2));
    const st = scene.strokes;
    for (let i = 0; i < scene.counts.strokes; i++) {
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
