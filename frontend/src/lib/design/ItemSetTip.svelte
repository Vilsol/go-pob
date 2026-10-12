<script lang="ts">
  import PobText from './PobText.svelte';
  import { besideAnchor } from './items';

  // ItemsTab.lua:4138-4147
  type Props = { anchor: HTMLElement | undefined; id: string; title: string; lines: string[] };
  let { anchor, id, title, lines }: Props = $props();
</script>

<div
  role="tooltip"
  {id}
  aria-label="Item set {title}"
  class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-30 flex flex-col gap-0.5 rounded-md px-3 py-2 text-12 shadow-tip"
  use:besideAnchor={anchor}>
  {#each lines as l, i (i)}<span><PobText text={l} /></span>{:else}<span class="text-text-3">No items</span>{/each}
</div>
