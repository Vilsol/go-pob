// Rasterises an SVG in headless Chromium, the renderer the favicon meets in a browser:
// LOGO_CHROMIUM=<chromium> node logo-render.mjs <svg> <out dir> <size>... writes <out dir>/<size>.png with a
// transparent background. go run ./scripts/logo passes the pinned Chromium.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const [svg, outDir, ...sizes] = process.argv.slice(2);
const executablePath = process.env.LOGO_CHROMIUM;
if (!executablePath || !svg || !outDir || sizes.length === 0 || sizes.some((s) => !/^\d+$/.test(s))) {
  console.error('usage: LOGO_CHROMIUM=<chromium> logo-render.mjs <svg> <out dir> <size>...');
  process.exit(2);
}
const src = `data:image/svg+xml;base64,${readFileSync(svg).toString('base64')}`;
const browser = await chromium.launch({ executablePath, args: ['--force-color-profile=srgb', '--disable-gpu'] });
try {
  for (const s of sizes) {
    const n = Number(s);
    const page = await browser.newPage({ viewport: { width: n, height: n }, deviceScaleFactor: 1 });
    await page.setContent(
      `<html style="background:transparent"><body style="margin:0"><img src="${src}" width="${n}" height="${n}" style="display:block"></body></html>`
    );
    await page.locator('img').evaluate((img) => img.decode());
    await page.screenshot({ path: join(outDir, `${n}.png`), omitBackground: true, clip: { x: 0, y: 0, width: n, height: n } });
    await page.close();
  }
} finally {
  await browser.close();
}
