import type { Scene, SceneState } from './scene';
import { GlTreeRenderer } from './gl_renderer';
import { CanvasTreeRenderer } from './canvas_renderer';

export type View = { offsetX: number; offsetY: number; scaling: number; width: number; height: number; dpr: number };

export type RendererKind = 'webgl' | 'canvas';

export interface TreeRenderer {
  readonly kind: RendererKind;
  setView(view: View): void;
  setState(state: SceneState): void;
  complete(): boolean;
  drawListArt(): string | undefined;
  selfCheck?(): string[];
  destroy(): void;
}

const SOFTWARE_GL = /swiftshader|llvmpipe|softpipe|software|basic render driver/i;
const hardwareWebGL2 = () => {
  const gl = document.createElement('canvas').getContext('webgl2', { failIfMajorPerformanceCaveat: true });
  if (!gl) return false;
  const info = gl.getExtension('WEBGL_debug_renderer_info');
  const name = String(gl.getParameter(info ? info.UNMASKED_RENDERER_WEBGL : gl.RENDERER));
  gl.getExtension('WEBGL_lose_context')?.loseContext();
  return !SOFTWARE_GL.test(name);
};

export const createRenderer = (
  canvas: HTMLCanvasElement,
  scene: Scene,
  cdnBase: string,
  afterDraw: (ms: number) => void,
  force?: RendererKind
): TreeRenderer => {
  if (force === 'webgl' || (force === undefined && hardwareWebGL2())) {
    const gl = canvas.getContext('webgl2', { alpha: false, antialias: false, premultipliedAlpha: true });
    if (gl) return new GlTreeRenderer(gl, scene, cdnBase, afterDraw);
  }
  return new CanvasTreeRenderer(canvas, scene, cdnBase, afterDraw);
};
