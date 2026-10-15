<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { title: string; count?: string; more?: string; class?: string; collapsed?: boolean; ontoggle?: () => void; children: Snippet };
  let { title, count, more, class: extra = '', collapsed = false, ontoggle, children }: Props = $props();
</script>

<section class={['option-group', extra]} aria-label={title}>
  <h3 class="option-head">
    <span>{title}</span>
    <span class="flex items-center gap-1.5"
      >{#if count}<span class="option-count tone-accent">{count}</span>{/if}{#if ontoggle}<button
          class="btn-bare px-1 font-mono text-13 text-text-2"
          aria-expanded={!collapsed}
          aria-label={`${collapsed ? 'Expand' : 'Collapse'} ${title}`}
          title={collapsed ? 'Expand' : 'Collapse'}
          onclick={ontoggle}>{collapsed ? '+' : '−'}</button
        >{/if}</span>
  </h3>
  {#if !collapsed}
    {@render children()}
    {#if more}<div class="option-more">{more}</div>{/if}
  {/if}
</section>
