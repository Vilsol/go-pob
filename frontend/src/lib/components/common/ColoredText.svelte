<script lang="ts">
  import { formatColors } from '$lib/display/colors';

  let {
    text
  }: {
    text: string;
  } = $props();
</script>

<!-- eslint-disable-next-line svelte/no-at-html-tags -->
{@html formatColors(text)}
