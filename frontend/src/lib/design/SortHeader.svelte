<script lang="ts">
  import type { ClassValue, HTMLAttributes } from 'svelte/elements';
  import Icon from './Icon.svelte';

  type Props = Omit<HTMLAttributes<HTMLSpanElement>, 'class' | 'title' | 'dir'> & {
    label: string;
    active: boolean;
    dir?: 'asc' | 'desc';
    align?: 'start' | 'end';
    title?: string;
    class?: ClassValue;
    buttonClass?: ClassValue;
    onsort: () => void;
  };
  let { label, active, dir = 'desc', align = 'start', title, class: extra = '', buttonClass = '', onsort, ...rest }: Props = $props();
</script>

<span
  role="columnheader"
  aria-sort={active ? (dir === 'asc' ? 'ascending' : 'descending') : 'none'}
  class={['flex min-w-0', align === 'end' && 'justify-end', extra]}
  {...rest}>
  <button
    class={['head-btn inline-flex items-center gap-1 whitespace-nowrap', align === 'end' && 'text-right', active && 'text-text', buttonClass]}
    {title}
    onclick={onsort}
    >{label}<span aria-hidden="true" class="inline-flex w-2.5 text-accent"
      >{#if active}<Icon name={dir === 'asc' ? 'chevron-up' : 'chevron-down'} size={10} />{/if}</span
    ></button>
</span>
