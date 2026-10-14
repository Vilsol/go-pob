import type { Locator } from '@playwright/test';

export type Box = { what: string; top: number; bottom: number; baseline: number };

export const boxes = (controls: Locator[]): Promise<Box[]> =>
  Promise.all(
    controls.map((c) =>
      c.evaluate((el: HTMLElement) => {
        const r = el.getBoundingClientRect();
        const probe = document.createElement('div');
        probe.style.cssText = 'position:absolute;left:0;top:0;visibility:hidden;display:flex;align-items:baseline';
        const copy = el.cloneNode(true) as HTMLElement;
        copy.style.width = `${r.width}px`;
        copy.style.height = `${r.height}px`;
        copy.style.flex = 'none';
        const mark = document.createElement('span');
        mark.style.cssText = 'display:inline-block;width:0;height:0';
        probe.append(copy, mark);
        el.parentElement!.append(probe);
        const baseline = mark.getBoundingClientRect().bottom - copy.getBoundingClientRect().top;
        probe.remove();
        const what = `${el.tagName.toLowerCase()}${el.getAttribute('role') ? `[${el.getAttribute('role')}]` : ''} "${el.getAttribute('aria-label') ?? el.textContent?.trim().slice(0, 20) ?? ''}"`;
        return { what, top: r.top, bottom: r.bottom, baseline: r.top + baseline };
      })
    )
  );
