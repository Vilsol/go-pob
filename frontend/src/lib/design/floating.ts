import { autoUpdate, computePosition, flip, offset, shift, size, type Middleware, type Placement, type VirtualElement } from '@floating-ui/dom';

export type Point = { x: number; y: number };
export type FloatOpts = {
  placement?: Placement;
  fallback?: Placement[];
  /** distance from the anchor along the placement side */
  gap?: number;
  /** offset along the anchor edge (a menu aligned to its trigger's right edge) */
  along?: number;
  /** keep this far inside the viewport */
  pad?: number;
  /** cap the panel's height to the room on the side it lands on, scrolling the rest */
  fit?: boolean;
  /** at least as wide as the anchor (capped to the viewport) */
  matchWidth?: boolean;
  /** write --x/--y for `top-(--y) left-(--x)` classes instead of inline left/top */
  vars?: boolean;
};

const virtual = (p: Point): VirtualElement => ({
  getBoundingClientRect: () => ({ x: p.x, y: p.y, left: p.x, top: p.y, right: p.x, bottom: p.y, width: 0, height: 0 })
});

/** Positions `panel` (position: fixed) beside `anchor` and keeps it there while either moves or resizes; returns the stop function. */
export function float(panel: HTMLElement, anchor: Element | Point, opts: FloatOpts = {}): () => void {
  const { placement = 'bottom-start', fallback, gap = 0, along = 0, pad = 8, fit = false, matchWidth = false, vars = false } = opts;
  const ref = anchor instanceof Element ? anchor : virtual(anchor);
  const middleware: Middleware[] = [offset({ mainAxis: gap, crossAxis: along }), flip({ padding: pad, crossAxis: false, fallbackPlacements: fallback })];
  // size before shift, so a capped panel is not slid over its anchor instead
  if (fit || matchWidth)
    middleware.push(
      size({
        padding: pad,
        apply({ availableHeight, rects }) {
          if (fit && availableHeight < rects.floating.height) Object.assign(panel.style, { maxHeight: `${Math.max(0, availableHeight)}px`, overflowY: 'auto' });
          if (matchWidth) panel.style.minWidth = `${Math.min(rects.reference.width, innerWidth - 2 * pad)}px`;
        }
      })
    );
  middleware.push(shift({ padding: pad, crossAxis: true }));
  let live = true;
  const update = () => {
    if (fit) panel.style.maxHeight = '';
    void computePosition(ref, panel, { placement, strategy: 'fixed', middleware }).then(({ x, y }) => {
      if (!live) return;
      if (vars) {
        panel.style.setProperty('--x', `${x}px`);
        panel.style.setProperty('--y', `${y}px`);
      } else Object.assign(panel.style, { left: `${x}px`, top: `${y}px` });
    });
  };
  const stop = autoUpdate(ref, panel, update);
  return () => {
    live = false;
    stop();
  };
}

/** Lifts `panel` into the top layer as a manual popover, above modal dialogs and clear of ancestor overflow and z-index. Without the Popover API it stays in place and keeps its z-index class. */
export function raise(panel: HTMLElement) {
  if (typeof panel.showPopover !== 'function' || !panel.isConnected) return;
  panel.setAttribute('popover', 'manual');
  try {
    panel.showPopover();
  } catch {
    panel.removeAttribute('popover');
  }
}

/** True when a press landed outside every given element. */
export const pressedOutside = (e: Event, ...roots: (Element | null | undefined)[]) => {
  const t = e.target;
  return t instanceof Node && roots.every((r) => !r?.contains(t));
};

/** Action: floats `tip` beside a point given in its parent's box (map and canvas overlays), in the top layer; flips to the left near the edge. */
export function besidePoint(tip: HTMLElement, at: Point | undefined) {
  const origin = tip.parentElement;
  raise(tip);
  let stop: (() => void) | undefined;
  const place = (p: Point | undefined) => {
    stop?.();
    if (!p || !origin) return;
    const r = origin.getBoundingClientRect();
    stop = float(tip, { x: r.left + p.x, y: r.top + p.y }, { placement: 'right-start', fallback: ['left-start'], gap: 18, along: -30, vars: true });
  };
  place(at);
  return { update: place, destroy: () => stop?.() };
}
