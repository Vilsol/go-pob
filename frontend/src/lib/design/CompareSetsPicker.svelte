<script lang="ts">
  import Input from './Input.svelte';
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import Dialog from './Dialog.svelte';
  import Disclosure from './Disclosure.svelte';
  import Field from './Field.svelte';
  import type { CompareSetKind, CompareSets } from './compare';

  // Build.lua:301-414
  // CompareTab.lua:290-340
  // CompareTab.lua:342-478
  // CompareTab.lua:775-831, 941-959
  // TreeTab.lua:649-666
  type Loadouts = { loadouts: string[]; selected: number };
  type PrimarySets = {
    tree: { labels: string[]; active: number };
    item: { labels: string[]; ids: number[]; active: number };
    versions: { a: string; b: string };
  };
  type Props = {
    sets: CompareSets | undefined;
    primarySets: PrimarySets | undefined;
    aConfigSets: { sets: { id: number; title?: string }[]; active: number };
    aLoadouts: Loadouts;
    bLoadouts: Loadouts;
    versionList: { value: string; label: string }[];
    /** the last tree conversion's message */
    message?: string;
    onloadout: (side: 'a' | 'b', name: string) => void;
    /** this build's tree, item or config set, by index in its list */
    onprimary: (kind: 'tree' | 'item' | 'config', index: number) => void;
    onset: (kind: CompareSetKind, index: number) => void;
    oncount: (kind: 'stage' | 'mine', text: string) => void;
    onconvert: (side: 'a' | 'b', version: string, replace: boolean) => void;
  };
  let { sets, primarySets, aConfigSets, aLoadouts, bLoadouts, versionList, message, onloadout, onprimary, onset, oncount, onconvert }: Props = $props();

  const uid = $props.id();
  let open = $state({ a: false, b: false });
  const SET_SELECTS: { kind: CompareSetKind; label: string }[] = [
    { kind: 'tree', label: 'Tree set' },
    { kind: 'skill', label: 'Skill set' },
    { kind: 'item', label: 'Item set' },
    { kind: 'config', label: 'Config set' }
  ];
  const SKILL_SELECTS: { kind: CompareSetKind; label: string }[] = [
    { kind: 'group', label: 'Socket group' },
    { kind: 'skills', label: 'Main skill' },
    { kind: 'parts', label: 'Skill part' },
    { kind: 'minion', label: 'Minion' },
    { kind: 'minionSkill', label: 'Minion skill' }
  ];
  let converting = $state<{ side: 'a' | 'b'; version: string } | null>(null);
  const convert = (replace: boolean) => {
    const c = converting;
    converting = null;
    if (c) onconvert(c.side, c.version, replace);
  };
</script>

