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
// asked once a page: each probe makes a context and loses it, which the browser logs
export const hardwareWebGL2 = () => (hardware ??= probe());
