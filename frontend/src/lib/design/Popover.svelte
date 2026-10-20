<script lang="ts">
  import type { Placement } from '@floating-ui/dom';
  import type { Snippet } from 'svelte';
  import type { ClassValue, HTMLAttributes } from 'svelte/elements';
  import { float, pressedOutside, raise } from './floating';

  type Props = Omit<HTMLAttributes<HTMLDivElement>, 'class' | 'role'> & {
    open: boolean;
    /** the trigger it opens under; a press on it is not a press outside */
    anchor: HTMLElement | undefined;
    role?: 'dialog' | 'group';
    label?: string;
    placement?: Placement;
    gap?: number;
    /** Escape, or a press outside the panel and its trigger */
    onclose: (via: 'escape' | 'outside') => void;
    /** Escape returns focus to the trigger */
    refocus?: boolean;
    class?: ClassValue;
    children: Snippet;
  };
  let { open, anchor, role, label, placement = 'bottom-start', gap = 6, onclose, refocus = true, class: extra = '', children, ...rest }: Props = $props();
  let panel = $state<HTMLElement>();

  const under = (node: HTMLElement, to: HTMLElement | undefined) => {
    raise(node);
    let stop = to ? float(node, to, { placement, gap, vars: true }) : undefined;
    return {
      update(next: HTMLElement | undefined) {
        stop?.();
        stop = next ? float(node, next, { placement, gap, vars: true }) : undefined;
      },
      destroy: () => stop?.()
    };
  };
</script>

<svelte:window onpointerdown={(e: PointerEvent) => open && pressedOutside(e, panel, anchor) && onclose('outside')} />

{#if open}
  <div
    bind:this={panel}
    {role}
    aria-label={label}
    tabindex="-1"
    class={['fixed top-(--y) left-(--x) z-dropdown box-border rounded-md border border-line-strong bg-surface-2 shadow-pop', extra]}
    use:under={anchor}
    onkeydown={(e) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      e.stopPropagation();
      onclose('escape');
      if (!refocus) return;
      if (anchor?.matches('button')) anchor.focus();
      else anchor?.querySelector('button')?.focus();
    }}
    {...rest}>
    {@render children()}
  </div>
{/if}
