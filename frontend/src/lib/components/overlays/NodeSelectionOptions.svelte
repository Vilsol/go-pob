<script lang="ts">
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

  const getLineOptions = (nodeToDescribe: Node): lineItem[] => {
    const allLines: lineItem[] = [];

    nodeToDescribe.masteryEffects?.forEach((effect) => {
      allLines.push({
        text: effect.stats.join('\n') ?? 'N/A'
      });
    });
    return allLines;
  };

  let selectionOption = (index: number) => () => {
    if (!node) {
      console.warn('Attempted to select an option without underlying node available.');
      onclose();
    }
    onSelectOption(node, index);
    onclose();
  };

  const getOptionOnSelect = (index: number) => selectionOption(index);
  let lineOptions = getLineOptions(node);
</script>

<div class="flex flex-col gap-2">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container">{node.name}</legend>
    <div class="container">
      <div class="flex flex-row gap-1">
        <ol class="options-list">
          {#each lineOptions as option, i}
            <li role="menuitem" id="option_${i}" onclick={getOptionOnSelect(i)} onkeydown={getOptionOnSelect(i)}>{option.text}</li>
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
    white-space: pre-wrap;
  }

  .options-list li {
    margin: 5px;
  }
</style>
