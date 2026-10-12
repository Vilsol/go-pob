const tree = async (version: string) => {
  const r = await fetch(`https://data.pob.gg/data/${version.replace('_', '.')}/tree/data.json.gz`);
  return new Response(r.body!.pipeThrough(new DecompressionStream('gzip'))).text();
};

const syncWrap = new Proxy({} as Record<string, unknown>, {
  get: (_, name: string) => (name === 'GetTree' ? tree : () => Promise.resolve(undefined))
});

(globalThis as { cachedWorker?: unknown }).cachedWorker ??= { syncWorker: undefined, syncWrap };
