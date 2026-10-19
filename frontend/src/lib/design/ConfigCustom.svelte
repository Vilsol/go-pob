<script lang="ts">
  import { untrack } from 'svelte';
  import ListOption from './ListOption.svelte';
  import ModCell from './ModCell.svelte';
  import Button from './Button.svelte';
  import { besideAnchor } from './items';
  import type { CustomGroup } from './fixtures';
  import type { CustomOps } from './config';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import ModBrowserDialog from './ModBrowserDialog.svelte';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

  // ConfigTab.lua:15-86
  type Props = { groups: CustomGroup[]; selected: number; ops: CustomOps; badHint?: string; revision?: unknown };
  let { groups, selected, ops, badHint = 'Not recognised: no modifier has this wording', revision }: Props = $props();

  const g = $derived(groups[selected]);
  const ok = $derived(g?.cells.filter((c) => c.k === 'ok').length ?? 0);
  const bad = $derived(g?.cells.filter((c) => c.k === 'bad').length ?? 0);

  let pending = $state<number | null>(null);
  let browsing = $state(false);
  let tip = $state<{ c: number; anchor: HTMLElement }>();
  const tipCell = $derived(tip && g?.enabled && g.cells[tip.c]?.k === 'ok' ? g.cells[tip.c] : undefined);
  const show = (c: number) => (e: Event) => (tip = { c, anchor: e.currentTarget as HTMLElement });
  const hide = (c: number) => () => {
    if (tip?.c === c) tip = undefined;
  };
  let toggleOver = $state.raw<{ i: number; anchor: HTMLElement }>();
  let toggleTip = $state.raw<{ i: number; header: string; anchor: HTMLElement; sections?: StatCompareSection[] }>();
  const ontoggleover = (i: number) => (e: Event) => {
    const box = (e.target as HTMLElement).closest('input[type="checkbox"]');
    if (!box || !ops.togglewhatif || toggleOver?.i === i) return;
    toggleOver = { i, anchor: e.currentTarget as HTMLElement };
  };
  const ontoggleout = (i: number) => (e: Event) => {
    if ((e.target as HTMLElement).closest('input[type="checkbox"]') && toggleOver?.i === i) toggleOver = undefined;
  };
  $effect(() => {
    void revision;
    const over = toggleOver;
    const enabled = over && groups[over.i]?.enabled;
    if (!over || !ops.togglewhatif) {
      toggleTip = undefined;
      return;
    }
    const { i, anchor } = over;
    const header = enabled ? 'Disabling this group will give you:' : 'Enabling this group will give you:';
    let live = true;
    const shown = untrack(() => toggleTip);
    toggleTip = { i, header, anchor, sections: shown?.i === i && shown.header === header ? shown.sections : undefined };
    ops
      .togglewhatif(i)
      .then((sections) => {
        if (live) toggleTip = { i, header, anchor, sections };
      })
      .catch(() => {});
    return () => (live = false);
  });
</script>

