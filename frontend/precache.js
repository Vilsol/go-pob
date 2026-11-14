// Files under static/ the service worker must not precache: the wasm builds (cached on first use) and the host's
// own configuration (_headers, _routes.json, _redirects), which Cloudflare Pages never serves, so precaching it
// fails the worker's install and no update can ever be applied.
/** @param {string} file */
export const keepInPrecache = (file) => !/(^|\/)(go-pob2?(-tiny)?\.wasm.*|_headers|_routes\.json|_redirects|\.DS_Store)$/.test(file);
