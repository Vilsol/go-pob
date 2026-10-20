<script lang="ts">
  import Button from '../Button.svelte';
  type Props = { text: string; onundo: () => void };
  let { text, onundo }: Props = $props();
</script>

<div
  class="absolute inset-x-2 bottom-2 z-toast flex items-center gap-2 rounded-md border border-line-strong bg-surface-2 py-1 pr-1 pl-3 shadow-dlg"
  role="status">
  <span class="min-w-0 flex-1 truncate text-13">{text}</span>
  <Button type="button" class="h-10 px-3" onclick={onundo}>Undo</Button>
</div>
