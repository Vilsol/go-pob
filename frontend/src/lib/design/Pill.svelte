<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { winner?: 'a' | 'b'; children: Snippet };
  let { winner, children }: Props = $props();

  const tones = { a: 'tone-build-a', b: 'tone-build-b' };
</script>

<span class={['pill', winner ? tones[winner] : 'pill-even']}>{@render children()}</span>
