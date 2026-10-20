<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue, HTMLButtonAttributes } from 'svelte/elements';
  import Icon from './Icon.svelte';

  type Props = Omit<HTMLButtonAttributes, 'class' | 'children' | 'onclick'> & {
    open?: boolean;
    controls?: string;
    indicator?: 'chevron' | 'caret' | 'plusminus';
    class?: ClassValue;
    ontoggle?: (open: boolean) => void;
    children?: Snippet;
  };
  let { open = $bindable(false), controls, indicator = 'chevron', class: extra = '', ontoggle, children, ...rest }: Props = $props();
</script>

<button
  type="button"
  class={extra}
  aria-expanded={open}
  aria-controls={controls}
  onclick={() => {
    open = !open;
    ontoggle?.(open);
  }}
  {...rest}
  >{#if indicator === 'chevron'}<span class="inline-flex w-2.5 shrink-0" aria-hidden="true"
      ><Icon name={open ? 'chevron-down' : 'chevron-right'} size={10} /></span
    >{/if}{@render children?.()}{#if indicator === 'caret'}<span class="inline-flex" aria-hidden="true"
      ><Icon name={open ? 'chevron-up' : 'chevron-down'} size={10} /></span
    >{:else if indicator === 'plusminus'}<span class="font-mono" aria-hidden="true">{open ? '−' : '+'}</span>{/if}</button>
