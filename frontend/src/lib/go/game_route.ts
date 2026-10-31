import { get } from 'svelte/store';
import { poe2Build, poe2Enabled } from '$lib/global';

// a build the PoE1 worker declined to open because it belongs to another game
export type ForeignBuild = { game: 'poe2' | 'poe2-legacy'; xml: string };

export const POE2_DISABLED = 'This is a Path of Exile 2 build. Path of Exile 2 builds are not supported yet.';
// Build.lua:2700-2703 (PoB-PoE2)
export const POE2_LEGACY = 'This build was saved by an older version of Path of Building for Path of Exile 2 and cannot be opened.';

// true when the build was another game's and is now open; throws when it cannot be opened
export const openForeign = async (foreign: ForeignBuild | undefined, clearPoE1: () => unknown) => {
  if (!foreign) return false;
  if (foreign.game === 'poe2-legacy') throw new Error(POE2_LEGACY);
  if (!get(poe2Enabled)) throw new Error(POE2_DISABLED);
  const { poe2Worker } = await import('./worker2');
  const view = await (await poe2Worker()).Open(foreign.xml);
  await clearPoE1();
  poe2Build.set(view);
  return true;
};
