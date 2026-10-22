<script lang="ts">
  import GemIcon from './GemIcon.svelte';
  import Input from './Input.svelte';
  import Field from './Field.svelte';
  import EmptyState from './EmptyState.svelte';
  import { tick } from 'svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import GemRow from './GemRow.svelte';
  import Icon from './Icon.svelte';
  import { Reorder, reorderState } from './reorder';
  import Dropdown from './Dropdown.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import NumberInput from './NumberInput.svelte';
  import Checkbox from './Checkbox.svelte';
  import Button from './Button.svelte';
  import Select from './Select.svelte';
  import { tone as toneClass } from './tone';
  import { besideAnchor, setLink, socketDescription, socketLine } from './items';
  import PobText from './PobText.svelte';
  import type { GemAttr, ItemTip, SkillGemFixture, SocketGroupFixture } from './fixtures';
  import { searchGems } from './gem_search';
  import { gemWikiName } from './wiki';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import TooltipLines from './TooltipLines.svelte';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

  type CatalogueGem = { name: string; attr: GemAttr; support: boolean; searchName?: string; tags?: string[]; levelReq?: number };
  type Props = {
    group: SocketGroupFixture;
    offset: number;
    catalogue: CatalogueGem[];
    deltas: Record<string, number>;
    slots: (string | { value: string; label: string; title?: string })[];
    sortOn: boolean;
    sortStat: string;
    onchange: (patch: Partial<SocketGroupFixture>) => void;
    ongem: (gem: number, patch: Partial<SkillGemFixture>) => void;
    onadd: (gem: SkillGemFixture) => void;
    onremove: (gem: number) => void;
    onreorder: (from: number, to: number) => void;
    groupCount?: boolean;
    ongemcount?: (gem: number, count: number) => void;
    maxLevel?: (gem: number) => number;
    socketsLocked?: string;
    /* SkillsTab.lua:242-288 */
    onoptimise?: () => void;
    /* SkillsTab.lua:294-313 */
    optimiseEnabled?: boolean;
    slotLocked?: string;
    /* GemSelectControl.lua:573-606 */
    gemwhatif?: (name: string) => Promise<StatCompareSection[] | undefined>;
    /* SkillsTab.lua:911-918 */
    qualitywhatif?: (gem: number) => Promise<StatCompareSection[] | undefined>;
    /* SkillsTab.lua:948-960 */
    enabledwhatif?: (gem: number) => Promise<StatCompareSection[] | undefined>;
    /* SkillsTab.lua:868-909 */
    gemtip?: (kind: 'gem' | 'list' | 'quality', key: number | string) => Promise<{ Text: string; Separator: boolean }[] | undefined>;
    /* GemSelectControl.lua:255-441 */
    sortCache?: { canSupport: Record<string, boolean>; dps: Record<string, number>; progress?: number };
    /* GemSelectControl.lua:40-43, 496-531 */
    onpicker?: (open: boolean) => void;
    /* SkillsTab.lua:190-198 */
    slottip?: (slot: string) => Promise<ItemTip | undefined>;
    /* SkillsTab.lua:357-391 */
    sourceNote?: string;
    supporting?: [number, number][];
    /* SkillsTab.lua:294-343 */
    imbued?: { value: string; options: string[]; enabled: boolean };
    onimbued?: (name: string) => void;
    revision?: unknown;
  };
  let {
    group,
    offset,
    catalogue,
    deltas,
    slots,
    sortOn,
    sortStat,
    onchange,
    ongem,
    onadd,
    onremove,
    onreorder,
    groupCount = true,
    ongemcount,
    maxLevel,
    socketsLocked,
    onoptimise,
    optimiseEnabled,
    slotLocked,
    gemwhatif,
    qualitywhatif,
    enabledwhatif,
    gemtip,
    sortCache,
    onpicker,
    slottip,
    sourceNote,
    supporting = [],
    imbued,
    onimbued,
    revision
  }: Props = $props();
  // GemSelectControl.lua:620-627
  let hoverRow = $state<number>();
  const supports = (i: number) =>
    hoverRow !== undefined && hoverRow !== i && supporting.some(([s, a]) => (s === hoverRow && a === i) || (s === i && a === hoverRow));

  const uid = $props.id();
  // SkillsTab.lua:187-189
  const SLOT_TIP =
    'Select the item in which this skill is socketed.\nThis will allow the skill to benefit from modifiers on the item that affect socketed gems.';
  const slotOptions = $derived(slots.map((s) => (typeof s === 'string' ? { value: s, label: s } : s)));
  // SkillsTab.lua:190-198
  let slotHover = $state<{ anchor: HTMLElement; slot: string; tip?: ItemTip }>();
  const hoverSlot = (i: number | null, el: HTMLElement | null) => {
    const slot = i === null ? undefined : slotOptions[i]?.value;
    if (!slot || !el || slot === 'None' || !slottip) {
      slotHover = undefined;
      return;
    }
    slotHover = { anchor: el, slot };
    slottip(slot)
      .then((tip) => {
        if (slotHover?.slot === slot && slotHover.anchor === el) slotHover = { anchor: el, slot, tip };
      })
      .catch(() => {});
  };
  let gemBox = $state<HTMLElement>();
  const dragState = $state(reorderState());
  const drag = new Reorder(dragState, { box: () => gemBox, name: (i) => group.gems[i]?.name ?? '', onmove: (f, t) => onreorder(f, t) });
  const line = $derived(socketLine(group.sockets));
  const sockAt = (i: number) => line.colours[offset + i];
  const linkAt = (i: number) => (i > 0 && sockAt(i) && sockAt(i - 1) ? line.links[offset + i - 1] : undefined);
  const summary = $derived.by(() => {
    // SkillsTab.lua:1210-1246
    if (!group.sockets) return `Links: all ${group.gems.length} linked`;
    const links = group.gems.slice(1).map((_, i) => linkAt(i + 1) ?? false);
    const broken = links.filter((l) => !l).length;
    return broken ? `Links: ${broken + 1} link groups (${broken} broken)` : `Links: all ${group.gems.length} linked`;
  });

  let q = $state('');
  // GemSelectControl.lua:651-681, 732-741
  let only = $state<'support' | 'grants_active_skill' | undefined>();
  let open = $state(false);
  $effect(() => onpicker?.(open));
  let active = $state(-1);
  const results = $derived.by(() => {
    const gems = catalogue
      .map((g) => ({ ...g, searchName: g.searchName ?? g.name, tags: g.tags ?? [], levelReq: g.levelReq ?? 1 }))
      .filter((g) => !only || g.tags.includes(only));
    const sc = sortCache;
    const key = sc ? (g: { name: string }) => sc.dps[g.name] : (g: { name: string }) => deltas[g.name];
    const hits = searchGems(gems, q, sortOn ? key : undefined, sc ? (g) => !!sc.canSupport[g.name] : undefined);
    return hits.map((g) => ({ ...g, socketed: group.gems.some((m) => m.name === g.name) }));
  });
  const add = (i: number) => {
    const g = results[i];
    if (!g || g.socketed) return;
    onadd({ name: g.name, level: 20, quality: 0, enabled: true, support: g.support, attr: g.attr, count: 1 });
    q = '';
    open = false;
    active = -1;
  };
  const rowEls: HTMLElement[] = $state([]);
  const optEls: HTMLElement[] = $state([]);
  const OVERSCAN = 8;
  let listEl = $state<HTMLElement>();
  let headEl = $state<HTMLElement>();
  let scrollTop = $state(0);
  let rowH = $state(29);
  $effect(() => {
    const h = optEls.find(Boolean)?.offsetHeight;
    if (h) rowH = h;
  });
  const span = $derived.by(() => {
    const rows = Math.ceil((listEl?.clientHeight || 420) / rowH);
    const top = Math.floor((scrollTop - (headEl?.offsetHeight ?? 0)) / rowH) - OVERSCAN;
    const from = Math.max(0, Math.min(top, results.length - rows - OVERSCAN));
    return { from, to: Math.min(results.length, from + rows + 2 * OVERSCAN) };
  });
  const shown = $derived(results.slice(span.from, span.to));
  $effect(() => {
    if (!open) scrollTop = 0;
  });
  const reveal = (i: number) => {
    if (i < 0) return;
    const head = headEl?.offsetHeight ?? 0;
    const view = listEl?.clientHeight || 420;
    const top = head + i * rowH;
    if (top - head < scrollTop) scrollTop = top - head;
    else if (top + rowH > scrollTop + view) scrollTop = top + rowH - view;
    void tick().then(() => {
      if (listEl) listEl.scrollTop = scrollTop;
    });
  };
  let boxEl = $state<HTMLInputElement>();
  function dropFrom(list: HTMLElement, field: HTMLElement | undefined) {
    const place = () => {
      if (!field) return;
      const a = field.getBoundingClientRect();
      const pad = 8;
      const gap = 4;
      const above = a.top - gap - pad;
      const below = innerHeight - a.bottom - gap - pad;
      const max = Math.min(420, innerHeight * 0.6);
      const up = above >= Math.min(max, below);
      list.style.left = `${a.left}px`;
      list.style.width = `${a.width}px`;
      list.style.maxHeight = `${Math.max(0, Math.min(max, up ? above : below))}px`;
      list.style.top = up ? '' : `${a.bottom + gap}px`;
      list.style.bottom = up ? `${innerHeight - a.top + gap}px` : '';
    };
    const onscroll = (e: Event) => e.target !== list && place();
    place();
    addEventListener('resize', place);
    addEventListener('scroll', onscroll, true);
    return {
      update: (f: HTMLElement | undefined) => {
        field = f;
        place();
      },
      destroy: () => {
        removeEventListener('resize', place);
        removeEventListener('scroll', onscroll, true);
      }
    };
  }
  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      open = false;
      active = -1;
    } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      open = true;
      const n = results.length;
      const step = e.key === 'ArrowDown' ? 1 : -1;
      active = !n ? -1 : active < 0 ? (step > 0 ? 0 : n - 1) : (active + step + n) % n;
      reveal(active);
    } else if (e.key === 'Enter' && open) add(active < 0 ? 0 : active);
    else return;
    e.preventDefault();
  }
  type TipLine = { Text: string; Separator: boolean };
  let hoverGem = $state<string>();
  let hoverQuality = $state<number>();
  let hoverEnabled = $state<number>();
  let hoverName = $state<number>();
  const listedGem = $derived(open ? (hoverGem ?? (active >= 0 ? results[active]?.name : undefined)) : undefined);
  let whatIf = $state.raw<{ key: string; header: string; head?: TipLine[]; sections?: StatCompareSection[]; done?: boolean }>();
  let whatIfRun = 0;
  $effect(() => {
    void revision;
    const name = listedGem;
    const qi = hoverQuality;
    const ei = hoverEnabled;
    const ni = hoverName;
    let key: string | undefined;
    let run: (() => Promise<StatCompareSection[] | undefined>) | undefined;
    let tip: (() => Promise<TipLine[] | undefined>) | undefined;
    let header = '';
    if (name !== undefined && (gemwhatif || gemtip) && !results.find((r) => r.name === name)?.socketed) {
      key = `gem:${name}`;
      header = 'Selecting this gem will give you:';
      run = gemwhatif && (() => gemwhatif(name));
      tip = gemtip && (() => gemtip('list', name));
    } else if (ei !== undefined && enabledwhatif && group.gems[ei]) {
      key = `enabled:${ei}`;
      header = group.gems[ei].enabled ? 'Disabling this gem will give you:' : 'Enabling this gem will give you:';
      run = () => enabledwhatif(ei);
    } else if (qi !== undefined && (qualitywhatif || gemtip)) {
      key = `quality:${qi}`;
      header = 'Setting to 20 quality will give you:';
      run = qualitywhatif && (() => qualitywhatif(qi));
      tip = gemtip && (() => gemtip('quality', qi));
    } else if (ni !== undefined && gemtip) {
      key = `tip:${ni}`;
      tip = () => gemtip('gem', ni);
    }
    const n = ++whatIfRun;
    if (!key) {
      whatIf = undefined;
      return;
    }
    const k = key;
    whatIf = { key: k, header };
    Promise.all([tip?.(), run?.()])
      .then(([head, sections]) => {
        if (n === whatIfRun) whatIf = { key: k, header, head, sections, done: true };
      })
      .catch(() => {});
  });
  const tipAnchor = $derived(
    whatIf?.key.startsWith('gem:')
      ? optEls[results.findIndex((r) => r.name === listedGem)]
      : rowEls[(whatIf?.key.startsWith('quality:') ? hoverQuality : whatIf?.key.startsWith('enabled:') ? hoverEnabled : hoverName) ?? -1]
  );
  const deltaText = (r: (typeof results)[number]) =>
    r.socketed
      ? 'already socketed'
      : deltas[r.name] !== undefined
        ? `${deltas[r.name] > 0 ? '▲ +' : deltas[r.name] < 0 ? '▼ ' : '= '}${deltas[r.name]}% ${sortStat}`
        : r.support
          ? '—'
          : 'active skill';
