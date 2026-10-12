const FONT_HOSTS: readonly string[] = ['fonts.googleapis.com', 'fonts.gstatic.com'];
const TREE_ASSETS = /^\/data\/[^/]+\/(tree\/assets|raw\/Art)\//;

export const isCdnAsset = (url: URL) => FONT_HOSTS.includes(url.hostname) || (url.hostname === 'data.pob.gg' && TREE_ASSETS.test(url.pathname));
