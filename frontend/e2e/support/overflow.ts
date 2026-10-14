import type { Page } from '@playwright/test';

export type Overflow = { key: string; dx: number; dy: number; text: string };

export const overflowing = (page: Page, root: string, skip: string[] = []): Promise<Overflow[]> =>
  page.evaluate(
    ([sel, skipSel]) => {
      const base = document.querySelector(sel);
      if (!base) throw new Error(`overflow census: no ${sel}`);
      const skipped = skipSel.flatMap((s) => [...document.querySelectorAll(s)]);
      const keyOf = (el: Element) => {
        const parts: string[] = [];
        for (let e: Element | null = el; e && e !== base && parts.length < 4; e = e.parentElement) {
          const label = e.getAttribute('aria-label') ?? e.getAttribute('role') ?? '';
          const cls = [...e.classList].filter((c) => !/[[:(]/.test(c)).slice(0, 3);
          parts.unshift(e.tagName.toLowerCase() + (cls.length ? '.' + cls.join('.') : '') + (label ? `[${label.slice(0, 30)}]` : ''));
        }
        return parts.join(' > ');
      };
      const hits: Element[] = [];
      const found: { el: Element; dx: number; dy: number }[] = [];
      for (const el of base.querySelectorAll('*')) {
        if (!(el instanceof HTMLElement) || el.closest('svg, canvas')) continue;
        if (skipped.some((s) => s.contains(el))) continue;
        if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) continue;
        if (el.clientWidth === 0 && el.clientHeight === 0) continue;
        const cs = getComputedStyle(el);
        if (cs.visibility === 'hidden') continue;
        if (cs.position === 'absolute' && (el.clientWidth <= 1 || el.clientHeight <= 1)) continue;
        const scrollX = /auto|scroll/.test(cs.overflowX) || cs.textOverflow === 'ellipsis';
        const scrollY = /auto|scroll/.test(cs.overflowY);
        const dx = scrollX ? 0 : el.scrollWidth - el.clientWidth;
        const dy = scrollY ? 0 : el.scrollHeight - el.clientHeight;
        if (dx > 1 || dy > 1) {
          hits.push(el);
          found.push({ el, dx: Math.max(dx, 0), dy: Math.max(dy, 0) });
        }
      }
      return found
        .filter((f) => !hits.some((h) => h !== f.el && f.el.contains(h)))
        .map((f) => ({ key: keyOf(f.el), dx: f.dx, dy: f.dy, text: (f.el.textContent ?? '').trim().slice(0, 60) }));
    },
    [root, skip] as const
  );

export const setFontScaling = async (page: Page, pt: number) => {
  await page.evaluate((size) => {
    document.documentElement.style.fontSize = `${(16 * size) / 12}px`;
  }, pt);
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(r, 50)))));
};
