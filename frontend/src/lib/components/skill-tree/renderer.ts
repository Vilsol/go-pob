import type { Scene, SceneState } from './scene';
import { GlTreeRenderer } from './gl_renderer';
import { CanvasTreeRenderer } from './canvas_renderer';
import { hardwareWebGL2 } from './webgl';

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
