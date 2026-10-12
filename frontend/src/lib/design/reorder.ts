import { tick } from 'svelte';

export type ReorderState = {
  grabbed: number;
  message: string;
};
export const reorderState = (): ReorderState => ({ grabbed: -1, message: '' });

type Opts = {
  box: () => HTMLElement | undefined;
  name: (i: number) => string;
  onmove: (from: number, to: number) => void;
  onactivate?: (i: number) => void;
};

const CONTROLS = 'input, select, textarea, button, a[href], [contenteditable], [role="spinbutton"], [role="combobox"]';
const THRESHOLD = 4;
const SETTLE = 150;
const EDGE = 40;
let uid = 0;

type Drag = {
  from: number;
  to: number;
  row: HTMLElement;
  pointer: number;
  x: number;
  y: number;
  started: boolean;
  items: HTMLElement[];
  rects: DOMRect[];
  gap: number;
  dx: number;
  dy: number;
  ghost?: HTMLElement;
  scroller?: HTMLElement;
  scroll0: number;
  frame: number;
};

const reduced = () => matchMedia('(prefers-reduced-motion: reduce)').matches || document.documentElement.dataset.motion === 'off';

function scrollParent(el: HTMLElement | null): HTMLElement | undefined {
  for (; el; el = el.parentElement) if (/(auto|scroll)/.test(getComputedStyle(el).overflowY)) return el;
}

export class Reorder {
  #origin = -1;
  #opts: Opts;
  #d: Drag | null = null;
  #busy = false;
  readonly hint = `reorder-hint-${++uid}`;
  readonly hintText = 'Drag to reorder, or Alt+↑ / Alt+↓, or Space to grab';
  s: ReorderState;

  constructor(s: ReorderState, opts: Opts) {
    this.s = s;
    this.#opts = opts;
  }

  row(i: number) {
    return {
      'data-reorder-item': '',
      'data-grabbed': this.s.grabbed === i ? '' : undefined,
      role: 'group',
      tabindex: 0,
      'aria-label': this.#opts.name(i),
      'aria-describedby': this.hint,
      onpointerdown: (e: PointerEvent) => this.#down(i, e),
      onkeydown: (e: KeyboardEvent) => this.#key(i, e)
    };
  }

