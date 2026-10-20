<script lang="ts">
  import Disclosure from './Disclosure.svelte';
  import Notice from './Notice.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import PobText from './PobText.svelte';
  import { tick } from 'svelte';
  import type { calculator } from '$lib/types';
  import { colorCodes } from '$lib/display/colors';
  import { VALUE_COLOR } from '$lib/display/format_stat';
  import type { Sidebar } from '$lib/display/sidebar';
  import type { StatLine } from '$lib/display/stat_list';
  import BreakdownTooltip from './BreakdownTooltip.svelte';
  import BreakdownSteps from './BreakdownSteps.svelte';
  import RadiusVisual from '$lib/components/radius/RadiusVisual.svelte';
  import type { SidebarSection } from '$lib/display/sidebar_breakdown';
  import Button from './Button.svelte';
  import Select from './Select.svelte';
  import StatGauge from './StatGauge.svelte';
  import StatGroup from './StatGroup.svelte';
  import type { Tone } from './tone';
  import { MINION_ITEM_DROP, draggedItem } from './item_drag';
  import { dropZone } from './reorder';

  type Control = 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill';
  type Breakdown = { formula?: string; groups: { title?: string; rows: { src: string; val: string; tone?: Tone }[] }[] };
  type Props = {
    sidebar: Sidebar;
    socketGroups: string[];
    mainSocketGroup: number;
    controls?: calculator.MainSkillControls;
    collapsed?: boolean;
    /** fills its container (the phone stats sheet): no rail, no resize */
    fill?: boolean;
    pinned?: string[];
    breakdown?: (line: StatLine) => Breakdown | undefined;
    /* Build.lua:1764-1908 */
    steps?: (line: StatLine) => Promise<SidebarSection[] | undefined> | undefined;
    onmaingroup?: (index: number) => void;
    oncontrol?: (field: Control, value: number | undefined) => void;
    /* Build.lua:552-554 */
    onspectres?: () => void;
    onbreakdown?: (line: StatLine) => void;
  };
  let {
    sidebar,
    socketGroups,
    mainSocketGroup,
    controls,
    collapsed = $bindable(false),
    fill = false,
    pinned = $bindable([]),
    breakdown,
    steps,
    onmaingroup,
    oncontrol,
    onspectres,
    onbreakdown
  }: Props = $props();

  const uid = $props.id();

  type Entry = { id: string; line: StatLine; skills?: { line: StatLine; details: StatLine[] }[] };
  const isSkillGroup = (g: StatLine[]) => g.length > 0 && g.every((l) => l.detail || l.labelColor === colorCodes.CUSTOM);
  const sections = $derived(
    sidebar.sections.map((s) => {
      const seen: Record<string, number> = {};
      const groups: Entry[][] = [];
      s.groups.forEach((g, gi) => {
        if (gi > 0 && isSkillGroup(g) && s.groups[gi - 1].some((l) => l.label === 'Full DPS')) return;
        const next = s.groups[gi + 1];
        groups.push(
          g.map((line) => {
            const n = (seen[line.label] = (seen[line.label] ?? 0) + 1);
            const entry: Entry = { id: `${s.header ?? ''}|${line.label}|${n}`, line };
            if (line.label === 'Full DPS' && next && isSkillGroup(next)) {
              entry.skills = [];
              for (const l of next) {
                if (l.detail) entry.skills.at(-1)?.details.push(l);
                else entry.skills.push({ line: l, details: [] });
              }
            }
            return entry;
          })
        );
      });
      return { title: s.header ?? 'Player', info: s.info, groups };
    })
  );
  const entries = $derived(sections.flatMap((s) => s.groups.flat()));

  const valueTone = (l: StatLine) => (l.valueColor === colorCodes.NEGATIVE ? 'tone-worse' : l.valueColor === VALUE_COLOR ? '' : 'tone-text-3');
  const tick3 = (l: StatLine) => (l.labelColor === VALUE_COLOR ? undefined : l.labelColor);

  const togglePin = (id: string) => (pinned = pinned.includes(id) ? pinned.filter((p) => p !== id) : [...pinned, id]);

  const short = (label: string) =>
    label
      .replace(/^(Total|Effective|Average) /, '')
      .split(' ')[0]
      .slice(0, 7);
  const compact = (value: string) => {
    const m = /^([+-]?)([\d,]+(?:\.\d+)?)(%?)$/.exec(value);
    if (!m) return value;
    const n = parseFloat(m[2].replace(/,/g, ''));
    const [d, s] = n >= 1e9 ? [1e9, 'B'] : n >= 1e6 ? [1e6, 'M'] : n >= 1e4 ? [1e3, 'k'] : [1, ''];
    return m[1] + (d === 1 ? m[2] : (n / d).toFixed(n / d >= 100 ? 0 : 1) + s) + m[3];
  };
  const pins = $derived(pinned.map((id) => entries.find((e) => e.id === id)).filter((e): e is Entry => !!e));

  let fullDpsOpen = $state(true);

  let aside = $state<HTMLElement>();
  let hover = $state.raw<{ entry: Entry; y: number }>();
  const show = (entry: Entry, e: Event) => {
    const row = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const box = aside!.getBoundingClientRect();
    hover = { entry, y: row.top - box.top + row.height / 2 };
  };
  const hide = (entry: Entry) => {
    if (hover?.entry === entry) hover = undefined;
  };
  // Build.lua:544-551
  let minionTipY = $state<number>();
  let minionEl = $state<HTMLElement>();
  const minionTip = $derived(controls?.Minions?.[controls.Minion - 1]?.Tooltip ?? []);
  // Build.lua:534-543
  const minionSetId = (c: calculator.MainSkillControls | undefined) => (c ? (c.Minions?.[c.Minion - 1]?.ItemSetID ?? 0) : 0);
  const showMinionTip = (e: Event) => {
    const row = (e.currentTarget as HTMLElement).getBoundingClientRect();
    minionTipY = row.top - aside!.getBoundingClientRect().top;
  };
  const tip = $derived(hover ? { ...(breakdown?.(hover.entry.line) ?? { groups: [] }), line: hover.entry.line } : undefined);
  const STEPS_DELAY = 150;
  let tipSteps = $state.raw<{ entry: Entry; sections?: SidebarSection[] }>();
  $effect(() => {
    const entry = hover?.entry;
    void sidebar;
    if (!entry || !steps) return;
    let live = true;
    const timer = setTimeout(() => {
      steps(entry.line)
        ?.then((shown) => {
          if (live) tipSteps = { entry, sections: shown };
        })
        .catch(() => {});
    }, STEPS_DELAY);
    return () => {
      live = false;
      clearTimeout(timer);
    };
  });
  let tipH = $state(0);
  const tipTop = $derived(hover && aside ? Math.max(8, Math.min(hover.y - tipH / 2, aside.clientHeight - tipH - 8)) : 0);

  const gemWritable = $derived(!!controls && controls.GemIndex >= 0);
  const statuses = $derived(sidebar.warnings.filter((w) => w.kind === 'cost' || w.kind === 'skill'));

  let expandBtn = $state<HTMLButtonElement>();
  let collapseBtn = $state<HTMLButtonElement>();
  async function setCollapsed(v: boolean) {
    collapsed = v;
    hover = undefined;
    await tick();
    (v ? expandBtn : collapseBtn)?.focus();
  }

  const selected = (e: Event) => parseInt((e.currentTarget as HTMLSelectElement).value);
  const digits = (e: Event) => {
    const raw = (e.currentTarget as HTMLInputElement).value.replace(/\D/g, '');
    return raw === '' ? undefined : parseInt(raw);
  };
  const opts = (names: string[] | undefined) => (names ?? []).map((label, i) => ({ value: String(i + 1), label }));

  const WIDTH_MIN = 240;
  const WIDTH_MAX = 640;
  const WIDTH_KEY = 'go-pob:sidebar-width';
  const storedWidth = () => {
    try {
      return Number(localStorage.getItem(WIDTH_KEY)) || 320;
    } catch {
      return 320;
    }
  };
  let width = $state(Math.min(WIDTH_MAX, Math.max(WIDTH_MIN, storedWidth())));
  let resizing: { x: number; w: number } | undefined;
  const setWidth = (w: number) => {
    width = Math.round(Math.min(WIDTH_MAX, Math.max(WIDTH_MIN, w)));
    try {
      localStorage.setItem(WIDTH_KEY, String(width));
    } catch {
      // empty
    }
  };
