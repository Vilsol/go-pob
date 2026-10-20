<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import NumberInput from './NumberInput.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass } from './tone';
  import EnchantDialog from './EnchantDialog.svelte';
  import AnointDialog from './AnointDialog.svelte';
  import CorruptDialog from './CorruptDialog.svelte';
  import AddImplicitDialog from './AddImplicitDialog.svelte';
  import AddModifierDialog from './AddModifierDialog.svelte';
  import AffixControls from './AffixControls.svelte';
  import CrucibleDialog from './CrucibleDialog.svelte';
  import CraftCardStepped from './craft-card/CraftCardStepped.svelte';
  import CraftCardTabs from './craft-card/CraftCardTabs.svelte';
  import { craftCardLayout, type CraftCardLayout } from '$lib/display/appearance';
  import { allAffixSliders, modTone, type DisplayEdit, type DisplayInfo } from './items';
  import { affixLines, statTemplate, type CraftApi } from './crafting';
  import type { Snippet } from 'svelte';

  // ItemsTab.lua:378-715, 1025-1070, 1522-1541, 2335-2382
  // ItemsTab.lua:1025-1089
  // ItemsTab.lua:509-553
  type Props = {
    info: DisplayInfo;
    onedit: (edit: DisplayEdit) => void;
    craft?: CraftApi;
    customRequest?: boolean;
    layout?: CraftCardLayout;
    head?: Snippet;
    result?: Snippet;
  };
  let { info, onedit, craft, customRequest = $bindable(false), layout = $craftCardLayout, head, result }: Props = $props();

  let enchantSlot = $state(0);
  let anointSlot = $state(0);
  let enchantOpen = $state(false);
  let anointOpen = $state(false);
  let corruptOpen = $state(false);
  let implicitOpen = $state(false);
  let customOpen = $state(false);
  let crucibleOpen = $state(false);
  $effect(() => {
    if (!customRequest) return;
    customRequest = false;
    if (info.addCustom && craft) customOpen = true;
  });
  let rangeLine = $state(0);
  const ranged = $derived(info.mods.flatMap((m, i) => (m.hasRange ? [i] : [])));
  const picked = $derived(ranged.includes(rangeLine) ? rangeLine : ranged[0]);
  const IMPLICIT = new Set(['enchant', 'scourge', 'implicit']);
  const implicitMods = $derived(info.mods.flatMap((m, i) => (IMPLICIT.has(m.section ?? '') ? [i] : [])));
  const explicitMods = $derived(info.mods.flatMap((m, i) => (IMPLICIT.has(m.section ?? '') ? [] : [i])));
  const slotted = $derived(
    craft && info.affixes
      ? affixLines(
          info.affixes,
          info.mods.map((m) => m.line),
          explicitMods
        )
      : { stats: [], slots: [], shared: new Set<number>(), rest: explicitMods }
  );
  const uid = $props.id();
  const lineId = (i: number) => `${uid}-line-${i}`;
  let lineRoll = $state.raw<{ index: number; value: number; from: DisplayInfo }>();

  // ItemsTab.lua:580-581
  const INFLUENCES = ['Shaper', 'Elder', 'Warlord', 'Hunter', 'Crusader', 'Redeemer', 'Searing Exarch', 'Eater of Worlds'];
  // ItemsTab.lua:659-660
  const CATALYSTS = [
    'Catalyst',
    'Abrasive (Attack)',
    'Accelerating (Speed)',
    'Dextral (Suffix)',
    'Fertile (Life & Mana)',
    'Imbued (Caster)',
    'Intrinsic (Attribute)',
    'Noxious (Physical & Chaos Damage)',
    'Prismatic (Resistance)',
    'Sinistral (Prefix)',
    'Tempering (Defense)',
    'Turbulent (Elemental)',
    'Unstable (Critical)'
  ];
  // ItemsTab.lua:28-36
  const COLOURS = ['R', 'G', 'B', 'W'];
  const ALT = ['Variant', 'Alt variant', 'Alt variant 2', 'Alt variant 3', 'Alt variant 4', 'Alt variant 5'];

  const sockets = $derived.by(() => {
    const out: { colour: string; linked: boolean }[] = [];
    for (const ch of info.sockets) {
      if (ch === '-') out[out.length - 1].linked = true;
      else if (ch !== ' ') out.push({ colour: ch, linked: false });
    }
    return out;
  });
  const writeSockets = (list: { colour: string; linked: boolean }[]) =>
    onedit({ kind: 'sockets', text: list.map((s, i) => s.colour + (i < list.length - 1 ? (s.linked ? '-' : ' ') : '')).join('') });
  const setInfluence = (slot: 0 | 1, name: string) => {
    const pair = [info.influence[0] ?? '', info.influence[1] ?? ''];
    pair[slot] = name;
    onedit({ kind: 'influence', text: pair.filter(Boolean).join(',') });
  };
  const selected = $derived([info.variant, ...info.altVariants]);

  type Cluster = NonNullable<DisplayInfo['cluster']>;
  let clusterSent = $state.raw<{ from: Cluster; skill: string; count: number }>();
  const clusterNow = (cl: Cluster) => (clusterSent?.from === cl ? clusterSent : { skill: cl.skill, count: cl.nodeCount });
  const clusterEdit = (cl: Cluster, skill: string, count: number) => {
    clusterSent = { from: cl, skill, count };
    onedit({ kind: 'cluster', text: skill, value: count });
  };
