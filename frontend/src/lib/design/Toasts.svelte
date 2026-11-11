<script lang="ts">
  import ActivityIndicator from './ActivityIndicator.svelte';
  import Button from './Button.svelte';
  import Spinner from './Spinner.svelte';
  import { dismissToast, toasts } from './toast';

  // ToastNotification.lua:166-257
  // every notice the app raises stacks here, in the bottom right corner; with activity, the activity indicator sits
  // under them, for a page without a top bar
  let { activity = false }: { activity?: boolean } = $props();
  const uid = $props.id();
  let expanded = $state<Record<string, boolean>>({});
</script>

{#if $toasts.length || activity}
  <div
    class="pointer-events-none fixed right-3 bottom-3 z-toast flex w-80 max-w-[calc(100vw-24px)] flex-col items-end gap-2 max-phone:bottom-[calc(3.5rem+env(safe-area-inset-bottom)+0.75rem)]">
    <!-- on a phone the page fills the screen, so a toast sits over the top bar and lets taps through to it, except on its own buttons -->
    {#if $toasts.length}
      <div
        class="flex w-full flex-col gap-2 max-phone:fixed max-phone:top-[calc(env(safe-area-inset-top)+0.25rem)] max-phone:right-3 max-phone:left-3 max-phone:w-auto">
        {#each $toasts as t, i (t.id)}
          {@const [title, ...body] = t.message.split('\n')}
          {@const open = !!(t.more?.lines.length && expanded[t.id])}
          <div
            role={t.role ?? 'status'}
            aria-labelledby="{uid}-{i}"
            class="panel pointer-events-auto flex flex-col gap-1 px-3 py-2.5 text-12 shadow-pop max-phone:pointer-events-none">
            <span class="flex items-center gap-2">
              {#if t.spinner}<Spinner />{/if}
              <span id="{uid}-{i}" class={['text-13 font-semibold', t.tone === 'error' && 'text-error']}>{title}</span>
            </span>
            {#if body.length}<span class="whitespace-pre-line text-text-2 [overflow-wrap:anywhere]">{body.join('\n')}</span>{/if}
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
            {#if open && t.more}
              <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
              <div
                id="{uid}-{i}-more"
                class="scroll pointer-events-auto flex max-h-[50vh] flex-col py-1 text-text-2"
                role="region"
                aria-label={t.more.name}
                tabindex="0">
                {#each t.more.lines as l, j (j)}
                  <span class={[l.head && ['font-semibold text-text', j > 0 && 'mt-3'], l.sub && 'text-text-3']}>{l.text || ' '}</span>
                {/each}
              </div>
            {/if}
            <span class="pointer-events-auto flex flex-wrap justify-end gap-1 self-end">
              {#each t.actions ?? [] as a (a.label)}
                <Button size="sm" variant={a.primary ? 'primary' : 'ghost'} onclick={() => a.run()}>{a.label}</Button>
              {/each}
              {#if t.more?.lines.length}
                <Button size="sm" variant="ghost" aria-expanded={open} aria-controls="{uid}-{i}-more" onclick={() => (expanded[t.id] = !open)}
                  >{t.more.label}</Button>
              {/if}
              {#if t.dismiss !== false}<Button size="sm" variant="ghost" onclick={() => dismissToast(t.id)}>{t.dismiss ?? 'Dismiss'}</Button>{/if}
            </span>
          </div>
        {/each}
      </div>
    {/if}
    {#if activity}<ActivityIndicator floating />{/if}
  </div>
{/if}
