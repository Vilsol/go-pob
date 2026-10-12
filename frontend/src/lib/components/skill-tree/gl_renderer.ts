import { fillScene, SPRITE_FLOATS, STROKE_FLOATS, STYLE_COLOURS, STYLE_WIDTHS, classArt, type Scene, type SceneState } from './scene';
import type { TreeRenderer, View } from './renderer';

const MAX_SHEETS = 16;

const SPRITE_VS = `#version 300 es
layout(location=0) in vec2 a_corner;
layout(location=1) in vec2 a_o;
layout(location=2) in vec4 a_uv;
layout(location=3) in vec4 a_src;
layout(location=4) in vec2 a_meta;
layout(location=5) in float a_tint;
uniform vec2 u_base;
uniform float u_scaling;
uniform vec2 u_size;
uniform vec2 u_texSize[${MAX_SHEETS}];
out vec2 v_uv;
out vec2 v_local;
flat out int v_sheet;
flat out int v_variant;
flat out vec2 v_srcSize;
flat out vec3 v_tint;
void main() {
  vec2 p = a_o + a_corner.x * a_uv.xy + a_corner.y * a_uv.zw;
  int t = int(a_tint + 0.5) - 1;
  v_tint = t < 0 ? vec3(1.0) : vec3(float((t >> 16) & 255), float((t >> 8) & 255), float(t & 255)) / 255.0;
  vec2 clip = (p + u_base) / u_scaling / u_size * 2.0 - 1.0;
  gl_Position = vec4(clip.x, -clip.y, 0.0, 1.0);
  int s = int(a_meta.x + 0.5);
  v_uv = (a_src.xy + a_corner * a_src.zw) / u_texSize[s];
  v_local = a_corner;
  v_sheet = s;
  v_variant = int(a_meta.y + 0.5);
  v_srcSize = a_src.zw;
}`;

const sampleChain = Array.from({ length: MAX_SHEETS }, (_, i) => `if (s == ${i}) return textureGrad(u_tex[${i}], uv, dx, dy);`).join('\n  ');

const SPRITE_FS = `#version 300 es
precision highp float;
in vec2 v_uv;
in vec2 v_local;
flat in int v_sheet;
flat in int v_variant;
flat in vec2 v_srcSize;
flat in vec3 v_tint;
uniform sampler2D u_tex[${MAX_SHEETS}];
out vec4 o;
vec4 sheet(int s, vec2 uv, vec2 dx, vec2 dy) {
  ${sampleChain}
  return vec4(0.0);
}
void main() {
  vec2 dx = dFdx(v_uv);
  vec2 dy = dFdy(v_uv);
  vec4 c = sheet(v_sheet, v_uv, dx, dy);
  if (v_variant > 0) {
    // Clipped to the circle of radius w/2 about the sprite's centre, in source texels.
    float d = length((v_local - 0.5) * v_srcSize);
    float r = v_srcSize.x * 0.5;
    float aa = max(fwidth(d), 1e-4);
    c *= clamp((r - d) / aa + 0.5, 0.0, 1.0);
    // brightness(50%) opacity(50%) on premultiplied colour.
    if (v_variant == 2) c = vec4(c.rgb * 0.25, c.a * 0.5);
  }
  o = vec4(c.rgb * v_tint, c.a);
}`;

const STROKE_VS = `#version 300 es
layout(location=0) in vec2 a_corner;
layout(location=1) in vec4 a_g;
layout(location=2) in vec3 a_m;
layout(location=3) in vec4 a_b;
uniform vec2 u_base;
uniform float u_scaling;
uniform vec2 u_size;
uniform float u_width[8];
out vec2 v_p;
out float v_off;
flat out vec4 v_g;
flat out vec3 v_m;
void main() {
  int style = int(a_m.y + 0.5);
  float w = u_width[style];
  vec2 p;
  if (a_m.z < 0.5) {
    vec2 d = a_g.zw - a_g.xy;
    float len = length(d);
    vec2 n = len > 0.0 ? vec2(-d.y, d.x) / len : vec2(0.0, 1.0);
    float h = w * 0.5 + u_scaling;
    float side = a_corner.y * 2.0 - 1.0;
    p = a_g.xy + d * a_corner.x + n * side * h;
    v_off = side * h;
  } else {
    float pad = w * 0.5 + 3.0 * u_scaling;
    p = mix(a_b.xy - pad, a_b.zw + pad, a_corner);
    v_off = 0.0;
  }
  vec2 clip = (p + u_base) / u_scaling / u_size * 2.0 - 1.0;
  gl_Position = vec4(clip.x, -clip.y, 0.0, 1.0);
  v_p = p;
  v_g = a_g;
  v_m = a_m;
}`;

