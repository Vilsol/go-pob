<script lang="ts">
  import Tiptap from '$lib/components/Tiptap.svelte';

  let value = $state('<p>Hello World! 🌍️ </p>');
</script>

<div class="w-full h-full">
  <Tiptap bind:value />
</div>
