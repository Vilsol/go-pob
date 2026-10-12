import { expect, within } from 'storybook/test';

const px = (v: string) => parseFloat(v) || 0;

const luminance = (c: string) => {
  const [r, g, b] = (c.match(/[\d.]+/g) ?? ['0', '0', '0']).slice(0, 3).map((n) => {
    const x = Number(n) / 255;
    return x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

export async function expectTopBarFits(root: HTMLElement) {
  const bar = root.querySelector('header');
  await expect(bar).not.toBeNull();
  const r = bar!.getBoundingClientRect();
  const cs = getComputedStyle(bar!);
  await expect(Math.round(r.height)).toBe(52);
  await expect(px(cs.paddingLeft)).toBeGreaterThanOrEqual(12);
  await expect(px(cs.paddingRight)).toBeGreaterThanOrEqual(12);
  await expect(bar!.scrollWidth).toBeLessThanOrEqual(bar!.clientWidth + 1);
  const outside = [...bar!.querySelectorAll<HTMLElement>('*')].filter((e) => {
    const b = e.getBoundingClientRect();
    return b.width > 0 && (b.right > r.right + 1 || b.bottom > r.bottom + 1);
  });
  await expect(outside.map((e) => e.outerHTML.slice(0, 80))).toEqual([]);
  for (const b of bar!.querySelectorAll<HTMLElement>('button, a.btn')) {
    await expect(luminance(getComputedStyle(b).backgroundColor) < 0.5 || getComputedStyle(b).backgroundColor === 'rgba(0, 0, 0, 0)').toBe(true);
  }
}

export async function expectPanelBelow(root: HTMLElement, panel: HTMLElement) {
  const bar = root.querySelector('header')!.getBoundingClientRect();
  const p = panel.getBoundingClientRect();
  await expect(Math.round(bar.height)).toBe(52);
  await expect(p.top).toBeGreaterThanOrEqual(bar.bottom - 1);
}

export async function expectInset(box: HTMLElement, kids: HTMLElement[], min: number, sides: ('left' | 'right' | 'top')[]) {
  const b = box.getBoundingClientRect();
  for (const k of kids) {
    const r = k.getBoundingClientRect();
    if (!r.width) continue;
    const gaps = { left: r.left - b.left, right: b.right - r.right, top: r.top - b.top };
    for (const s of sides) await expect(gaps[s], `${s} gap of ${k.outerHTML.slice(0, 60)}`).toBeGreaterThanOrEqual(min);
  }
}

export async function expectFloatingLabel(label: HTMLElement, control: HTMLElement) {
  const l = label.getBoundingClientRect();
  const c = control.getBoundingClientRect();
  await expect(l.width).toBeGreaterThan(0);
  await expect(l.top).toBeLessThan(c.top);
  await expect(l.bottom).toBeGreaterThan(c.top);
  await expect(l.left).toBeGreaterThanOrEqual(c.left);
  await expect(l.right).toBeLessThanOrEqual(c.right);
}

export async function expectLabelled(label: HTMLElement, value: HTMLElement, stacked: boolean) {
  const l = label.getBoundingClientRect();
  const v = value.getBoundingClientRect();
  await expect(l.width * v.width).toBeGreaterThan(0);
  if (stacked) await expect(l.bottom, `${label.textContent} above its value`).toBeLessThanOrEqual(v.top + 1);
  else {
    await expect(l.right, `${label.textContent} left of its value`).toBeLessThanOrEqual(v.left + 1);
    await expect(Math.abs(l.top + l.height / 2 - (v.top + v.height / 2))).toBeLessThanOrEqual(3);
  }
}

export async function expectIconButton(b: HTMLElement) {
  const r = b.getBoundingClientRect();
  await expect(Math.abs(r.width - r.height), `${b.getAttribute('aria-label')} is square`).toBeLessThanOrEqual(1);
  await expect(b.querySelector('svg')?.getBoundingClientRect().width ?? 0).toBeGreaterThan(0);
  await expect(b.innerText.trim()).toBe('');
}

export async function expectSelectArrowRoom(sel: HTMLSelectElement) {
  const cs = getComputedStyle(sel);
  await expect(cs.appearance, 'native arrow replaced').toBe('none');
  await expect(cs.backgroundImage).not.toBe('none');
  const pos = cs.backgroundPositionX;
  const m = pos.match(/^(?:right (\d+(?:\.\d+)?)px|calc\(100% - (\d+(?:\.\d+)?)px\))$/);
  await expect(m, `background-position-x ${pos}`).not.toBeNull();
  const inset = Number(m![1] ?? m![2]);
  const arrowW = px(cs.backgroundSize.split(' ')[0]);
  await expect(inset).toBeGreaterThanOrEqual(10);
  await expect(arrowW).toBeGreaterThan(0);
  const r = sel.getBoundingClientRect();
  const innerRight = r.right - px(cs.borderRightWidth);
  const textRight = innerRight - px(cs.paddingRight);
  await expect(textRight, 'text box ends before the arrow').toBeLessThanOrEqual(innerRight - inset - arrowW - 2);
}

export async function expectPageFits(main: HTMLElement) {
  await expect(document.documentElement.scrollWidth, 'the document scrolls sideways').toBeLessThanOrEqual(window.innerWidth);
  await expect(main.scrollWidth, 'main scrolls sideways').toBeLessThanOrEqual(main.clientWidth + 1);
  const m = main.getBoundingClientRect();
  await expect(Math.round(m.right)).toBeLessThanOrEqual(window.innerWidth);
  const cs = getComputedStyle(main);
  await expect(px(cs.paddingLeft)).toBeGreaterThanOrEqual(12);
  await expect(px(cs.paddingRight)).toBeGreaterThanOrEqual(12);
  const blocks = [...main.querySelectorAll<HTMLElement>('section, aside, [role="table"], [role="grid"], [role="tablist"]')];
  await expect(blocks.length, 'the page rendered its blocks').toBeGreaterThan(0);
  const right = m.right - px(cs.paddingRight) - (main.offsetWidth - main.clientWidth);
  const out = blocks.filter((e) => {
    const b = e.getBoundingClientRect();
    return b.width > 0 && (b.left < m.left + px(cs.paddingLeft) - 1 || b.right > right + 1);
  });
  await expect(out.map((e) => e.getAttribute('aria-label') ?? e.outerHTML.slice(0, 60))).toEqual([]);
}

export async function expectFloating(trigger: HTMLElement, pop: HTMLElement) {
  const t = trigger.getBoundingClientRect();
  const p = pop.getBoundingClientRect();
  await expect(p.width * p.height, 'the popover is drawn').toBeGreaterThan(0);
  const ox = Math.min(t.right, p.right) - Math.max(t.left, p.left);
  const oy = Math.min(t.bottom, p.bottom) - Math.max(t.top, p.top);
  await expect(ox <= 1 || oy <= 1, `${pop.getAttribute('role')} covers its trigger`).toBe(true);
  await expect(p.left).toBeGreaterThanOrEqual(0);
  await expect(p.top).toBeGreaterThanOrEqual(0);
  await expect(p.right).toBeLessThanOrEqual(window.innerWidth);
  await expect(p.bottom).toBeLessThanOrEqual(window.innerHeight);
}

export async function expectModalTrap(dialog: HTMLElement, tab: (shift: boolean) => Promise<void>) {
  const d = dialog.getBoundingClientRect();
  await expect(d.left).toBeGreaterThanOrEqual(0);
  await expect(d.top).toBeGreaterThanOrEqual(0);
  await expect(d.right).toBeLessThanOrEqual(window.innerWidth);
  await expect(d.bottom).toBeLessThanOrEqual(window.innerHeight);
  const n = dialog.querySelectorAll('button, input, select, textarea, a[href]').length;
  for (let i = 0; i < n + 2; i++) {
    await tab(false);
    await expect(dialog.contains(document.activeElement), `Tab ${i + 1} left the dialog`).toBe(true);
  }
  for (let i = 0; i < 3; i++) {
    await tab(true);
    await expect(dialog.contains(document.activeElement), `Shift+Tab ${i + 1} left the dialog`).toBe(true);
  }
}

export async function expectValuesVisible(root: HTMLElement) {
  const inputs = [...root.querySelectorAll<HTMLInputElement>('[role="spinbutton"]')].filter((i) => i.getBoundingClientRect().width);
  await expect(inputs.length).toBeGreaterThan(0);
  for (const i of inputs) await expect(i.scrollWidth, `${i.getAttribute('aria-label')} = ${i.value}`).toBeLessThanOrEqual(i.clientWidth);
}

export async function expectDocked(main: HTMLElement, panel: HTMLElement, content: HTMLElement) {
  const m = main.getBoundingClientRect();
  const p = panel.getBoundingClientRect();
  const c = content.getBoundingClientRect();
  await expect(p.width * p.height, 'the panel is drawn').toBeGreaterThan(0);
  await expect(p.left).toBeGreaterThanOrEqual(m.left - 1);
  await expect(p.right).toBeLessThanOrEqual(m.right + 1);
  await expect(p.left, 'the panel starts where the content ends').toBeGreaterThanOrEqual(c.right - 1);
  const under = [...content.querySelectorAll<HTMLElement>('section')].filter((e) => {
    const b = e.getBoundingClientRect();
    return b.width > 0 && b.right > p.left + 1 && b.bottom > p.top && b.top < p.bottom;
  });
  await expect(
    under.map((e) => e.getAttribute('aria-label')),
    'sections under the panel'
  ).toEqual([]);
}

export async function expectNotClipped(els: HTMLElement[]) {
  await expect(els.length).toBeGreaterThan(0);
  for (const e of els) await expect(e.scrollWidth, `"${e.textContent}" is clipped`).toBeLessThanOrEqual(e.clientWidth + 1);
}

export async function expectSingleLine(els: HTMLElement[]) {
  await expect(els.length).toBeGreaterThan(0);
  for (const e of els) {
    const range = document.createRange();
    range.selectNodeContents(e);
    const lines = new Set([...range.getClientRects()].filter((r) => r.width).map((r) => Math.round(r.top)));
    await expect(lines.size, `"${e.textContent}" wraps`).toBeLessThanOrEqual(1);
  }
}

export async function expectDarkCheckboxes(root: HTMLElement) {
  const boxes = [...root.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')].filter((b) => b.getBoundingClientRect().width);
  await expect(boxes.length, 'visible checkboxes').toBeGreaterThan(0);
  const probe = document.createElement('div');
  probe.style.background = 'var(--color-accent)';
  document.body.append(probe);
  const accent = getComputedStyle(probe).backgroundColor;
  probe.remove();
  for (const b of boxes) {
    const cs = getComputedStyle(b);
    const name = b.getAttribute('aria-label') ?? b.closest('label')?.textContent?.trim() ?? b.outerHTML.slice(0, 60);
    await expect(cs.appearance, `${name}: native box replaced`).toBe('none');
    if (b.checked) await expect(cs.backgroundColor, `${name}: ticked = accent`).toBe(accent);
    else await expect(luminance(cs.backgroundColor), `${name}: unticked is dark (${cs.backgroundColor})`).toBeLessThan(0.2);
  }
}

export async function expectCapped(row: HTMLElement, cap: number) {
  const gauge = row.querySelector<HTMLElement>('[data-gauge]');
  await expect(gauge, `${row.textContent}: gauge`).not.toBeNull();
  const hatch = gauge!.querySelector<HTMLElement>('[data-cap-zone]');
  const tick = gauge!.querySelector<HTMLElement>('[data-cap-tick]');
  await expect(hatch, `${row.textContent}: hatch beyond the cap`).not.toBeNull();
  await expect(tick, `${row.textContent}: cap tick`).not.toBeNull();
  const g = gauge!.getBoundingClientRect();
  const t = tick!.getBoundingClientRect();
  const h = hatch!.getBoundingClientRect();
  const at = g.left + (g.width * cap) / 100;
  await expect(Math.abs(t.left + t.width / 2 - at), `${row.textContent}: tick at ${cap}%`).toBeLessThanOrEqual(1.5);
  await expect(Math.abs(h.left - at), `${row.textContent}: hatch from ${cap}%`).toBeLessThanOrEqual(1);
  await expect(h.width).toBeGreaterThan(0);
}

export async function expectGauge(row: HTMLElement, g: { pct: number; toneClass?: string; res?: number; slack?: number }) {
  const gauge = row.querySelector<HTMLElement>('[data-gauge]');
  await expect(gauge, `${row.textContent}: gauge`).not.toBeNull();
  const fill = gauge!.querySelector<HTMLElement>('[data-fill]');
  await expect(fill, `${row.textContent}: fill`).not.toBeNull();
  const w = gauge!.getBoundingClientRect().width;
  const want = (w * Math.max(0, Math.min(100, g.pct))) / 100;
  const slack = 1 + (w * (g.slack ?? 0)) / 100;
  await expect(Math.abs(fill!.getBoundingClientRect().width - want), `${row.textContent}: fill at ${g.pct}%`).toBeLessThanOrEqual(slack);
  await expect(getComputedStyle(fill!).backgroundColor, `${row.textContent}: fill colour`).not.toBe('rgba(0, 0, 0, 0)');
  if (g.toneClass) {
    const probe = document.createElement('span');
    probe.className = g.toneClass;
    gauge!.append(probe);
    const tone = getComputedStyle(probe).getPropertyValue('--tone').trim();
    probe.remove();
    await expect(tone, `${g.toneClass} resolves`).not.toBe('');
    await expect(getComputedStyle(fill!).getPropertyValue('--tone').trim(), `${row.textContent}: fill tone`).toBe(tone);
  }
  const band = gauge!.querySelector<HTMLElement>('[data-res]');
  if (g.res === undefined) {
    await expect(band, `${row.textContent}: no reserved band`).toBeNull();
  } else {
    await expect(band, `${row.textContent}: reserved band`).not.toBeNull();
    await expect(Math.abs(band!.getBoundingClientRect().width - (w * g.res) / 100), `${row.textContent}: ${g.res}% reserved`).toBeLessThanOrEqual(slack);
  }
}

export async function expectCollapsed(el: HTMLElement) {
  await expect(getComputedStyle(el).display, 'display of a collapsed element').toBe('none');
  await expect(el.getBoundingClientRect().height).toBe(0);
}

export async function expectSelectsFloating(root: HTMLElement, bare: string[] = []) {
  const names = new Map<Element, string>();
  within(root).queryAllByRole('combobox', {
    name: (n: string, el: Element | null) => {
      if (el) names.set(el, n.trim());
      return true;
    }
  });
  const sels = [...root.querySelectorAll<HTMLSelectElement>('select')].filter((s) => s.getBoundingClientRect().width);
  await expect(sels.length, 'visible selects').toBeGreaterThan(0);
  for (const s of sels) {
    const name = names.get(s) ?? '';
    await expect(name, `${s.outerHTML.slice(0, 80)} has an accessible name`).not.toBe('');
    if (bare.includes(name)) continue;
    const label = s.closest('label')?.querySelector<HTMLElement>(':scope > [data-floating-label]');
    await expect(label, `${name}: floating label`).toBeTruthy();
    const l = label!.getBoundingClientRect();
    const c = s.getBoundingClientRect();
    await expect(l.width, `${name}: label drawn`).toBeGreaterThan(0);
    await expect(l.top < c.top && l.bottom > c.top, `${name}: label on the top border`).toBe(true);
    await expect(l.left >= c.left - 1 && l.right <= c.right + 1, `${name}: label within the select`).toBe(true);
    label!.scrollIntoView({ block: 'nearest' });
    const lv = label!.getBoundingClientRect();
    const top = document.elementFromPoint(lv.left + lv.width / 2, s.getBoundingClientRect().top);
    await expect(top === label || label!.contains(top), `${name}: label drawn over the select`).toBe(true);
  }
}

export async function expectThemedScrollbars(root: HTMLElement) {
  const scrollers = [root, ...root.querySelectorAll<HTMLElement>('*')].filter((e) =>
    /(auto|scroll)/.test(getComputedStyle(e).overflowY + getComputedStyle(e).overflowX)
  );
  await expect(scrollers.length, 'scrolling areas').toBeGreaterThan(0);
  for (const e of scrollers) {
    const cs = getComputedStyle(e);
    await expect(cs.scrollbarWidth, `${e.tagName}.${e.className.toString().slice(0, 40)} scrollbar width`).toBe('thin');
    await expect(cs.scrollbarColor, `${e.tagName}.${e.className.toString().slice(0, 40)} scrollbar colour`).not.toBe('auto');
  }
}
