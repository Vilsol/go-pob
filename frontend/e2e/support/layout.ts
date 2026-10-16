import type { Page } from '@playwright/test';

export type Finding = { check: 'page-scroll' | 'clipped' | 'scrollbar' | 'overlap'; where: string; detail: string };

export type Scroller = { selector: string; axis: 'x' | 'y'; why: string };

const CHECKED = 'button, a[href], input:not([type=hidden]), select, textarea, [role=tab], [role=separator], h1, h2, h3, h4, [role=columnheader]';
const FLOATING = '[role=tooltip], [role=dialog], dialog, [role=listbox], [role=menu], .tooltip-float';

export const layoutFindings = (page: Page, root: string, allow: Scroller[]): Promise<Finding[]> =>
  page.evaluate(
    ([rootSel, allowed, checkedSel, floatingSel]) => {
      const base = document.querySelector(rootSel);
      if (!base) throw new Error(`layout check: no ${rootSel}`);
      const out: { check: 'page-scroll' | 'clipped' | 'scrollbar' | 'overlap'; where: string; detail: string }[] = [];
      const where = (el: Element) => {
        const parts: string[] = [];
        for (let e: Element | null = el; e && e !== document.body && parts.length < 4; e = e.parentElement) {
          const label = e.getAttribute('aria-label') ?? (e.matches(checkedSel) ? (e.textContent ?? '').trim().slice(0, 30) : '');
          parts.unshift(e.tagName.toLowerCase() + (label ? `[${label}]` : ''));
        }
        return parts.join(' > ');
      };
      const px = (n: number) => Math.round(n);

      const doc = document.scrollingElement ?? document.documentElement;
      if (doc.scrollWidth > window.innerWidth + 1)
        out.push({ check: 'page-scroll', where: 'document', detail: `scrollWidth ${doc.scrollWidth} > viewport ${window.innerWidth}` });

      for (const el of [base, ...base.querySelectorAll('*')]) {
        if (!(el instanceof HTMLElement) || (!el.scrollLeft && !el.scrollTop)) continue;
        const cs = getComputedStyle(el);
        if ((el.scrollLeft && !/auto|scroll/.test(cs.overflowX)) || (el.scrollTop && !/auto|scroll/.test(cs.overflowY)))
          out.push({
            check: 'page-scroll',
            where: where(el),
            detail: `a clipping box scrolled to ${el.scrollLeft},${el.scrollTop} (content ${el.scrollWidth}×${el.scrollHeight} in ${el.clientWidth}×${el.clientHeight})`
          });
      }

      const shown = (el: Element): el is HTMLElement => {
        if (!(el instanceof HTMLElement) || el.closest('svg, canvas')) return false;
        const r = el.getBoundingClientRect();
        if (r.width <= 1 || r.height <= 1) return false;
        const cs = getComputedStyle(el);
        return cs.visibility !== 'hidden' && !el.closest('.sr-only');
      };
      const scrolls = (cs: CSSStyleDeclaration, axis: 'x' | 'y') => /auto|scroll/.test(axis === 'x' ? cs.overflowX : cs.overflowY);
      const clips = (cs: CSSStyleDeclaration, axis: 'x' | 'y') => (axis === 'x' ? cs.overflowX : cs.overflowY) !== 'visible';

      for (const el of [base, ...base.querySelectorAll('*')]) {
        if (!shown(el) || el.closest(floatingSel)) continue;
        const cs = getComputedStyle(el);
        for (const axis of ['x', 'y'] as const) {
          if (!scrolls(cs, axis)) continue;
          const over = axis === 'x' ? el.scrollWidth - el.clientWidth : el.scrollHeight - el.clientHeight;
          const forced = (axis === 'x' ? cs.overflowX : cs.overflowY) === 'scroll';
          if (over <= 1 && !forced) continue;
          if (allowed.some((a) => a.axis === axis && el.matches(a.selector))) continue;
          out.push({ check: 'scrollbar', where: where(el), detail: `${axis} scrolls by ${over}px` });
        }
      }

      const checked = [...base.querySelectorAll(checkedSel)].filter((el): el is HTMLElement => shown(el) && !el.closest(floatingSel));
      for (const el of checked) {
        const r = el.getBoundingClientRect();
        let [l, t, ri, b] = [r.left, r.top, r.right, r.bottom];
        let freeX = false;
        let freeY = false;
        for (let a = el.parentElement; a; a = a.parentElement) {
          if (getComputedStyle(a).position === 'fixed') break;
          const cs = getComputedStyle(a);
          const ar = a.getBoundingClientRect();
          if (scrolls(cs, 'x')) freeX = true;
          if (scrolls(cs, 'y')) freeY = true;
          if (!freeX && clips(cs, 'x')) [l, ri] = [Math.max(l, ar.left), Math.min(ri, ar.right)];
          if (!freeY && clips(cs, 'y')) [t, b] = [Math.max(t, ar.top), Math.min(b, ar.bottom)];
        }
        if (!freeX) [l, ri] = [Math.max(l, 0), Math.min(ri, window.innerWidth)];
        if (!freeY) [t, b] = [Math.max(t, 0), Math.min(b, window.innerHeight)];
        const cutX = r.width - Math.max(0, ri - l);
        const cutY = r.height - Math.max(0, b - t);
        if (cutX > 1 || cutY > 1)
          out.push({ check: 'clipped', where: where(el), detail: `${px(cutX)}px of ${px(r.width)} wide, ${px(cutY)}px of ${px(r.height)} high cut` });
      }

      const visible = (el: HTMLElement) => {
        const r = el.getBoundingClientRect();
        let [left, top, right, bottom] = [
          Math.max(r.left, 0),
          Math.max(r.top, 0),
          Math.min(r.right, window.innerWidth),
          Math.min(r.bottom, window.innerHeight)
        ];
        for (let a = el.parentElement; a; a = a.parentElement) {
          const cs = getComputedStyle(a);
          if (cs.position === 'fixed') break;
          const ar = a.getBoundingClientRect();
          if (clips(cs, 'x')) [left, right] = [Math.max(left, ar.left), Math.min(right, ar.right)];
          if (clips(cs, 'y')) [top, bottom] = [Math.max(top, ar.top), Math.min(bottom, ar.bottom)];
        }
        return new DOMRect(left, top, right - left, bottom - top);
      };
      const boxes = checked.map((el) => ({ el, r: visible(el) })).filter(({ r }) => r.width > 0 && r.height > 0);
      for (let i = 0; i < boxes.length; i++) {
        for (let j = i + 1; j < boxes.length; j++) {
          const [p, q] = [boxes[i], boxes[j]];
          if (p.el.contains(q.el) || q.el.contains(p.el)) continue;
          const w = Math.min(p.r.right, q.r.right) - Math.max(p.r.left, q.r.left);
          const h = Math.min(p.r.bottom, q.r.bottom) - Math.max(p.r.top, q.r.top);
          const inside = (a: DOMRect, b: DOMRect) => a.left >= b.left - 1 && a.right <= b.right + 1 && a.top >= b.top - 1 && a.bottom <= b.bottom + 1;
          if ((p.el.matches('input, textarea') && inside(q.r, p.r)) || (q.el.matches('input, textarea') && inside(p.r, q.r))) continue;
          if (w > 2 && h > 2) out.push({ check: 'overlap', where: `${where(p.el)}  ×  ${where(q.el)}`, detail: `${px(w)}×${px(h)}px` });
        }
      }
      return out;
    },
    [root, allow, CHECKED, FLOATING] as const
  );