  #items() {
    return [...(this.#opts.box()?.querySelectorAll<HTMLElement>(':scope [data-reorder-item]') ?? [])];
  }

  #down(i: number, e: PointerEvent) {
    if (e.button || !e.isPrimary || this.#d || this.#busy) return;
    const row = e.currentTarget as HTMLElement;
    const ctrl = (e.target as Element).closest(CONTROLS);
    if (ctrl && ctrl !== row && row.contains(ctrl) && !ctrl.hasAttribute('data-reorder-press')) return;
    e.preventDefault();
    this.#d = {
      from: i,
      to: i,
      row,
      pointer: e.pointerId,
      x: e.clientX,
      y: e.clientY,
      started: false,
      items: [],
      rects: [],
      gap: 0,
      dx: 0,
      dy: 0,
      scroll0: 0,
      frame: 0
    };
    addEventListener('pointermove', this.#onmove);
    addEventListener('pointerup', this.#onup);
    addEventListener('pointercancel', this.#oncancel);
    addEventListener('keydown', this.#onkey, true);
  }

  #onmove = (e: PointerEvent) => {
    const d = this.#d;
    if (!d || e.pointerId !== d.pointer) return;
    const from = { x: d.x, y: d.y };
    d.x = e.clientX;
    d.y = e.clientY;
    if (!d.started) {
      if (Math.hypot(d.x - from.x, d.y - from.y) < THRESHOLD) {
        d.x = from.x;
        d.y = from.y;
        return;
      }
      this.#start(d, from.x, from.y);
    }
    this.#track(d);
  };

  #onup = (e: PointerEvent) => {
    const d = this.#d;
    if (!d || e.pointerId !== d.pointer) return;
    if (!d.started) return this.#release();
    d.x = e.clientX;
    d.y = e.clientY;
    const stop = (c: Event) => c.stopPropagation();
    addEventListener('click', stop, { capture: true, once: true });
    setTimeout(() => removeEventListener('click', stop, { capture: true }), 0);
    this.#settle(this.#inside(d));
  };

  #oncancel = () => {
    const d = this.#d;
    if (d?.started) this.#settle(false);
    else this.#release();
  };

  #onkey = (e: KeyboardEvent) => {
    if (e.key !== 'Escape' || !this.#d) return;
    e.preventDefault();
    e.stopPropagation();
    this.#oncancel();
  };

  #start(d: Drag, x: number, y: number) {
    d.started = true;
    d.items = this.#items();
    d.rects = d.items.map((el) => el.getBoundingClientRect());
    d.gap = d.rects.length > 1 ? d.rects[1].top - d.rects[0].bottom : 0;
    const r = d.rects[d.from];
    d.dx = x - r.left;
    d.dy = y - r.top;
    d.scroller = scrollParent(d.row.parentElement);
    d.scroll0 = d.scroller?.scrollTop ?? 0;

    const copy = d.row.cloneNode(true) as HTMLElement;
    const src = d.row.querySelectorAll<HTMLInputElement>('input, select, textarea');
    copy.querySelectorAll<HTMLInputElement>('input, select, textarea').forEach((el, k) => {
      if (el.type === 'checkbox' || el.type === 'radio') el.checked = src[k].checked;
      else el.value = src[k].value;
    });
    for (const el of [copy, ...copy.querySelectorAll('[id]')]) el.removeAttribute('id');
    for (const a of ['data-reorder-item', 'data-grabbed', 'role', 'tabindex', 'aria-label', 'aria-describedby']) copy.removeAttribute(a);
    copy.style.height = '100%';
    const ghost = document.createElement('div');
    ghost.append(copy);
    ghost.setAttribute('data-reorder-ghost', '');
    ghost.setAttribute('aria-hidden', 'true');
    ghost.inert = true;
    ghost.classList.add('reorder-ghost');
    Object.assign(ghost.style, { left: `${r.left}px`, top: `${r.top}px`, width: `${r.width}px`, height: `${r.height}px` });
    d.row.parentElement!.append(ghost);
    d.ghost = ghost;
    d.row.setAttribute('data-reorder-source', '');
    this.#opts.box()?.setAttribute('data-reorder-active', '');
    try {
      d.row.setPointerCapture(d.pointer);
    } catch {
      // empty
    }
    const scroll = () => {
      if (this.#d !== d) return;
      const s = d.scroller;
      if (s) {
        const b = s.getBoundingClientRect();
        const step = d.y < b.top + EDGE ? -Math.ceil((b.top + EDGE - d.y) / 4) : d.y > b.bottom - EDGE ? Math.ceil((d.y - b.bottom + EDGE) / 4) : 0;
        const was = s.scrollTop;
        if (step) s.scrollTop += step;
        if (s.scrollTop !== was) this.#track(d);
      }
      d.frame = requestAnimationFrame(scroll);
    };
    d.frame = requestAnimationFrame(scroll);
  }

  #scrolled(d: Drag) {
    return (d.scroller?.scrollTop ?? 0) - d.scroll0;
  }

