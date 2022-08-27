<script lang="ts">
  import { colorCodes, formatColors } from '../../display/colors';

  type Section = {
    large?: boolean;
    color?: string;
    italic?: boolean;
    text: string[];
  };

  export let sections: Section[] = [];
  export let color = colorCodes.WHITE;
</script>

<div class="grid grid-flow-row min-w-[300px] max-w-[25vw] w-fit h-fit border-2" style="border-color: {color}">
  {#each sections as section}
    <div
      class="grid grid-flow-row section p-1.5 text-center"
      class:text-xl={section.large}
      class:italic={section.italic}
      style="--border-color: {color}; color: {section.color || '#ffffff'}">
      {#each section.text as line}
        <div>
          {@html formatColors(line)}
        </div>
      {/each}
    </div>
  {/each}
</div>

<style lang="postcss">
  .section:not(:last-child) {
    border-bottom: 1px solid var(--border-color, #ffffff);
  }
</style>
