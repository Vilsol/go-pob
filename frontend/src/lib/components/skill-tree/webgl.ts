const SOFTWARE_GL = /swiftshader|llvmpipe|softpipe|software|basic render driver/i;
const probe = () => {
  const gl = document.createElement('canvas').getContext('webgl2', { failIfMajorPerformanceCaveat: true });
  if (!gl) return false;
  const info = gl.getExtension('WEBGL_debug_renderer_info');
  const name = String(gl.getParameter(info ? info.UNMASKED_RENDERER_WEBGL : gl.RENDERER));
  gl.getExtension('WEBGL_lose_context')?.loseContext();
  return !SOFTWARE_GL.test(name);
};
let hardware: boolean | undefined;
// the attributes of the PoE2 tree's context
export const GL2_ATTRS: WebGLContextAttributes = { alpha: false, antialias: false, premultipliedAlpha: true };
// asked once a page: each probe makes a context and loses it, which the browser logs
export const hardwareWebGL2 = () => (hardware ??= probe());
// a hardware WebGL2 context on canvas, or null where WebGL2 is missing or software; it stands in for the probe's own
// context, a second context costing the page tens of milliseconds
export const hardwareContext = (canvas: HTMLCanvasElement, attrs: WebGLContextAttributes): WebGL2RenderingContext | null => {
  if (hardware === false || typeof WebGL2RenderingContext === 'undefined') return null;
  const gl = canvas.getContext('webgl2', { ...attrs, failIfMajorPerformanceCaveat: true });
  if (gl && hardware === undefined) {
    const info = gl.getExtension('WEBGL_debug_renderer_info');
    hardware = !SOFTWARE_GL.test(String(gl.getParameter(info ? info.UNMASKED_RENDERER_WEBGL : gl.RENDERER)));
    if (!hardware) gl.getExtension('WEBGL_lose_context')?.loseContext();
  }
  hardware = !!gl && hardware !== false;
  return hardware ? gl : null;
};
