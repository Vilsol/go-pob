<script lang="ts">
  import Editor from '@tinymce/tinymce-svelte';
  import { onMount } from 'svelte';

  const config = {
    skin: 'oxide-dark',
    content_css: 'dark',
    resize: false,
    height: 200,
    plugins: ['code', 'table', 'image', 'lists', 'help', 'link'],
    menubar: 'edit insert view format table help',
    toolbar:
      'undo redo | styles | bold italic underline strikethrough subscript superscript | link image table | aligncenter alignjustify alignleft alignright | backcolor forecolor | fontsize | numlist bullist | hr | removeformat'
  };

  let parentElement: HTMLElement;

  let debounceTimer;
  const resized = () => {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    debounceTimer = setTimeout(() => {
      config.height = parentElement.offsetHeight;
    }, 50);
  };

  onMount(() => {
    resized();
  });

  let value = '';
</script>

<svelte:window on:resize={resized} />

<div class="w-full h-full" bind:this={parentElement}>
  {#key config.height}
    <Editor scriptSrc="tinymce/tinymce.min.js" conf={config} bind:value />
  {/key}
</div>
