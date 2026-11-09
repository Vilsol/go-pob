<script lang="ts">
  import Button from './Button.svelte';
  import Spinner from './Spinner.svelte';
  import { dismissToast, toasts } from './toast';

  // ToastNotification.lua:166-257
  const uid = $props.id();
</script>

{#if $toasts.length}
  <div
    class="pointer-events-none fixed bottom-3 left-3 z-toast flex max-phone:bottom-[calc(3.5rem+env(safe-area-inset-bottom)+0.75rem)] w-80 max-w-[calc(100vw-24px)] flex-col gap-2">
    {#each $toasts as t, i (t.id)}
      {@const [title, ...body] = t.message.split('\n')}
      <div role="status" aria-labelledby="{uid}-{i}" class="panel pointer-events-auto flex flex-col gap-1 px-3 py-2.5 text-12 shadow-pop">
        <span class="flex items-center gap-2">
          {#if t.spinner}<Spinner />{/if}
          <span id="{uid}-{i}" class="text-13 font-semibold">{title}</span>
        </span>
        {#if body.length}<span class="whitespace-pre-line text-text-2">{body.join('\n')}</span>{/if}
        {#if t.progress !== undefined}
          <div
            class="h-1 overflow-hidden rounded-full bg-surface-3"
            role="progressbar"
            aria-label={title}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(t.progress * 100)}>
            <div class="h-full bg-accent" style:width="{Math.min(1, Math.max(0, t.progress)) * 100}%"></div>
          </div>
        {/if}
        <span class="flex justify-end gap-1">
          {#if t.action}{@const action = t.action}<Button size="sm" variant="ghost" onclick={() => action.run()}>{action.label}</Button>{/if}
          <Button size="sm" variant="ghost" onclick={() => dismissToast(t.id)}>Dismiss</Button>
        </span>
      </div>
    {/each}
  </div>
{/if}
