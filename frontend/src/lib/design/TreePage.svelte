<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Dropdown from './Dropdown.svelte';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import NumberInput from './NumberInput.svelte';
  import { tick, untrack, type Snippet } from 'svelte';
  import type { PlayerProgress } from '$lib/display/points';
  import type { TreeFixture, TreeNodeFx } from './fixtures';
  import { KIND_NAME, SEARCH_HELP, nodeMatches, searchWords, type CanvasApi, type SpecOp } from './tree';
  import WhatIfTooltip from './WhatIfTooltip.svelte';
  import { Reorder, reorderState } from './reorder';
  import { progressTooltip } from '$lib/display/points';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

  type MenuAction = 'tattoo' | 'socket' | 'copy' | 'power';

  type Props = {
    tree: TreeFixture;
    className: string;
    ascendancy: string;
    progress?: PlayerProgress;
    alloc: Record<number, boolean>;
    q?: string;
    filter?: string;
    spec?: number;
    mastery?: number | null;
    project?: (id: number) => { x: number; y: number } | undefined;
    canvas: Snippet<[CanvasApi]>;
    onpick?: (n: TreeNodeFx) => void;
    onmastery?: (id: number, effect: number) => void;
    onmenuaction?: (action: 'tattoo' | 'socket' | 'copy' | 'power', n: TreeNodeFx) => void;
    onzoom?: (how: 'in' | 'out' | 'fit') => void;
    ontimeless?: () => void;
    /* TreeTab.lua:268-271 */
    onpowerreport?: () => void;
    /* TreeTab.lua:194-205, PassiveTreeView.lua:204-205 */
    nodePower?: boolean;
    onnodepower?: (on: boolean) => void;
    nodePowerBar?: boolean;
    /* TreeTab.lua:207-254 */
    nodePowerControls?: Snippet;
    onpathmode?: (mode: string) => void;
    onclass?: (className: string) => void;
    onascendancy?: (ascendancy: string) => void;
    /* Build.lua:285-299, 1093-1138 */
    secondary?: { entries: { id: number; label: string }[]; selected: number; enabled: boolean };
    onsecondary?: (ascendClassId: number) => void;
    level?: number;
    onlevel?: (level: number) => void;
    /* Build.lua:211-217 */
    levelAuto?: boolean;
    onlevelauto?: (auto: boolean) => void;
    recordedVersion?: string;
    whole?: boolean;
    diff?: {
      node: number;
      pending: boolean;
      empty?: string;
      blocks: { header: string; sections: StatCompareSection[]; readyAt?: number; calcMs?: number }[];
    };
    /* PassiveMasteryControl.lua:37-43 */
    onmasteryhover?: (id: number, effect: number | null) => void;
    masteryDiff?: {
      node: number;
      effect: number;
      pending: boolean;
      empty?: string;
      blocks: { header: string; sections: StatCompareSection[]; readyAt?: number; calcMs?: number }[];
    };
    menuOff?: Partial<Record<MenuAction, string>>;
    menuOffFor?: (action: MenuAction, n: TreeNodeFx) => string | undefined;
    hint?: string;
    onspec?: (index: number) => void;
    specDetails?: string[];
    onspecs?: (op: SpecOp) => void;
    onspecsopen?: () => void;
    /* ItemsTab.lua:203-205 */
    manage?: boolean;
    /* TreeTab.lua:51-104 */
    onspechover?: (index: number | null) => void;
    /* TreeTab.lua:156-175 */
    versions?: { value: string; label: string }[];
    specVersion?: string;
    /* TreeTab.lua:588-612, 649-666 */
    onconvert?: (version: string, replace: boolean) => Promise<string>;
    /* TreeTab.lua:614-627, 668-680 */
    onconvertall?: (version: string) => Promise<string>;
    /* TreeTab.lua:129-154 */
    onresettree?: () => void;
    onremovetattoos?: () => void;
    /* TreeTab.lua:812-839 */
    onexporttree?: () => Promise<string>;
    /* TreeTab.lua:682-810 */
    onimporttree?: (title: string, link: string) => Promise<string>;
    /* TreeTab.lua:818-830 */
    onshrinktree?: (link: string) => Promise<string>;
    specTip?: {
      index: number;
      title: string;
      lines: string[];
      footer: string;
      pending: boolean;
      empty?: string;
      blocks: { header: string; sections: StatCompareSection[]; readyAt?: number; calcMs?: number }[];
    };
    details?: Snippet;
    detailsOpen?: boolean;
    /* TreeTab.lua:106-128 */
    compareSpec?: number;
    oncompare?: (spec: number) => void;
    nodeTipExtra?: Snippet<[TreeNodeFx]>;
    nodeTipTail?: Snippet<[TreeNodeFx]>;
    /* Main.lua:1011-1013 */
    edgeMarkers?: boolean;
    onhovered?: (id: number | null) => void;
    /* PassiveTreeView.lua:207 */
    showDiff?: boolean;
  };
  let {
    tree,
    className,
    ascendancy,
    progress,
    alloc,
    q = $bindable(''),
    filter = $bindable('All'),
    spec = $bindable(tree.activeSpec),
    mastery = $bindable(null),
    project,
    canvas,
    onpick,
    onmastery,
    onmenuaction,
    onzoom,
    ontimeless,
    onpowerreport,
    compareSpec = -1,
    oncompare,
    nodeTipExtra,
    nodeTipTail,
    edgeMarkers = true,
    onhovered,
    nodePower = false,
    onnodepower,
    nodePowerBar = true,
    nodePowerControls,
    onpathmode,
    onclass,
    onascendancy,
    secondary,
    onsecondary,
    level,
    onlevel,
    levelAuto = false,
    onlevelauto,
    recordedVersion,
    whole = false,
    diff,
    onmasteryhover,
    masteryDiff,
    menuOff = {},
    menuOffFor,
    hint = 'Click: allocate / deallocate · Right-click or the menu key: more · Ctrl+D: stat diff on hover',
    onspec,
    specDetails = [],
    onspecs,
    onspecsopen,
    onspechover,
    specTip,
    onexporttree,
    onimporttree,
    onshrinktree,
    versions = [],
    specVersion,
    onresettree,
    onremovetattoos,
    onconvert,
    onconvertall,
    details,
    detailsOpen = false,
    showDiff = $bindable(true),
    manage = $bindable(false)
  }: Props = $props();

  // TreeTab.lua:629-647
  let managing = $state(false);
  let selRow = $state(0);
  let naming = $state<{ op: 'new' | 'copy' | 'rename'; title: string } | null>(null);
  let confirmDelete = $state(false);
  // TreeTab.lua:156-175, 322-340, 588-680
  const versionLabel = (v: string) => versions.find((o) => o.value === v)?.label ?? v.replace('_', '.');
  const shownVersion = $derived(specVersion ?? tree.version);
  const versionOptions = $derived(
    versions.some((o) => o.value === shownVersion) ? versions : [...versions, { value: shownVersion, label: versionLabel(shownVersion) }]
  );
  let converting = $state<{ version: string } | null>(null);
  let convertingAll = $state(false);
  let resetting = $state(false);
  let converted = $state<{ version: string } | null>(null);
  let convertMsg = $state('');
  let versionSelect = $state<HTMLSelectElement>();
  const convert = async (version: string, replace: boolean, success: boolean) => {
    converting = null;
    const msg = (await onconvert?.(version, replace)) ?? '';
    convertMsg = msg;
    if (!msg && success) converted = { version };
  };
  const cancelConvert = () => {
    converting = null;
    if (versionSelect) versionSelect.value = shownVersion;
  };

  // TreeTab.lua:629-647
  let linkForm = $state<
    | { op: 'import'; title: string; link: string; msg: string; busy: boolean }
    | { op: 'export'; link: string; copied: boolean; shrink: 'idle' | 'busy' | 'done'; msg: string }
    | null
  >(null);
  const exportTree = async () => {
    linkForm = { op: 'export', link: (await onexporttree?.()) ?? '', copied: false, shrink: 'idle', msg: '' };
  };
  // TreeTab.lua:818-830
  const shrinkTree = async () => {
    const f = linkForm;
    if (f?.op !== 'export' || !onshrinktree) return;
    f.shrink = 'busy';
    try {
      f.link = await onshrinktree(f.link);
      f.copied = false;
    } catch {
      f.msg = 'Failed to get PoEURL link. Try again later.';
    }
    f.shrink = 'done';
  };
  const importTree = async () => {
    const f = linkForm;
    if (f?.op !== 'import' || !/\S/.test(f.title) || !/\S/.test(f.link)) return;
    f.busy = true;
    const msg = (await onimporttree?.(f.title, f.link.trim())) ?? '';
    f.busy = false;
    if (msg) f.msg = msg;
    else linkForm = null;
  };

  const openManage = () => {
    selRow = spec;
    naming = null;
    linkForm = null;
    confirmDelete = false;
    managing = true;
    onspecsopen?.();
    onspechover?.(spec);
  };
  $effect(() => {
    if (!manage || !onspecs) return;
    manage = false;
    untrack(openManage);
  });
  // PassiveSpecListControl.lua:82-87
  let specBox = $state<HTMLElement>();
  const specDragState = $state(reorderState());
  const specDrag = new Reorder(specDragState, {
    box: () => specBox,
    name: (i) => tree.specs[i] ?? '',
    onmove: (from, to) => {
      onspecs?.({ op: 'move', from, to });
      selRow = to;
    },
    onactivate: (i) => {
      if (i !== spec) {
        spec = i;
        onspec?.(i);
      }
    }
  });

  // TreeTab.lua:367-383
  let searchEl = $state<HTMLInputElement>();
  let selectHovered = false;

  const bareTitle = (i: number) => (tree.specs[i] ?? '').replace(/^\[[^\]]*\] /, '');
  const saveName = () => {
    if (!naming || !/\S/.test(naming.title)) return;
    const { op, title } = naming;
    if (op === 'new') onspecs?.({ op, title });
    else onspecs?.({ op, index: selRow, title });
    naming = null;
  };

  const uid = $props.id();
  const byId = $derived(new Map(tree.nodes.map((n) => [n.id, n])));
  let hovered = $state<number | null>(null);
  let menu = $state<{ id: number; x: number; y: number } | null>(null);
  let box = $state<HTMLElement>();
  let boxW = $state(0);
  let boxH = $state(0);
  let menuEl = $state<HTMLElement>();

  const pathOf = (n: TreeNodeFx) => {
    const out = [n.id];
    for (let x = n; x.next !== undefined && !alloc[x.next] && byId.get(x.next); x = byId.get(x.next)!) out.push(x.next);
    return out;
  };
  const H = $derived(hovered !== null ? byId.get(hovered) : undefined);
  $effect(() => onhovered?.(hovered));
  const path = $derived(H && !alloc[H.id] && H.k !== 'c' ? pathOf(H) : []);
  const lose = $derived(H && alloc[H.id] && H.a ? (H.loseIds ?? []).filter((id) => alloc[id]) : []);

  function pageKey(e: KeyboardEvent) {
    if (document.querySelector('dialog[open]')) return;
    const t = e.target as HTMLElement | null;
    const k = e.key.toLowerCase();
    // PassiveTreeView.lua:204-220
    if (e.key === 'PageUp' || e.key === 'PageDown') {
      e.preventDefault();
      for (let i = 0; i < (e.shiftKey ? 3 : 1); i++) onzoom?.(e.key === 'PageUp' ? 'in' : 'out');
      return;
    }
    if (e.key === 'F1' && H) {
      e.preventDefault();
      window.open(`https://www.poewiki.net/wiki/${H.name.replace(/ /g, '_')}`, '_blank', 'noopener');
      return;
    }
    if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey) {
      if (k === 'd') {
        e.preventDefault();
        showDiff = !showDiff;
      } else if (k === 'c' && H && H.k !== 'j' && !window.getSelection()?.toString()) {
        e.preventDefault();
        void navigator.clipboard?.writeText(`# ${H.name}\n${(H.stats ?? []).map((l) => l + '\n').join('')}`).catch(() => undefined);
      } else if (k === 'm' && onspecs) {
        e.preventDefault();
        openManage();
      } else if (k === 'f') {
        e.preventDefault();
        searchEl?.focus();
        searchEl?.select();
      }
      return;
    }
    if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey || selectHovered) return;
    if (t && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName) || t.closest('[role="menu"], [role="listbox"], [role="combobox"]')))
      return;
    if (k === 'p' && onnodepower) {
      e.preventDefault();
      onnodepower(!nodePower);
      return;
    }
    if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return;
    const to = spec + (e.key === 'ArrowUp' ? -1 : 1);
    if (to < 0 || to >= tree.specs.length || !onspec) return;
    e.preventDefault();
    spec = to;
    onspec(to);
  }

  const words = $derived(searchWords(q.trim()));
  const KIND_FILTER: Record<string, string[]> = { All: ['s', 'n', 'k', 'm', 'j'], Notables: ['n'], Keystones: ['k'], Masteries: ['m'] };
  const matchList = $derived(q.trim().length >= 2 ? tree.nodes.filter((n) => KIND_FILTER[filter].includes(n.k) && nodeMatches(n, words)) : []);
  const matches = $derived(new Set(matchList.map((n) => n.id)));
  const big = (n: TreeNodeFx) => n.k === 'n' || n.k === 'k' || n.k === 'm';
  const listed = $derived([...matchList.filter(big), ...matchList.filter((n) => !big(n))].slice(0, 12));

  const markers = $derived.by(() => {
    if (!project || !boxW || !edgeMarkers) return [];
    const side: Record<'left' | 'right' | 'top' | 'bottom', number[]> = { left: [], right: [], top: [], bottom: [] };
    for (const n of matchList) {
      const p = project(n.id);
      if (!p) continue;
      if (p.x < 0) side.left.push(p.y);
      else if (p.x > boxW) side.right.push(p.y);
      else if (p.y < 0) side.top.push(p.x);
      else if (p.y > boxH) side.bottom.push(p.x);
    }
    const mid = (a: number[], hi: number) => Math.max(60, Math.min(hi - 60, a.reduce((s, v) => s + v, 0) / a.length));
    return (Object.keys(side) as (keyof typeof side)[])
      .filter((k) => side[k].length)
      .map((k) => ({ side: k, count: side[k].length, at: mid(side[k], k === 'left' || k === 'right' ? boxH : boxW) }));
  });
  const ARROW = { left: '‹', right: '›', top: '˄', bottom: '˅' };
  const MARK_CLS = {
    left: 'left-1.5 top-(--p) -translate-y-1/2',
    right: 'right-12 top-(--p) -translate-y-1/2',
    top: 'top-11 left-(--p) -translate-x-1/2',
    bottom: 'bottom-11 left-(--p) -translate-x-1/2'
  };

  const tipSections = (blocks: { header: string; sections: StatCompareSection[]; readyAt?: number; calcMs?: number }[]) =>
    blocks.flatMap((b) =>
      b.sections.map((s, i) => ({
        header: b.header,
        actor: s.actor,
        readyAt: b.readyAt,
        calcMs: b.calcMs,
        title: `${i === 0 ? b.header : ''}${b.sections.length > 1 ? `${i === 0 ? ' ' : ''}${s.actor ?? ''}` : ''}` || undefined,
        lines: s.lines.map((l) => ({
          label: l.label,
          text: `${l.value}${l.percent ? ` ${l.percent}` : ''}`,
          dir: l.better ? ('up' as const) : ('down' as const),
          perPoint: l.perPoint,
          color: l.color
        }))
      }))
    );

  const tip = $derived.by(() => {
    if (!H || H.k === 'c') return null;
    const a = !!alloc[H.id];
    // PassiveTreeView.lua:1578-1599
    const chosen = H.k === 'm' && H.effect !== undefined && H.effect >= 0 ? H.effects?.[H.effect] : undefined;
    const mods =
      H.k === 'm'
        ? chosen
          ? [chosen.join(' / ')]
          : (H.effects ?? []).map((e) => e.join(' / '))
        : H.k === 'j' && tree.sockets.some((s) => s.id === H.id)
          ? [tree.sockets.find((s) => s.id === H.id)?.jewel ?? 'Empty socket']
          : (H.stats ?? []);
    const action = a
      ? `Deallocating${H.lose && H.a ? ` removes ${H.lose + 1} nodes (orange)` : ''}`
      : path.length
        ? `Allocating the path: ${H.pts ?? path.length} point${(H.pts ?? path.length) === 1 ? '' : 's'} (blue)`
        : 'Not reachable from your tree';
    const live = diff?.node === H.id ? diff : undefined;
    const blocks: { header: string; sections: StatCompareSection[]; readyAt?: number; calcMs?: number }[] = live
      ? live.blocks
      : !!H.a === a
        ? (tree.tips[H.id] ?? [])
        : [];
    const sections = showDiff ? tipSections(blocks) : [];
    const p = project?.(H.id);
    return {
      name: H.name,
      kind: `${KIND_NAME[H.k]}${a ? ' · allocated' : ''}`,
      mods,
      modsTitle: H.k === 'm' && !chosen && H.effects?.length ? 'Available Mastery node options are:' : undefined,
      action,
      sections,
      p,
      node: live?.node,
      pending: showDiff ? live?.pending : false,
      empty: showDiff ? live?.empty : 'Tip: Press Ctrl+D to enable the display of stat differences.'
    };
  });
  let tipEl = $state<HTMLElement>();
  let tipH = $state(0);
  let tipW = $state(360);
  const tipPos = $derived.by(() => {
    if (!tip?.p) return null;
    const w = Math.max(tipW, 360);
    const x = tip.p.x + 18 + w <= boxW ? tip.p.x + 18 : Math.max(tip.p.x - 18 - w, 4);
    const y = Math.min(Math.max(tip.p.y - 40, 4), Math.max(boxH - tipH - 4, 4));
    return { x, y };
  });

  const menuNode = $derived(menu ? byId.get(menu.id) : undefined);
  const menuItems = $derived.by(() => {
    const n = menuNode;
    if (!n) return [];
    const items = [
      {
        action: 'tattoo' as const,
        label: 'Tattoo / runegraft…',
        // PassiveTreeView.lua:520-532
        off: !['s', 'n', 'k', 'm'].includes(n.k),
        why: 'Tattoos and runegrafts replace passives and masteries, not sockets'
      },
      {
        action: 'socket' as const,
        label: 'Socket a jewel…',
        off: n.k !== 'j' || !alloc[n.id],
        why: n.k !== 'j' ? 'Only jewel sockets take a jewel' : 'Allocate the socket first'
      },
      { action: 'copy' as const, label: 'Copy node text', off: false, why: '' },
      { action: 'power' as const, label: 'Show in power report', off: false, why: '' }
    ];
    return items.map((it) => {
      const why = it.off ? undefined : (menuOff[it.action] ?? menuOffFor?.(it.action, n));
      return why ? { ...it, off: true, why } : it;
    });
  });
  async function openMenu(n: TreeNodeFx, at: { x: number; y: number }) {
    hovered = null;
    menu = { id: n.id, x: Math.min(at.x + 12, boxW - 232), y: Math.min(at.y + 8, boxH - 190) };
    await tick();
    menuEl?.querySelector<HTMLElement>('[role="menuitem"]:not(:disabled)')?.focus();
  }
  function closeMenu(refocus = true) {
    const id = menu?.id;
    menu = null;
    if (refocus && id !== undefined) box?.querySelector<HTMLElement>(`[data-node="${id}"]`)?.focus();
  }
  function menuKey(e: KeyboardEvent) {
    const list = [...(menuEl?.querySelectorAll<HTMLElement>('[role="menuitem"]:not(:disabled)') ?? [])];
    const i = list.indexOf(document.activeElement as HTMLElement);
    const to = { ArrowDown: (i + 1) % list.length, ArrowUp: (i - 1 + list.length) % list.length, Home: 0, End: list.length - 1 }[e.key];
    if (to !== undefined) {
      e.preventDefault();
      list[to]?.focus();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      closeMenu();
    } else if (e.key === 'Tab') closeMenu(false);
  }
  const outside = (e: PointerEvent) => {
    if (menu && menuEl && !menuEl.contains(e.target as Node)) closeMenu(false);
  };

  const api: CanvasApi = {
    get hovered() {
      return hovered;
    },
    describedBy: `${uid}-tip`,
    get path() {
      return path;
    },
    get lose() {
      return lose;
    },
    get matches() {
      return matches;
    },
    enter: (n) => {
      if (!menu) hovered = n.id;
    },
    leave: (n) => {
      if (hovered === n.id) hovered = null;
    },
    pick: (n) => {
      if (n.k === 'c') return;
      // PassiveTreeView.lua:500-502
      if (n.k !== 'm' || !onmastery) onpick?.(n);
      else if (alloc[n.id] || n.pts) mastery = n.id;
    },
    menu: (n, at) => void openMenu(n, at)
  };

  let helpHover = $state(false);
  let helpFocus = $state(false);

  const DETAILS_MIN = 280;
  const DETAILS_MAX = 720;
  const DETAILS_KEY = 'go-pob:tree-details-width';
  const storedWidth = () => {
    try {
      return Number(localStorage.getItem(DETAILS_KEY)) || 360;
    } catch {
      return 360;
    }
  };
  let detailsWidth = $state(Math.min(DETAILS_MAX, Math.max(DETAILS_MIN, storedWidth())));
  let resizing: { x: number; w: number } | undefined;
  const setDetailsWidth = (w: number) => {
    detailsWidth = Math.round(Math.min(DETAILS_MAX, Math.max(DETAILS_MIN, w)));
    try {
      localStorage.setItem(DETAILS_KEY, String(detailsWidth));
    } catch {
      // empty
    }
  };

  const M = $derived(mastery !== null ? byId.get(mastery) : undefined);
  const closeMastery = () => {
    if (mastery !== null) onmasteryhover?.(mastery, null);
    mastery = null;
  };
  const allocCount = $derived(tree.nodes.filter((n) => alloc[n.id]).length);