</script>

<section aria-label="Selected group" class="panel scroll relative flex max-h-full min-w-0 flex-[0_1_1000px] flex-col gap-2.25 self-start p-3">
  <div class="flex flex-wrap items-end gap-2">
    <Field label="Label" class="w-42.5 gap-0.5 text-11">
      <Input size="sm" value={group.label} onchange={(e: Event & { currentTarget: HTMLInputElement }) => onchange({ label: e.currentTarget.value })} /></Field>
    <FloatingLabel label="Socketed in"
      ><Dropdown
        class="h-7.5 min-w-32"
        options={slotOptions}
        disabled={!!slotLocked}
        title={slotLocked ?? SLOT_TIP}
        value={group.slot || 'None'}
        onhover={hoverSlot}
        onchange={(v: string) => onchange({ slot: v === 'None' ? '' : v })} /></FloatingLabel>
    {#if slotHover?.tip}
      <FloatingItemTip
        anchor={slotHover.anchor}
        id="{uid}-slot-tip"
        item={slotHover.slot}
        base=""
        rarity="normal"
        mods={[]}
        impactTitle=""
        impact={[]}
        tooltip={slotHover.tip} />
    {/if}
    <Checkbox
      compact
      class="h-7.5"
      checked={group.enabled}
      onchange={(e: Event & { currentTarget: HTMLInputElement }) => onchange({ enabled: e.currentTarget.checked })}>Enabled</Checkbox>
    <Checkbox
      compact
      class="h-7.5"
      checked={group.inFull}
      onchange={(e: Event & { currentTarget: HTMLInputElement }) => onchange({ inFull: e.currentTarget.checked })}>Include in Full DPS</Checkbox>
    {#if groupCount}
      <label class="check-label h-7.5 text-text-2"
        >Count <NumberInput size="xs" class="w-14" value={group.count} min={1} max={20} onchange={(count) => onchange({ count })} /></label>
    {/if}
  </div>
  {#if imbued}
    <div class="flex items-center gap-2 text-12">
      <span class={['toned', toneClass('crafted')]}>Imbued Support:</span>
      <Select
        size="xs"
        class="w-62.5"
        aria-label="Imbued Support"
        disabled={!imbued.enabled}
        value={imbued.value}
        options={[{ value: '', label: '' }, ...imbued.options.map((o) => ({ value: o, label: o }))]}
        onchange={(e: Event) => onimbued?.((e.currentTarget as HTMLSelectElement).value)} />
      <Button size="xs" disabled={!imbued.enabled} title="Remove this imbued support." aria-label="Remove this imbued support." onclick={() => onimbued?.('')}
        >x</Button>
    </div>
  {/if}
  <div class="flex items-center gap-2.5 text-12 text-text-2">
    <!-- SkillsTab.lua:228-237 -->
    {#if group.sockets}<span>Item sockets: <span class="font-mono"><PobText text={socketDescription(group.sockets)} /></span></span>{/if}
    <Button
      size="xs"
      class="text-11"
      disabled={(!onoptimise && !!socketsLocked) || optimiseEnabled === false}
      title={onoptimise
        ? "Rebuild the item's sockets to match the groups assigned to it."
        : (socketsLocked ?? "Rebuild the item's sockets to match the socket groups assigned to it")}
      onclick={onoptimise}>Match item sockets to gems</Button>
  </div>
  {#if sourceNote}<p class="m-0 text-12 whitespace-pre-line text-text-2"><PobText text={sourceNote} /></p>{/if}
  <div role="group" aria-label="Gems" class="relative flex flex-col text-13" bind:this={gemBox}>
    <div
      aria-hidden="true"
      class="grid grid-cols-[1.125rem_2.75rem_minmax(1rem,1fr)_repeat(23,minmax(0,1fr))] gap-1.5 border-b border-line px-1.5 py-1 text-10 tracking-caps text-text-2 max-phone:grid-cols-[2.75rem_repeat(24,minmax(0,1fr))]">
      <span class="max-phone:hidden"></span><span>SOCKET</span><span class="max-phone:hidden">ON</span><span class={ongemcount ? 'col-span-10' : 'col-span-13'}
        >GEM</span
      ><span class="col-span-4">LEVEL</span><span class="col-span-4">QUAL.</span>{#if ongemcount}<span class="col-span-3">COUNT</span>{/if}<span
        class="col-span-2"></span>
    </div>
    {#each group.gems as gem, i (i)}
      <div
        {...drag.row(i)}
        class={['relative', supports(i) && 'bg-code-link/25']}
        data-supporting={supports(i) || undefined}
        bind:this={rowEls[i]}
        onmouseenter={() => (hoverRow = hoverName = i)}
        onmouseleave={() => (hoverRow = hoverName = hoverRow === i ? undefined : hoverRow)}>
        <GemRow
          n={offset + i + 1}
          name={gem.name}
          level={gem.level}
          quality={gem.quality}
          tone={gem.attr}
          socket={sockAt(i) ?? 'W'}
          support={gem.support}
          linked={linkAt(i)}
          onlinkchange={(linked: boolean) => onchange({ sockets: setLink(group.sockets, offset + i - 1, linked) })}
          enabled={gem.enabled}
          onenabled={(enabled) => ongem(i, { enabled })}
          onlevel={(level) => ongem(i, { level })}
          onquality={(quality) => ongem(i, { quality })}
          onremove={() => onremove(i)}
          count={gem.count}
          oncount={ongemcount && ((count: number) => ongemcount(i, count))}
          maxLevel={maxLevel?.(i)}
          error={gem.error}
          globals={gem.globals}
          onqualityhover={(qualitywhatif || gemtip) && ((on: boolean) => (hoverQuality = on ? i : hoverQuality === i ? undefined : hoverQuality))}
          onenabledhover={enabledwhatif && ((on: boolean) => (hoverEnabled = on ? i : hoverEnabled === i ? undefined : hoverEnabled))}
          onglobal={(index, on) => ongem(i, { globals: (gem.globals ?? []).map((g, k) => (k === index ? { ...g, on } : g)) })}
          linkLocked={socketsLocked}>
          {#snippet grip()}<span aria-hidden="true" class="flex text-text-3"><Icon name="grip" /></span>{/snippet}
        </GemRow>
      </div>
    {:else}
      <EmptyState inset="row" text="No gems in this group yet; add one below." />
    {/each}
    <div class="flex gap-3.5 px-1 pt-1.5 text-11 text-text-2">
      <span>Ring = the item socket’s colour; the filled circle is the gem in it.</span>
      <span class="ml-auto">{summary}</span>
    </div>
    <div data-reorder-live aria-live="assertive" class="sr-only">{dragState.message}</div>
    <span id={drag.hint} class="sr-only">{drag.hintText}</span>
  </div>
  <div class="relative">
    <Input
      type="text"
      bind:ref={boxEl}
      role="combobox"
      size="md"
      class="w-full border-dashed border-accent px-2.5"
      placeholder="Add a gem — type to filter, Esc to close"
      aria-label="Add a gem"
      aria-controls="{uid}-results"
      aria-autocomplete="list"
      aria-expanded={open}
      aria-activedescendant={open && active >= 0 ? `${uid}-opt-${active}` : undefined}
      data-wiki={open && results.length ? gemWikiName(results[Math.max(active, 0)].name, results[Math.max(active, 0)].support) : undefined}
      bind:value={q}
      oninput={() => {
        open = true;
        active = -1;
      }}
      onfocus={() => (open = true)}
      onblur={() => (open = false)}
      {onkeydown} />
    <div class="absolute top-1 right-1 flex gap-0.5">
      {#each [['grants_active_skill', 'A', 'Only show Active gems'], ['support', 'S', 'Only show Support gems']] as [tag, letter, tip] (tag)}
        <button
          type="button"
          class={['h-6.5 w-5 rounded border border-line bg-surface-1 font-mono text-11', only === tag ? 'text-text' : 'text-text-3']}
          title={tip}
          aria-label={tip}
          aria-pressed={only === tag}
          onmousedown={(e) => e.preventDefault()}
          onclick={() => {
            only = only === tag ? undefined : (tag as typeof only);
            open = true;
            active = -1;
          }}>{letter}</button>
      {/each}
    </div>
    {#if open}
      <div
        role="listbox"
        id="{uid}-results"
        aria-label="Gem results"
        bind:this={listEl}
        use:dropFrom={boxEl}
        onscroll={(e) => (scrollTop = e.currentTarget.scrollTop)}
        class="fixed z-dock overflow-y-auto overscroll-contain rounded-md border border-line-strong bg-surface-2 shadow-pop">
        <div bind:this={headEl} class="sticky top-0 z-raised border-b border-line bg-surface-2 px-2.5 py-1.5 text-11 text-text-2">
          {results.length} gems{sortOn ? ` · sorted by ${sortStat}` : ''}{#if sortCache?.progress !== undefined && sortCache.progress < 100}<span
              class="float-right"
              role="status"
              aria-label="Sort progress">Sorting {sortCache.progress}%</span
            >{/if}
        </div>
        {#if span.from}<div aria-hidden="true" style:height="{span.from * rowH}px"></div>{/if}
        {#each shown as r, k (r.name)}
          {@const i = span.from + k}
          <div
            bind:this={optEls[i]}
            role="option"
            id="{uid}-opt-{i}"
            data-wiki={gemWikiName(r.name, r.support)}
            tabindex="-1"
            aria-selected={i === active}
            aria-disabled={r.socketed}
            class="grid grid-cols-12 gap-1.5 border-b border-surface-3 px-2.5 py-1.25 text-13 aria-disabled:opacity-55 aria-selected:bg-row-selected"
            onmouseenter={() => (hoverGem = r.name)}
            onmouseleave={() => (hoverGem = hoverGem === r.name ? undefined : hoverGem)}
            onmousedown={(e) => {
              e.preventDefault();
              add(i);
            }}>
            <span class={['toned col-span-6 truncate', toneClass(r.attr)]}
              ><GemIcon name={r.name} size={16} decorative class="mr-1 align-[-3px]" />{r.name}</span>
            <span class="col-span-2 text-11 text-text-2">{r.support ? 'Support' : 'Active'}</span>
            <span class={['col-span-4 text-right font-mono text-12', deltas[r.name] > 0 && !r.socketed ? 'text-better' : 'text-text-2']}>{deltaText(r)}</span>
          </div>
        {:else}
          <EmptyState inset="row" text="No gems match “{q}”." />
        {/each}
        {#if span.to < results.length}<div aria-hidden="true" style:height="{(results.length - span.to) * rowH}px"></div>{/if}
      </div>
    {/if}
  </div>
  {#if whatIf && !(whatIf.done && !whatIf.head?.length && !whatIf.sections)}
    <div data-floating class="pointer-events-none fixed top-(--y) left-(--x) z-float w-max max-w-[480px]" use:besideAnchor={tipAnchor}>
      <div role="status" aria-label="What-if" class="tooltip rounded-md px-3 py-2.5 shadow-tip" data-whatif={whatIf.key}>
        {#if !whatIf.done}<span class="note">Calculating…</span>{:else}
          {#if whatIf.head?.length}<TooltipLines lines={whatIf.head} center={!whatIf.key.startsWith('quality:')} label="Gem tooltip" />{/if}
          {#if whatIf.sections}<ImpactBlocks blocks={[{ header: whatIf.header, sections: whatIf.sections }]} />{/if}
        {/if}
      </div>
    </div>
  {/if}
</section>
