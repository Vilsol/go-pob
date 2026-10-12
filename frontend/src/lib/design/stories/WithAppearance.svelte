<script lang="ts">
  import type { Snippet } from 'svelte';
  import { rootFontSize } from '$lib/display/appearance';

  type Props = { theme?: 'dark' | 'light' | 'contrast'; pt?: number; children: Snippet };
  let { theme = 'dark', pt = 12, children }: Props = $props();

  $effect(() => {
    const r = document.documentElement;
    r.dataset.theme = theme;
    r.style.colorScheme = theme === 'light' ? 'light' : 'dark';
    r.style.fontSize = rootFontSize(pt);
    return () => {
      delete r.dataset.theme;
      r.style.colorScheme = '';
      r.style.fontSize = '';
    };
  });
</script>

{@render children()}
