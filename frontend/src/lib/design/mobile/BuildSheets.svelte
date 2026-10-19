<script lang="ts" module>
  export type BuildSheet = 'history' | 'loadout' | 'issues';
</script>

<script lang="ts">
  import type { BuildWarning } from '$lib/display/sidebar';
  import IssueRow from '../IssueRow.svelte';
  import { issueKinds } from '../issue_kinds';
  import Sheet from './Sheet.svelte';

  // the TopBar's undo history, loadout picker and build issues, as phone sheets
  type Props = {
    open?: BuildSheet;
    history: { action: string; page: string; when: string }[];
    undone: number;
    loadouts: string[];
    loadout?: string;
    issues: BuildWarning[];
    onhistory?: (i: number) => void;
    onloadout?: (name: string) => void;
  };
  let { open = $bindable(), history, undone, loadouts, loadout, issues, onhistory, onloadout }: Props = $props();

  const close = (which: BuildSheet, v: boolean) => {
    if (!v && open === which) open = undefined;
  };
  const row = 'flex min-h-12 w-full items-center gap-2 border-0 border-b border-line bg-transparent px-4 py-1 text-left font-sans text-14 text-text';
</script>

<Sheet bind:open={() => open === 'history', (v: boolean) => close('history', v)} title="Undo history" note="Tap a step to go back to it">
  {#if history.length}
    <ul class="m-0 flex list-none flex-col p-0" aria-label="Undo history">
      {#each history as h, i (i)}
        <li>
          <button
            type="button"
            class={row}
            aria-current={i === undone ? 'step' : undefined}
            onclick={() => {
              open = undefined;
              onhistory?.(i);
            }}>
            <span class="w-3 shrink-0 text-accent" aria-hidden="true">{i === undone ? '●' : ''}</span>
            <span class={['min-w-0 flex-1 truncate', i < undone && 'line-through']}>{h.action}</span>
            <span class="shrink-0 text-11 text-text-3">{h.page} · {h.when}</span>
          </button>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="note px-4 py-3">Nothing to undo yet.</p>
  {/if}
</Sheet>

<Sheet bind:open={() => open === 'loadout', (v: boolean) => close('loadout', v)} title="Loadout" note="Switches tree, skill, item and config sets together">
  <ul class="m-0 flex list-none flex-col p-0" aria-label="Loadouts">
    {#each loadouts as l (l)}
      <li>
        <button
          type="button"
          class={row}
          aria-current={l === loadout ? 'true' : undefined}
          onclick={() => {
            open = undefined;
            onloadout?.(l);
          }}
          ><span class="min-w-0 flex-1 truncate">{l}</span>{#if l === loadout}<span class="text-11 text-accent">current</span>{/if}</button>
      </li>
    {/each}
  </ul>
</Sheet>

<Sheet
  bind:open={() => open === 'issues', (v: boolean) => close('issues', v)}
  title="Build issues"
  note="Problems the calculation found; they clear themselves when fixed">
  {#if issues.length}
    <ul aria-label="Build issues" class="m-0 list-none p-0">
      {#each issues as it, i (i)}
        <li><IssueRow text={it.text} kind={issueKinds[it.kind][0]} where={issueKinds[it.kind][1]} /></li>
      {/each}
    </ul>
  {:else}
    <p class="note px-4 py-3">No build issues.</p>
  {/if}
</Sheet>
