const TREE_ASSETS = /^\/data\/[^/]+\/(tree\/assets|raw\/Art)\//;

export const isCdnAsset = (url: URL) => url.hostname === 'data.pob.gg' && TREE_ASSETS.test(url.pathname);

export const ART_CACHE = 'pob-art-v1';
