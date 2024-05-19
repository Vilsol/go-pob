<script lang="ts">
  import { formatColors } from '../display/colors';

  export let isActive = false;
  export const isFirst = false;
  export let isHover = false;
  export const isSelectable = false;
  export const getOptionLabel = undefined;
  export const filterText = '';

  // TODO Tooltip
  export let item: {
    label: string;
  };

  let itemClasses = '';

  $: {
    const classes = [];
    if (isActive) {
      classes.push('active');
    }
    if (isHover) {
      classes.push('hover');
    }
    itemClasses = classes.join(' ');
  }
</script>

<div class="item formatted {itemClasses}">
  {@html formatColors(item.label)}
</div>

<style lang="postcss">
  .item {
    cursor: default;
    padding: 0.25em 0.5em;
    color: #929292;
    text-overflow: ellipsis;
    overflow: hidden;
    white-space: nowrap;
  }

  .item:active {
    color: #ffffff;
  }

  .item:active {
    background: #000000;
    color: #ffffff;
  }

  :global(.formatted span) {
      position: relative;
      top: -0.12em;
  }
</style>
