import { syncWrap } from '$lib/go/worker';
import { backendLoaded, currentBuildPath } from '$lib/global.js';
import { downloadUrl, importLink, SITES, type Site } from '$lib/design/share';
import { opening, unsavedName } from './build_name.svelte';
import { resetAccountSession } from './account_session.svelte';
import { DESKTOP } from '$lib/pwa/desktop';
import { addToast } from '$lib/design/toast';

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

const OPENED = 'gopob-file:';

const openedDownload = (raw: string): Download | undefined =>
  raw.startsWith(OPENED) ? { site: 'the build file', url: `/desktop/opened/${encodeURIComponent(raw.slice(OPENED.length))}` } : undefined;

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

/* ImportTab.lua:539-554, BuildSiteTools.lua:47-82 */
export const shareBuild = async (code: string, label: string) => {
  const site = SITES.find((s) => s.label === label);
  if (!site?.post) throw new Error(`${label} does not take uploads`);
  let res: Response;
  try {
    res = await fetch(`/api/share?site=${site.id}`, { method: 'POST', headers: { 'Content-Type': 'text/plain' }, body: code });
  } catch {
    throw new Error("pob.gg's share proxy is unreachable");
  }
  if (!res.headers.get('Content-Type')?.startsWith('text/plain')) throw new Error("pob.gg's share proxy is not available here");
  const text = (await res.text()).trim();
  if (!res.ok) throw new Error(text);
  return text;
};

const poeUrl = async (query: string) => {
  let res: Response;
  try {
    res = await fetch(`/api/poeurl?${query}`);
  } catch {
    throw new Error('PoEURL is unreachable');
  }
  if (!res.headers.get('Content-Type')?.startsWith('text/plain')) throw new Error("pob.gg's PoEURL proxy is not available here");
  const text = (await res.text()).trim();
  if (!res.ok) throw new Error(text);
  return text;
};

/* TreeTab.lua:759-789 */
export const resolvePoeUrl = async (link: string) => {
  const id = /poeurl\.com\/(\w+)/.exec(link)?.[1];
  if (!id) throw new Error('Failed to resolve PoEURL link');
  return poeUrl(new URLSearchParams({ id }).toString());
};

/* TreeTab.lua:818-830 */
export const shrinkTreeLink = (link: string) => poeUrl(new URLSearchParams({ shrink: link }).toString());

export const downloadBuild = async (text: string) => {
  const link = linkDownload(text);
  if (!link) throw new Error('not a build site link');
  return download(link);
};

/* Main.lua:56-85 */
export const importParam = (url: URL) => url.searchParams.get('import') ?? undefined;

export const importLanding = (url: URL, base: string) => {
  const path = url.pathname.slice(base.length).replace(/\/+$/, '');
  if (!path) return `${base}/tree`;
  const params = new URLSearchParams(url.search);
  params.delete('import');
  const query = params.toString();
  return `${url.pathname}${query ? `?${query}` : ''}`;
};

/* ImportTab.lua:590-606, 684-699 */
export const linkCode = async (text: string) => (downloadUrl(text) ? downloadBuild(text) : importLink(text));

/* Main.lua:62-85, ImportTab.lua:663-666 */
export const openImportLink = async (text: string) => {
  const protocol = protocolDownload(text) ?? openedDownload(text);
  const code = protocol ? await download(protocol) : await linkCode(text);
  opening.modFlag = true;
  await syncWrap?.ImportCode(code);
  currentBuildPath.current = '';
  unsavedName.imported = true;
  resetAccountSession();
};

/* Build.lua:1421-1448 */
export const beforeReplace: { ask?: (open: () => void) => void; reloading?: boolean } = {};

if (DESKTOP && typeof window !== 'undefined') {
  let handedOver: Promise<unknown> = backendLoaded;
  const handOver = (text: string) => {
    const open = () =>
      openImportLink(text)
        .then(async () => (await import('$app/navigation')).goto('/tree'))
        .catch((err: Error) => addToast(err.message));
    handedOver = handedOver.then(() => (beforeReplace.ask ? beforeReplace.ask(() => void open()) : open()));
  };
  window.addEventListener('pob:desktop-import', (e) => handOver((e as CustomEvent<string>).detail));
  window.addEventListener('pob:desktop-toast', (e) => addToast((e as CustomEvent<string>).detail));
  void fetch('/desktop/hint', { method: 'POST' })
    .then(async (res) => res.status === 200 && addToast(await res.text()))
    .catch(() => {});
  window.addEventListener('pob:desktop-builds-folder', (e) => {
    const pick = () => void fetch(`/desktop/builds-folder${(e as CustomEvent<string>).detail === 'default' ? '?default' : ''}`, { method: 'POST' });
    if (beforeReplace.ask) beforeReplace.ask(pick);
    else pick();
  });
  window.addEventListener('pob:desktop-reload', () => {
    beforeReplace.reloading = true;
    location.reload();
  });
  const host = window as Window & { pobDesktopListening?: boolean; pobDesktopPending?: string[] };
  host.pobDesktopListening = true;
  host.pobDesktopPending?.splice(0).forEach(handOver);
}

/* CompareTab.lua:1485, 1495, 1511 */
export const compareName = (text: string, name: string) => {
  if (name !== '') return name;
  const link = downloadUrl(text);
  return link ? `Imported from ${link.site}` : 'Imported build';
};
