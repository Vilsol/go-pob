<script lang="ts">
  import Tooltip from './Tooltip.svelte';
  import PobText from './PobText.svelte';

  // ItemsTab.lua:4138-4147
  type Props = { anchor: HTMLElement | undefined; id: string; title: string; lines: string[] };
  let { anchor, id, title, lines }: Props = $props();
</script>

<Tooltip {id} label="Item set {title}" placement="beside" {anchor} class="flex flex-col gap-0.5">
  {#each lines as l, i (i)}<span><PobText text={l} /></span>{:else}<span class="text-text-3">No items</span>{/each}
</Tooltip>
