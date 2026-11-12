import { readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { missingFile, onRequest } from '../functions/[[path]]';

const frontend = resolve(import.meta.dirname, '..');
const read = (path: string) => readFileSync(resolve(frontend, path), 'utf8');
const html = read('src/app.html');
const head = html.slice(0, html.indexOf('</head>'));

const meta = (attr: 'name' | 'property', key: string) => new RegExp(`<meta\\s+${attr}="${key}"\\s+content="([^"]*)"\\s*/>`).exec(head)?.[1];

describe('app.html head', () => {
  it('names the site and describes it for bots and chats', () => {
    const title = /<title>([^<]+)<\/title>/.exec(head)?.[1];
    const description = meta('name', 'description');
    expect(title).toBe('go-pob: Path of Building in your browser');
    expect(description).toMatch(/Path of Exile 2/);
    expect(description!.length).toBeLessThanOrEqual(160);
    expect(head).toContain('<link rel="canonical" href="https://pob.gg/" />');
    expect(meta('property', 'og:type')).toBe('website');
    expect(meta('property', 'og:site_name')).toBe('go-pob');
    expect(meta('property', 'og:url')).toBe('https://pob.gg/');
    expect(meta('property', 'og:title')).toBe(title);
    expect(meta('property', 'og:description')).toBe(description);
    expect(meta('name', 'twitter:card')).toBe('summary');
    expect(meta('name', 'twitter:title')).toBe(title);
    expect(meta('name', 'twitter:description')).toBe(description);
  });

  it('carries no image tags', () => {
    expect(head).not.toMatch(/og:image|twitter:image/);
  });

  it('describes the app as a free WebApplication in JSON-LD', () => {
    const blocks = [...html.matchAll(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/g)];
    expect(blocks).toHaveLength(1);
    const ld = JSON.parse(blocks[0][1]) as Record<string, unknown>;
    expect(ld).toMatchObject({
      '@context': 'https://schema.org',
      '@type': 'WebApplication',
      name: 'go-pob',
      description: meta('name', 'description'),
      url: 'https://pob.gg/',
      applicationCategory: 'GameApplication',
      operatingSystem: 'Any',
      isAccessibleForFree: true,
      offers: { '@type': 'Offer', price: '0' },
      sameAs: ['https://github.com/Vilsol/go-pob']
    });
  });

  it('tells a visitor without JavaScript what the site is', () => {
    const noscript = /<body>\s*<noscript>([\s\S]*?)<\/noscript>/.exec(html)?.[1];
    expect(noscript).toMatch(/Path of Exile 2/);
    expect(noscript).toMatch(/needs JavaScript/);
  });
});

describe('static site files', () => {
  it('robots.txt points at the sitemap and keeps build links out', () => {
    const robots = read('static/robots.txt');
    expect(robots).toMatch(/^Sitemap: https:\/\/pob\.gg\/sitemap\.xml$/m);
    expect(robots).toMatch(/^Disallow: \/\*\?import=$/m);
    expect(robots).toMatch(/^Disallow: \/\*&import=$/m);
  });

  it('sitemap.xml lists the home page', () => {
    const sitemap = read('static/sitemap.xml');
    expect(sitemap).toContain('xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"');
    expect([...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1])).toEqual(['https://pob.gg/']);
  });

  it('security.txt has the RFC 9116 fields and has not expired', () => {
    const fields = Object.fromEntries(
      read('static/.well-known/security.txt')
        .trim()
        .split('\n')
        .map((l) => [l.slice(0, l.indexOf(':')), l.slice(l.indexOf(':') + 1).trim()])
    );
    expect(fields.Contact).toBe('https://github.com/Vilsol/go-pob/security/advisories/new');
    expect(fields.Canonical).toBe('https://pob.gg/.well-known/security.txt');
    expect(fields['Preferred-Languages']).toBe('en');
    // RFC 9116 2.5.5: renew Expires (at most a year out) when this fails
    expect(Date.parse(fields.Expires)).toBeGreaterThan(Date.now());
  });

  it('the manifest names both games and has no screenshots', () => {
    const manifest = JSON.parse(read('static/manifest.webmanifest')) as Record<string, unknown>;
    expect(manifest.description).toMatch(/Path of Exile 2/);
    expect(manifest).toMatchObject({ lang: 'en', dir: 'ltr', categories: ['games', 'utilities'] });
    expect(manifest).not.toHaveProperty('screenshots');
  });
});

describe('missing files answer 404', () => {
  const routes = JSON.parse(read('static/_routes.json')) as { version: number; include: string[]; exclude: string[] };
  const page = (type: string) => new Response('<!doctype html>', { headers: { 'Content-Type': type } });

  it('_routes.json stays inside the Pages limits and skips every app route', () => {
    const rules = [...routes.include, ...routes.exclude];
    expect(routes.include.length).toBeGreaterThan(0);
    expect(rules.length).toBeLessThanOrEqual(100);
    for (const r of rules) expect(r.length).toBeLessThanOrEqual(100);
    const appRoutes = readdirSync(resolve(frontend, 'src/routes'), { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => `/${d.name}`);
    expect(appRoutes.length).toBeGreaterThan(0);
    for (const r of ['/', ...appRoutes]) expect(routes.exclude).toContain(r);
    expect(routes.exclude.some((r) => r.startsWith('/api'))).toBe(false);
  });

  // every real file and folder the app serves bypasses the function, so a page load costs no invocation; the wasm
  // files are listed by their compressed names, as the deploy serves them
  it('_routes.json skips every file and folder in static/', () => {
    const entries = readdirSync(resolve(frontend, 'static'), { withFileTypes: true }).filter(
      (d) => !d.name.startsWith('_') && !/\.wasm(\.gz|\.br)?$/.test(d.name)
    );
    expect(entries.length).toBeGreaterThan(0);
    for (const d of entries) expect(routes.exclude).toContain(d.isDirectory() ? `/${d.name}/*` : `/${d.name}`);
  });

  it('only a file path answered with the app page is missing', () => {
    expect(missingFile('/apple-touch-icon.png', page('text/html; charset=utf-8'))).toBe(true);
    expect(missingFile('/.well-known/nothing.TXT', page('text/html'))).toBe(true);
    expect(missingFile('/robots.txt', page('text/plain; charset=utf-8'))).toBe(false);
    expect(missingFile('/manifest.webmanifest', page('application/manifest+json'))).toBe(false);
    expect(missingFile('/tree', page('text/html'))).toBe(false);
    expect(missingFile('/build.png/tree', page('text/html'))).toBe(false);
  });

  it('the function turns the app page into a 404 and passes everything else through', async () => {
    const ctx = (url: string, res: Response) => ({ request: new Request(url), params: {}, waitUntil: () => {}, next: () => Promise.resolve(res) }) as never;
    const missing = await onRequest(ctx('https://pob.gg/sitemap2.xml', page('text/html')));
    expect(missing.status).toBe(404);
    expect(missing.headers.get('Content-Type')).toMatch(/^text\/plain/);
    const route = page('text/html');
    expect(await onRequest(ctx('https://pob.gg/party?import=x.json', route))).toBe(route);
    const file = page('application/xml');
    expect(await onRequest(ctx('https://pob.gg/sitemap.xml', file))).toBe(file);
  });
});
