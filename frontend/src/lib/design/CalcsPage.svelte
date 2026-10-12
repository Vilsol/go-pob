<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import StatGauge from './StatGauge.svelte';
  import PobText from './PobText.svelte';
  import TooltipLines from './TooltipLines.svelte';
  import Button from './Button.svelte';
  import CalcsBreakdown, { type BreakdownPart, type ModRow, type ModTable } from './CalcsBreakdown.svelte';
  import { tone as toneClass } from './tone';
  import { pobPlain } from './pob_text';
  import { isHeaderRow, rowKey, sectionTone } from './calcs';
  import type { CalcRow, CalcSectionFixture, CalcsFixture, ItemTip } from './fixtures';
  import type { Snippet } from 'svelte';
  import type { calculator } from '$lib/types';

  const MODES = ['Unbuffed', 'Buffed', 'In Combat', 'Effective DPS'];
  type Props = {
    calcs: CalcsFixture;
    socketGroups: string[];
    mainSocketGroup: number;
    controls?: calculator.MainSkillControls;
    mode?: string;
    showAll?: boolean;
    q?: string;
    pin?: string;
    onmaingroup: (i: number) => void;
    oncontrol: (key: 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill', value: number | undefined) => void;
    /* CalcsTab.lua:106-110 */
    onspectres?: () => void;
    /* CalcsTab.lua:46-51 */
    grouptip?: (group: number) => Promise<{ Text: string; Separator: boolean }[] | undefined>;
    modrows?: (sd: unknown) => Promise<ModRow[] | undefined>;
    itemtip?: (name: string) => Promise<ItemTip | undefined>;
    nodeview?: Snippet<[number]>;
    onmode?: (mode: string) => void;
    /* CalcsTab.lua:87-91 */
    showMinion?: boolean;
    onshowminion?: (show: boolean) => void;
    /* CalcsTab.lua:174-189 */
    collapsed?: Record<string, boolean>;
    oncollapse?: (id: string, collapsed: boolean) => void;
    /* CalcsTab.lua:135-137 */
    /* CalcsTab.lua:135-137 */
    lists?: { label: string; text: string; bd?: string }[];
    listrows?: (bd: string) => Promise<ModRow[] | undefined>;
    /* Calcs.lua:37-70 */
    moddebug?: () => Promise<string>;
  };
  let {
    calcs,
    socketGroups,
    mainSocketGroup,
    controls,
    mode = 'Effective DPS',
    showAll = $bindable(false),
    q = $bindable(''),
    pin,
    onmaingroup,
    oncontrol,
    onmode,
    showMinion,
    onshowminion,
    collapsed,
    oncollapse,
    lists = [],
    onspectres,
    grouptip,
    modrows,
    listrows,
    itemtip,
    nodeview,
    moddebug
  }: Props = $props();
  let debugText = $state<string>();
  const toggleDebug = async () => {
    debugText = debugText === undefined ? await moddebug?.() : undefined;
  };
  let groupTip = $state.raw<{ Text: string; Separator: boolean }[]>();
  let groupTipFor = -1;
  const showGroupTip = () => {
    if (!grouptip) return;
    const i = (groupTipFor = mainSocketGroup);
    grouptip(i)
      .then((lines) => {
        if (groupTipFor === i) groupTip = lines;
      })
      .catch(() => {});
  };
  const hideGroupTip = () => {
    groupTipFor = -1;
    groupTip = undefined;
  };
  const digits = (e: Event) => {
    const raw = (e.currentTarget as HTMLInputElement).value.replace(/\D/g, '');
    return raw === '' ? undefined : parseInt(raw);
  };

  let pinned = $state<string | undefined>(pin);
  let hover = $state<string>();
  let closed = $state<Record<string, boolean>>({});
  $effect(() => {
    if (collapsed) closed = { ...collapsed };
  });

  const needle = $derived(q.trim().toLowerCase());
  const matches = (r: CalcRow) => !needle || pobPlain(r.label).toLowerCase().includes(needle);
  const visible = (s: CalcSectionFixture) => (s.applicable || showAll) && (!needle || (s.blocks ?? []).some((b) => b.rows.some((r) => r.label && matches(r))));
  const shown = $derived(calcs.sections.filter(visible));
  const wideTop = $derived(shown.filter((s) => s.wide && s.id !== 'DamageTaken'));
  const wideBottom = $derived(shown.filter((s) => s.wide && s.id === 'DamageTaken'));

  // CalcSectionControl.lua:56-61, 223-356
  let overlays = $state<{ id: string; x: number; y: number }[]>([]);
  const cols = $derived([1, 2, 3].map((c) => shown.filter((s) => !s.wide && s.col === c && !overlays.some((o) => o.id === s.id))));
  const overlaid = $derived(overlays.map((o) => ({ o, s: shown.find((x) => x.id === o.id) })).filter((e) => e.s));
  const popOut = (id: string, el: HTMLElement | null) => {
    const r = el?.getBoundingClientRect();
    overlays = [...overlays, { id, x: r?.left ?? 40, y: r?.top ?? 40 }];
  };
  const raise = (id: string) => {
    if (overlays.at(-1)?.id !== id) overlays = [...overlays.filter((o) => o.id !== id), ...overlays.filter((o) => o.id === id)];
  };
  const drag = (id: string) => (e: PointerEvent) => {
    if ((e.target as HTMLElement).closest('button')) return;
    const o = overlays.find((x) => x.id === id);
    if (!o) return;
    const [dx, dy] = [e.clientX - o.x, e.clientY - o.y];
    const move = (m: PointerEvent) => {
      o.x = m.clientX - dx;
      o.y = m.clientY - dy;
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };

  const rows = $derived(
    new Map(
      calcs.sections.flatMap((s) =>
        (s.blocks ?? []).flatMap((b, j) => {
          const head = b.rows.find(isHeaderRow);
          return b.rows.map((r, k) => [rowKey(s.id, j, k), { r, head, section: s }] as const);
        })
      )
    )
  );
  const shownKey = $derived(hover ?? pinned);
  const shownRow = $derived(shownKey ? rows.get(shownKey) : undefined);
  const LIST = 'list:';
  const shownList = $derived(shownKey?.startsWith(LIST) ? lists.find((l) => l.bd === shownKey.slice(LIST.length)) : undefined);
  let listTables = $state.raw<Record<string, ModRow[]>>({});
  let modTables = $state.raw<Record<string, ModTable[]>>({});
  const parts = $derived<BreakdownPart[]>(
    shownList?.bd
      ? listTables[shownList.bd]
        ? [{ text: shownList.text, modTables: [{ rows: listTables[shownList.bd] }] }]
        : []
      : shownRow
        ? shownRow.r.cells
            .map((c, i) => ({
              head: shownRow.head?.cells[i]?.text,
              text: c.text,
              bd: c.bd ? calcs.breakdowns[c.bd] : undefined,
              mods: c.mods,
              modTables: modTables[`${shownKey}|${i}`]
            }))
            .filter((p) => p.bd || p.mods || p.modTables)
        : []
  );
  $effect(() => {
    void calcs;
    modTables = {};
    listTables = {};
  });
  $effect(() => {
    const bd = shownList?.bd;
    if (!listrows || !bd || listTables[bd]) return;
    void listrows(bd)
      .then((r) => (listTables = { ...listTables, [bd]: r ?? [] }))
      .catch(() => {});
  });
  $effect(() => {
    const key = shownKey;
    const row = shownRow;
    if (!modrows || !key || !row) return;
    row.r.cells.forEach((c, i) => {
      const sections = (c as { sections?: { label?: string; sd: unknown }[] }).sections;
      const k = `${key}|${i}`;
      if (!sections?.length || modTables[k]) return;
      void Promise.all(sections.map((s) => modrows(s.sd)))
        .then((tables) => {
          modTables = { ...modTables, [k]: sections.map((s, j) => ({ label: s.label, rows: tables[j] ?? [] })) };
        })
        .catch(() => {});
    });
  });
  const title = $derived(shownList ? shownList.label : shownRow ? `${pobPlain(shownRow.section.title)} · ${pobPlain(shownRow.r.label)}` : undefined);

  const enter = (key: string) => () => (hover = key);
  const leave = (key: string) => () => {
    if (hover === key) hover = undefined;
  };
  const toggle = (key: string) => () => {
    pinned = pinned === key ? undefined : key;
    hover = undefined;
  };
  const described = (key: string) => (shownKey === key ? 'calcs-breakdown' : undefined);
  const opts = (names: string[] | undefined) => (names ?? []).map((label, i) => ({ value: String(i + 1), label }));
  const num = (e: Event) => Number((e.currentTarget as HTMLSelectElement).value);
</script>

{#snippet label(key: string, r: CalcRow)}
  <button
    class="min-w-0 truncate border-0 bg-transparent p-0 text-left font-sans text-11 text-text-2"
    aria-pressed={pinned === key}
    aria-describedby={described(key)}
    onfocus={enter(key)}
    onblur={leave(key)}><PobText text={r.label} /></button>
{/snippet}

{#snippet sectionBody(s: CalcSectionFixture)}
  <div class="border-t border-line">
    {#each s.blocks ?? [] as b, j (j)}
      {#if b.label !== s.title || (s.blocks?.length ?? 0) > 1}
        <h3 class="m-0 border-t border-line-faint px-2 pt-1 font-sans text-10 font-semibold tracking-caps-tight text-text-3 uppercase first:border-t-0">
          {b.label}{#if b.extra}<span class="ml-1.5 font-mono tracking-normal normal-case"><PobText text={b.extra} /></span>{/if}
        </h3>
      {/if}
      {#each b.rows as r, k (k)}
        {@const key = rowKey(s.id, j, k)}
        {#if r.label && matches(r)}
          <div
            data-key={key}
            class={['relative flex cursor-pointer justify-between gap-1.5 px-2 text-11 leading-[18px] hover:bg-surface-3', shownKey === key && 'bg-surface-3']}
            onclick={toggle(key)}
            onmouseenter={enter(key)}
            onmouseleave={leave(key)}
            role="presentation">
            {#if r.gauge}<StatGauge {...r.gauge} />{/if}
            <span class="relative flex min-w-0">{@render label(key, r)}</span>
            <span class="relative flex shrink-0 gap-2 font-mono">
              {#each r.cells as c, i (i)}<span><PobText text={c.text} /></span>{/each}
            </span>
          </div>
        {/if}
      {/each}
    {/each}
  </div>
{/snippet}

{#snippet wide(s: CalcSectionFixture)}
  <section aria-label={pobPlain(s.title)} class="rounded border border-line bg-surface-1">
    <h2 class={['toned block border-b border-line px-2 py-0.75 text-11 font-bold tracking-caps-tight uppercase', toneClass(sectionTone(s))]}>▾ {s.title}</h2>
    {#each s.blocks ?? [] as b, j (j)}
      {@const head = b.rows.find(isHeaderRow)}
      <div role="table" aria-label={pobPlain(b.label)} class="font-mono text-11" style:--rest={(head?.cells.length ?? 6) - 1}>
        {#if b.label !== s.title || b.extra}
          <div
            role="row"
            class="border-t border-line-faint px-2 pt-1.5 pb-0.5 font-sans text-10 font-semibold tracking-caps-tight text-text-3 uppercase first:border-t-0">
            <span role="rowheader"
              >{b.label}{#if b.extra}<span class="ml-1.5 font-mono tracking-normal normal-case"><PobText text={b.extra} /></span>{/if}</span>
          </div>
        {/if}
        {#each b.rows as r, k (k)}
          {@const key = rowKey(s.id, j, k)}
          {#if isHeaderRow(r)}
            <div
              role="row"
              class="grid grid-cols-[130px_minmax(min-content,2fr)_repeat(var(--rest),minmax(min-content,1fr))] px-2 py-px font-sans text-10 min-[1800px]:grid-cols-[210px_minmax(min-content,2fr)_repeat(var(--rest),minmax(min-content,1fr))]">
              <span role="columnheader"><span class="sr-only">Stat</span></span>
              {#each r.cells as c, i (i)}<span role="columnheader" class="truncate text-right"><PobText text={c.text} /></span>{/each}
            </div>
          {:else if matches(r)}
            <!-- eslint-disable-next-line svelte/no-unused-svelte-ignore -- svelte-check warns on both, eslint-plugin-svelte sees one -->
            <!-- svelte-ignore a11y_interactive_supports_focus, a11y_click_events_have_key_events -->
            <div
              role="row"
              data-key={key}
              class={[
                'grid cursor-pointer grid-cols-[130px_minmax(min-content,2fr)_repeat(var(--rest),minmax(min-content,1fr))] px-2 py-px leading-[17px] hover:bg-surface-3 min-[1800px]:grid-cols-[210px_minmax(min-content,2fr)_repeat(var(--rest),minmax(min-content,1fr))]',
                shownKey === key && 'bg-surface-3'
              ]}
              onclick={toggle(key)}
              onmouseenter={enter(key)}
              onmouseleave={leave(key)}>
              <span role="rowheader" class="flex min-w-0">{@render label(key, r)}</span>
              {#each r.cells as c, i (i)}<span role="cell" class="truncate text-right" title={pobPlain(c.text)}><PobText text={c.text} /></span>{/each}
            </div>
          {/if}
        {/each}
      </div>
    {/each}
  </section>
{/snippet}

<div class="flex shrink-0 flex-wrap items-end gap-2 border-b border-line px-3 py-2">
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="relative" onmouseenter={showGroupTip} onmouseleave={hideGroupTip}>
    <FloatingLabel label="Socket Group"
      ><Select
        size="xs"
        class="w-42.5"
        options={socketGroups.map((l, i) => ({ value: String(i), label: l }))}
        value={String(mainSocketGroup)}
        onchange={(e: Event) => onmaingroup(num(e))} /></FloatingLabel>
    {#if groupTip?.length}
      <div role="tooltip" aria-label="Socket group tooltip" class="tooltip absolute top-full left-0 z-10 mt-1 w-80 rounded-md px-3 py-2.5">
        <TooltipLines lines={groupTip} />
      </div>
    {/if}
  </div>
  {#if controls && (controls.Skills?.length ?? 0) > 0}
    <FloatingLabel label="Active Skill"
      ><Select
        size="xs"
        class="w-32.5"
        options={opts(controls.Skills)}
        value={String(controls.MainActiveSkill)}
        onchange={(e: Event) => oncontrol('MainActiveSkill', num(e))} /></FloatingLabel>
  {/if}
  {#if controls?.PartShown}
    <FloatingLabel label="Skill Part"
      ><Select
        size="xs"
        class="w-27.5"
        options={opts(controls.Parts)}
        value={String(controls.SkillPart)}
        onchange={(e: Event) => oncontrol('SkillPart', num(e))} /></FloatingLabel>
  {/if}
  {#if controls?.StageShown}
    <label class="check-label h-6.5 text-12 text-text-2"
      >Skill Stages<input
        class="input h-6.5 w-13 font-mono"
        inputmode="numeric"
        aria-label="Skill Stages"
        value={controls.StageCount}
        oninput={(e: Event) => oncontrol('StageCount', digits(e))} /></label>
  {/if}
  {#if controls?.MineShown}
    <label class="check-label h-6.5 text-12 text-text-2"
      >Active Mines<input
        class="input h-6.5 w-13 font-mono"
        inputmode="numeric"
        aria-label="Active Mines"
        value={controls.MineCount}
        oninput={(e: Event) => oncontrol('MineCount', digits(e))} /></label>
  {/if}
  {#if controls?.MinionShown}
    <FloatingLabel label="Minion"
      ><Select
        size="xs"
        class="w-32.5"
        options={(controls.Minions ?? []).map((m, i) => ({ value: String(i + 1), label: m.Label }))}
        value={String(controls.Minion)}
        onchange={(e: Event) => oncontrol('Minion', num(e))} /></FloatingLabel>
  {/if}
  {#if controls?.MinionLibraryShown && onspectres}
    <Button size="xs" class="h-6.5 text-12" onclick={onspectres}>Manage Spectres...</Button>
  {/if}
  {#if controls?.MinionSkillShown}
    <FloatingLabel label="Minion Skill"
      ><Select
        size="xs"
        class="w-30"
        options={opts(controls.MinionSkills)}
        value={String(controls.MinionSkill)}
        onchange={(e: Event) => oncontrol('MinionSkill', num(e))} /></FloatingLabel>
  {/if}
  {#if showMinion !== undefined}
    <label class="check-label h-6.5 text-text-2"
      ><input
        type="checkbox"
        class="check size-3.75"
        checked={showMinion}
        title="Show stats for the minion instead of the player."
        onchange={(e) => onshowminion?.(e.currentTarget.checked)} /> Show Minion Stats</label>
  {/if}
  {#if onmode}
    <FloatingLabel label="Calculation Mode"
      ><Select
        size="xs"
        class="w-30"
        options={MODES}
        value={mode}
        onchange={(e: Event) => onmode((e.currentTarget as HTMLSelectElement).value)} /></FloatingLabel>
  {/if}
  <label class="check-label h-6.5 text-text-2"><input type="checkbox" class="check size-3.75" bind:checked={showAll} /> Show all panels</label>
  {#if moddebug}
    <Button size="xs" class="h-6.5 text-12" aria-expanded={debugText !== undefined} onclick={toggleDebug}>Mod Debug</Button>
  {/if}
  <input type="search" placeholder="Search stats" aria-label="Search stats" class="input input-xs ml-auto w-42.5 px-2" bind:value={q} />
</div>
<div class="flex min-h-0 grow">
  <div aria-label="Calculations" role="region" class="scroll flex min-w-0 grow flex-col gap-1.5 px-2.5 pt-2 pb-10">
    {#if debugText !== undefined}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
      <pre
        aria-label="Mod debug"
        role="region"
        tabindex="0"
        class="scroll m-0 max-h-[60vh] shrink-0 rounded border border-line bg-surface-1 px-2 py-1.5 font-mono text-11 whitespace-pre text-text-2">{debugText}</pre>
    {/if}
    {#if lists.length}
      <dl
        aria-label="Buffs and debuffs"
        class="m-0 grid grid-cols-[max-content_1fr] gap-x-3 gap-y-0.5 rounded border border-line bg-surface-1 px-2 py-1.5 text-11">
        {#each lists as l (l.label)}
          {#if l.bd}
            {@const key = LIST + l.bd}
            <dt class="text-text-2">
              <button
                class="border-0 bg-transparent p-0 text-left font-sans text-11 text-text-2"
                aria-pressed={pinned === key}
                aria-describedby={described(key)}
                onclick={toggle(key)}
                onmouseenter={enter(key)}
                onmouseleave={leave(key)}
                onfocus={enter(key)}
                onblur={leave(key)}>{l.label}</button>
            </dt>
          {:else}<dt class="text-text-2">{l.label}</dt>{/if}
          <dd class="m-0 min-w-0 break-words">{l.text || '—'}</dd>{/each}
      </dl>
    {/if}
    {#each wideTop as s (s.id)}{@render wide(s)}{/each}
    <div class="grid grid-cols-3 items-start gap-1.5">
      {#each cols as col, ci (ci)}
        <div class="flex min-w-0 flex-col gap-1.5">
          {#each col as s (s.id)}
            {@const open = s.applicable && !closed[s.id]}
            <section aria-label={pobPlain(s.title)} class={['rounded border border-line bg-surface-1', !s.applicable && 'opacity-55']}>
              <h2 class="m-0 flex">
                <button
                  class={[
                    'toned box-border flex min-w-0 grow gap-1.25 border-0 bg-transparent px-2 py-0.75 text-left font-sans text-11 font-semibold tracking-caps-tight uppercase',
                    toneClass(sectionTone(s))
                  ]}
                  aria-expanded={open}
                  disabled={!s.applicable}
                  onclick={() => {
                    const collapse = open;
                    closed = { ...closed, [s.id]: collapse };
                    oncollapse?.(s.id, collapse);
                  }}
                  >{open ? '▾' : '▸'}
                  {s.title}
                  {#if !s.applicable}<span class="font-normal text-text-3 normal-case">not applicable</span>{/if}</button>
                {#if s.applicable}<button
                    class="border-0 bg-transparent px-1.5 text-11 text-text-3 hover:text-text"
                    aria-label="Pop out {pobPlain(s.title)}"
                    title="Pop out"
                    onclick={(e) => popOut(s.id, e.currentTarget.closest('section'))}>^</button
                  >{/if}
              </h2>
              {#if open}{@render sectionBody(s)}{/if}
            </section>
          {/each}
        </div>
      {/each}
    </div>
    {#each wideBottom as s (s.id)}{@render wide(s)}{/each}
  </div>
  {#each overlaid as { o, s } (o.id)}
    {#if s}
      {@const open = !closed[s.id]}
      <div
        role="dialog"
        aria-label="{pobPlain(s.title)} (popped out)"
        class="fixed z-20 flex max-h-[80vh] w-90 flex-col rounded border border-line bg-surface-1 shadow-lg"
        style:left="{o.x}px"
        style:top="{o.y}px"
        onpointerdown={() => raise(o.id)}>
        <h2 class="m-0 flex cursor-move items-center" onpointerdown={drag(o.id)}>
          <button
            class={[
              'toned box-border flex min-w-0 grow gap-1.25 border-0 bg-transparent px-2 py-0.75 text-left font-sans text-11 font-semibold tracking-caps-tight uppercase',
              toneClass(sectionTone(s))
            ]}
            aria-expanded={open}
            onclick={() => {
              const collapse = open;
              closed = { ...closed, [s.id]: collapse };
              oncollapse?.(s.id, collapse);
            }}>{open ? '▾' : '▸'} {s.title}</button>
          <button
            class="border-0 bg-transparent px-1.5 text-13 text-text-3 hover:text-text"
            aria-label="Close {pobPlain(s.title)}"
            title="Close"
            onclick={() => (overlays = overlays.filter((x) => x.id !== o.id))}>×</button>
        </h2>
        {#if open}<div class="scroll min-h-0">{@render sectionBody(s)}</div>{/if}
      </div>
    {/if}
  {/each}
  <CalcsBreakdown
    id="calcs-breakdown"
    {title}
    {parts}
    pinned={!!pinned && shownKey === pinned}
    pinnedTitle={pinned && shownKey !== pinned ? pobPlain(rows.get(pinned)?.r.label ?? lists.find((l) => LIST + l.bd === pinned)?.label ?? '') : undefined}
    {itemtip}
    {nodeview}
    onunpin={() => {
      pinned = undefined;
      hover = undefined;
    }} />
</div>
