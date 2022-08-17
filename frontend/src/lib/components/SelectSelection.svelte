<script lang="ts">
  import { formatColors } from '../display/colors';

  export const getSelectionLabel: unknown = undefined;

  // TODO Tooltip
  export let item: {
    label: string;
  };
</script>

<div class="selection">
  {@html formatColors(item.label)}
</div>

<style>
  .selection {
    text-overflow: ellipsis;
    overflow-x: hidden;
    white-space: nowrap;
  }
</style>
