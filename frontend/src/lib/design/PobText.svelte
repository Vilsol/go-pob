<script lang="ts">
  import { notSupportedNote, pobSegments } from './pob_text';

  type Props = { text: string; plain?: boolean };
  let { text, plain = false }: Props = $props();

  const segs = $derived(pobSegments(text, $notSupportedNote));
</script>

{#if plain}{segs.map((s) => s.text).join('')}{:else}{#each segs as s, i (i)}{#if s.color}<span data-pob-color class="toned" style:--tone={s.color}
        >{s.text}</span
      >{:else}{s.text}{/if}{/each}{/if}