</script>

{#if collapsed && !fill}
  <aside
    aria-label="Main skill and stats (collapsed)"
    class="relative box-border flex h-full max-h-screen w-15 shrink-0 flex-col items-center gap-3 border-r border-line bg-surface-1 px-1.5 py-3 font-sans text-text">
    <button
      bind:this={expandBtn}
      class="btn btn-icon btn-sm size-7.5"
      onclick={() => setCollapsed(false)}
      aria-label="Expand the sidebar"
      title="Expand the sidebar"
      aria-expanded="false">›</button>
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      class="flex min-h-0 w-full flex-1 flex-col items-center gap-3 overflow-y-auto [scrollbar-width:none]"
      role="group"
      aria-label="Pinned stats"
      tabindex="0"
      data-rail-pins>
      {#each pins as p (p.id)}
        <div
          class="flex w-full shrink-0 flex-col items-center gap-0.5 border-t border-line pt-2"
          style:--tone={tick3(p.line)}
          title="{p.line.label}: {p.line.value}">
          <span class="sr-only">{p.line.label}: {p.line.value}</span>
          <span class="h-0.75 w-5 rounded-xs bg-(--tone,var(--color-line-strong))" aria-hidden="true"></span>
          <span class="text-10 text-text-3" aria-hidden="true">{short(p.line.label)}</span>
          <span class={['font-mono text-11 toned', valueTone(p.line) || 'tone-text']} aria-hidden="true">{compact(p.line.value)}</span>
        </div>
      {:else}
        <span class="text-14 text-text-3" title="Star a stat in the full sidebar to pin it here" aria-hidden="true">☆</span>
      {/each}
    </div>
  </aside>
{:else}
  <aside
    bind:this={aside}
    aria-label="Main skill and stats"
    class={[
      'relative box-border flex h-full max-h-screen shrink-0 flex-col overflow-visible bg-surface-1 font-sans text-text',
      fill ? 'w-full max-w-none' : 'max-w-[50vw] border-r border-line'
    ]}
    style:width={fill ? undefined : `${width}px`}>
    <div class="flex flex-col gap-2.5 border-b border-line px-4 py-3.5">
      <div class="flex items-end gap-2">
        <FloatingLabel label="Main skill" class="min-w-0 grow"
          ><Select
            size="md"
            class="w-full min-w-0 text-14"
            options={socketGroups.map((label, i) => ({ value: String(i), label }))}
            value={String(mainSocketGroup)}
            onchange={(e: Event) => onmaingroup?.(selected(e))} /></FloatingLabel>
        {#if !fill}<button
            bind:this={collapseBtn}
            class="btn btn-icon btn-md shrink-0"
            onclick={() => setCollapsed(true)}
            aria-label="Collapse the sidebar to a rail"
            title="Collapse to a rail of pinned stats"
            aria-expanded="true">‹</button
          >{/if}
      </div>
      {#if controls}
        {#if (controls.Skills?.length ?? 0) > 1}
          <FloatingLabel label="Active skill"
            ><Select
              size="md"
              options={opts(controls.Skills)}
              value={String(controls.MainActiveSkill)}
              disabled={!controls.SkillEnabled}
              onchange={(e: Event) => oncontrol?.('MainActiveSkill', selected(e))} /></FloatingLabel>
        {/if}
        {#if controls.PartShown}
          <FloatingLabel label="Skill part"
            ><Select
              size="md"
              options={opts(controls.Parts)}
              value={String(controls.SkillPart)}
              disabled={!gemWritable}
              onchange={(e: Event) => oncontrol?.('SkillPart', selected(e))} /></FloatingLabel>
        {/if}
        {#if controls.StageShown || controls.MineShown}
          <div class="grid grid-cols-2 gap-2">
            {#if controls.StageShown}
              <label class="field"
                >Stages<input
                  class="input input-md font-mono"
                  inputmode="numeric"
                  value={controls.StageCount}
                  disabled={!gemWritable}
                  oninput={(e: Event) => oncontrol?.('StageCount', digits(e))} /></label>
            {/if}
            {#if controls.MineShown}
              <label class="field"
                >Active mines<input
                  class="input input-md font-mono"
                  inputmode="numeric"
                  value={controls.MineCount}
                  disabled={!gemWritable}
                  oninput={(e: Event) => oncontrol?.('MineCount', digits(e))} /></label>
            {/if}
          </div>
        {/if}
        {#if controls.MinionLibraryShown && onspectres}
          <Button size="md" class="self-start" onclick={onspectres}>Manage Spectres...</Button>
        {/if}
        {#if controls.MinionShown || controls.MinionSkillShown}
          <div class="grid grid-cols-2 gap-2">
            {#if controls.MinionShown}
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="min-w-0"
                bind:this={minionEl}
                data-minion-select
                onmouseenter={showMinionTip}
                onmouseleave={() => (minionTipY = undefined)}
                onfocusin={showMinionTip}
                onfocusout={() => (minionTipY = undefined)}
                use:dropZone={{
                  accepts: (d) => !!minionSetId(controls) && !!draggedItem(d),
                  drop: (d, _hit, e) => {
                    const setId = minionSetId(controls);
                    const drag = draggedItem(d);
                    // Build.lua:541-543
                    if (setId && drag)
                      minionEl?.dispatchEvent(
                        new CustomEvent(MINION_ITEM_DROP, { bubbles: true, detail: { setId, drag, second: e.shiftKey, uses: controls?.MinionUses ?? [] } })
                      );
                  }
                }}>
                <FloatingLabel label="Minion" class="min-w-0"
                  ><Select
                    size="md"
                    class="min-w-0"
                    options={(controls.Minions ?? []).map((m, i) => ({ value: String(i + 1), label: m.Label }))}
                    value={String(controls.Minion)}
                    disabled={!controls.MinionEnabled}
                    aria-describedby={minionTipY !== undefined && minionTip.length ? `${uid}-minion` : undefined}
                    onchange={(e: Event) => oncontrol?.('Minion', selected(e))} /></FloatingLabel>
              </div>
            {/if}
            {#if controls.MinionSkillShown}
              <FloatingLabel label="Minion skill" class="min-w-0"
                ><Select
                  size="md"
                  class="min-w-0"
                  options={opts(controls.MinionSkills)}
                  value={String(controls.MinionSkill)}
                  disabled={!controls.MinionSkillEnabled}
                  onchange={(e: Event) => oncontrol?.('MinionSkill', selected(e))} /></FloatingLabel>
            {/if}
          </div>
        {/if}
      {/if}
      {#each statuses as w, i (i)}
        <Notice kind="warn" size="sm" class="text-worse">
          <span aria-hidden="true">⚠</span>
          {w.text}
        </Notice>
      {/each}
    </div>

    <div role="region" aria-label="Build stats" class="scroll relative flex grow flex-col gap-4 px-4 py-2.5">
      {#if sidebar.info?.length}<div class="flex flex-col items-center text-12" aria-label="Main skill info">
          {#each sidebar.info as l, i (i)}<span><PobText text={l} /></span>{/each}
        </div>{/if}
      {#each sections as s (s.title)}
        <StatGroup title={s.title}>
          {#if s.info?.length}<div class="flex flex-col items-center text-12" aria-label="Skill info">
              {#each s.info as l, i (i)}<span><PobText text={l} /></span>{/each}
            </div>{/if}
          <div class="flex flex-col gap-3.5">
            {#each s.groups as g, gi (gi)}
              <div data-stat-group class="flex flex-col">
                {#each g as e, i (e.id)}
                  {@const on = pinned.includes(e.id)}
                  <div class={['group/row stat-row stat-fit', i % 2 && 'bg-bar']} style:--tone={tick3(e.line)}>
                    {#if e.line.gauge}<StatGauge {...e.line.gauge} />{/if}
                    <button
                      class="flex min-w-0 grow cursor-default items-center border-0 bg-transparent p-0 pl-2.25 text-left [font:inherit] text-inherit"
                      aria-describedby={hover?.entry === e ? `${uid}-bd` : undefined}
                      onmouseenter={(ev) => show(e, ev)}
                      onmouseleave={() => hide(e)}
                      onfocus={(ev) => show(e, ev)}
                      onblur={() => hide(e)}
                      onclick={() => onbreakdown?.(e.line)}>
                      <span class="stat-label w-auto max-w-39.5 grow pl-0 text-clip"
                        >{e.line.label}{#if e.line.labelNote}<span class="text-worse">{e.line.labelNote}</span>{/if}</span
                      ><span class="sr-only">: </span>
                      <span class={['stat-value stat-fade toned', valueTone(e.line) || 'tone-text']}
                        >{e.line.value}{#if e.line.note}<span class="text-11 text-text-2">{e.line.note}</span>{/if}</span>
                    </button>
                    <button
                      class="relative z-raised mr-1 ml-auto shrink-0 border-0 bg-transparent p-0 text-12 leading-none text-text-3 opacity-0 group-hover/row:opacity-100 group-focus-within/row:opacity-100 focus-visible:opacity-100 aria-pressed:text-accent aria-pressed:opacity-100 max-phone:opacity-100"
                      aria-pressed={on}
                      onclick={() => togglePin(e.id)}
                      aria-label={on ? `Unpin ${e.line.label}` : `Pin ${e.line.label} to the collapsed rail`}
                      title={on ? `Unpin ${e.line.label}` : `Pin ${e.line.label} to the collapsed rail`}>{on ? '★' : '☆'}</button>
                  </div>
                  {#if e.skills}
                    <Disclosure class="stat-group-head pl-2.25 font-normal tracking-normal normal-case" bind:open={fullDpsOpen}
                      >Full DPS by skill ({e.skills.length})</Disclosure>
                    {#if fullDpsOpen}
                      <ul class="m-0 flex list-none flex-col p-0 pl-4.5 text-12" aria-label="Full DPS by skill">
                        {#each e.skills as k, ki (ki)}
                          <li class="flex flex-col leading-5">
                            <span class="stat-fit flex justify-between"
                              ><span class="min-w-0 grow overflow-hidden pr-2 whitespace-nowrap text-text-2"
                                >{k.line.label}{#if k.line.labelNote}<span class="text-worse">{k.line.labelNote}</span>{/if}</span
                              ><span class="sr-only">:&nbsp;</span><span class="stat-fade font-mono">{k.line.value}</span></span>
                            {#each k.details as d, di (di)}<span class="note-sm pl-2">{d.value}</span>{/each}
                          </li>
                        {/each}
                      </ul>
                    {/if}
                  {/if}
                {/each}
              </div>
            {/each}
          </div>
        </StatGroup>
      {/each}
    </div>

    {#if minionTipY !== undefined && minionTip.length}
      <div
        id="{uid}-minion"
        role="tooltip"
        class="tooltip-float tooltip z-drag flex flex-col gap-0.5 rounded-lg px-3 py-2.5 text-12 whitespace-nowrap shadow-bd"
        style:--x="calc(100% + 6px)"
        style:--y="{minionTipY}px">
        {#each minionTip as line, i (i)}{#if line}<span><PobText text={line} /></span>{:else}<hr class="my-1 border-line" />{/if}{/each}
      </div>
    {/if}
    {#if hover && tip}
      <div class="tooltip-float z-drag shadow-bd" style:--x="calc(100% + 6px)" style:--y="{tipTop}px" bind:clientHeight={tipH}>
        <BreakdownTooltip
          id="{uid}-bd"
          title={tip.line.label}
          value={tip.line.value + (tip.line.note ?? '')}
          formula={tip.formula}
          groups={tip.groups}
          footer="Click for the full breakdown on the Calcs page">
          {#each tipSteps?.entry === hover.entry ? (tipSteps.sections ?? []) : [] as s, i (i)}
            {#if s.label}<span class="mt-1 text-10 tracking-caps text-text-3">{s.label}</span>{/if}
            {#if s.bd}
              {#if s.bd.lines.length}<div class="mt-1 text-12"><BreakdownSteps lines={s.bd.lines} /></div>{/if}
              {#if s.bd.radius}<RadiusVisual radius={s.bd.radius} />{/if}
              {#if s.bd.columns && s.bd.rows?.length}
                <div role="table" aria-label="Sources" class="grid grid-cols-[repeat(var(--n),max-content)] gap-x-2 text-11" style:--n={s.bd.columns.length}>
                  <div role="row" class="col-span-full grid grid-cols-subgrid border-b border-line text-10 text-text-3">
                    {#each s.bd.columns as c (c.key)}<span role="columnheader">{c.label}</span>{/each}
                  </div>
                  {#each s.bd.rows as r, j (j)}
                    <div role="row" class="col-span-full grid grid-cols-subgrid">
                      {#each s.bd.columns as c (c.key)}<span role="cell"><PobText text={r[c.key] ?? ''} /></span>{/each}
                    </div>
                  {/each}
                </div>
              {/if}
            {/if}
            {#each s.mods ?? [] as m, j (j)}
              <div class="kv gap-2.5 text-11" data-mod-row>
                <span class="truncate"><PobText text={m.SourceName || m.Source} /></span><span class="font-mono"><PobText text={m.Value} /></span>
              </div>
            {/each}
          {/each}
        </BreakdownTooltip>
      </div>
    {/if}
    {#if !fill}
      <!-- eslint-disable-next-line svelte/no-unused-svelte-ignore -->
      <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
      <div
        role="separator"
        aria-label="Resize the sidebar"
        aria-orientation="vertical"
        aria-valuenow={width}
        aria-valuemin={WIDTH_MIN}
        aria-valuemax={WIDTH_MAX}
        tabindex="0"
        class="absolute top-0 -right-0.75 bottom-0 z-overlay w-1.5 cursor-col-resize hover:bg-surface-3 focus-visible:bg-surface-3"
        onkeydown={(e: KeyboardEvent) => {
          const d = e.key === 'ArrowLeft' ? -16 : e.key === 'ArrowRight' ? 16 : 0;
          if (!d) return;
          e.preventDefault();
          setWidth(width + d);
        }}
        onpointerdown={(e: PointerEvent) => {
          if (e.button > 0) return;
          resizing = { x: e.clientX, w: width };
          (e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId);
        }}
        onpointermove={(e: PointerEvent) => {
          if (resizing) setWidth(resizing.w + e.clientX - resizing.x);
        }}
        onpointerup={() => (resizing = undefined)}
        onpointercancel={() => (resizing = undefined)}>
      </div>
    {/if}
  </aside>
{/if}
