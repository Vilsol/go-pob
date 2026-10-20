<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue, HTMLAttributes } from 'svelte/elements';
  import { besidePoint, type Point } from './floating';
  import { besideAnchor } from './items';

  type Props = Omit<HTMLAttributes<HTMLDivElement>, 'class'> & {
    /** float beside this element (pass `anchor` even while it is still undefined) */
    anchor?: HTMLElement;
    /** float beside this point, in the parent element's box */
    at?: Point;
    placement?: 'beside' | 'point' | 'inline';
    size?: 'none' | 'sm' | 'md' | 'lg';
    interactive?: boolean;
    /** no shadow, for a tooltip drawn in the page flow */
    flat?: boolean;
    textSize?: 11 | 12;
    label?: string;
    class?: ClassValue;
    children: Snippet;
  };
  let {
    anchor,
    at,
    placement = 'inline',
    size = 'md',
    interactive = false,
    flat = false,
    textSize = 12,
    label,
    class: extra = '',
    children,
    ...rest
  }: Props = $props();
  const pads = { none: '', sm: 'px-2.5 py-1.5', md: 'px-3 py-2', lg: 'px-3 py-2.5' };
  const shell = $derived([
    'tooltip rounded-md',
    textSize === 11 ? 'text-11' : 'text-12',
    !flat && 'shadow-tip',
    pads[size],
    !interactive && 'pointer-events-none',
    placement !== 'inline' && 'fixed top-(--y) left-(--x) z-float',
    extra
  ]);
</script>

{#if placement === 'beside'}
  <div role="tooltip" aria-label={label} class={shell} use:besideAnchor={anchor} {...rest}>{@render children()}</div>
{:else if placement === 'point'}
  <div role="tooltip" aria-label={label} class={shell} use:besidePoint={at} {...rest}>{@render children()}</div>
{:else}
  <div role="tooltip" aria-label={label} class={shell} {...rest}>{@render children()}</div>
{/if}
