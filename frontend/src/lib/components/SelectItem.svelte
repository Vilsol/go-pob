<script lang="ts">
  import ColoredText from '$lib/components/common/ColoredText.svelte';

  let {
    isActive = false,
    isHover = false,
    item
  }: {
    isActive?: boolean;
    isHover?: boolean;
    // TODO Tooltip
    item: {
      label: string;
    };
  } = $props();

  let itemClasses = $state('');

  $effect(() => {
    const classes = [];
    if (isActive) {
      classes.push('active');
    }
    if (isHover) {
      classes.push('hover');
    }
    itemClasses = classes.join(' ');
  });
</script>

<div class="item formatted {itemClasses}">
  <ColoredText text={item.label} />
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
