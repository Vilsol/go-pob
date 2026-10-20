<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';
  import type { MenuItem } from './menu';
  import { tone as toneClass } from './tone';

  type Props = { items: MenuItem[]; itemClass?: ClassValue; row?: Snippet<[MenuItem, number]>; onpick: (it: MenuItem) => void };
  let { items, itemClass = '', row, onpick }: Props = $props();
</script>

{#each items as it, i (i)}
  {#if it.separator}<div role="separator" class="my-1 border-t border-line"></div>{/if}
  {#if it.caption}<span class="px-2 pt-1.5 pb-0.5 text-11 tracking-caps text-text-3">{it.caption}</span>{/if}
  <button
    role="menuitem"
    tabindex="-1"
    class={['menu-item', toneClass(it.tone), itemClass]}
    aria-current={it.current ? 'true' : undefined}
    data-wiki={it.wiki}
    disabled={it.disabled}
    onmouseenter={it.onhover && ((e) => it.onhover?.(e.currentTarget))}
    onmouseleave={it.onhover && (() => it.onhover?.(null))}
    onfocus={it.onhover && ((e) => it.onhover?.(e.currentTarget))}
    onblur={it.onhover && (() => it.onhover?.(null))}
    onclick={() => onpick(it)}
    >{#if row}{@render row(it, i)}{:else}{it.label}{#if it.note}<span class="note ml-auto">{it.note}</span>{/if}{/if}</button>
{/each}
