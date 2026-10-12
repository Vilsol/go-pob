import { syncWrap } from '$lib/go/worker';
import { currentBuildPath } from '$lib/global.js';
import { downloadUrl, importLink } from '$lib/design/share';
import { unsavedName } from './build_name.svelte';

// ImportTab.lua:684-699, BuildSiteTools.lua:88-113
export const downloadBuild = async (site: string, url: string) => {
  let res: Response;
  try {
    res = await fetch(url);
  } catch {
    throw new Error(`could not read the build from ${site}: the site is unreachable or does not allow pob.gg (CORS). Paste the code instead.`);
  }
  if (!res.ok) throw new Error(`${site} answered ${res.status}`);
  return (await res.text()).trim();
};

/* Main.lua:56-85 */
export const importParam = (url: URL) => url.searchParams.get('import') ?? undefined;

/* ImportTab.lua:590-606, 684-699 */
export const linkCode = async (text: string) => {
  const link = downloadUrl(text);
  return link ? downloadBuild(link.site, link.url) : importLink(text);
};

/* Main.lua:62-85, ImportTab.lua:663-666 */
export const openImportLink = async (text: string) => {
  const code = await linkCode(text);
  await syncWrap?.ImportCode(code);
  currentBuildPath.current = '';
  unsavedName.imported = true;
};
/* CompareTab.lua:1485, 1495, 1511 */
export const compareName = (text: string, name: string) => {
  if (name !== '') return name;
  const link = downloadUrl(text);
  return link ? `Imported from ${link.site}` : 'Imported build';
};
