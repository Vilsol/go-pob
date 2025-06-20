<script lang="ts">

  import { onMount, onDestroy } from 'svelte';
  import { Color } from '@tiptap/extension-color'
  import { Editor } from '@tiptap/core';
  import TextStyle from '@tiptap/extension-text-style'
  import StarterKit from '@tiptap/starter-kit';
  import { colorCodes } from '$lib/display/colors';

  let element: HTMLElement;
  let editor = $state<Editor>();

  const textColorCodes: Record<string, string> = {
    NORMAL: colorCodes.NORMAL,
    FIRE: colorCodes.FIRE,
    STRENGTH: colorCodes.STRENGTH,
    MAGIC: colorCodes.MAGIC,
    COLD: colorCodes.COLD,
    DEXTERITY: colorCodes.DEXTERITY,
    RARE: colorCodes.RARE,
    LIGHTNING: colorCodes.LIGHTNING,
    INTELLIGENCE: colorCodes.INTELLIGENCE,
    UNIQUE: colorCodes.UNIQUE,
    CHAOS: colorCodes.CHAOS,
    DEFAULT: colorCodes.WHITE
  };

  onMount(() => {
    editor = new Editor({
      autofocus: true,
      element: element,
      extensions: [StarterKit, Color, TextStyle],
      content: '',
      editorProps: {
        attributes: {
          class: 'prose max-w-none prose-sm sm:prose-base lg:prose-lg xl:prose-2xl m-2 focus:outline-none text-white w-full'
        }
      },
      onTransaction: () => {}
    });
  });

  onDestroy(() => {
    if (editor) {
      editor.destroy();
    }
  });

  const toggleTextColor = (color: string) => {
    return () => {
      editor?.chain().focus().setColor(color).run()
    }
  }

  function chunkArray<T>(arr: T[], columns: number): T[][] {
    return Array.from({ length: Math.ceil(arr.length / columns) }, (_, i) =>
      arr.slice(i * columns, i * columns + columns)
    );
  }

  const chunkedEntries = chunkArray(Object.entries(textColorCodes), 3);

</script>

{#if editor}
  <div class="button-group flex flex-row justify-between text-sm p-3 border-b-2 border-white">   
    <div class="color-buttons flex flex-row w-1/3 gap-4">
      {#each chunkedEntries as row}
        <div class="grid w-full">
            {#each row as [colorName, colorHexValue]}
                <button class="w-full border border-gray-400" style="color: {colorHexValue}" onclick={toggleTextColor(colorHexValue)} data-testid="setPurple">{colorName}</button>
            {/each}
        </div>
      {/each}
    </div> 
    <!-- TODO: Implement behavior of "Show Color Codes" button -->
    <div class="display-color-code-button content-center w-1/6">
      <button class="w-full border border-gray-400">Show Color Codes</button>
    </div>
  </div>
{/if}

<div class="div-editor w-full h-full" bind:this={element}></div>

<style>
</style>