const STROKE_FS = `#version 300 es
precision highp float;
in vec2 v_p;
in float v_off;
flat in vec4 v_g;
flat in vec3 v_m;
uniform float u_scaling;
uniform vec4 u_colour[8];
uniform float u_width[8];
out vec4 o;
const float TAU = 6.28318530718;
void main() {
  int style = int(v_m.y + 0.5);
  float hw = u_width[style] * 0.5 / u_scaling;
  float d;
  if (v_m.z < 0.5) {
    d = abs(v_off) / u_scaling;
  } else {
    vec2 r = v_p - v_g.xy;
    // The Canvas 2D renderer stroked orbit arcs one CSS pixel outside the orbit.
    d = abs(length(r) - (v_g.z + u_scaling)) / u_scaling;
    if (mod(atan(r.y, r.x) - v_g.w, TAU) > v_m.x) discard;
  }
  float cov = clamp(max(hw, 0.5) + 0.5 - d, 0.0, 1.0) * min(1.0, 2.0 * hw);
  if (cov <= 0.0) discard;
  o = u_colour[style] * cov;
}`;

const hexColour = (h: string) => [parseInt(h.slice(1, 3), 16) / 255, parseInt(h.slice(3, 5), 16) / 255, parseInt(h.slice(5, 7), 16) / 255, 1];

type Tex = { tex: WebGLTexture; ready: boolean; w: number; h: number; requested: boolean };

export class GlTreeRenderer implements TreeRenderer {
  readonly kind = 'webgl';
  private canvas: HTMLCanvasElement;
  private spriteProg: WebGLProgram;
  private strokeProg: WebGLProgram;
  private sheets: Tex[];
  private art = new Map<string, Tex>();
  private bufs: { background: WebGLBuffer; strokes: WebGLBuffer; sprites: WebGLBuffer };
  private vaos: { background: WebGLVertexArrayObject; strokes: WebGLVertexArrayObject; sprites: WebGLVertexArrayObject };
  private view: View = { offsetX: 0, offsetY: 0, scaling: 10, width: 1, height: 1, dpr: 1 };
  private state: SceneState = { active: new Set(), hoverPath: new Set(), hovered: undefined, className: undefined, ascendancy: undefined };
  private stateDirty = true;
  private frame = 0;
  private lost = false;
  private artPath: string | undefined;

  constructor(
    private gl: WebGL2RenderingContext,
    private scene: Scene,
    private cdnBase: string,
    private afterDraw: (ms: number) => void
  ) {
    if (scene.sheets.length >= MAX_SHEETS) throw new Error(`the tree uses ${scene.sheets.length} sprite sheets; the renderer binds ${MAX_SHEETS - 1}`);
    this.canvas = gl.canvas as HTMLCanvasElement;
    this.spriteProg = this.program(SPRITE_VS, SPRITE_FS);
    this.strokeProg = this.program(STROKE_VS, STROKE_FS);

    const empty = this.texture(0);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, new Uint8Array(4));
    for (let i = 0; i < MAX_SHEETS; i++) {
      gl.activeTexture(gl.TEXTURE0 + i);
      gl.bindTexture(gl.TEXTURE_2D, empty);
    }
    this.sheets = scene.sheets.map(() => ({ tex: empty, ready: false, w: 1, h: 1, requested: false }));

