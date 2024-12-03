<script lang="ts">
  import { fontScaling } from '$lib/global';
  import type { Node } from '../../skill_tree/types';

  interface NodeSelectionOptionsProps {
    node: Node;
    onSelectOption: (node: Node, optionIdex: number) => void;
  }

  let {
    node,
    onSelectOption,
    onclose
  }: {
    onclose: () => void;
  } & NodeSelectionOptionsProps = $props();

  type lineItem = {
    text: string;
  };

  const getLineOptions = (node: Node): lineItem[] => {
    const allLines: lineItem[] = [];

    node.masteryEffects?.forEach((effect) => {
      effect.stats.forEach((stat) => {
        stat.split('\n').forEach(() => {
          allLines.push({
            text: stat ?? 'N/A'
          });
        });
      });
    });
    return allLines;
  };

  const getOptionOnClick = (index: number) => {
    return clickSelectionOption(index);
  };

  let clickSelectionOption = (index: number) => (_: Event) => {
    if (!node) {
      console.warn('Attempted to select an option without underlying node available.');
      onclose();
    }
    onSelectOption(node, index);
    onclose();
  };

  let lineOptions = getLineOptions(node);
</script>

<div class="flex flex-col gap-4">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container"></legend>
    <div class="side-by-side-max-content w-full">
      <div class="flex flex-row gap-1">
        <ol class="options-list">
          {#each lineOptions as option, i}
            <li class="option_${i}" id="test" onclick={getOptionOnClick(i)}>{option.text}</li>
          {/each}
        </ol>
      </div>
    </div>
  </fieldset>

  <div class="flex flex-row items-center justify-center">
    <button class="container" onclick={onclose}>Close</button>
  </div>
</div>

<style>
  .options-list li:hover {
    width: 100%;
    border-radius: 0.2em;
    background-color: var(--color, hsla(225, 29%, 65%, 0.672));
  }

  .options-list {
    text-align: left;
  }
</style>
