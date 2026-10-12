<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { kind: 'a' | 'b' | 'level' | 'same'; children: Snippet };
  let { kind, children }: Props = $props();

  const kinds = { a: 'tone-build-a', b: 'tone-build-b', level: 'tag-level', same: 'tag-same' };
</script>

<span class={['tag', kinds[kind]]}>{@render children()}</span>