{#snippet loadout(side: 'a' | 'b', l: Loadouts)}
  <Field label="Loadout:" inline>
    <Select
      size="xs"
      class="max-w-60"
      aria-label={side === 'a' ? 'Primary loadout' : 'Comparison loadout'}
      disabled={!l.loadouts.length}
      value={String(l.selected)}
      onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
        const name = l.loadouts[Number(e.currentTarget.value)];
        e.currentTarget.value = String(l.selected);
        if (name !== undefined) onloadout(side, name);
      }}>
      {#if l.selected < 0}<option value="-1" disabled>{l.loadouts.length ? 'Custom sets' : 'No loadouts'}</option>{/if}
      {#each l.loadouts as n, i (i)}<option value={String(i)}>{n}</option>{/each}
    </Select></Field>
  <Disclosure class="btn btn-2xs" indicator="caret" bind:open={open[side]} controls="{uid}-sets-{side}"
    >{side === 'a' ? "This build's sets" : 'Comparison sets'}</Disclosure>
{/snippet}

{#if sets}
  {@const cur = sets}
  <div role="group" aria-label="Comparison build sets" class="grid shrink-0 grid-cols-[repeat(auto-fit,minmax(min(100%,420px),1fr))] gap-2 text-12">
    <section aria-label="This build" class="flex min-w-0 flex-col gap-1.5 rounded-md border border-line border-l-3 border-l-build-a px-2.5 py-1.5">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-11 font-semibold tracking-caps text-build-a">THIS BUILD</span>
        {@render loadout('a', aLoadouts)}
      </div>
      <div id="{uid}-sets-a" hidden={!open.a} class="flex flex-wrap items-center gap-2">
        {#if primarySets}
          {@const ps = primarySets}
          <Field label="Tree set:" inline>
            <Select
              size="xs"
              class="max-w-50"
              aria-label="Primary tree set"
              disabled={ps.tree.labels.length < 2}
              value={String(ps.tree.active)}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onprimary('tree', Number(e.currentTarget.value))}
              >{#each ps.tree.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</Select
            ></Field>
          <Field label="Item set:" inline>
            <Select
              size="xs"
              class="max-w-50"
              aria-label="Primary item set"
              disabled={ps.item.labels.length < 2}
              value={String(ps.item.active)}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onprimary('item', Number(e.currentTarget.value))}
              >{#each ps.item.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</Select
            ></Field>
        {/if}
        <Field label="Config set:" inline>
          <Select
            size="xs"
            class="max-w-50"
            aria-label="Primary config set"
            disabled={aConfigSets.sets.length < 2}
            value={String(aConfigSets.active)}
            onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
              onprimary('config', Number(e.currentTarget.value));
            }}
            >{#each aConfigSets.sets as c, i (i)}<option value={String(i)}>{c.title || 'Default'}</option>{/each}</Select
          ></Field>
        {#if primarySets}
          {@const ps = primarySets}
          <Field label="Tree version:" inline>
            <Select
              size="xs"
              class="max-w-50"
              aria-label="Primary tree version"
              value={ps.versions.a}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
                const version = e.currentTarget.value;
                e.currentTarget.value = ps.versions.a;
                if (version !== ps.versions.a) converting = { side: 'a', version };
              }}
              >{#each versionList as o (o.value)}<option value={o.value}>{o.label}</option>{/each}</Select
            ></Field>
        {/if}
      </div>
    </section>
    <section aria-label="Comparison" class="flex min-w-0 flex-col gap-1.5 rounded-md border border-line border-l-3 border-l-build-b px-2.5 py-1.5">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-11 font-semibold tracking-caps text-build-b">COMPARISON</span>
        {@render loadout('b', bLoadouts)}
      </div>
      <div id="{uid}-sets-b" hidden={!open.b} class="flex flex-wrap items-center gap-2">
        {#each SET_SELECTS as sel (sel.kind)}
          {@const list = cur[sel.kind]}
          {#if list}
            <Field label={`${sel.label}:`} inline>
              <Select
                size="xs"
                class="max-w-50"
                aria-label="Comparison {sel.label.toLowerCase()}"
                disabled={list.labels.length < 2}
                value={String(list.active)}
                onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onset(sel.kind, Number(e.currentTarget.value))}
                >{#each list.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</Select
              ></Field>
          {/if}
        {/each}
        {#if primarySets}
          {@const ps = primarySets}
          <Field label="Tree version:" inline>
            <Select
              size="xs"
              class="max-w-50"
              aria-label="Comparison tree version"
              value={ps.versions.b}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
                const version = e.currentTarget.value;
                e.currentTarget.value = ps.versions.b;
                if (version !== ps.versions.b) converting = { side: 'b', version };
              }}
              >{#each versionList as o (o.value)}<option value={o.value}>{o.label}</option>{/each}</Select
            ></Field>
        {/if}
      </div>
      <div class="flex flex-wrap items-center gap-2">
        {#each SKILL_SELECTS as sel (sel.kind)}
          {@const list = cur[sel.kind]}
          {#if list}
            <Field label={`${sel.label}:`} inline>
              <Select
                size="xs"
                class="max-w-50"
                aria-label="Comparison {sel.label.toLowerCase()}"
                disabled={list.labels.length < 2 ||
                  (sel.kind === 'skills' && !cur.skills.enabled) ||
                  ((sel.kind === 'minion' || sel.kind === 'minionSkill') && !cur[sel.kind]?.enabled)}
                value={String(list.active)}
                onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onset(sel.kind, Number(e.currentTarget.value))}
                >{#each list.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</Select
              ></Field>
          {/if}
        {/each}
        {#each [{ kind: 'stage' as const, label: 'Stages' }, { kind: 'mine' as const, label: 'Mines' }] as box (box.kind)}
          {#if cur[box.kind] !== undefined}
            <Field label={`${box.label}:`} inline>
              <Input
                size="xs"
                class="w-13"
                inputmode="numeric"
                aria-label="Comparison {box.label.toLowerCase()}"
                value={cur[box.kind]}
                onchange={(e: Event & { currentTarget: HTMLInputElement }) => oncount(box.kind, e.currentTarget.value)} /></Field>
          {/if}
        {/each}
      </div>
    </section>
  </div>
{/if}
{#if message}<div role="alert" class="note text-12">{message}</div>{/if}
<Dialog
  phone="centred"
  open={converting !== null}
  onclose={() => (converting = null)}
  title={converting ? `Convert to Version ${versionList.find((o) => o.value === converting?.version)?.label ?? converting.version}` : 'Convert'}
  size="md">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">Warning: some or all of the passives may be de-allocated due to changes in the tree.</p>
    <p class="m-0">Convert will replace your current tree.<br />Copy + Convert will backup your current tree.</p>
    <div class="flex gap-1.5">
      <Button size="sm" variant="primary" onclick={() => convert(true)}>Convert</Button>
      <Button size="sm" onclick={() => convert(false)}>Copy + Convert</Button>
      <Button size="sm" onclick={() => (converting = null)}>Cancel</Button>
    </div>
  </div>
</Dialog>
