import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

const CORPUS = resolve(import.meta.dirname, '../../../testdata/corpus-3.29');
const BUILD = process.env.UNLIGHTHOUSE_BUILD || '0280-elementalist-fireball';
const PORT = Number(process.env.UNLIGHTHOUSE_PORT || 4392);
const ROUTES = ['tree', 'skills', 'items', 'calcs', 'configuration', 'notes', 'party', 'compare', 'import'];

const code = deflateSync(readFileSync(resolve(CORPUS, `${BUILD}.xml`)))
  .toString('base64')
  .replace(/\+/g, '-')
  .replace(/\//g, '_');
const query = new URLSearchParams({ import: code, sw: 'off' }).toString();

export default {
  site: process.env.UNLIGHTHOUSE_SITE || `http://localhost:${PORT}`,
  urls: ['/', ...ROUTES.map((r) => `/${r}?${query}`)],
  outputPath: resolve(import.meta.dirname, '../.results/unlighthouse'),
  cache: false,
  chrome: { useSystem: true },
  puppeteerOptions: {
    ...(process.env.E2E_CHROMIUM ? { executablePath: process.env.E2E_CHROMIUM } : {}),
    args: process.env.UNLIGHTHOUSE_MARK ? [process.env.UNLIGHTHOUSE_MARK] : []
  },
  scanner: {
    device: 'mobile',
    crawler: false,
    sitemap: false,
    robotsTxt: false,
    dynamicSampling: false,
    samples: 1
  },
  lighthouseOptions: {
    formFactor: 'mobile',
    maxWaitForLoad: 180_000,
    networkQuietThresholdMs: 5_000,
    cpuQuietThresholdMs: 5_000,
    pauseAfterLoadMs: 5_000,
    screenEmulation: { mobile: true, width: 390, height: 769, deviceScaleFactor: 3, disabled: false }
  },
  ci: { buildStatic: true, reporter: 'jsonExpanded' }
};
