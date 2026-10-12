<script lang="ts">
  // Main.lua:1300-1320
  import { applyUpdate, online, updateAvailable, updateChanges } from '$lib/pwa';

  let changes = $state(false);
</script>

<div class="fixed bottom-2 left-2 z-50 flex flex-col items-start gap-1">
  {#if !$online}
    <div role="status" class="rounded-md border border-line bg-surface-2 px-3 py-1 text-13 text-text-2 shadow-pop">
      Offline — saved builds and calculations work; importing from links needs the network
    </div>
  {/if}
  {#if $updateAvailable}
    <div role="alert" class="flex max-w-150 flex-col gap-1 rounded-md border border-line bg-surface-2 px-3 py-1 text-13 text-text-2 shadow-pop">
      {#if changes && $updateChanges.length}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <div id="pwa-changes" class="scroll flex max-h-[50vh] flex-col py-1 text-12" role="region" aria-label="Changes since this version" tabindex="0">
          {#each $updateChanges as v, i (i)}
            <span class={['font-semibold text-text', i > 0 && 'mt-3']}>Version {v.version} ({v.date})</span>
            {#each v.lines as l, j (j)}<span class={[l.sub && 'text-text-3']}>{l.text || '\u00a0'}</span>{/each}
          {/each}
        </div>
      {/if}
      <div class="flex items-center gap-2">
        <span>A new version of go-pob is available. Reloading discards unsaved changes.</span>
        <button type="button" class="btn btn-sm btn-primary" onclick={applyUpdate}>Reload</button>
        {#if $updateChanges.length}
          <button type="button" class="btn btn-sm" aria-expanded={changes} aria-controls="pwa-changes" onclick={() => (changes = !changes)}>Changes…</button>
        {/if}
      </div>
    </div>
  {/if}
</div>
