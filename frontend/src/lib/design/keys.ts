const TYPING_TAGS = ['INPUT', 'TEXTAREA', 'SELECT'];

/** True when a key event's target takes text (or, with `widgets`, is inside a menu, listbox or combobox), so a page shortcut must not fire. */
export function isTypingTarget(t: EventTarget | null, opts: { widgets?: boolean } = {}): boolean {
  const el = t as HTMLElement | null;
  if (!el || typeof el.tagName !== 'string') return false;
  if (el.isContentEditable || TYPING_TAGS.includes(el.tagName)) return true;
  return !!opts.widgets && !!el.closest('[role="menu"], [role="listbox"], [role="combobox"]');
}

/** The index an arrow/Home/End key moves a roving focus to among `n` items, or undefined when the key does not move it. */
export function rovingIndex(
  e: KeyboardEvent,
  i: number,
  n: number,
  opts: { orientation?: 'horizontal' | 'vertical'; wrap?: boolean } = {}
): number | undefined {
  if (n <= 0) return undefined;
  const [next, prev] = opts.orientation === 'vertical' ? ['ArrowDown', 'ArrowUp'] : ['ArrowRight', 'ArrowLeft'];
  const wrap = opts.wrap ?? true;
  if (e.key === 'Home') return 0;
  if (e.key === 'End') return n - 1;
  if (e.key !== next && e.key !== prev) return undefined;
  if (i < 0) return e.key === next ? 0 : n - 1;
  const to = i + (e.key === next ? 1 : -1);
  return wrap ? (to + n) % n : Math.min(n - 1, Math.max(0, to));
}