<section aria-label="Custom modifiers" class="panel relative mb-1 shrink-0">
  <div class="flex flex-wrap items-center justify-between gap-x-3 border-b border-line px-3 py-1.5">
    <h2 class="subhead text-12 uppercase">
      Custom Modifiers <span class="font-normal tracking-normal text-text-3 normal-case"
        >· {groups.length} groups · {groups.filter((x) => x.enabled).length} enabled</span>
    </h2>
    <span class="note">Each mod group has a title, an on/off switch and its lines · one modifier per cell · hover a parsed cell</span>
  </div>
  <div class="flex min-w-0 max-phone:flex-col">
    <div class="flex flex-[0_0_240px] flex-col gap-1 border-r border-line p-2 max-phone:flex-none max-phone:border-r-0 max-phone:border-b">
      <div role="list" aria-label="Mod groups" class="flex flex-col gap-1">
        {#each groups as x, i (i)}
          <!-- svelte-ignore a11y_mouse_events_have_key_events -->
          <div role="listitem" class="relative" onmouseover={ontoggleover(i)} onmouseout={ontoggleout(i)}>
            {#if toggleTip?.i === i}
              <div
                role="tooltip"
                aria-label="Group toggle what-if"
                class="tooltip fixed top-(--y) left-(--x) z-30 w-80 rounded-md px-3 py-2.5"
                use:besideAnchor={toggleTip.anchor}>
                {#if toggleTip.sections}<ImpactBlocks blocks={[{ header: toggleTip.header, sections: toggleTip.sections }]} />{:else}<span class="note"
                    >Calculating…</span
                  >{/if}
              </div>
            {/if}
            <ListOption
              label={x.title}
              count={x.cells.length}
              selected={i === selected}
              enabled={x.enabled}
              onselect={() => ops.onselect(i)}
              ontoggle={(on: boolean) => ops.ontoggle(i, on)} />
          </div>
        {/each}
      </div>
      <div class="mt-1 flex gap-1">
        <Button size="sm" variant="dashed" class="h-7 flex-[1_1_0]" onclick={ops.onaddgroup}>+ New group</Button>
        <Button
          size="sm"
          variant="danger"
          class="h-7 px-2"
          disabled={groups.length <= 1}
          aria-label="Delete selected group"
          onclick={() => ops.ondeletegroup(selected)}>Delete</Button>
      </div>
    </div>
    {#if g}
      <div class="flex min-w-0 flex-[1_1_0] flex-col">
        <div class="flex flex-wrap items-center gap-2 px-3 pt-2">
          <label class="field field-inline"
            >Group name
            <input
              class="input input-xs h-7 w-55 px-2 max-phone:w-full max-phone:min-w-0"
              value={g.title}
              oninput={(e) => ops.onrename(selected, e.currentTarget.value)} />
          </label>
          {#if ops.modbrowser}<Button size="sm" class="h-7" onclick={() => (browsing = true)}>Mod browser</Button>{/if}
          <span class={['text-12', g.enabled ? 'text-text-3' : 'text-worse']} aria-live="polite"
            >{g.enabled ? `${ok} applied${bad ? ` · ${bad} not recognised` : ''}` : 'Group disabled — its lines are not applied'}</span>
        </div>
        <div class="flex flex-wrap items-center gap-1.5 px-3 py-2">
          <div role="list" aria-label="Modifier lines" class="contents">
            {#each g.cells as c, i (i)}
              <div role="listitem" class="contents">
                <ModCell
                  text={c.text}
                  k={c.k}
                  off={!g.enabled}
                  hover={tipCell === c}
                  title={c.k === 'bad' ? badHint : c.k === 'ok' ? `Parsed: ${c.parsed}` : undefined}
                  aria-describedby={tipCell === c ? 'custom-mod-tip' : undefined}
                  onmouseenter={show(i)}
                  onmouseleave={hide(i)}
                  onfocusin={show(i)}
                  onfocusout={hide(i)}
                  edit={{
                    label: `Modifier line ${i + 1}`,
                    oninput: (t: string) => ops.oncell(selected, i, t),
                    oncommit: (t: string) => ops.oncommit(selected, i, t),
                    field: (el: HTMLInputElement) => {
                      if (pending === i) {
                        el.focus();
                        pending = null;
                      }
                    }
                  }}>
                  {#snippet before()}<button class="btn-bare px-0.5 text-11" aria-label="Move line {i + 1} earlier" onclick={() => ops.onmove(selected, i, -1)}
                      >◂</button
                    >{/snippet}
                  {#snippet after()}<button class="btn-bare px-0.5 text-11" aria-label="Move line {i + 1} later" onclick={() => ops.onmove(selected, i, 1)}
                      >▸</button
                    ><button class="btn-bare px-0.5" aria-label="Remove line {i + 1}" onclick={() => ops.onremove(selected, i)}>×</button>{/snippet}
                </ModCell>
              </div>
            {/each}
          </div>
          <Button
            size="sm"
            variant="dashed"
            onclick={() => {
              pending = g.cells.length;
              ops.onaddcell(selected);
            }}>+ Add modifier</Button>
        </div>
      </div>
    {/if}
  </div>
  {#if ops.modbrowser}
    <ModBrowserDialog
      bind:open={browsing}
      list={ops.modbrowser.list}
      sources={ops.modbrowser.sources}
      onadd={(line: string) => ops.modbrowser?.onadd(selected, line)} />
  {/if}
  {#if tipCell && tip}
    <div
      role="tooltip"
      id="custom-mod-tip"
      class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-30 flex w-81.5 flex-col gap-1.5 rounded-lg px-3 py-2.5 shadow-tip"
      use:besideAnchor={tip.anchor}>
      <div class="text-13 font-semibold">{tipCell.text}</div>
      <div class="font-mono text-11 break-all text-text-3">parsed as: {tipCell.parsed}</div>
    </div>
  {/if}
</section>
