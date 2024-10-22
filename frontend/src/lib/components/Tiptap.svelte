<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Editor } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';

  let element: HTMLElement;
  let editor = $state<Editor>();

  let {
    value = $bindable()
  }: {
    value: string;
  } = $props();

  onMount(() => {
    editor = new Editor({
      element: element,
      extensions: [StarterKit],
      content: value,
      editorProps: {
        attributes: {
          class: 'prose prose-sm sm:prose-base lg:prose-lg xl:prose-2xl m-5 focus:outline-none'
        }
      },
      onTransaction: () => {
        value = editor?.getHTML() || '';
      }
    });
  });

  onDestroy(() => {
    if (editor) {
      editor.destroy();
    }
  });
</script>

{#if editor}
  <button onclick={() => editor?.chain().focus().toggleHeading({ level: 1 }).run()} class:active={editor.isActive('heading', { level: 1 })}>H1</button>
  <button onclick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()} class:active={editor.isActive('heading', { level: 2 })}>H2</button>
  <button onclick={() => editor?.chain().focus().setParagraph().run()} class:active={editor.isActive('paragraph')}>P</button>
{/if}

<div bind:this={element}></div>

<style>
  button.active {
    background: black;
    color: white;
  }
</style>
