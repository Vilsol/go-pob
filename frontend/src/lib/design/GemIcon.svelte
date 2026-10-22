<script lang="ts">
  import type { ClassValue } from 'svelte/elements';
  import ItemIcon from './ItemIcon.svelte';

  type Props = { name: string; size?: number; decorative?: boolean; src?: string; class?: ClassValue };
  let { name, size = 24, decorative = false, src, class: extra = '' }: Props = $props();
</script>

<ItemIcon {name} gem {size} {decorative} {src} class={extra} />
