<script lang="ts">
  import { isTypingTarget } from './keys';
  import { untrack } from 'svelte';
  import TooltipLines from './TooltipLines.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import NumberInput from './NumberInput.svelte';
  import Checkbox from './Checkbox.svelte';
  import Button from './Button.svelte';
  import SkillGroupEditor from './SkillGroupEditor.svelte';
  import Icon from './Icon.svelte';
  import Dialog from './Dialog.svelte';
  import { Reorder, reorderState } from './reorder';
  import { socketLine } from './items';
  import type { GemAttr, ItemTip, SkillGemFixture, SocketGroupFixture } from './fixtures';
  import type { GemOptions, SkillSetOp } from './skills';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

  type Props = {
    sets: { title: string; groups: (SocketGroupFixture & { display?: string })[] }[];
    set: number;
    selected: number;
    mainGroup: number;
    catalogue: { name: string; attr: GemAttr; support: boolean; searchName?: string; tags?: string[]; levelReq?: number }[];
    deltas: Record<string, number>;
    slots: (string | { value: string; label: string; title?: string })[];
    sortOn?: boolean;
    sortStat?: string;
    onset?: (i: number) => void;
    /* SkillsTab.lua:1415-1422 */
    onsets?: (op: SkillSetOp) => void;
    onselect: (i: number) => void;
    ongroup: (i: number, patch: Partial<SocketGroupFixture>) => void;
    ongem: (i: number, gem: number, patch: Partial<SkillGemFixture>) => void;
    onaddgem: (i: number, gem: SkillGemFixture) => void;
    onremovegem: (i: number, gem: number) => void;
    onnewgroup: () => void;
    onreordergroups?: (from: number, to: number) => void;
    onreordergems: (from: number, to: number) => void;
    setLocked?: string;
    groupsLocked?: string;
    onmain?: (i: number) => void;
    ondeletegroup?: (i: number) => void;
    ondeleteall?: () => void;
    onsort?: (on: boolean, stat: string) => void;
    /* SkillListControl.lua:148-152 */
    oncopygroup?: (i: number) => void;
    /* SkillsTab.lua:628-629 */
    onpastegroup?: (text: string) => void;
    options?: GemOptions;
    onoption?: (patch: Partial<GemOptions>) => void;
    editor?: {
      /* SkillsTab.lua:345-356 */
      groupCount?: boolean | ((group: number) => boolean);
      /* SkillsTab.lua:242-288 */
      onoptimise?: (group: number) => void;
      /* SkillsTab.lua:294-313 */
      optimiseEnabled?: (group: number) => boolean;
      /* SkillsTab.lua:357-391 */
      sourceNote?: (group: number) => string | undefined;
      /* GemSelectControl.lua:620-627 */
      supporting?: (group: number) => [number, number][];
      /* SkillsTab.lua:294-343 */
      imbued?: (group: number) => { value: string; options: string[]; enabled: boolean } | undefined;
      onimbued?: (group: number, name: string) => void;
      ongemcount?: (group: number, gem: number, count: number) => void;
      maxLevel?: (group: number, gem: number) => number;
      socketsLocked?: string;
      slotLocked?: (group: number) => string | undefined;
      gemwhatif?: (group: number, name: string) => Promise<StatCompareSection[] | undefined>;
      qualitywhatif?: (group: number, gem: number) => Promise<StatCompareSection[] | undefined>;
      enabledwhatif?: (group: number, gem: number) => Promise<StatCompareSection[] | undefined>;
      gemtip?: (group: number, kind: 'gem' | 'list' | 'quality', key: number | string) => Promise<{ Text: string; Separator: boolean }[] | undefined>;
      sortCache?: { canSupport: Record<string, boolean>; dps: Record<string, number>; progress?: number };
      onpicker?: (group: number, open: boolean) => void;
      /* SkillsTab.lua:190-198 */
      slottip?: (slot: string) => Promise<ItemTip | undefined>;
    };
    /* SkillListControl.lua:117-120 */
    grouptip?: (group: number) => Promise<{ Text: string; Separator: boolean }[] | undefined>;
    /* SkillListControl.lua:234-260 */
    groupicon?: (slot: string) => string | undefined;
    revision?: unknown;
  };
  let {
    sets,
    set,
    selected,
    mainGroup,
    catalogue,
    deltas,
    slots,
    sortOn = $bindable(true),
    sortStat = $bindable('Combined DPS'),
    onset,
    onsets,
    onselect,
    ongroup,
    ongem,
    onaddgem,
    onremovegem,
    onnewgroup,
    onreordergroups,
    onreordergems,
    setLocked,
    groupsLocked,
    onmain,
    ondeletegroup,
    ondeleteall,
    onsort,
    oncopygroup,
    onpastegroup,
    options,
    onoption,
    editor,
    grouptip,
    groupicon,
    revision
  }: Props = $props();
  let groupTip = $state.raw<{ i: number; lines?: { Text: string; Separator: boolean }[] }>();
  const showGroupTip = (i: number) => {
    if (!grouptip) return;
    groupTip = { i };
    grouptip(i)
      .then((lines) => {
        if (groupTip?.i === i) groupTip = { i, lines };
      })
      .catch(() => {});
  };
  $effect(() => {
    void revision;
    const i = untrack(() => groupTip?.i);
    if (i === undefined || !grouptip) return;
    grouptip(i)
      .then((lines) => {
        if (groupTip?.i === i) groupTip = { i, lines };
      })
      .catch(() => {});
  });

  const SORTS = ['Full DPS', 'Combined DPS', 'Hit DPS', 'Average Hit', 'DoT DPS', 'Bleed DPS', 'Ignite DPS', 'Poison DPS', 'Effective Hit Pool'];
  const LEVELS = [
    { value: 'normalMaximum', label: 'Normal Maximum', description: 'All gems default to their highest valid non-corrupted gem level.' },
    {
      value: 'corruptedMaximum',
      label: 'Corrupted Maximum',
      description: 'Normal gems default to their highest valid corrupted gem level.\nAwakened gems default to their highest valid non-corrupted gem level.'
    },
    { value: 'awakenedMaximum', label: 'Awakened Maximum', description: 'All gems default to their highest valid corrupted gem level.' },
    {
      value: 'characterLevel',
      label: 'Match Character Level',
      description:
        'All gems default to their highest valid non-corrupted gem level that your character meets the level requirement for.\nThis hides gems with a minimum level requirement above your character level, preventing them from showing up in the dropdown list.'
    },
    { value: 'levelOne', label: 'Level 1', description: 'All gems default to level 1.' }
  ];
  const SUPPORTS = [
    { value: 'ALL', label: 'All' },
    { value: 'NORMAL', label: 'Non-Exceptional' },
    { value: 'EXCEPTIONAL', label: 'Exceptional' }
  ];
  const levelDescription = $derived(LEVELS.find((l) => l.value === options?.defaultLevel)?.description);
  const reorderable = $derived(!!onreordergroups);

  let managing = $state(false);
  let selRow = $state(0);
  let naming = $state<{ op: 'new' | 'copy' | 'rename'; title: string } | null>(null);
  let confirmDelete = $state(false);
  const openManage = () => {
    selRow = set;
    naming = null;
    confirmDelete = false;
    managing = true;
  };
  const saveName = () => {
    if (!naming || !/\S/.test(naming.title)) return;
    const { op, title } = naming;
    if (op === 'new') onsets?.({ op, title });
    else onsets?.({ op, index: selRow, title });
    naming = null;
  };
  const groups = $derived(sets[set]?.groups ?? []);
  const group = $derived(groups[selected]);
  const offset = $derived(
    groups
      .slice(0, selected)
      .filter((g) => group && g.slot && g.slot === group.slot)
      .reduce((n, g) => n + g.gems.length, 0)
  );
  const socketCount = (g: SocketGroupFixture) => socketLine(g.sockets).colours.length;
  const groupLabel = (g: (SocketGroupFixture & { display?: string }) | undefined) => g?.display || g?.label || '<No active skills>';
  let groupBox = $state<HTMLElement>();
  const dragState = $state(reorderState());
  const drag = new Reorder(dragState, {
    box: () => groupBox,
    name: (i) => groupLabel(groups[i]),
    onmove: (f, t) => onreordergroups?.(f, t),
    onactivate: (i) => onselect(i)
  });