  #offset(d: Drag, to: number) {
    let o = 0;
    for (let k = d.from + 1; k <= to; k++) o += d.rects[k].height + d.gap;
    for (let k = to; k < d.from; k++) o -= d.rects[k].height + d.gap;
    return o;
  }

  #track(d: Drag) {
    d.ghost!.style.left = `${d.x - d.dx}px`;
    d.ghost!.style.top = `${d.y - d.dy}px`;
    const y = d.y + this.#scrolled(d);
    const to = d.rects.filter((r, k) => k !== d.from && r.top + r.height / 2 < y).length;
    if (to === d.to) return;
    d.to = to;
    const stride = d.rects[d.from].height + d.gap;
    d.items.forEach((el, k) => {
      const shift = k === d.from ? this.#offset(d, to) : d.from < k && k <= to ? -stride : to <= k && k < d.from ? stride : 0;
      el.style.transform = shift ? `translateY(${shift}px)` : '';
    });
  }

  #inside(d: Drag) {
    const top = Math.min(...d.rects.map((r) => r.top)) - EDGE - this.#scrolled(d);
    const bottom = Math.max(...d.rects.map((r) => r.bottom)) + EDGE - this.#scrolled(d);
    const left = Math.min(...d.rects.map((r) => r.left)) - EDGE;
    const right = Math.max(...d.rects.map((r) => r.right)) + EDGE;
    return d.x >= left && d.x <= right && d.y >= top && d.y <= bottom;
  }

  #settle(drop: boolean) {
    const d = this.#d!;
    cancelAnimationFrame(d.frame);
    this.#busy = true;
    this.#release();
    const to = drop ? d.to : d.from;
    if (!drop) d.items.forEach((el) => (el.style.transform = ''));
    const r = d.rects[d.from];
    const ms = reduced() ? 0 : SETTLE;
    Object.assign(d.ghost!.style, {
      transition: `left ${ms}ms ease, top ${ms}ms ease, transform ${ms}ms ease`,
      left: `${r.left}px`,
      top: `${r.top + this.#offset(d, to) - this.#scrolled(d)}px`
    });
    d.ghost!.setAttribute('data-settling', '');
    const done = () => {
      d.ghost!.remove();
      for (const el of d.items) {
        el.style.transition = 'none';
        el.style.transform = '';
      }
      d.row.removeAttribute('data-reorder-source');
      this.#opts.box()?.removeAttribute('data-reorder-active');
      void d.row.offsetHeight;
      for (const el of d.items) el.style.transition = '';
      this.#busy = false;
      if (to !== d.from) this.#move(d.from, to);
    };
    if (ms) setTimeout(done, ms);
    else done();
  }

  #release() {
    const d = this.#d;
    if (!d) return;
    try {
      if (d.row.hasPointerCapture(d.pointer)) d.row.releasePointerCapture(d.pointer);
    } catch {
      // empty
    }
    removeEventListener('pointermove', this.#onmove);
    removeEventListener('pointerup', this.#onup);
    removeEventListener('pointercancel', this.#oncancel);
    removeEventListener('keydown', this.#onkey, true);
    this.#d = null;
  }

  #key(i: number, e: KeyboardEvent) {
    if (e.target !== e.currentTarget) return;
    const n = this.#items().length;
    const name = this.#opts.name(i);
    if (e.key === 'Enter' && this.s.grabbed !== i && this.#opts.onactivate) {
      e.preventDefault();
      this.#opts.onactivate(i);
    } else if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      if (this.s.grabbed === i) {
        this.s.grabbed = -1;
        this.s.message = `Dropped ${name} at position ${i + 1} of ${n}`;
      } else {
        this.s.grabbed = this.#origin = i;
        this.s.message = `Grabbed ${name}, position ${i + 1} of ${n}. Up and down arrows move it, Space drops, Escape cancels`;
      }
    } else if (e.key === 'Escape' && this.s.grabbed === i) {
      e.preventDefault();
      const back = this.#origin;
      this.s.grabbed = -1;
      if (back !== i) this.#opts.onmove(i, back);
      this.s.message = `Cancelled; ${name} back at position ${back + 1} of ${n}`;
      void this.#focus(back);
    } else if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') && (e.altKey || this.s.grabbed === i)) {
      e.preventDefault();
      const to = i + (e.key === 'ArrowUp' ? -1 : 1);
      if (to < 0 || to >= n) return;
      if (this.s.grabbed === i) this.s.grabbed = to;
      this.#move(i, to);
    }
  }

  #move(from: number, to: number) {
    const name = this.#opts.name(from);
    const n = this.#items().length;
    this.#opts.onmove(from, to);
    this.s.message = `Moved ${name} to position ${to + 1} of ${n}`;
    void this.#focus(to);
  }

  async #focus(i: number) {
    await tick();
    this.#items()[i]?.focus();
  }
}
