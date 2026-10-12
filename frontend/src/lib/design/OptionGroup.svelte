<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { title: string; count?: string; more?: string; class?: string; children: Snippet };
  let { title, count, more, class: extra = '', children }: Props = $props();
</script>

<section class={['option-group', extra]} aria-label={title}>
  <h3 class="option-head">
    <span>{title}</span>{#if count}<span class="option-count tone-accent">{count}</span>{/if}
  </h3>
  {@render children()}
  {#if more}<div class="option-more">{more}</div>{/if}
</section>
