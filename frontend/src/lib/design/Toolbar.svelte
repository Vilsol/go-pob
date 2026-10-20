<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue, HTMLAttributes } from 'svelte/elements';
  import { isTypingTarget, rovingIndex } from './keys';

  type Props = Omit<HTMLAttributes<HTMLDivElement>, 'class'> & {
    label?: string;
    edge?: 'none' | 'top' | 'bottom';
    surface?: 'none' | 'bar' | 'surface-1';
    wrap?: boolean;
    class?: ClassValue;
    children: Snippet;
  };
  let { label, edge = 'none', surface = 'none', wrap = false, class: extra = '', children, ...rest }: Props = $props();
  const edges = { none: '', top: 'border-t border-line', bottom: 'border-b border-line' };
  const surfaces = { none: '', bar: 'bg-bar', 'surface-1': 'bg-surface-1' };
  const NESTED = '[role="dialog"], [role="menu"], [role="listbox"], [popover]';

  // ArrowLeft/ArrowRight/Home/End move focus between the toolbar's own buttons
  const onkeydown = (e: KeyboardEvent) => {
    const bar = e.currentTarget as HTMLElement;
    const from = e.target as HTMLElement;
    if (isTypingTarget(from, { widgets: true }) || from.closest(NESTED)?.closest('[role="toolbar"]') === bar) return;
    const items = [...bar.querySelectorAll<HTMLElement>('button:not(:disabled), a[href]')].filter((el) => !el.closest(NESTED) && !el.closest('[hidden]'));
    const to = rovingIndex(e, items.indexOf(from), items.length, { wrap: false });
    if (to === undefined) return;
    e.preventDefault();
    items[to].focus();
  };
</script>

<div
  role={label ? 'toolbar' : undefined}
  aria-label={label}
  class={['flex items-center', wrap && 'flex-wrap', edges[edge], surfaces[surface], extra]}
  onkeydown={label ? onkeydown : undefined}
  {...rest}>
  {@render children()}
</div>
