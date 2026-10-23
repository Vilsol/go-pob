<script lang="ts">
  import type { ClassValue } from 'svelte/elements';
  import ItemIcon from './ItemIcon.svelte';

  type Props = { name: string; size?: number; decorative?: boolean; src?: string; lastFrame?: boolean; class?: ClassValue };
  let { name, size = 24, decorative = false, src, lastFrame = false, class: extra = '' }: Props = $props();
</script>

<ItemIcon {name} gem {size} {decorative} {src} {lastFrame} class={extra} />