</script>

{#snippet modRow(i: number, preview?: string[], also = false)}
  {@const m = info.mods[i]}
  {#if also}
    <!-- Item.lua:2085-2093 -->
    <li class="note">Also adds to {m.line}</li>
  {:else}
    <li class="flex flex-col gap-0.5">
      <div class="flex min-w-0 items-center gap-1.5">
        <button
          class={['btn-bare grow text-left font-sans text-13 text-(--tone)', toneClass(m.disabled ? 'muted' : modTone(m.kind)), m.disabled && 'line-through']}
          aria-pressed={!m.disabled}
          title={m.disabled ? 'Disabled: click to enable it' : 'Click to disable this modifier'}
          onclick={() => onedit({ kind: 'disable', index: i })}
          ><span id={lineId(i)}>{(!slotted.shared.has(i) && preview?.find((l) => statTemplate(l) === statTemplate(m.line))) || m.line}</span></button>
        {#if m.removable}
          <span class="text-11 text-text-3">{m.removal}</span>
          <Button size="xs" aria-label="Remove {m.line}" onclick={() => onedit({ kind: 'remove', index: i })}>Remove</Button>
        {/if}
        {#if info.foulborn?.[i]?.mutable}
          <label class="field field-inline shrink-0 gap-1 text-11 text-text-2" title="Swap this modifier for its Foulborn counterpart, or back"
            ><input
              type="checkbox"
              aria-label="Foulborn: {m.line}"
              checked={info.foulborn[i].mutated}
              onchange={() => onedit({ kind: 'mutate', index: i })} />Foulborn</label>
        {/if}
      </div>
      {#if m.hasRange && $allAffixSliders}
        <div class="flex items-center gap-2">
          <input
            type="range"
            class="min-w-0 grow"
            min="0"
            max="1"
            step="0.01"
            aria-label="Roll of {m.line}"
            aria-describedby={lineId(i)}
            value={m.range}
            oninput={(e) => (lineRoll = { index: i, value: Number(e.currentTarget.value), from: info })}
            onchange={(e) => onedit({ kind: 'range', index: i, float: Number(e.currentTarget.value) })} /><output
            class="w-10 shrink-0 text-right font-mono text-12 text-text-2"
            >{Math.round((lineRoll?.index === i && lineRoll.from === info ? lineRoll.value : m.range) * 100)}%</output>
        </div>
      {/if}
    </li>
  {/if}
{/snippet}

{#snippet sub(title: string)}
  <h4 class="caps text-11 text-text-3">{title}</h4>
{/snippet}

{#snippet actionsHead(title: string)}
  <div class="mt-1 border-t border-line-faint pt-2.5">{@render sub(title)}</div>
{/snippet}

{#snippet modList(list: number[], label: string)}
  {#if list.length}
    <ul aria-label={label} class="flex flex-col gap-1">
      {#each list as i (i)}
        {@render modRow(i)}
      {/each}
    </ul>
  {/if}
{/snippet}

{#snippet variants()}
  {#if info.variants.length > 1}
    {#each selected as sel, i (i)}
      <FloatingLabel label={ALT[i]}
        ><Select
          size="sm"
          aria-label={ALT[i]}
          options={info.variants.map((v, k) => ({ value: String(k + 1), label: v }))}
          value={String(sel)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) =>
            onedit({ kind: 'variant', index: i, value: Number(e.currentTarget.value) })} /></FloatingLabel>
    {/each}
  {/if}
  {#if info.variantGroups}
    {@const vg = info.variantGroups}
    {#if vg.versions.length > 1}
      <FloatingLabel label="Version"
        ><Select
          size="sm"
          aria-label="Version"
          options={vg.versions.map((v, k) => ({ value: String(k + 1), label: v }))}
          value={String(vg.version)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onedit({ kind: 'version', value: Number(e.currentTarget.value) })} /></FloatingLabel>
    {/if}
    {#each vg.groups as g, i (i)}
      <FloatingLabel label={ALT[i]}
        ><Select
          size="sm"
          aria-label={ALT[i]}
          disabled={!g.enabled}
          options={g.options.map((o) => ({ value: String(o.variant), label: o.label }))}
          value={String(g.selected)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) =>
            onedit({ kind: 'variantGroup', index: g.group, value: Number(e.currentTarget.value) })} /></FloatingLabel>
    {/each}
  {/if}
{/snippet}

{#snippet socketRow()}
  {#if info.socketLimit > 0}
    <div role="group" aria-label="Sockets" class="flex flex-wrap items-center gap-1.5">
      {#each sockets as s, i (i)}
        <Select
          size="sm"
          class="w-14"
          aria-label="Socket {i + 1} colour"
          options={COLOURS}
          value={s.colour}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) =>
            writeSockets(sockets.map((x, k) => (k === i ? { ...x, colour: e.currentTarget.value } : x)))} />
        {#if i < sockets.length - 1}
          <input
            class="check"
            type="checkbox"
            aria-label="Link sockets {i + 1} and {i + 2}"
            checked={s.linked}
            onchange={(e) => writeSockets(sockets.map((x, k) => (k === i ? { ...x, linked: e.currentTarget.checked } : x)))} />
        {/if}
      {/each}
      {#if sockets.length < info.socketLimit}
        <Button size="xs" aria-label="Add a socket" onclick={() => writeSockets([...sockets, { colour: info.defaultColor || 'R', linked: false }])}>+</Button>
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet properties()}
  {#if info.canInfluence}
    <div class="grid grid-cols-2 gap-2">
      {#each [0, 1] as const as slot (slot)}
        <FloatingLabel label="Influence {slot + 1}"
          ><Select
            size="sm"
            aria-label="Influence {slot + 1}"
            title="Selecting an influence here will also allow the modifier dropdowns to contain influenced mods."
            options={[{ value: '', label: 'Influence' }, ...INFLUENCES.map((n) => ({ value: n, label: n }))]}
            value={info.influence[slot] ?? ''}
            onchange={(e: Event & { currentTarget: HTMLSelectElement }) => setInfluence(slot, e.currentTarget.value)} /></FloatingLabel>
      {/each}
    </div>
  {/if}
  {#if info.hasQuality}
    <label class="field field-inline gap-2"
      >Quality<NumberInput value={info.quality} min={0} max={99} label="Quality" onchange={(v: number) => onedit({ kind: 'quality', value: v })} /></label>
  {/if}
  {#if info.showsCatalyst}
    <div class="flex items-end gap-2">
      <FloatingLabel label="Catalyst" class="grow"
        ><Select
          size="sm"
          aria-label="Catalyst"
          options={CATALYSTS.map((c, k) => ({ value: String(k), label: c }))}
          value={String(info.catalyst)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onedit({ kind: 'catalyst', value: Number(e.currentTarget.value) })} /></FloatingLabel>
      {#if info.catalyst > 0}
        <NumberInput
          value={info.catalystQuality}
          min={0}
          max={99}
          label="Catalyst quality"
          onchange={(v: number) => onedit({ kind: 'catalyst', value: info.catalyst, second: v })} />
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet clusterControls()}
  {#if info.cluster}
    {@const cl = info.cluster}
    <div role="group" aria-label="Cluster jewel" class="flex flex-col gap-1.5">
      <FloatingLabel label="Cluster skill"
        ><Select
          size="sm"
          aria-label="Cluster skill"
          options={cl.skills.map((s) => ({ value: s.id, label: s.label }))}
          value={cl.skill}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => clusterEdit(cl, e.currentTarget.value, clusterNow(cl).count)} /></FloatingLabel>
      <label class="field field-inline gap-2 text-12"
        >Added Passives: {cl.nodeCount}<input
          type="range"
          class="w-37.5"
          min={cl.min}
          max={cl.max}
          step="1"
          aria-label="Added Passives"
          value={cl.nodeCount}
          onchange={(e) => clusterEdit(cl, clusterNow(cl).skill, Number(e.currentTarget.value))} /></label>
    </div>
  {/if}
{/snippet}

{#snippet base()}
  {@render head?.()}
  {#if info.variants.length > 1 || (info.variantGroups && (info.variantGroups.versions.length > 1 || info.variantGroups.groups.length))}
    <div class="flex flex-col gap-2 border-t border-line-faint pt-2.5">
      {@render sub('Variant')}
      {@render variants()}
    </div>
  {/if}
  {#if info.socketLimit > 0}
    <div class="flex flex-col gap-1.5 border-t border-line-faint pt-2.5">
      {@render sub('Sockets and links')}
      {@render socketRow()}
    </div>
  {/if}
  {#if info.canInfluence || info.hasQuality || info.showsCatalyst}
    <div class="flex flex-col gap-2 border-t border-line-faint pt-2.5">
      {@render sub('Properties')}
      {@render properties()}
    </div>
  {/if}
  {#if info.cluster}
    <div class="flex flex-col gap-2 border-t border-line-faint pt-2.5">
      {@render sub('Cluster jewel')}
      {@render clusterControls()}
    </div>
  {/if}
{/snippet}

{#snippet implicits()}
  {#if implicitMods.length}
    {@render modList(implicitMods, 'Implicit modifiers')}
  {:else}
    <p class="note">No implicit modifiers.</p>
  {/if}
  {#if craft && ((info.enchants ?? 0) > 0 || (info.anoints ?? 0) > 0 || info.corruptible || info.addImplicit)}
    {@render actionsHead('Change')}
    <div role="group" aria-label="Enchant, anoint and corrupt" class="flex flex-wrap gap-1.5">
      {#each Array.from({ length: info.enchants ?? 0 }, (_, k) => k + 1) as slot (slot)}
        <Button
          size="xs"
          onclick={() => {
            enchantSlot = slot;
            enchantOpen = true;
          }}>{slot > 1 ? `Change Enchantment ${slot}...` : 'Change Enchantment...'}</Button>
      {/each}
      {#each Array.from({ length: info.anoints ?? 0 }, (_, k) => k + 1) as slot (slot)}
        <Button
          size="xs"
          onclick={() => {
            anointSlot = slot;
            anointOpen = true;
          }}>{slot > 1 ? `Anoint ${slot}...` : 'Anoint...'}</Button>
      {/each}
      {#if info.corruptible}
        <Button size="xs" onclick={() => (corruptOpen = true)}>Corrupt...</Button>
      {/if}
      {#if info.addImplicit}
        <Button size="xs" onclick={() => (implicitOpen = true)}>Add Implicit...</Button>
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet affixSection()}
  {#if craft && info.affixes}
    <AffixControls affixes={info.affixes} {craft} slots={slotted.slots} changes={slotted.stats} {lineId} line={modRow} />
  {/if}
  {#if craft && info.affixes && slotted.rest.length}
    <div class="flex flex-col gap-1.5 border-t border-line-faint pt-2.5">
      {@render sub('Other modifiers')}
      {@render modList(slotted.rest, 'Other explicit modifiers')}
    </div>
  {:else}
    {@render modList(slotted.rest, craft && info.affixes ? 'Other explicit modifiers' : 'Explicit modifiers')}
  {/if}
  {#if !$allAffixSliders && picked !== undefined}
    {@const m = info.mods[picked]}
    <div class="flex items-center gap-2">
      <Select
        size="sm"
        class="min-w-0 grow"
        aria-label="Modifier range"
        options={ranged.map((i) => ({ value: String(i), label: info.mods[i].line }))}
        value={String(picked)}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (rangeLine = Number(e.currentTarget.value))} />
      <input
        type="range"
        class="w-25 shrink-0"
        min="0"
        max="1"
        step="0.01"
        aria-label="Roll of {m.line}"
        value={m.range}
        onchange={(e) => onedit({ kind: 'range', index: picked, float: Number(e.currentTarget.value) })} />
    </div>
  {/if}
  {#if craft && (info.addCustom || info.addCrucible)}
    {@render actionsHead('Add')}
    <div role="group" aria-label="Add modifiers" class="flex flex-wrap gap-1.5">
      {#if info.addCustom}
        <Button size="xs" onclick={() => (customOpen = true)}>Add modifier...</Button>
      {/if}
      {#if info.addCrucible}
        <Button size="xs" onclick={() => (crucibleOpen = true)}>Add Crucible mod...</Button>
      {/if}
    </div>
  {/if}
{/snippet}

<section aria-label="Item controls" class="flex flex-col border-t border-line pt-2.5 text-13">
  {#if layout === 'tabs'}
    <CraftCardTabs {base} {implicits} affixes={affixSection} {result} implicitCount={implicitMods.length} explicitCount={explicitMods.length} />
  {:else}
    <CraftCardStepped {base} {implicits} affixes={affixSection} {result} />
  {/if}
</section>

{#if craft && enchantSlot}
  <EnchantDialog bind:open={enchantOpen} slot={enchantSlot} {craft} {onedit} />
{/if}
{#if craft && anointSlot}
  <AnointDialog bind:open={anointOpen} slot={anointSlot} anointed={info.anointed ?? []} {craft} {onedit} />
{/if}
{#if craft && corruptOpen}
  <CorruptDialog bind:open={corruptOpen} {craft} />
{/if}
{#if craft && implicitOpen}
  <AddImplicitDialog bind:open={implicitOpen} {craft} />
{/if}
{#if craft && customOpen}
  <AddModifierDialog bind:open={customOpen} {craft} />
{/if}
{#if craft && crucibleOpen}
  <CrucibleDialog bind:open={crucibleOpen} {craft} />
{/if}