</script>

<svelte:window
  onpaste={(e: ClipboardEvent) => {
    if (!onpastegroup || isTypingTarget(e.target)) return;
    const text = e.clipboardData?.getData('text/plain');
    if (text) onpastegroup(text);
  }} />
<section aria-label="Socket groups" class="scroll flex w-72.5 shrink-0 flex-col gap-1.5 pr-1" bind:this={groupBox}>
  <FloatingLabel label="Skill set" class="[--float-bg:var(--color-surface-1)]"
    ><Select
      class="h-7.5 bg-surface-1"
      options={[...sets.map((s, i) => ({ value: String(i), label: s.title })), ...(onsets ? [{ value: '-1', label: 'Manage skill sets…' }] : [])]}
      value={String(set)}
      disabled={!onset}
      title={onset ? undefined : setLocked}
      onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
        const v = Number(e.currentTarget.value);
        if (v === -1) {
          e.currentTarget.value = String(set);
          openManage();
          return;
        }
        onset?.(v);
      }} /></FloatingLabel>
  <div class="mt-0.5 flex items-baseline justify-between"><h2 class="subhead uppercase">Socket groups</h2></div>
  {#each groups as g, i (i)}
    <div {...reorderable ? drag.row(i) : {}} class="relative flex items-stretch gap-1" title={reorderable ? undefined : groupsLocked}>
      <span aria-hidden="true" class={['flex w-4 shrink-0 items-center text-text-3', !reorderable && 'invisible']}><Icon name="grip" /></span>
      <button
        data-reorder-press
        tabindex={reorderable ? -1 : undefined}
        class={[
          'flex min-w-0 grow flex-col items-start gap-0.5 rounded-md border border-line bg-surface-1 px-2.5 py-2 text-left font-sans text-13 text-text aria-[current=true]:border-accent aria-[current=true]:bg-surface-2',
          reorderable && 'cursor-grab'
        ]}
        aria-current={i === selected}
        onmouseenter={() => showGroupTip(i)}
        onmouseleave={() => (groupTip = groupTip?.i === i ? undefined : groupTip)}
        onclick={(e: MouseEvent) => {
          onselect(i);
          // SkillListControl.lua:218-224
          if (e.ctrlKey) ongroup(i, { enabled: !g.enabled });
        }}
        onkeydown={(e: KeyboardEvent) => {
          if (e.ctrlKey && e.key.toLowerCase() === 'c' && oncopygroup) {
            e.preventDefault();
            oncopygroup(i);
          }
        }}
        oncontextmenu={(e: MouseEvent) => {
          e.preventDefault();
          // SkillListControl.lua:202-217
          if (e.ctrlKey) ongroup(i, { inFull: !g.inFull });
          else onmain?.(i);
        }}>
        <span class="flex w-full justify-between gap-1.5"
          ><span class={['font-semibold', !g.enabled && 'text-text-2 line-through']}
            >{#if g.slot && groupicon?.(g.slot)}<img src={groupicon(g.slot)} alt="" data-slot-icon class="mr-1 inline size-4 align-[-2px]" />{/if}{groupLabel(
              g
            )}{#if i === mainGroup}<span class="text-10 font-semibold tracking-[0.04em] text-accent"> ★ MAIN</span>{/if}</span
          ><span class="font-mono text-11 text-text-2">{g.gems.length}/{socketCount(g) || '–'}{g.slot ? ' · ' + g.slot : ''}</span></span>
        <span class="max-w-full truncate text-11 text-text-2">{g.gems.map((m) => m.name.replace(/ Support$/, '')).join(', ') || 'No gems'}</span>
      </button>
      {#if groupTip?.i === i && groupTip.lines?.length}
        <div role="tooltip" aria-label="Socket group tooltip" class="tooltip absolute top-0 left-full z-sticky ml-2 w-80 rounded-md px-3 py-2.5">
          <TooltipLines lines={groupTip.lines} />
        </div>
      {/if}
    </div>
  {/each}
  <div data-reorder-live aria-live="assertive" class="sr-only">{dragState.message}</div>
  <span id={drag.hint} class="sr-only">{drag.hintText}; Enter selects</span>
  <Button variant="dashed" class="text-12" onclick={onnewgroup}>+ New group</Button>
  {#if onmain || ondeletegroup || ondeleteall}
    <div class="flex gap-1.5">
      {#if onmain}<Button size="xs" class="grow text-11" disabled={selected === mainGroup || !group} onclick={() => onmain(selected)}>Set as main</Button>{/if}
      {#if ondeletegroup}<Button size="xs" class="grow text-11" disabled={groups.length <= 1 || !group} onclick={() => ondeletegroup(selected)}
          >Delete group</Button
        >{/if}
      {#if ondeleteall}<Button size="xs" class="grow text-11" disabled={groups.length <= 1} onclick={ondeleteall}>Delete all</Button>{/if}
    </div>
  {/if}
  <section aria-label="Gem options" class="panel mt-1 flex flex-col gap-1.75 p-2.5 text-12">
    <h3 class="subhead">GEM OPTIONS</h3>
    <div class="flex items-center justify-between gap-1.5">
      <Checkbox compact bind:checked={sortOn} onchange={(e: Event & { currentTarget: HTMLInputElement }) => onsort?.(e.currentTarget.checked, sortStat)}
        >Sort gems by DPS</Checkbox>
      <Select
        size="xs"
        class="w-30"
        aria-label="Sort by"
        options={SORTS}
        bind:value={sortStat}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onsort?.(sortOn, e.currentTarget.value)} />
    </div>
    <FloatingLabel label="Default gem level"
      ><Select
        size="xs"
        options={LEVELS}
        value={options?.defaultLevel}
        title={levelDescription}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onoption?.({ defaultLevel: e.currentTarget.value })} /></FloatingLabel>
    <label class="flex items-center justify-between"
      >Default gem quality<NumberInput
        size="xs"
        value={options?.defaultQuality ?? 0}
        min={0}
        max={23}
        class="w-16"
        label={options ? 'Default gem quality' : undefined}
        onchange={(defaultQuality) => onoption?.({ defaultQuality })} /></label>
    <FloatingLabel label="Show support gems"
      ><Select
        size="xs"
        options={SUPPORTS}
        value={options?.showSupport}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onoption?.({ showSupport: e.currentTarget.value })} /></FloatingLabel>
    {#if options}
      <Checkbox
        compact
        checked={options.showAlt}
        onchange={(e: Event & { currentTarget: HTMLInputElement }) => onoption?.({ showAlt: e.currentTarget.checked })}>Show gem quality variants</Checkbox>
      <Checkbox
        compact
        checked={options.showLegacy}
        onchange={(e: Event & { currentTarget: HTMLInputElement }) => onoption?.({ showLegacy: e.currentTarget.checked })}>Show legacy gems</Checkbox>
    {:else}
      <Checkbox compact disabled>Show legacy gems</Checkbox>
    {/if}
  </section>
</section>
{#if onsets}
  <Dialog bind:open={managing} title="Manage skill sets" class="w-105 p-4.5">
    <div class="flex flex-col gap-2">
      <div role="listbox" aria-label="Skill sets" class="flex max-h-75 flex-col gap-1 overflow-y-auto">
        {#each sets as s, i (i)}
          <button
            role="option"
            aria-selected={i === selRow}
            class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13 aria-selected:border-accent aria-selected:bg-surface-3"
            onclick={() => {
              selRow = i;
              naming = null;
              confirmDelete = false;
            }}
            ondblclick={() => {
              if (i !== set) onset?.(i);
            }}
            ><span class="truncate">{s.title}</span>{#if i === set}<span class="font-mono text-11 text-accent">current</span>{/if}</button>
        {/each}
      </div>
      {#if naming}
        <form
          class="flex items-center gap-1.5"
          onsubmit={(e) => {
            e.preventDefault();
            saveName();
          }}>
          <input class="input h-7.5 grow" aria-label="Name for this skill set" bind:value={naming.title} maxlength="100" />
          <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(naming.title)}>Save</Button>
          <Button size="sm" type="button" onclick={() => (naming = null)}>Cancel</Button>
        </form>
      {:else if confirmDelete}
        <div class="flex items-center gap-1.5 text-13" role="group" aria-label="Confirm delete">
          <span class="grow">Are you sure you want to delete '{sets[selRow]?.title}'?</span>
          <Button
            size="sm"
            variant="danger"
            onclick={() => {
              onsets({ op: 'delete', index: selRow });
              confirmDelete = false;
              selRow = Math.max(0, selRow - 1);
            }}>Delete</Button>
          <Button size="sm" onclick={() => (confirmDelete = false)}>Cancel</Button>
        </div>
      {:else}
        <div class="flex gap-1.5">
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'new', title: '' })}>New</Button>
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'copy', title: sets[selRow]?.title ?? '' })}>Copy</Button>
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'rename', title: sets[selRow]?.title ?? '' })}>Rename</Button>
          <Button size="sm" variant="danger" disabled={sets.length < 2} onclick={() => (confirmDelete = true)}>Delete</Button>
        </div>
      {/if}
      <span class="note">Double-click a skill set to switch to it.</span>
    </div>
    {#snippet actions()}
      <Button size="md" variant="primary" class="px-4" onclick={() => (managing = false)}>Done</Button>
    {/snippet}
  </Dialog>
{/if}
{#if group}
  {#key `${set}:${selected}`}
    <SkillGroupEditor
      {group}
      {offset}
      {catalogue}
      {deltas}
      {slots}
      {sortOn}
      {sortStat}
      onchange={(patch: Partial<SocketGroupFixture>) => ongroup(selected, patch)}
      ongem={(g: number, patch: Partial<SkillGemFixture>) => ongem(selected, g, patch)}
      onadd={(gem: SkillGemFixture) => onaddgem(selected, gem)}
      onremove={(g: number) => onremovegem(selected, g)}
      onreorder={onreordergems}
      groupCount={typeof editor?.groupCount === 'function' ? editor.groupCount(selected) : editor?.groupCount}
      sourceNote={editor?.sourceNote?.(selected)}
      supporting={editor?.supporting?.(selected)}
      imbued={editor?.imbued?.(selected)}
      onimbued={editor?.onimbued && ((name: string) => editor.onimbued!(selected, name))}
      ongemcount={editor?.ongemcount && ((g: number, count: number) => editor.ongemcount!(selected, g, count))}
      maxLevel={editor?.maxLevel && ((g: number) => editor.maxLevel!(selected, g))}
      socketsLocked={editor?.socketsLocked}
      onoptimise={editor?.onoptimise && (() => editor.onoptimise!(selected))}
      optimiseEnabled={editor?.optimiseEnabled?.(selected)}
      slotLocked={editor?.slotLocked?.(selected)}
      gemwhatif={editor?.gemwhatif && ((name: string) => editor.gemwhatif!(selected, name))}
      qualitywhatif={editor?.qualitywhatif && ((g: number) => editor.qualitywhatif!(selected, g))}
      enabledwhatif={editor?.enabledwhatif && ((g: number) => editor.enabledwhatif!(selected, g))}
      gemtip={editor?.gemtip && ((kind: 'gem' | 'list' | 'quality', key: number | string) => editor.gemtip!(selected, kind, key))}
      sortCache={editor?.sortCache}
      onpicker={editor?.onpicker && ((open: boolean) => editor.onpicker!(selected, open))}
      slottip={editor?.slottip}
      {revision} />
  {/key}
{/if}
