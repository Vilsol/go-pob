import { syncWrap } from '$lib/go/worker';

// the PoE2 worker serializes; the PoE1 worker owns the build storage
export const savePoe2Build = async (path: string) => {
  const { poe2Worker } = await import('$lib/go/worker2');
  const xml = await (await poe2Worker()).Serialize();
  await syncWrap.SaveForeignBuild(path, xml);
};
