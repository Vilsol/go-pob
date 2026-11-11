import { get } from 'svelte/store';
import { currentBuild, poe2Build, poe2Enabled } from '$lib/global';
import { logError } from '$lib/utils';
import { addToast } from '$lib/design/toast';
import { poe2Xml } from './game_detect';

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
  // through the engine's boot, which runs as a job
  const { poe2Worker } = await import('./engine_poe2.svelte');
  const view = await (await poe2Worker()).Open(foreign.xml);
  // with no PoE1 build open the clear is not waited for: the PoE1 worker may still be loading its game data
  const cleared = Promise.resolve(clearPoE1());
  if (get(currentBuild)) await cleared;
  else cleared.catch(logError);
  poe2Build.set(view);
  const worker = await poe2Worker();
  void worker
    .TreeConversion()
    .then((c) => c && addToast(treeConversionNotice(c), { key: 'tree-conversion', role: 'status' }))
    .catch(logError);
  return true;
};

// TreeTab.lua:609 (PoB-PoE2): go-pob-data publishes one PoE2 tree, so an older spec opens on it as Convert would move it
export const treeConversionNotice = (c: { from: string; to: string; saved: number; lost: number }) =>
  `This tree was saved on ${c.from} and has been converted to ${c.to}, the only PoE2 tree go-pob has. ` +
  (c.lost > 0
    ? `${c.lost} of its ${c.saved} passives are no longer allocated due to changes in the tree.`
    : `All ${c.saved} of its passives are still allocated.`);

// a code that is a PoE2 build opens in the PoE2 worker without the PoE1 one; false for any other code, which goes the
// way it always has
export const openPoe2Code = async (code: string, clearPoE1: () => unknown) => {
  const xml = await poe2Xml(code);
  return xml !== undefined && openForeign({ game: 'poe2', xml }, clearPoE1);
};