</script>

{#snippet tipExtra()}{#if H}{@render nodeTipExtra?.(H)}{/if}{/snippet}
{#snippet tipTail()}{#if H}{@render nodeTipTail?.(H)}{/if}{/snippet}

<svelte:window onpointerdown={outside} onkeydown={pageKey} />

<div class="box-border flex h-13 shrink-0 items-center gap-2.5 border-b border-line px-4" data-toolbar>
  <FloatingLabel label="Class" class="[--float-bg:var(--color-surface-1)]"
    ><select class="select h-8.5 bg-surface-1" value={className} onchange={(e) => onclass?.(e.currentTarget.value)}
      >{#each tree.classes as o (o)}<option>{o}</option>{/each}</select
    ></FloatingLabel>
  <FloatingLabel label="Ascendancy" class="[--float-bg:var(--color-surface-1)]"
    ><select class="select h-8.5 bg-surface-1" value={ascendancy} onchange={(e) => onascendancy?.(e.currentTarget.value)}
      >{#each tree.ascendancies as o (o)}<option>{o}</option>{/each}</select
    ></FloatingLabel>
  {#if secondary}
    <!-- Build.lua:285 -->
    <FloatingLabel label="Bloodline" class="[--float-bg:var(--color-surface-1)]"
      ><select
        class="select h-8.5 max-w-40 bg-surface-1"
        value={String(secondary.selected)}
        disabled={!secondary.enabled}
        onchange={(e) => onsecondary?.(Number(e.currentTarget.value))}
        >{#each secondary.entries as o (o.id)}<option value={String(o.id)}>{o.label}</option>{/each}</select
      ></FloatingLabel>
  {/if}
  {#if onlevel}
    <FloatingLabel label="Level" class="[--float-bg:var(--color-surface-1)]"
      ><div
        class="contents"
        role="presentation"
        oninput={(e: Event) => {
          const v = parseInt((e.target as HTMLInputElement).value, 10);
          if (Number.isFinite(v)) onlevel(Math.min(100, Math.max(1, v)));
        }}>
        <NumberInput
          class="h-8.5 w-20 bg-surface-1 font-mono"
          label="Character level"
          min={1}
          max={100}
          value={level}
          onchange={(v: number) => {
            if (v !== level) onlevel(v);
          }} />
      </div></FloatingLabel>
    {#if onlevelauto}
      <button
        class="btn btn-md px-3"
        aria-pressed={levelAuto}
        title="Auto: the level follows the passive points used; Manual: the level you type"
        onclick={() => onlevelauto(!levelAuto)}>{levelAuto ? 'Auto' : 'Manual'}</button>
    {/if}
  {/if}
  <div class="grow"></div>
  {#if progress}
    <output class="text-13 whitespace-nowrap text-text-2" aria-label="Points used" title={progressTooltip(progress)} data-points
      >{#if !onlevel}Level <span class="font-mono text-text">{progress.level}</span> ·&nbsp;{/if}Points
      <span class="font-mono text-text">{progress.used} / {progress.usedMax}</span>
      · Ascendancy
      <span class="font-mono text-text">{progress.ascUsed} / {progress.ascMax}</span></output>
  {/if}
  <FloatingLabel label="Tree" class="[--float-bg:var(--color-surface-1)]"
    ><select
      class="select h-8.5 bg-surface-1"
      aria-label="Version"
      bind:this={versionSelect}
      value={shownVersion}
      disabled={!onconvert}
      onchange={(e) => {
        const v = e.currentTarget.value;
        if (v !== shownVersion) converting = { version: v };
      }}
      >{#each versionOptions as o (o.value)}<option value={o.value}>{o.label}</option>{/each}</select
    ></FloatingLabel>
</div>

<div class="flex min-h-0 flex-[1_1_0]">
  <section
    aria-label="Passive tree"
    class="relative min-w-0 flex-[1_1_0] overflow-hidden bg-tree-ground"
    bind:this={box}
    bind:clientWidth={boxW}
    bind:clientHeight={boxH}>
    {@render canvas(api)}
    {#if onconvert && specVersion && specVersion !== tree.version}
      <div
        role="status"
        aria-label="Convert older tree"
        class="absolute top-2.5 left-1/2 z-4 flex -translate-x-1/2 items-center gap-2 rounded-md border border-line bg-surface-1/95 px-3 py-1.5 text-12">
        <span>This is an older tree version, which may not be fully compatible with the current game version.</span>
        <Button size="sm" variant="primary" onclick={() => void convert(tree.version, false, true)}>Convert to {versionLabel(tree.version)}</Button>
        {#if onconvertall}<Button size="sm" onclick={() => (convertingAll = true)}>Convert all trees to {versionLabel(tree.version)}</Button>{/if}
      </div>
    {/if}
    {#each markers as m (m.side)}
      <div
        role="note"
        aria-label="{m.count} search matches off screen to the {m.side}"
        class={[
          'pointer-events-none absolute z-3 flex items-center gap-1 rounded-full border border-stat-light bg-surface-1/90 px-2 py-0.5 font-mono text-11 text-stat-light',
          MARK_CLS[m.side]
        ]}
        style:--p="{m.at}px"
        data-marker={m.side}>
        <span aria-hidden="true">{ARROW[m.side]}</span>{m.count}
      </div>
    {/each}
    <div class="absolute top-2.5 right-2.5 z-3 flex flex-col gap-1" data-zoom>
      <button class="btn btn-icon size-7.5 bg-surface-1 text-16 font-normal" aria-label="Zoom in" onclick={() => onzoom?.('in')}>+</button>
      <button class="btn btn-icon size-7.5 bg-surface-1 text-16 font-normal" aria-label="Zoom out" onclick={() => onzoom?.('out')}>−</button>
      <button class="btn btn-icon size-7.5 bg-surface-1 text-11 font-normal" aria-label="Fit tree" onclick={() => onzoom?.('fit')}>fit</button>
    </div>
    <div class="map-note pointer-events-none top-2.5 left-2.5 z-3">
      <output aria-label="Tree version"
        >Tree {tree.version.replace('_', '.')}{recordedVersion && recordedVersion !== tree.version
          ? ` (build records ${recordedVersion.replace('_', '.')})`
          : ''}</output>
      · {allocCount} allocated{whole ? '' : ' in view'}
    </div>
    <div class="map-note pointer-events-none bottom-2.5 left-2.5 z-3">{hint}</div>
    {#if tip && tipPos}
      <div
        class="pointer-events-none absolute top-(--y) left-(--x) z-4"
        style:--x="{tipPos.x}px"
        style:--y="{tipPos.y}px"
        bind:this={tipEl}
        bind:clientHeight={tipH}
        bind:clientWidth={tipW}>
        <WhatIfTooltip
          translucent
          id="{uid}-tip"
          title={tip.name}
          kind={tip.kind}
          mods={tip.mods}
          modsTitle={tip.modsTitle}
          action={tip.action}
          sections={tip.sections}
          node={tip.node}
          pending={tip.pending}
          empty={tip.empty}
          extra={nodeTipExtra ? tipExtra : undefined}
          tail={nodeTipTail ? tipTail : undefined} />
      </div>
    {/if}
    {#if menu && menuNode}
      <div
        bind:this={menuEl}
        role="menu"
        aria-label="Actions for {menuNode.name}"
        tabindex="-1"
        class="menu top-(--y) right-auto left-(--x) z-5 w-56"
        style:--x="{menu.x}px"
        style:--y="{menu.y}px"
        onkeydown={menuKey}>
        <div class="truncate px-2 pt-0.5 pb-1 text-11 text-text-3">{menuNode.name}</div>
        {#each menuItems as it (it.action)}
          <button
            role="menuitem"
            tabindex="-1"
            class="menu-item"
            disabled={it.off}
            title={it.why || undefined}
            onclick={() => {
              onmenuaction?.(it.action, menuNode);
              closeMenu();
            }}>{it.label}</button>
        {/each}
      </div>
    {/if}
  </section>
  {#if q.trim() || detailsOpen}
    <!-- eslint-disable-next-line svelte/no-unused-svelte-ignore -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
    <div
      role="separator"
      aria-label="Resize tree details"
      aria-orientation="vertical"
      aria-valuenow={detailsWidth}
      aria-valuemin={DETAILS_MIN}
      aria-valuemax={DETAILS_MAX}
      tabindex="0"
      class="relative z-5 -mr-1.5 w-1.5 shrink-0 cursor-col-resize hover:bg-surface-3 focus-visible:bg-surface-3"
      onkeydown={(e: KeyboardEvent) => {
        const d = e.key === 'ArrowLeft' ? 16 : e.key === 'ArrowRight' ? -16 : 0;
        if (!d) return;
        e.preventDefault();
        setDetailsWidth(detailsWidth + d);
      }}
      onpointerdown={(e: PointerEvent) => {
        if (e.button > 0) return;
        resizing = { x: e.clientX, w: detailsWidth };
        (e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId);
      }}
      onpointermove={(e: PointerEvent) => {
        if (resizing) setDetailsWidth(resizing.w + resizing.x - e.clientX);
      }}
      onpointerup={() => (resizing = undefined)}
      onpointercancel={() => (resizing = undefined)}>
    </div>
    <aside aria-label="Tree details" class="box-border flex shrink-0 flex-col overflow-hidden border-l border-line bg-surface-1" style:width="{detailsWidth}px">
      {@render details?.()}
      <section aria-label="Search results" class="scroll flex grow flex-col gap-1.5 px-4 py-3">
        <div class="flex justify-between gap-2">
          <h2 class="subhead text-12">SEARCH{q.trim() ? ` “${q.trim()}”` : ''}</h2>
          <span class="note whitespace-nowrap">{matchList.length} matches · {matchList.filter((n) => alloc[n.id]).length} allocated</span>
        </div>
        {#if !q.trim()}<span class="note">Type in the search below; matches light up on the tree.</span>{/if}
        {#each listed as n (n.id)}
          <button
            class="choice-card flex-col items-start gap-px px-2 py-1.25 text-12"
            data-result={n.id}
            aria-label="{n.name} #{n.id}"
            aria-pressed={!!alloc[n.id]}
            onclick={() => api.pick(n)}
            onmouseenter={() => (hovered = n.id)}
            onfocus={() => (hovered = n.id)}
            onmouseleave={() => (hovered = null)}
            onblur={() => (hovered = null)}
            ><span class="flex w-full justify-between gap-1.5"
              ><span class="truncate font-semibold">{n.name}</span><span class={['whitespace-nowrap', alloc[n.id] ? 'text-accent' : 'text-text-3']}
                >{alloc[n.id] ? 'allocated' : n.pts ? `${n.pts} pts` : '—'}</span
              ></span
            ><span class="max-w-75 truncate text-text-3">{n.stats?.[0] ?? (n.k === 'm' ? 'Mastery — choose an effect' : '')}</span></button>
        {/each}
      </section>
    </aside>
  {/if}
</div>

<!-- TreeTab.lua:1011-1036, PassiveMasteryControl.lua:37-43 -->
<Dialog open={!!M} title={M?.name ?? ''} class="w-[min(1000px,95vw)] p-4" onclose={closeMastery}>
  {#if M}
    <div
      role="presentation"
      class="mt-2 grid h-[min(460px,calc(100dvh-8rem))] grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-3"
      onmouseleave={() => M && onmasteryhover?.(M.id, null)}>
      <div class="scroll flex flex-col gap-2">
        <span class="note">{alloc[M.id] ? 'Choose one effect' : 'Choose one effect to allocate it'}</span>
        <div role="group" aria-label="Mastery effects" class="flex flex-col gap-1.5">
          {#each M.effects ?? [] as e, i (i)}
            <button
              class={['choice-card items-center justify-between gap-2.5 px-2.25 py-1.75 text-12 aria-pressed:bg-surface-3', i === M.picked && 'border-accent']}
              aria-pressed={i === M.effect}
              data-power-pick={i === M.picked || undefined}
              onclick={() => {
                onmastery?.(M.id, i);
                closeMastery();
              }}
              onmouseenter={() => M && onmasteryhover?.(M.id, i)}
              onfocus={() => M && onmasteryhover?.(M.id, i)}
              onblur={() => M && onmasteryhover?.(M.id, null)}
              ><span class="text-magic">{e.join(' / ')}</span>{#if i === M.effect}<span class="font-mono text-11 text-accent">selected</span
                >{:else if i === M.picked}<span class="font-mono text-11 text-accent">power pick</span>{/if}</button>
          {/each}
        </div>
        {#if alloc[M.id] && onpick}
          <button
            class="btn btn-sm self-start"
            onclick={() => {
              onpick(M);
              closeMastery();
            }}>Deallocate</button>
        {/if}
      </div>
      <div class="scroll">
        {#if masteryDiff && masteryDiff.node === M.id && M.effects?.[masteryDiff.effect]}
          <div aria-label="Mastery effect tooltip" class="[&>[role=tooltip]]:w-auto [&>[role=tooltip]]:max-w-full [&>[role=tooltip]]:min-w-0">
            <WhatIfTooltip
              title={M.name}
              kind={KIND_NAME.m}
              mods={M.effects[masteryDiff.effect]}
              action={alloc[M.id] ? 'Choosing this effect' : 'Allocating with this effect'}
              sections={tipSections(masteryDiff.blocks)}
              node={masteryDiff.node}
              pending={masteryDiff.pending}
              empty={masteryDiff.empty} />
          </div>
        {/if}
      </div>
    </div>
  {/if}
</Dialog>

<div class="box-border flex h-13 shrink-0 items-center gap-2.5 border-t border-line bg-surface-1 px-4">
  <div
    class="relative flex items-center gap-1.5"
    role="group"
    aria-label="Tree search"
    onmouseenter={() => (helpHover = true)}
    onmouseleave={() => (helpHover = false)}
    onfocusin={() => (helpFocus = true)}
    onfocusout={() => (helpFocus = false)}>
    <input
      type="search"
      class="input h-8.5 w-60 px-2.5 text-14 max-[1800px]:w-48"
      bind:value={q}
      placeholder="Search nodes and stats"
      aria-label="Search the tree"
      bind:this={searchEl}
      aria-describedby="{uid}-help" />
    <button class="info-btn" aria-label="Search syntax" aria-describedby="{uid}-help">?</button>
    <div
      role="tooltip"
      id="{uid}-help"
      class={['tooltip tooltip-info top-auto bottom-10.5 left-0 z-20 w-100 flex-col gap-1.5', helpHover || helpFocus ? 'flex' : 'hidden']}>
      <span class="font-semibold">Search syntax</span>
      {#each SEARCH_HELP as [k, v] (k)}<span class="grid grid-cols-[130px_minmax(0,1fr)] gap-2"
          ><code class="font-mono text-accent">{k}</code><span class="text-text-2">{v}</span></span
        >{/each}
      <span class="note-sm border-t border-line pt-1.5">The search is saved with the build and comes back when you reopen it.</span>
    </div>
  </div>
  <span class="note whitespace-nowrap max-[1800px]:hidden">{matchList.length} matches</span>
  <FloatingLabel label="Filter"
    ><select class="select h-8.5" bind:value={filter}
      >{#each Object.keys(KIND_FILTER) as o (o)}<option>{o}</option>{/each}</select
    ></FloatingLabel>
  {#if onpathmode}
    <FloatingLabel label="Path"
      ><select class="select h-8.5 max-[1800px]:max-w-32" onchange={(e) => onpathmode(e.currentTarget.value)}
        ><option>Shortest</option><option>Best DPS pp</option></select
      ></FloatingLabel>
  {/if}
  <div class="grow"></div>
  {#if onresettree}
    <button class="btn btn-md px-3 whitespace-nowrap" onclick={() => (resetting = true)}>Reset Tree/Tattoos</button>
  {/if}
  {#if onnodepower && nodePowerBar}
    <label class="field field-inline gap-1 text-12 whitespace-nowrap" title="An estimate of each unallocated passive's power for the chosen stat"
      ><input class="check" type="checkbox" checked={nodePower} onchange={(e) => onnodepower(e.currentTarget.checked)} /><span>Show Node Power</span></label>
    {#if nodePower}{@render nodePowerControls?.()}{/if}
  {/if}
  {#if onpowerreport}
    <button class="btn btn-md px-3 whitespace-nowrap" onclick={onpowerreport}>Power<span class="max-[1800px]:hidden"> report</span></button>
  {/if}
  {#if ontimeless}
    <button class="btn btn-md px-3 whitespace-nowrap" title="Timeless jewel search" onclick={ontimeless}
      >Timeless<span class="max-[1800px]:hidden"> search</span></button>
  {/if}
  {#if tree.specs.length}
    <!-- TreeTab.lua:51-104 -->
    <FloatingLabel label="Tree set"
      ><div
        role="presentation"
        class="tree-set relative"
        onmouseenter={() => {
          selectHovered = true;
          onspechover?.(spec);
        }}
        onmouseleave={() => {
          selectHovered = false;
          onspechover?.(null);
        }}>
        <Dropdown
          class="h-8.5 max-w-60 min-w-40"
          value={String(spec)}
          options={[...tree.specs.map((o, i) => ({ value: String(i), label: o })), ...(onspecs ? [{ value: '-1', label: 'Manage trees…' }] : [])]}
          onhover={(i: number | null) => onspechover?.(i !== null && i < tree.specs.length ? i : selectHovered ? spec : null)}
          onchange={(v: string) => {
            const n = Number(v);
            if (n < 0) {
              openManage();
              return false;
            }
            spec = n;
            onspec?.(n);
          }} />
        {#if specTip && !managing && specTip.index < tree.specs.length}
          <div class="pointer-events-none absolute right-full bottom-0 z-40 mr-2" aria-label="Tree tooltip">
            <WhatIfTooltip
              title={specTip.title}
              info={specTip.lines}
              footer={specTip.footer}
              action=""
              sections={tipSections(specTip.blocks)}
              node={specTip.index}
              pending={specTip.pending}
              empty={specTip.empty} />
          </div>
        {/if}
      </div></FloatingLabel>
    {#if oncompare}
      <label class="field field-inline gap-1 text-12 whitespace-nowrap"
        ><input
          class="check"
          type="checkbox"
          checked={compareSpec >= 0}
          onchange={(e) => oncompare(e.currentTarget.checked ? (tree.specs.length > 1 ? (spec === 0 ? 1 : 0) : 0) : -1)} /><span>Compare:</span></label>
      {#if compareSpec >= 0}
        <select
          class="select h-8.5 max-w-48 text-12"
          aria-label="Compared tree"
          value={String(compareSpec)}
          onchange={(e) => oncompare(Number(e.currentTarget.value))}
          >{#each tree.specs as o, i (i)}<option value={String(i)}>{o}</option>{/each}</select>
      {/if}
    {/if}
  {/if}
</div>

{#if onspecs}
  <Dialog bind:open={managing} title="Manage passive trees" class="w-105 p-4.5" onclose={() => onspechover?.(null)}>
    <div class="flex flex-col gap-2">
      <!-- PassiveSpecListControl.lua:95-117 -->
      <div
        role="listbox"
        aria-label="Passive trees"
        tabindex="-1"
        class="flex max-h-75 flex-col gap-1 overflow-y-auto"
        bind:this={specBox}
        onkeydown={(e: KeyboardEvent) => {
          if (e.key !== 'Delete' && e.key !== 'F2') return;
          const i = [...(specBox?.querySelectorAll('[data-reorder-item]') ?? [])].indexOf((e.target as HTMLElement).closest('[data-reorder-item]')!);
          if (i < 0) return;
          e.preventDefault();
          selRow = i;
          linkForm = null;
          if (e.key === 'F2') {
            confirmDelete = false;
            naming = { op: 'rename', title: bareTitle(i) };
          } else if (tree.specs.length > 1) {
            naming = null;
            confirmDelete = true;
          }
        }}>
        {#each tree.specs as o, i (i)}
          <div {...specDrag.row(i)} class="relative">
            <button
              data-reorder-press
              tabindex="-1"
              role="option"
              aria-selected={i === selRow}
              class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13 aria-selected:border-accent aria-selected:bg-surface-3"
              onclick={() => {
                selRow = i;
                naming = null;
                confirmDelete = false;
              }}
              ondblclick={() => {
                if (i !== spec) {
                  spec = i;
                  onspec?.(i);
                }
              }}
              onmouseenter={() => onspechover?.(i)}
              onfocus={() => onspechover?.(i)}
              ><span class="truncate"
                >{o}{#if specDetails[i]}<span class="text-text-3">{` (${specDetails[i]})`}</span>{/if}</span
              >{#if i === spec}<span class="font-mono text-11 text-accent">current</span>{/if}</button>
          </div>
        {/each}
      </div>
      <div aria-live="assertive" class="sr-only">{specDragState.message}</div>
      <span id={specDrag.hint} class="sr-only">{specDrag.hintText}; Enter switches to the tree</span>
      <div class="scroll h-64 shrink-0">
        {#if managing && specTip && specTip.index < tree.specs.length}
          <div aria-label="Tree tooltip">
            <WhatIfTooltip
              title={specTip.title}
              info={specTip.lines}
              footer={specTip.footer}
              action=""
              sections={tipSections(specTip.blocks)}
              node={specTip.index}
              pending={specTip.pending}
              empty={specTip.empty} />
          </div>
        {/if}
      </div>
      {#if linkForm?.op === 'import'}
        <form
          class="flex flex-col gap-1.5"
          aria-label="Import Tree"
          onsubmit={(e) => {
            e.preventDefault();
            void importTree();
          }}>
          <input class="input h-7.5" aria-label="Enter name for this passive tree" placeholder="Name" bind:value={linkForm.title} maxlength="100" />
          <input
            class="input h-7.5"
            aria-label="Enter passive tree link"
            placeholder="https://www.pathofexile.com/passive-skill-tree/…"
            bind:value={linkForm.link}
            oninput={() => linkForm?.op === 'import' && (linkForm.msg = '')} />
          {#if linkForm.msg}<span role="alert" class="text-12 text-worse">{linkForm.msg}</span>{/if}
          <div class="flex gap-1.5">
            <Button size="sm" variant="primary" type="submit" disabled={linkForm.busy || !/\S/.test(linkForm.title) || !/\S/.test(linkForm.link)}
              >Import</Button>
            <Button size="sm" type="button" onclick={() => (linkForm = null)}>Cancel</Button>
          </div>
        </form>
      {:else if linkForm?.op === 'export'}
        <div class="flex flex-col gap-1.5" role="group" aria-label="Export Tree">
          <span class="text-12 text-text-2">Passive tree link:</span>
          <input
            class="input h-7.5 font-mono text-12"
            aria-label="Passive tree link"
            readonly
            value={linkForm.link}
            onfocus={(e) => e.currentTarget.select()} />
          {#if linkForm.msg}<span role="alert" class="text-12 text-worse">{linkForm.msg}</span>{/if}
          <div class="flex gap-1.5">
            {#if onshrinktree}<Button size="sm" disabled={linkForm.shrink !== 'idle'} onclick={() => void shrinkTree()}
                >{linkForm.shrink === 'busy' ? 'Shrinking...' : linkForm.shrink === 'done' ? 'Done' : 'Shrink with PoEURL'}</Button
              >{/if}
            <Button
              size="sm"
              onclick={() => {
                if (linkForm?.op !== 'export') return;
                const f = linkForm;
                void navigator.clipboard?.writeText(f.link).then(() => (f.copied = true));
              }}>{linkForm.copied ? 'Copied' : 'Copy'}</Button>
            <Button size="sm" onclick={() => (linkForm = null)}>Done</Button>
          </div>
        </div>
      {:else if naming}
        <form
          class="flex items-center gap-1.5"
          onsubmit={(e) => {
            e.preventDefault();
            saveName();
          }}>
          <input class="input h-7.5 grow" aria-label="Name for this passive tree" bind:value={naming.title} maxlength="100" />
          <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(naming.title)}>Save</Button>
          <Button size="sm" type="button" onclick={() => (naming = null)}>Cancel</Button>
        </form>
      {:else if confirmDelete}
        <div class="flex items-center gap-1.5 text-13" role="group" aria-label="Confirm delete">
          <span class="grow">Delete '{tree.specs[selRow]}'?</span>
          <Button
            size="sm"
            variant="danger"
            onclick={() => {
              onspecs({ op: 'delete', index: selRow });
              confirmDelete = false;
              selRow = Math.max(0, selRow - 1);
            }}>Delete</Button>
          <Button size="sm" onclick={() => (confirmDelete = false)}>Cancel</Button>
        </div>
      {:else}
        <div class="flex gap-1.5">
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'new', title: '' })}>New</Button>
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'copy', title: bareTitle(selRow) })}>Copy</Button>
          <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'rename', title: bareTitle(selRow) })}>Rename</Button>
          <Button size="sm" variant="danger" disabled={tree.specs.length < 2} onclick={() => (confirmDelete = true)}>Delete</Button>
        </div>
        {#if onimporttree || onexporttree}
          <div class="flex gap-1.5">
            {#if onimporttree}<Button size="sm" class="bg-surface-1" onclick={() => (linkForm = { op: 'import', title: '', link: '', msg: '', busy: false })}
                >Import Tree</Button
              >{/if}
            {#if onexporttree}<Button size="sm" class="bg-surface-1" onclick={() => void exportTree()}>Export Tree</Button>{/if}
          </div>
        {/if}
      {/if}
      <span class="note">Double-click a tree to switch to it; drag to reorder.</span>
    </div>
    {#snippet actions()}
      <Button size="md" variant="primary" class="px-4" onclick={() => (managing = false)}>Done</Button>
    {/snippet}
  </Dialog>
{/if}

<Dialog
  open={converting !== null}
  onclose={cancelConvert}
  title={converting ? `Convert to Version ${versionLabel(converting.version)}` : 'Convert'}
  class="w-115 p-4.5">
  {#if converting}
    {@const v = converting.version}
    <div class="flex flex-col gap-3 text-13">
      <p class="m-0">Warning: some or all of the passives may be de-allocated due to changes in the tree.</p>
      <p class="m-0">Convert will replace your current tree.<br />Copy + Convert will backup your current tree.</p>
      <div class="flex gap-1.5">
        <Button size="sm" variant="primary" onclick={() => void convert(v, true, false)}>Convert</Button>
        <Button size="sm" onclick={() => void convert(v, false, false)}>Copy + Convert</Button>
        <Button size="sm" variant="ghost" onclick={cancelConvert}>Cancel</Button>
      </div>
    </div>
  {/if}
</Dialog>

<Dialog bind:open={convertingAll} title="Convert all to Version {versionLabel(tree.version)}" class="w-115 p-4.5">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">Warning: some or all of the passives may be de-allocated due to changes in the tree.</p>
    <p class="m-0">Convert will replace all trees that are not Version {versionLabel(tree.version)}.<br />This action cannot be undone.</p>
    <div class="flex gap-1.5">
      <Button
        size="sm"
        variant="primary"
        onclick={async () => {
          convertingAll = false;
          convertMsg = (await onconvertall?.(tree.version)) ?? '';
        }}>Convert</Button>
      <Button size="sm" variant="ghost" onclick={() => (convertingAll = false)}>Cancel</Button>
    </div>
  </div>
</Dialog>

<Dialog
  open={converted !== null || !!convertMsg}
  onclose={() => ((converted = null), (convertMsg = ''))}
  title={convertMsg ? 'Convert' : 'Tree Converted'}
  class="w-115 p-4.5">
  <div class="flex flex-col gap-3 text-13">
    {#if convertMsg}
      <p class="m-0" role="alert">{convertMsg}</p>
    {:else if converted}
      <p class="m-0">The tree has been converted to {versionLabel(converted.version)}.</p>
      <p class="m-0">Note that some or all of the passives may have been de-allocated due to changes in the tree.</p>
      <p class="m-0">You can switch back to the old tree using the Tree set select.</p>
    {/if}
    <div class="flex"><Button size="sm" onclick={() => ((converted = null), (convertMsg = ''))}>OK</Button></div>
  </div>
</Dialog>

<Dialog bind:open={resetting} title="Reset Tree/Tattoos" class="w-115 p-4.5">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">Warning: resetting your passive tree or removing all tattoos cannot be undone.</p>
    <div class="flex gap-1.5">
      <Button
        size="sm"
        variant="danger"
        onclick={() => {
          resetting = false;
          onresettree?.();
        }}>Reset Tree</Button>
      {#if onremovetattoos}<Button
          size="sm"
          onclick={() => {
            resetting = false;
            onremovetattoos?.();
          }}>Remove All Tattoos</Button
        >{/if}
      <Button size="sm" variant="ghost" onclick={() => (resetting = false)}>Cancel</Button>
    </div>
  </div>
</Dialog>

<style>
  .tree-set :global([role='listbox']) {
    top: auto;
    bottom: 100%;
    margin-top: 0;
    margin-bottom: 0.25rem;
  }
</style>
