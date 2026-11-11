<script lang="ts">
  import Icon from './Icon.svelte';
  import { GLYPH_ICONS, GLYPH_SPLIT } from './glyph_icons';

  let { text }: { text: string } = $props();
</script>

{#each text.split(GLYPH_SPLIT) as part, i (i)}{#if GLYPH_ICONS[part]}<span class="contents" data-glyph={part}
      ><Icon name={GLYPH_ICONS[part]} em={0.85} class="inline-block align-[-0.1em]" /></span
    >{:else}{part}{/if}{/each}