    const corners = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, corners);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]), gl.STATIC_DRAW);
    this.bufs = { background: gl.createBuffer(), strokes: gl.createBuffer(), sprites: gl.createBuffer() };
    const spriteVao = (buf: WebGLBuffer) =>
      this.vao(corners, buf, SPRITE_FLOATS, [
        [1, 2, 0],
        [2, 4, 2],
        [3, 4, 6],
        [4, 2, 10],
        [5, 1, 12]
      ]);
    this.vaos = {
      background: spriteVao(this.bufs.background),
      sprites: spriteVao(this.bufs.sprites),
      strokes: this.vao(corners, this.bufs.strokes, STROKE_FLOATS, [
        [1, 4, 0],
        [2, 3, 4],
        [3, 4, 7]
      ])
    };

    gl.useProgram(this.spriteProg);
    gl.uniform1iv(
      gl.getUniformLocation(this.spriteProg, 'u_tex'),
      Array.from({ length: MAX_SHEETS }, (_, i) => i)
    );
    gl.useProgram(this.strokeProg);
    gl.uniform4fv(gl.getUniformLocation(this.strokeProg, 'u_colour'), STYLE_COLOURS.flatMap(hexColour));
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);

    this.canvas.addEventListener('webglcontextlost', this.onLost);
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
    this.canvas.removeEventListener('webglcontextlost', this.onLost);
    this.gl.getExtension('WEBGL_lose_context')?.loseContext();
  }

  private onLost = () => {
    this.lost = true;
  };

  private request() {
    if (this.frame === 0) this.frame = requestAnimationFrame(() => this.draw());
  }

  private draw() {
    this.frame = 0;
    if (this.lost) return;
    const t0 = performance.now();
    const { gl, view } = this;
    const w = Math.round(view.width * view.dpr);
    const h = Math.round(view.height * view.dpr);
    if (this.canvas.width !== w || this.canvas.height !== h) {
      this.canvas.width = w;
      this.canvas.height = h;
    }
    if (this.stateDirty) this.fill();

    gl.viewport(0, 0, w, h);
    gl.clearColor(0x08 / 255, 0x0c / 255, 0x11 / 255, 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    this.uniforms(this.spriteProg, false);
    this.uniforms(this.strokeProg, true);
    const { counts } = this.scene;
    gl.useProgram(this.spriteProg);
    gl.bindVertexArray(this.vaos.background);
    gl.drawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, counts.background);
    gl.useProgram(this.strokeProg);
    gl.bindVertexArray(this.vaos.strokes);
    gl.drawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, counts.strokes);
    gl.useProgram(this.spriteProg);
    gl.bindVertexArray(this.vaos.sprites);
    gl.drawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, counts.sprites);
    gl.bindVertexArray(null);
    this.afterDraw(performance.now() - t0);
  }

  private uniforms(prog: WebGLProgram, stroke: boolean) {
    const { gl, view } = this;
    gl.useProgram(prog);
    gl.uniform2f(gl.getUniformLocation(prog, 'u_base'), Math.abs(this.scene.tree.min_x) + view.offsetX, Math.abs(this.scene.tree.min_y) + view.offsetY);
    gl.uniform1f(gl.getUniformLocation(prog, 'u_scaling'), view.scaling);
    gl.uniform2f(gl.getUniformLocation(prog, 'u_size'), view.width, view.height);
    if (stroke) {
      gl.uniform1fv(gl.getUniformLocation(prog, 'u_width'), STYLE_WIDTHS);
      return;
    }
    gl.uniform2fv(
      gl.getUniformLocation(prog, 'u_texSize[0]'),
      Array.from({ length: MAX_SHEETS }, (_, i) => {
        const t = i < this.sheets.length ? this.sheets[i] : this.artTex();
        return t ? [t.w, t.h] : [1, 1];
      }).flat()
    );
  }

  private capture(vs: string, stroke: boolean, varyings: string[], vao: WebGLVertexArrayObject, count: number, floats: number) {
    const { gl } = this;
    const p = this.program(vs, '#version 300 es\nprecision highp float;\nout vec4 o;\nvoid main() { o = vec4(0.0); }', varyings);
    this.uniforms(p, stroke);
    const buf = gl.createBuffer();
    gl.bindBuffer(gl.TRANSFORM_FEEDBACK_BUFFER, buf);
    gl.bufferData(gl.TRANSFORM_FEEDBACK_BUFFER, Math.max(1, count * 4 * floats * 4), gl.STREAM_READ);
    gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, buf);
    gl.bindVertexArray(vao);
    gl.enable(gl.RASTERIZER_DISCARD);
    gl.beginTransformFeedback(gl.POINTS);
    gl.drawArraysInstanced(gl.POINTS, 0, 4, count);
    gl.endTransformFeedback();
    gl.disable(gl.RASTERIZER_DISCARD);
    gl.bindVertexArray(null);
    gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, null);
    const out = new Float32Array(count * 4 * floats);
    gl.bindBuffer(gl.COPY_READ_BUFFER, buf);
    gl.getBufferSubData(gl.COPY_READ_BUFFER, 0, out);
    gl.deleteBuffer(buf);
    gl.deleteProgram(p);
    return out;
  }

  selfCheck(): string[] {
    const { gl, scene, view } = this;
    const problems: string[] = [];
    this.sheets.forEach((s, i) => {
      if (!s.ready) return;
      gl.activeTexture(gl.TEXTURE0 + i);
      if (gl.getParameter(gl.TEXTURE_BINDING_2D) !== s.tex) problems.push(`sheet ${i} is not bound to unit ${i}`);
    });
    const units = Array.from(
      { length: MAX_SHEETS },
      (_, i) => gl.getUniform(this.spriteProg, gl.getUniformLocation(this.spriteProg, `u_tex[${i}]`)!) as number
    );
    if (units.some((u, i) => u !== i)) problems.push(`sampler units ${units.join(',')}`);

    const bx = Math.abs(scene.tree.min_x) + view.offsetX;
    const by = Math.abs(scene.tree.min_y) + view.offsetY;
    const clip = (x: number, y: number) => [((x + bx) / view.scaling / view.width) * 2 - 1, -(((y + by) / view.scaling / view.height) * 2 - 1)];
    const near = (a: number, b: number) => Math.abs(a - b) <= 1e-3 * Math.max(1, Math.abs(b));
    const corners = [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1]
    ];

    for (const [name, vao, arr, count] of [
      ['background', this.vaos.background, scene.background, scene.counts.background],
      ['sprites', this.vaos.sprites, scene.sprites, scene.counts.sprites]
    ] as const) {
      const out = this.capture(SPRITE_VS, false, ['gl_Position', 'v_uv'], vao, count, 6);
      let bad = 0;
      for (let i = 0; i < count; i++) {
        const o = i * SPRITE_FLOATS;
        const sheet = arr[o + 10];
        const t = sheet < this.sheets.length ? this.sheets[sheet] : this.artTex()!;
        corners.forEach(([cx, cy], c) => {
          const [ex, ey] = clip(arr[o] + cx * arr[o + 2] + cy * arr[o + 4], arr[o + 1] + cx * arr[o + 3] + cy * arr[o + 5]);
          const g = out.subarray((i * 4 + c) * 6, (i * 4 + c + 1) * 6);
          const ok =
            near(g[0] / g[3], ex) &&
            near(g[1] / g[3], ey) &&
            near(g[4], (arr[o + 6] + cx * arr[o + 8]) / t.w) &&
            near(g[5], (arr[o + 7] + cy * arr[o + 9]) / t.h);
          if (!ok) bad++;
        });
      }
      if (bad) problems.push(`${name}: ${bad} of ${count * 4} corners are not where the scene puts them`);
      if (count === 0) problems.push(`${name}: nothing to check`);
    }

    const out = this.capture(STROKE_VS, true, ['gl_Position'], this.vaos.strokes, scene.counts.strokes, 4);
    let bad = 0;
    let lines = 0;
    for (let i = 0; i < scene.counts.strokes; i++) {
      const o = i * STROKE_FLOATS;
      if (scene.strokes[o + 6] !== 0) continue;
      lines++;
      const [x0, y0, x1, y1] = scene.strokes.subarray(o, o + 4);
      const len = Math.hypot(x1 - x0, y1 - y0);
      const h = STYLE_WIDTHS[scene.strokes[o + 5]] / 2 + view.scaling;
      const [nx, ny] = len > 0 ? [-(y1 - y0) / len, (x1 - x0) / len] : [0, 1];
      corners.forEach(([cx, cy], c) => {
        const side = cy * 2 - 1;
        const [ex, ey] = clip(x0 + (x1 - x0) * cx + nx * side * h, y0 + (y1 - y0) * cx + ny * side * h);
        const g = out.subarray((i * 4 + c) * 4, (i * 4 + c + 1) * 4);
        if (!(near(g[0] / g[3], ex) && near(g[1] / g[3], ey))) bad++;
      });
    }
    if (bad) problems.push(`strokes: ${bad} of ${lines * 4} line corners are not where the scene puts them`);
    if (lines === 0) problems.push('strokes: no line to check');
    const err = gl.getError();
    if (err !== gl.NO_ERROR) problems.push(`GL error ${err}`);
    return problems;
  }

  private artTex() {
    const art = classArt(this.scene.tree, this.state.className);
    return art && this.art.get(art.image);
  }

  private fill() {
    const { gl, scene } = this;
    this.stateDirty = false;
    const art = classArt(scene.tree, this.state.className);
    let artInfo: { sheet: number; w: number; h: number } | undefined;
    this.artPath = undefined;
    if (art) {
      let t = this.art.get(art.image);
      if (!t) {
        t = { tex: this.texture(MAX_SHEETS - 1), ready: false, w: 1, h: 1, requested: true };
        this.art.set(art.image, t);
        this.load(t, `${this.cdnBase}/raw/${art.image}`, MAX_SHEETS - 1);
      }
      if (t.ready) {
        gl.activeTexture(gl.TEXTURE0 + MAX_SHEETS - 1);
        gl.bindTexture(gl.TEXTURE_2D, t.tex);
        artInfo = { sheet: MAX_SHEETS - 1, w: t.w, h: t.h };
        this.artPath = `raw/${art.image}`;
      }
    }
    fillScene(scene, this.state, (i) => this.sheetReady(i), artInfo);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.bufs.background);
    gl.bufferData(gl.ARRAY_BUFFER, scene.background.subarray(0, scene.counts.background * SPRITE_FLOATS), gl.DYNAMIC_DRAW);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.bufs.strokes);
    gl.bufferData(gl.ARRAY_BUFFER, scene.strokes.subarray(0, scene.counts.strokes * STROKE_FLOATS), gl.DYNAMIC_DRAW);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.bufs.sprites);
    gl.bufferData(gl.ARRAY_BUFFER, scene.sprites.subarray(0, scene.counts.sprites * SPRITE_FLOATS), gl.DYNAMIC_DRAW);
  }

  private sheetReady(i: number) {
    const s = this.sheets[i];
    if (!s.requested) {
      s.requested = true;
      s.tex = this.texture(i);
      this.load(s, `${this.cdnBase}/${this.scene.sheets[i]}`, i);
    }
    return s.ready;
  }

  private load(t: Tex, url: string, unit: number) {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = async () => {
      if (this.frame === -1 || this.lost) return;
      const { gl } = this;
      const max = gl.getParameter(gl.MAX_TEXTURE_SIZE) as number;
      const fit = Math.min(1, max / img.naturalWidth, max / img.naturalHeight);
      const src =
        fit < 1
          ? await createImageBitmap(img, {
              resizeWidth: Math.floor(img.naturalWidth * fit),
              resizeHeight: Math.floor(img.naturalHeight * fit),
              resizeQuality: 'high',
              premultiplyAlpha: 'premultiply'
            })
          : img;
      if (this.frame === -1 || this.lost) return;
      gl.activeTexture(gl.TEXTURE0 + unit);
      gl.bindTexture(gl.TEXTURE_2D, t.tex);
      gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, true);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, src);
      gl.generateMipmap(gl.TEXTURE_2D);
      t.w = img.naturalWidth;
      t.h = img.naturalHeight;
      t.ready = true;
      this.stateDirty = true;
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

  private program(vs: string, fs: string, capture?: string[]) {
    const { gl } = this;
    const p = gl.createProgram();
    if (capture) gl.transformFeedbackVaryings(p, capture, gl.INTERLEAVED_ATTRIBS);
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
