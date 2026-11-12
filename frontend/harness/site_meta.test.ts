import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
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

  it('shows the square logo on link cards', () => {
    // chat apps cache a card image by URL, so the URL carries icon-512.png's hash (mise run logo writes it)
    const cardImage = `https://pob.gg/icon-512.png?v=${createHash('sha256')
      .update(readFileSync(resolve(frontend, 'static/icon-512.png')))
      .digest('hex')
      .slice(0, 8)}`;
    const alt = meta('property', 'og:image:alt');
    expect(meta('property', 'og:image')).toBe(cardImage);
    expect(meta('property', 'og:image:type')).toBe('image/png');
    expect(meta('property', 'og:image:width')).toBe('512');
    expect(meta('property', 'og:image:height')).toBe('512');
    expect(alt).toMatch(/logo/);
    expect(meta('name', 'twitter:image')).toBe(cardImage);
    expect(meta('name', 'twitter:image:alt')).toBe(alt);
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

// width and height from a PNG's IHDR chunk
const pngSize = (b: Buffer) => {
  expect(b.subarray(0, 8).toString('latin1')).toBe('\x89PNG\r\n\x1a\n');
  return [b.readUInt32BE(16), b.readUInt32BE(20)];
};
const staticFile = (path: string) => readFileSync(resolve(frontend, 'static', path));

describe('icons', () => {
  it('each PNG icon has its size', () => {
    for (const [file, n] of [
      ['favicon.png', 64],
      ['apple-touch-icon.png', 180],
      ['icon-192.png', 192],
      ['icon-512.png', 512],
      ['icon-maskable-512.png', 512]
    ] as const)
      expect([file, ...pngSize(staticFile(file))]).toEqual([file, n, n]);
  });

  it('favicon.ico holds 16, 32 and 48 px PNGs', () => {
    const ico = staticFile('favicon.ico');
    expect([ico.readUInt16LE(0), ico.readUInt16LE(2)]).toEqual([0, 1]);
    const sizes = Array.from({ length: ico.readUInt16LE(4) }, (_, i) => {
      const at = 6 + 16 * i;
      const png = ico.subarray(ico.readUInt32LE(at + 12), ico.readUInt32LE(at + 12) + ico.readUInt32LE(at + 8));
      const [w, h] = pngSize(png);
      expect([ico[at] || 256, ico[at + 1] || 256]).toEqual([w, h]);
      return w;
    });
    expect(sizes).toEqual([16, 32, 48]);
  });

  it('the apple-touch and maskable icons are opaque', () => {
    // colour type 2 (RGB) or a palette without a tRNS chunk
    for (const file of ['apple-touch-icon.png', 'icon-maskable-512.png']) {
      const png = staticFile(file);
      expect([file, png[25] === 2 || (png[25] === 3 && !png.includes('tRNS'))]).toEqual([file, true]);
    }
  });

  it('the SVG favicon is the minified logo with its title', () => {
    const svg = staticFile('favicon.svg').toString('utf8');
    expect(svg).toMatch(/^<svg [^>]*viewBox="0 0 512 512"/);
    expect(svg).toMatch(/<title>[^<]+<\/title>/);
    expect(svg).not.toMatch(/<desc|<metadata|<!--|c2pa/);
  });

  it('the manifest and app.html name only files that exist', () => {
    const manifest = JSON.parse(read('static/manifest.webmanifest')) as { icons: { src: string; sizes: string; type: string }[] };
    expect(manifest.icons.map((i) => i.src)).toContain('favicon.svg');
    for (const i of manifest.icons) {
      if (i.sizes !== 'any') expect(pngSize(staticFile(i.src)).join('x')).toBe(i.sizes);
      expect(i.type).toBe(i.src.endsWith('.svg') ? 'image/svg+xml' : 'image/png');
    }
    const links = [...head.matchAll(/href="%sveltekit\.assets%\/([^"]+)"/g)].map((m) => m[1]);
    expect(links).toEqual(expect.arrayContaining(['favicon.png', 'favicon.svg', 'apple-touch-icon.png']));
    for (const l of links) expect(() => staticFile(l)).not.toThrow();
    for (const url of [meta('property', 'og:image'), meta('name', 'twitter:image')]) expect(() => staticFile(new URL(url!).pathname.slice(1))).not.toThrow();
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
    // what git tracks in static/: a local build artifact (a gitignored .wasm.br.sha256) is never deployed
    const tracked = execFileSync('git', ['ls-files', 'static'], { cwd: frontend, encoding: 'utf8' }).split('\n').filter(Boolean);
    const entries = [...new Set(tracked.map((f) => f.slice('static/'.length).split('/')[0]))]
      .filter((n) => !n.startsWith('_') && !/\.wasm(\.gz|\.br)?$/.test(n))
      .map((n) => (tracked.some((f) => f.startsWith(`static/${n}/`)) ? `/${n}/*` : `/${n}`));
    expect(entries.length).toBeGreaterThan(0);
    for (const e of entries) expect(routes.exclude).toContain(e);
  });

  it('only a file path answered with the app page is missing', () => {
    expect(missingFile('/apple-touch-icon-precomposed.png', page('text/html; charset=utf-8'))).toBe(true);
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
