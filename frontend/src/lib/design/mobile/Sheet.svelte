<script lang="ts">
  import type { Snippet } from 'svelte';
  import Dialog from '../Dialog.svelte';

  type Props = { open?: boolean; title: string; note?: string; full?: boolean; head?: Snippet; children: Snippet; onclose?: () => void };
  let { open = $bindable(false), title, note, full = false, head, children, onclose }: Props = $props();
</script>

<Dialog
  bind:open
  sheet
  backdrop
  {title}
  {note}
  {head}
  {onclose}
  class={[
    'w-full max-w-none',
    full ? 'm-0 h-dvh max-h-none rounded-none' : 'sheet-bottom mx-0 mt-auto mb-0 max-h-[85dvh] rounded-b-none pb-[env(safe-area-inset-bottom)]'
  ].join(' ')}>
  <div class="scroll flex min-h-0 flex-1 flex-col">{@render children()}</div>
</Dialog>
