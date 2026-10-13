import { syncWrap } from '$lib/go/worker';
import { currentBuildPath } from '$lib/global.js';
import { downloadUrl, importLink, SITES, type Site } from '$lib/design/share';
import { unsavedName } from './build_name.svelte';

// ImportTab.lua:684-699, BuildSiteTools.lua:88-113
const DIRECT = new Set(['PoBCodes']);

type Download = { site: string; url: string };

const viaProxy = (site: Site, id: string, fields: Record<string, string> = {}) => `/api/build?${new URLSearchParams({ site: site.id, id, ...fields })}`;

/* BuildSiteTools.lua:100 */
const linkDownload = (text: string): Download | undefined => {
  const link = downloadUrl(text);
  const site = SITES.find((s) => s.label === link?.site);
  if (!link || !site) return undefined;
  const id = DIRECT.has(site.id) ? undefined : site.regex.exec(importLink(text))?.[1];
  return { site: link.site, url: id === undefined ? link.url : viaProxy(site, id) };
};

/* BuildSiteTools.lua:91-98 */
export const protocolDownload = (raw: string): Download | undefined => {
  for (const site of SITES) {
    const rest = new RegExp(`^pob:[/\\\\]*${site.id.toLowerCase()}[/\\\\]+(.+)`).exec(raw)?.[1];
    if (rest === undefined) continue;
    if (DIRECT.has(site.id)) return { site: site.label, url: `https://${site.download.replace('$1', rest)}` };
    const [path, query] = rest.split(/\?(.*)/s);
    if (site.id === 'PoeNinja' && query !== undefined) return { site: site.label, url: viaProxy(site, path, Object.fromEntries(new URLSearchParams(query))) };
    return { site: site.label, url: viaProxy(site, rest) };
  }
  return undefined;
};

const unreadable = (site: string, why: string) => new Error(`could not read the build from ${site}: ${why}. Paste the code instead.`);

const download = async ({ site, url }: Download) => {
  let res: Response;
  try {
    res = await fetch(url);
  } catch {
    throw unreadable(site, 'the site is unreachable or does not allow pob.gg (CORS)');
  }
  const proxied = url.startsWith('/api/');
  if (proxied && !res.headers.get('Content-Type')?.startsWith('text/plain')) throw unreadable(site, "pob.gg's import proxy is not available here");
  if (!res.ok) throw unreadable(site, proxied ? await res.text() : `${site} answered ${res.status}`);
  return (await res.text()).trim();
};

export const downloadBuild = async (text: string) => {
  const link = linkDownload(text);
  if (!link) throw new Error('not a build site link');
  return download(link);
};

/* Main.lua:56-85 */
export const importParam = (url: URL) => url.searchParams.get('import') ?? undefined;

/* ImportTab.lua:590-606, 684-699 */
export const linkCode = async (text: string) => (downloadUrl(text) ? downloadBuild(text) : importLink(text));

/* Main.lua:62-85, ImportTab.lua:663-666 */
export const openImportLink = async (text: string) => {
  const protocol = protocolDownload(text);
  const code = protocol ? await download(protocol) : await linkCode(text);
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
