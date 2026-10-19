<script lang="ts">
  import Button from './Button.svelte';
  import { dismissToast, toasts } from './toast';

  // ToastNotification.lua:166-257
  const uid = $props.id();
</script>

{#if $toasts.length}
  <div
    class="pointer-events-none fixed bottom-3 left-3 z-toast flex max-phone:bottom-[calc(3.5rem+env(safe-area-inset-bottom)+0.75rem)] w-80 max-w-[calc(100vw-24px)] flex-col gap-2">
    {#each $toasts as t, i (t.id)}
      {@const [title, ...body] = t.message.split('\n')}
      <div role="status" aria-labelledby="{uid}-{i}" class="panel pointer-events-auto flex flex-col gap-1 px-3 py-2.5 text-12 shadow-lg">
        <span id="{uid}-{i}" class="text-13 font-semibold">{title}</span>
        {#if body.length}<span class="whitespace-pre-line text-text-2">{body.join('\n')}</span>{/if}
        <Button size="sm" variant="ghost" class="self-end" onclick={() => dismissToast(t.id)}>Dismiss</Button>
      </div>
    {/each}
  </div>
{/if}
