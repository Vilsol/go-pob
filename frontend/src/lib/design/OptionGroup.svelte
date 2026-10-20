<script lang="ts">
  import Disclosure from './Disclosure.svelte';
  import type { Snippet } from 'svelte';

  type Props = { title: string; count?: string; more?: string; class?: string; collapsed?: boolean; ontoggle?: () => void; children: Snippet };
  let { title, count, more, class: extra = '', collapsed = false, ontoggle, children }: Props = $props();
</script>

<section class={['option-group', extra]} aria-label={title}>
  <h3 class="option-head">
    <span>{title}</span>
    <span class="flex items-center gap-1.5"
      >{#if count}<span class="option-count tone-accent">{count}</span>{/if}{#if ontoggle}<Disclosure
          class="btn-bare px-1 text-13 text-text-2"
          indicator="plusminus"
          open={!collapsed}
          aria-label={`${collapsed ? 'Expand' : 'Collapse'} ${title}`}
          title={collapsed ? 'Expand' : 'Collapse'}
          ontoggle={() => ontoggle()} />{/if}</span>
  </h3>
  {#if !collapsed}
    {@render children()}
    {#if more}<div class="option-more">{more}</div>{/if}
  {/if}
</section>
