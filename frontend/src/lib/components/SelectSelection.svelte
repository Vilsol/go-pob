<script lang="ts">
  import ColoredText from '$lib/components/common/ColoredText.svelte';

  let {
    item
  }: {
    // TODO Tooltip
    item: {
      label: string;
    };
  } = $props();
</script>

<div class="selection">
  <ColoredText text={item.label} />
</div>

<style>
  .selection {
    text-overflow: ellipsis;
    overflow-x: hidden;
    white-space: nowrap;
  }
</style>
