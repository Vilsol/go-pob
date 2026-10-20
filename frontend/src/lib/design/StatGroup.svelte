<script lang="ts">
  import Disclosure from './Disclosure.svelte';
  import Badge from './Badge.svelte';
  import type { Snippet } from 'svelte';

  type Props = { title: string; warn?: string; open?: boolean; children: Snippet };
  let { title, warn, open = $bindable(true), children }: Props = $props();
  const id = $props.id();
</script>

<div class="flex flex-col">
  <Disclosure class="stat-group-head" bind:open controls={id}
    >{title}
    {#if warn}<Badge tone="warn" class="tracking-normal normal-case">⚠ {warn}</Badge>{/if}
  </Disclosure>
  <div {id} class="flex flex-col" hidden={!open}>{@render children()}</div>
</div>
