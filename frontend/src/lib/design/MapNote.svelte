<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { class?: string; children: Snippet };
  let { class: extra = 'top-1.5 right-1.5', children }: Props = $props();
</script>

<div class={['map-note', extra]}>{@render children()}</div>
