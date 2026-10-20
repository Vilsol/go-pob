<script lang="ts">
  import Button from './Button.svelte';

  type Props = { message: string; label?: string; confirm?: string; onconfirm: () => void; oncancel: () => void };
  let { message, label = 'Confirm delete', confirm = 'Delete', onconfirm, oncancel }: Props = $props();
</script>

<div class="flex items-center gap-1.5 text-13" role="group" aria-label={label}>
  <span class="grow">{message}</span>
  <Button size="sm" variant="danger" onclick={onconfirm}>{confirm}</Button>
  <Button size="sm" onclick={oncancel}>Cancel</Button>
</div>
