<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { title: string; warn?: string; open?: boolean; children: Snippet };
  let { title, warn, open = $bindable(true), children }: Props = $props();
  const id = $props.id();
</script>

<div class="flex flex-col">
  <button class="stat-group-head" aria-expanded={open} aria-controls={id} onclick={() => (open = !open)}>
    <span class="w-2.5" aria-hidden="true">{open ? '▾' : '▸'}</span>{title}
    {#if warn}<span class="badge-warn">⚠ {warn}</span>{/if}
  </button>
  <div {id} class="flex flex-col" hidden={!open}>{@render children()}</div>
</div>
