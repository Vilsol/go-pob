<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { loadSkillTree, skillTree } from '$lib/skill_tree';
  import type { Tree } from '$lib/skill_tree/types';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { base as appBase } from '$app/paths';
  import { treeManageRequest } from './tree_manage';
  import ItemsPage from '$lib/design/ItemsPage.svelte';
  import ItemDB from '$lib/design/ItemDB.svelte';
  import { defaultItemDBFilter, type ItemDBFilterInput, type ItemDBKind, type ItemDBLists, type ItemDBRowPlain } from '$lib/design/item_db';
  import { MINION_ITEM_DROP, type ItemDrag, type MinionItemDrop } from '$lib/design/item_drag';
  import type { ItemSet, ItemSummary, ItemTip, ModLine, StatDiff } from '$lib/design/fixtures';
  import type { PoolItem } from '$lib/design/fixtures/items';
  import {
    itemTip as tipOf,
    type CraftChoice,
    type DisplayEdit,
    type DisplayInfo,
    type ImpactBlock,
    type ItemSetOp,
    type TipProps,
    type WhatIf
  } from '$lib/design/items';
  import { itemUsage, type UsageContext } from '$lib/design/item_usage';
  import { buildRevision, currentBuild, outputs } from '$lib/global.js';
  import { migrateEldritchImplicits, slotOnlyTooltips } from '$lib/display/item_options';
  import { itemsLayout } from '$lib/display/appearance';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import type { calculator, exposition, pob } from '$lib/types';
  import type { ItemStatDiffBlock } from '$lib/components/items/item_diff';
  import type { CraftApi } from '$lib/design/crafting';
  import BuySimilarDialog from '$lib/design/BuySimilarDialog.svelte';
  import { tradeDefences, type TradeDefence, type TradeItem } from '$lib/design/trade';

  // ItemsTab.lua:1679-1703
  // ItemSlotControl.lua:33-38
  // ItemsTab.lua:99-102
  // ItemSlotControl.lua:12-19, 82-93
  // ItemsTab.lua:5003-5158
  // ItemsTab.lua:224-263
  // ItemListControl.lua:23-75, 331-369, ItemsTab.lua:1768-1815
  // ItemsTab.lua:1731-1765, ItemListControl.lua:290-329

  type Editing = { item: ItemSummary; slot?: string; impact?: StatDiff; isNew?: boolean; key?: string };
  type Diff = { blocks: ItemStatDiffBlock[]; calcMs: number };
  type View = {
    activeSet: string;
    items: Record<string, ItemSummary>;
    texts: Record<string, string>;
    sets: ItemSet[];
    jewels: { node: number; item: number; choices: number[] }[];
    abyssal: string[];
    extraSlots: string[];
    stash: { item: number; used: string }[];
    equipped: exposition.EquippedSlot[];
    slotItems: Record<string, number[]>;
  };

  const READONLY = 'Edit as text to change it in the paste panel, under its controls';
  const AFFIXES = { tiers: [], sources: [], sorts: [], rows: [] };

  const slotLabel = (slot: string) => {
    const abyssal = /Abyssal Socket (\d+)$/.exec(slot);
    return abyssal ? `${slot.replace(/ ?Swap/, '').replace(/ Abyssal Socket \d+$/, '')} Abyssal #${abyssal[1]}` : slot.replace(/ Swap$/, '');
  };
  const unescapeXml = (t: string) => t.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');

  const itemTip = (t: calculator.ItemTooltip): ItemTip => ({
    influence: [...(t.Influence ?? [])],
    title: [...(t.Title ?? [])],
    lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })),
    children: (t.Children ?? []).map((c) => (c ?? []).map((l) => ({ text: l.Text, sep: l.Separator })))
  });
  const lines = (l: exposition.ItemModLine[] | null | undefined): ModLine[] => (l ?? []).map((m) => ({ text: m.Line, kind: m.Kind.toLowerCase() }));
  const summary = (p: exposition.ParsedItem): ItemSummary => ({
    name: p.Title || p.Name,
    base: p.BaseName || p.Type,
    rarity: (p.Rarity.toLowerCase() || 'normal') as ItemSummary['rarity'],
    sockets: p.Sockets ?? '',
    implicits: lines(p.Implicits),
    explicits: lines(p.Explicits),
    flask: p.Flask,
    fits: p.FitSlots ?? [],
    tooltip: itemTip(p.Tooltip)
  });
  const fromSlot = (e: exposition.EquippedSlot): ItemSummary => ({
    name: e.Name,
    base: '',
    rarity: (e.Rarity.toLowerCase() || 'normal') as ItemSummary['rarity'],
    sockets: '',
    implicits: [],
    explicits: [],
    flask: e.Flask,
    fits: []
  });

  const parsed = new Map<string, ItemSummary | null>();
  let parsedFor: unknown;
  const parse = async (text: string) => {
    if (!parsed.has(text)) {
      const r = await syncWrap?.ParseItem(text);
      parsed.set(text, r?.item ? summary(r.item) : null);
    }
    return parsed.get(text) ?? undefined;
  };

  // ItemListControl.lua:77-183
  let layout = $state(get(itemsLayout));
  let stashFilter = $state('Any Loadout');
  let stashFilters = $state<string[]>([]);
  let sharedSets = $state<{ title: string; tip: string[] }[]>([]);
  let sharedItems = $state<{ text: string; row: PoolItem }[]>([]);

  // ItemsTab.lua:193-206
  let treeSpecs = $state<{ active: number; labels: string[] }>({ active: 1, labels: [] });
  let treeVersion = $state<string>();

  let loaded: { revision: number; epoch: number; items?: pob.Items; sockets: pob.Socket[]; usage: Pick<UsageContext, 'specs' | 'activeSpec'> } | undefined;
  let epoch = 0;
  let saved = $state<{ id: number; text: string }[]>([]);

  let view = $state<View>();
  let loadSeq = 0;

  const load = async (revision: number, calculated?: unknown) => {
    const token = ++loadSeq;
    const build = $currentBuild;
    if (!syncWrap || !build) return;
    const at = epoch;
    if (parsedFor !== calculated) {
      parsed.clear();
      parsedFor = calculated;
    }
    if (loaded?.revision !== revision || loaded.epoch !== at) {
      const [items, tree] = await Promise.all([syncWrap.GetItems(), Promise.resolve(build.Tree).catch(() => undefined)]);
      const spec = tree?.Specs?.[tree.ActiveSpec - 1] ?? tree?.Specs?.[0];
      // ItemListControl.lua:198-219
      const specs = (tree?.Specs ?? []).map((sp) => ({
        title: sp.Title ?? 'Default',
        sockets: (sp.Sockets ?? []).map((x) => ({ node: x.NodeID, item: x.ItemID })),
        alloc: new Set((sp.NodesAttr.match(/\d+/g) ?? []).map(Number))
      }));
      loaded = { revision, epoch: at, items, sockets: spec?.Sockets ?? [], usage: { specs, activeSpec: Math.max(0, (tree?.ActiveSpec ?? 1) - 1) } };
      saved = [];
    }
    const [equipped, slotItems, tabSlots, setTips, extraSlots, filterInfo, specList] = await Promise.all([
      syncWrap.GetEquippedItems().then((e) => e ?? []),
      syncWrap.GetSlotItemLists().then((l) => l ?? {}),
      syncWrap.GetItemsTabSlots(),
      syncWrap.GetItemSetTooltips().catch((e: Error) => {
        logError(e);
        return {} as Record<string, string[]>;
      }),
      syncWrap.GetConditionalSlots().catch((e: Error) => {
        logError(e);
        return [] as string[];
      }),
      syncWrap.GetItemListFilter('Any Loadout').catch((e: Error) => {
        logError(e);
        return { filters: [] as string[], items: [] as number[] };
      }),
      syncWrap.GetTreeSpecs().catch((e: Error) => {
        logError(e);
        return undefined;
      })
    ]);
    stashFilters = filterInfo.filters;
    if (specList) {
      treeSpecs = { active: specList.active, labels: specList.specs.map((sp) => sp.Label) };
      treeVersion = specList.specs[specList.active - 1]?.TreeVersion;
    }
    const raw = loaded.items;
    const texts: Record<string, string> = {};
    for (const it of raw?.Items ?? []) texts[it.ID] = it.Text;
    for (const e of equipped) texts[e.ItemID] = e.Text;
    for (const s of saved) texts[s.id] = s.text;

    const items: Record<string, ItemSummary> = {};
    await Promise.all(
      Object.entries(texts).map(async ([id, text]) => {
        const s = await parse(text);
        if (s) items[id] = s;
      })
    );
    for (const e of equipped) items[e.ItemID] ??= fromSlot(e);
    if (token !== loadSeq) return;

    const activeSet = String(raw?.ActiveItemSet ?? 1);
    const rawSets = raw?.ItemSets?.length ? raw.ItemSets : [{ ID: activeSet, Slots: [] } as unknown as pob.ItemSet];
    const sets: ItemSet[] = rawSets.map((s) => {
      const swap = !!s.UseSecondWeaponSet;
      const slots = (s.Slots ?? []).filter((x) => x.ItemID > 0 && items[x.ItemID]).map((x) => ({ slot: x.Name, item: x.ItemID, active: !!x.Active }));
      if (s.ID !== activeSet) return { id: s.ID, title: s.Title || 'Default', swap, slots, tip: setTips[s.ID] };
      // ItemsTab.lua:147-162
      const other = slots.filter((x) => x.slot.startsWith('Weapon') && x.slot.includes('Swap') !== swap);
      return {
        id: s.ID,
        title: s.Title || 'Default',
        swap,
        slots: [...equipped.map((e) => ({ slot: e.SlotName, item: e.ItemID, active: e.Active })), ...other],
        tip: setTips[s.ID]
      };
    });
    // ItemsTab.lua:208-222, 1652-1670
    const jewels = tabSlots.sockets.map((s) => ({ node: s.node, item: items[s.item] ? s.item : 0, choices: s.choices }));
    // ItemListControl.lua:113-121, 242-258
    const ctx: UsageContext = { sets, activeSet, ...loaded.usage };
    // ItemsTab.lua:1731-1765
    const usage = (item: number) => itemUsage(item, { jewel: /Jewel$/.test(items[item].base), abyss: /Abyss/.test(items[item].base) }, ctx);
    // ItemListControl.lua:113-183
    const filtered = stashFilter !== 'Any Loadout' ? await syncWrap.GetItemListFilter(stashFilter) : undefined;
    if (token !== loadSeq) return;
    const stash = filtered
      ? filtered.items.filter((item) => items[item]).map((item) => ({ item, used: usage(item) || 'Equipped' }))
      : [...new Set([...(raw?.Items ?? []).map((it) => it.ID), ...Object.keys(items).map(Number)])]
          .filter((item) => items[item])
          .map((item) => ({ item, used: usage(item) || 'Equipped' }));
    view = { activeSet, items, texts, sets, jewels, abyssal: tabSlots.abyssal, extraSlots, stash, equipped, slotItems };
  };

  $effect(() => {
    const revision = $buildRevision;
    const calculated = $outputs;
    if (!$currentBuild) return;
    untrack(() => load(revision, calculated)).catch(logError);
  });

  $effect(() => {
    void $outputs;
    syncWrap?.PrepareStatCompare().catch(logError);
  });

  const flasks = $derived(Object.fromEntries((view?.equipped ?? []).filter((e) => e.Flask).map((e) => [e.SlotName, e.Active])));

  const stamp = (r: Diff): ImpactBlock[] => {
    const readyAt = performance.now();
    return r.blocks.map((b) => ({ ...b, readyAt, calcMs: r.calcMs }));
  };

  let pageView = $state<'equipped' | 'stash' | 'shared' | 'browse'>('equipped');
  let hover = $state<string | null>(null);
  let editing = $state<Editing | null>(null);
  let dbKind = $state<ItemDBKind>('UNIQUE');
  let dbHover = $state<string | null>(null);
  const target = $derived(editing?.key ?? hover ?? (dbHover === null ? null : `db:${dbKind}:${dbHover}`));
  let whatIf = $state<WhatIf>();
  // ItemsTab.lua:1481-1484, 4795-4801
  let statDiffs = $state(true);
  $effect(() => {
    void $buildRevision;
    void Promise.resolve(syncWrap?.StatDifferencesShown())
      .then((on) => (statDiffs = on ?? true))
      .catch(logError);
  });

  const whatIfOf = (key: string): (() => Promise<Diff | undefined>) | undefined => {
    const e = view?.equipped.find((x) => x.SlotName === key);
    if (e) {
      return e.Flask ? async () => syncWrap?.FlaskToggleCompare(e.Text) : async () => syncWrap?.ItemUnequipCompare(e.SlotName, slotLabel(e.SlotName));
    }
    // ItemsTab.lua:5081-5088
    const inSlot = /^(-?\d+)@(.+)$/.exec(key);
    if (inSlot) {
      const t = view?.texts[inSlot[1]];
      if (t === undefined) return undefined;
      const only = $slotOnlyTooltips;
      return async () => {
        const r = await syncWrap?.ItemCompare(t);
        return r && only ? { ...r, blocks: r.blocks.filter((b) => b.slot === inSlot[2]) } : r;
      };
    }
    // ItemDBControl.lua:320-328
    const db = /^db:(UNIQUE|RARE):(.+)$/.exec(key);
    if (db) {
      return async () => {
        const t = await syncWrap?.GetItemDBText(db[1], db[2]);
        return t === undefined ? undefined : syncWrap?.ItemCompare(t);
      };
    }
    const text = /^-?\d+$/.test(key) ? view?.texts[key] : undefined;
    return text === undefined ? undefined : async () => syncWrap?.ItemCompare(text);
  };

  let whatIfBusy = false;
  let whatIfAgain = false;
  const runWhatIf = async () => {
    if (whatIfBusy) {
      whatIfAgain = true;
      return;
    }
    whatIfBusy = true;
    try {
      do {
        whatIfAgain = false;
        const key = target;
        const job = key ? whatIfOf(key) : undefined;
        if (!key || !job) {
          whatIf = undefined;
          continue;
        }
        if (!statDiffs) {
          whatIf = { key, pending: false, blocks: [], tip: 'Tip: Press Ctrl+D to enable the display of stat differences.' };
          continue;
        }
        whatIf = { key, pending: true, blocks: [] };
        const r = await job();
        if (whatIfAgain || target !== key) continue;
        if (r) whatIf = { key, pending: false, blocks: stamp(r) };
      } while (whatIfAgain);
    } finally {
      whatIfBusy = false;
    }
  };

  $effect(() => {
    void target;
    void view;
    void statDiffs;
    untrack(() => runWhatIf()).catch(logError);
  });

  // ItemsTab.lua:4580-4592
  let socketTip = $state<{ node: number; tip: ItemTip }>();
  $effect(() => {
    const m = hover ? /^Jewel (\d+)$/.exec(hover) : null;
    void $outputs;
    if (!m) return;
    const node = Number(m[1]);
    void untrack(() =>
      syncWrap
        ?.SocketJewelTooltip(node)
        .then((t) => {
          if (t && hover === `Jewel ${node}`) socketTip = { node, tip: itemTip(t) };
        })
        .catch(logError)
    );
  });

  // PassiveTreeView.lua:511-519
  let socketAsked = false;
  $effect(() => {
    const node = page.url.searchParams.get('socket');
    if (!view || !node || socketAsked) return;
    socketAsked = true;
    pageView = 'equipped';
    void tick().then(() => {
      const select = document.querySelector<HTMLSelectElement>(`[data-socket-node="${CSS.escape(node)}"] select`);
      select?.focus();
      try {
        select?.showPicker();
      } catch {
        // empty
      }
    });
  });

  // ItemSlotControl.lua:132-149
  let socketView = $state<{ node: number; tree: Tree; radii: { inner: number; outer: number; col: string; nodes: number[] }[]; alloc: Set<number> }>();
  $effect(() => {
    const m = hover ? /^Jewel (\d+)$/.exec(hover) : null;
    const t = $skillTree;
    void $outputs;
    if (!m) return;
    const node = Number(m[1]);
    if (!t) {
      if (treeVersion) loadSkillTree(treeVersion).catch(logError);
      return;
    }
    const alloc = loaded?.usage.specs[loaded.usage.activeSpec]?.alloc ?? new Set<number>();
    void untrack(() =>
      syncWrap
        ?.GetTreeSocketRadii(node)
        .then((radii) => {
          if (hover === `Jewel ${node}`) socketView = { node, tree: t, radii, alloc };
        })
        .catch(logError)
    );
  });

  $effect(() => {
    void pageView;
    untrack(() => (hover = null));
  });

  let text = $state('');
  let paste = $state<{ item?: ItemSummary; name?: string; error?: string; pending: boolean; blocks: ImpactBlock[] }>({
    pending: false,
    blocks: []
  });
  // ItemsTab.lua:378-715
  let display = $state<DisplayInfo>();
  let customFor: string | undefined;
  let customRequest = $state(false);
  // ItemsTab.lua:342-350
  let pasteTrade = $state<{ item: TradeItem; slot?: string; defences: TradeDefence[] }>();
  let buying = $state(false);
  let seq = 0;
  let running = false;
  let dirty = false;

  const runPaste = async () => {
    const mine = ++seq;
    const current = () => mine === seq;
    if (!text.trim()) {
      paste = { pending: false, blocks: [] };
      display = undefined;
      pasteTrade = undefined;
      return;
    }
    paste = { ...paste, pending: true, blocks: [] };
    const p = (await syncWrap?.ParseItem(text)) ?? {};
    if (!current()) return;
    if (!p.item) {
      paste = { error: p.error, pending: false, blocks: [] };
      pasteTrade = undefined;
      return;
    }
    paste = { item: summary(p.item), name: p.item.Name, pending: true, blocks: [] };
    const tradeItem = { rarity: p.item.Rarity, title: p.item.Title, name: p.item.Name, baseName: p.item.BaseName, type: p.item.Type };
    pasteTrade = { item: tradeItem, slot: p.item.FitSlots?.[0], defences: tradeDefences(tradeItem, p.item.ArmourData) };
    const [d, c, affixes, extras, bases] = await Promise.all([
      syncWrap?.GetDisplayItem(text),
      syncWrap?.GetDisplayItemCraft(text),
      syncWrap?.GetAffixControls(text),
      syncWrap?.GetDisplayItemExtras(text),
      syncWrap?.RebaseChoices(text)
    ]);
    display = d && { ...d, ...c, affixes: affixes && { ...affixes, base: p.item.BaseName, bases }, ...extras };
    if (!current()) return;
    if (customFor === text) {
      customFor = undefined;
      customRequest = true;
    }
    const result = await syncWrap?.ItemCompare(text);
    if (!current()) return;
    if (result) paste = { ...paste, pending: false, blocks: stamp(result) };
  };

  const refreshPaste = async () => {
    if (running) {
      seq++;
      dirty = true;
      return;
    }
    running = true;
    try {
      do {
        dirty = false;
        await runPaste();
      } while (dirty);
    } finally {
      running = false;
    }
  };

  $effect(() => {
    void $outputs;
    void text;
    untrack(() => refreshPaste()).catch(logError);
  });

  let message = $state<string>();
  const say = (m: string) => (message = m);

  // ItemsTab.lua:335-337
  let edited = $state<{ id: number; name: string }>();

  // ItemsTab.lua:338-340
  const oncancelpaste = () => {
    edited = undefined;
    text = '';
  };

  const onsave = async (equip: boolean, slot: string | undefined) => {
    const t = text;
    if (!equip && edited) {
      // ItemsTab.lua:2626-2629
      const error = await syncWrap?.ReplaceItem(edited.id, t);
      if (!error) oncancelpaste();
      say(error ?? 'Saved.');
      return;
    }
    if (equip && slot) {
      const error = await syncWrap?.EquipItem(t, slot);
      say(error ?? `Equipped in ${slotLabel(slot)}.`);
      return;
    }
    const r = await syncWrap?.SaveItem(t);
    if (!r?.error && !r?.slotName) saved = [...saved, { id: -(saved.length + 1), text: t }];
    say(r?.error ?? (r?.slotName ? `Saved and equipped in ${slotLabel(r.slotName)}.` : 'Saved to the item list.'));
  };

  const onequip = async (slot: string, id: number | null) => {
    if (hover === slot) hover = null;
    const error = await syncWrap?.EquipItemID(slot, id ?? 0);
    if (error) say(error);
  };

  const onset = async (id: string) => {
    hover = null;
    await syncWrap?.SetActiveItemSet(Number(id));
    epoch++;
    await load($buildRevision, $outputs);
  };

  const onswap = async (second: boolean) => {
    hover = null;
    await syncWrap?.SetWeaponSet(second);
    epoch++;
    await load($buildRevision, $outputs);
  };

  const onsets = async (op: ItemSetOp) => {
    hover = null;
    if (op.op === 'new' || op.op === 'move') await syncWrap?.ItemSetOp(op);
    else await syncWrap?.ItemSetOp({ ...op, id: Number(op.id) });
    epoch++;
    await load($buildRevision, $outputs);
  };

  const reread = async () => {
    hover = null;
    epoch++;
    await load($buildRevision, $outputs);
  };
  const ondelete = async (id: number, confirmed: boolean) => {
    if (!confirmed) {
      const prompt = await syncWrap?.DeleteItemPrompt(id);
      if (prompt) return prompt;
    }
    if (editing?.key === String(id)) editing = null;
    await syncWrap?.DeleteItem(id);
    await reread();
    return undefined;
  };

  // ItemSlotControl.lua:12-19, 61-68
  const onjewel = (node: number, id: number | null) => {
    syncWrap
      ?.SocketTreeJewel(node, id ?? 0)
      .then(reread)
      .catch(logError);
  };

  const onsort = () => {
    syncWrap?.SortItems().then(reread).catch(logError);
  };
  const ontoggle = (id: number, second: boolean) => {
    syncWrap?.ToggleEquipItem(id, second).then(reread).catch(logError);
  };
  // ItemListControl.lua:326-329
  const oncopy = (id: number) => {
    const t = view?.texts[id];
    if (t !== undefined) navigator.clipboard?.writeText(unescapeXml(t).replace(/\r?\n/g, '\r\n')).catch(logError);
  };

  const ondisplayedit = async (edit: DisplayEdit) => {
    const r = await syncWrap?.EditItemText(text, edit);
    if (r?.text !== undefined) text = r.text;
    else if (r?.error) say(r.error);
  };

  // ItemsTab.lua:2856, 3335, 3653, 4368
  const popupTip = async (t: string): Promise<TipProps | undefined> => {
    const [p, r] = await Promise.all([syncWrap?.ParseItem(t), syncWrap?.ItemCompare(t)]);
    if (!p?.item) return undefined;
    const tip = { ...tipOf(summary(p.item), undefined, 'candidate'), impactTitle: '', impactSub: undefined, impact: [] };
    return { ...tip, whatIf: { key: 'popup', pending: false, blocks: r ? stamp(r) : [] } };
  };

  // ItemsTab.lua:2661-3015
  const displayCraft: CraftApi = {
    enchantChoices: async () => syncWrap?.GetEnchantChoices(text),
    notables: async () => (await syncWrap?.GetAnointNotables()) ?? [],
    sortStats: async () => ((await syncWrap?.GetPowerStats()) ?? []).map((s) => ({ stat: s.Stat, label: s.Label })),
    preview: async (edit) => {
      const r = await syncWrap?.EditItemText(text, edit);
      return r?.text !== undefined ? popupTip(r.text) : undefined;
    },
    enchantSort: async (slot, list, stat) => (await syncWrap?.EnchantSortValues(text, slot, list, stat)) ?? [],
    anointPowers: async (slot, ids, stat) => (await syncWrap?.AnointPowers(text, slot, ids, stat)) ?? [],
    anointCompare: async (slot, id) => {
      const r = await syncWrap?.AnointCompare(text, slot, id);
      if (!r) return undefined;
      return r.note ? { note: r.note, blocks: [] } : { blocks: stamp({ blocks: r.block ? [r.block] : [], calcMs: 0 }) };
    },
    corruptChoices: async () => syncWrap?.GetCorruptChoices(text),
    rangeLine: async (index, value) => (await syncWrap?.CorruptedRangeLine(text, index, value)) ?? '',
    corruptText: async (source, ids, downside, rolls) => syncWrap?.CorruptItemText(text, source, ids, downside, rolls),
    corruptSort: async (source, modType, ids, stat) => (await syncWrap?.CorruptSortValues(text, source, modType, ids, stat)) ?? [],
    modCompare: async (modLines) => {
      const b = await syncWrap?.AddedModCompare(text, modLines, '');
      return b ? stamp({ blocks: [b], calcMs: 0 }) : [];
    },
    implicitChoices: async () => syncWrap?.GetImplicitChoices(text),
    implicitText: async (source, modId, custom) => syncWrap?.ImplicitText(text, source, modId, custom),
    implicitSort: async (source, ids, stat) => (await syncWrap?.ImplicitSortValues(text, source, ids, stat)) ?? [],
    customModChoices: async () => syncWrap?.GetCustomModChoices(text),
    customModText: async (source, modId, custom) => syncWrap?.CustomModText(text, source, modId, custom),
    customModSort: async (source, ids, stat) => (await syncWrap?.CustomModSortValues(text, source, ids, stat)) ?? [],
    affixEdit: async (slot, series, slider) => {
      const t = await syncWrap?.AffixEditText(text, slot, series, slider);
      if (t !== undefined) text = t;
      return t;
    },
    affixRollTooltip: async (slot, slider) => (await syncWrap?.AffixRollTooltip(text, slot, slider)) ?? [],
    affixPowers: async (slot, stat) => (await syncWrap?.AffixPowers(text, slot, stat)) ?? [],
    rebase: async (base) => {
      const r = await syncWrap?.RebaseItemText(text, base);
      if (r) text = r.text;
      return r?.dropped;
    },
    crucibleChoices: async () => syncWrap?.GetCrucibleChoices(text),
    crucibleText: async (ids) => syncWrap?.CrucibleText(text, ids),
    tipOf: popupTip,
    settext: (t) => {
      text = t;
    }
  };

  // ItemsTab.lua:2519-2603
  let craftBases = $state<{ type: string; bases: string[] }[]>([]);
  let craftFrom = $state<string>();
  let craftPreview = $state<TipProps>();
  let craftBasesAsked = false;
  $effect(() => {
    if ($currentBuild && !craftBasesAsked) {
      craftBasesAsked = true;
      void untrack(() =>
        syncWrap
          ?.GetCraftBases()
          .then((g) => (craftBases = g))
          .catch(logError)
      );
    }
  });
  // ItemsTab.lua:2582-2587
  let craftSeq = 0;
  const oncraftpreview = async (c: CraftChoice) => {
    const mine = ++craftSeq;
    const t = await syncWrap?.CraftItemText(c);
    const p = t ? await syncWrap?.ParseItem(t) : undefined;
    if (mine !== craftSeq) return;
    if (!t || !p?.item) {
      craftPreview = undefined;
      return;
    }
    const tip = { ...tipOf(summary(p.item), undefined, 'candidate'), impactTitle: '', impactSub: undefined, impact: [] };
    craftPreview = { ...tip, whatIf: { key: 'craft', pending: true, blocks: [] } };
    const r = await syncWrap?.ItemCompare(t);
    if (mine === craftSeq) craftPreview = { ...tip, whatIf: { key: 'craft', pending: false, blocks: r ? stamp(r) : [] } };
  };
  const oncraftcreate = async (c: CraftChoice) => {
    const t = await syncWrap?.CraftItemText(c);
    if (t === undefined) return;
    edited = undefined;
    text = t;
    pageView = 'equipped';
  };

  const onflask = (slot: string, active: boolean) => {
    syncWrap?.SetFlaskActive(slot, active).catch(logError);
  };

  // ItemsTab.lua:273-296

  const dbFilters = $state<Record<ItemDBKind, ItemDBFilterInput>>({ UNIQUE: defaultItemDBFilter(), RARE: defaultItemDBFilter() });
  let dbSort = $state('');
  let dbLists = $state<Partial<Record<ItemDBKind, ItemDBLists>>>({});
  let dbRows = $state<ItemDBRowPlain[]>([]);
  let dbBusy = $state(false);
  let dbNote = $state('');
  let dbTip = $state<{ name: string; tip: ItemTip }>();
  let dbSeq = 0;

  $effect(() => {
    const kind = dbKind;
    if (pageView !== 'browse' || dbLists[kind]) return;
    void untrack(() =>
      syncWrap
        ?.GetItemDBLists(kind)
        .then((l) => (dbLists = { ...dbLists, [kind]: l }))
        .catch(logError)
    );
  });

  // ItemDBControl.lua:288-296
  $effect(() => {
    const kind = dbKind;
    const filter = $state.snapshot(dbFilters[kind]);
    const sort = kind === 'UNIQUE' ? dbSort : '';
    void $outputs;
    if (pageView !== 'browse') return;
    const mine = ++dbSeq;
    untrack(() => {
      dbBusy = true;
      syncWrap
        ?.GetItemDBRows(kind, filter, sort)
        .then((rows) => {
          if (mine !== dbSeq) return;
          dbRows = rows;
          dbBusy = false;
        })
        .catch(logError);
    });
  });

  $effect(() => {
    const name = dbHover;
    const kind = dbKind;
    void $outputs;
    if (name === null) return;
    void untrack(() =>
      syncWrap
        ?.GetItemDBTooltip(kind, name)
        .then((t) => {
          if (t && dbHover === name) dbTip = { name, tip: itemTip(t) };
        })
        .catch(logError)
    );
  });
  const dbTipProps = $derived.by((): TipProps | undefined => {
    const row = dbRows.find((r) => r.name === dbHover);
    if (!row || dbTip?.name !== row.name) return undefined;
    const key = `db:${dbKind}:${row.name}`;
    return {
      item: row.title || row.name,
      base: row.base,
      rarity: row.rarity.toLowerCase() as TipProps['rarity'],
      mods: [],
      impactTitle: '',
      impact: [],
      tooltip: dbTip.tip,
      whatIf: whatIf?.key === key ? whatIf : undefined
    };
  });

  const dbAdd = async (name: string, equip: boolean, second: boolean) => {
    if (!(await syncWrap?.AddDBItem(dbKind, name, equip, second))) return;
    dbNote = equip ? `Added ${name} and equipped it in its slot` : `Added ${name} to the build's items`;
    await reread();
  };
  // ItemsTab.lua:1859-1869
  const normalise = async (t: string) => (await syncWrap?.NormaliseDisplayItem(t, $migrateEldritchImplicits)) ?? t;
  // ItemDBControl.lua:361-367
  const dbOpen = async (name: string) => {
    const t = await syncWrap?.GetItemDBText(dbKind, name);
    if (t === undefined) return;
    edited = undefined;
    text = await normalise(t);
    dbHover = null;
    pageView = 'equipped';
  };
  // ItemDBControl.lua:371-373
  const dbCopy = async (name: string) => {
    const t = await syncWrap?.GetItemDBRaw(dbKind, name);
    if (t !== undefined) await navigator.clipboard?.writeText(t.replace(/\r?\n/g, '\r\n'));
  };

  // ItemSlotControl.lua:114-130
  const ondropslot = async (slot: string, d: ItemDrag) => {
    hover = null;
    const ok =
      d.shared !== undefined ? await syncWrap?.AddSharedItem(d.shared, slot) : await syncWrap?.DropItemOnSlot(slot, d.id ?? 0, d.db ?? '', d.name ?? '');
    if (!ok) {
      say(`That item does not fit ${slotLabel(slot)}.`);
      return;
    }
    await reread();
  };
  // Build.lua:541-543
  $effect(() => {
    const listener = (e: Event) => {
      const { setId, drag, second, uses } = (e as CustomEvent<MinionItemDrop>).detail;
      // Build.lua:534-540
      syncWrap
        ?.ItemPrimarySlot(drag.id ?? 0, drag.db ?? '', drag.name ?? '')
        .then((slot) => (uses.includes(slot) ? syncWrap?.EquipItemInSet(setId, drag.id ?? 0, drag.db ?? '', drag.name ?? '', second) : false))
        .then((ok) => (ok ? reread() : undefined))
        .catch(logError);
    };
    window.addEventListener(MINION_ITEM_DROP, listener);
    return () => window.removeEventListener(MINION_ITEM_DROP, listener);
  });

  // ItemListControl.lua:275-284
  // ListControl.lua:428-436
  const ondropstash = async (d: ItemDrag, before?: number) => {
    if (d.id) {
      if (before && before !== d.id) {
        await syncWrap?.MoveItem(d.id, before);
        await reread();
      }
      return;
    }
    if (d.shared !== undefined) {
      if (await syncWrap?.AddSharedItem(d.shared, '', before ?? 0)) await reread();
      return;
    }
    if (before) {
      if (d.db && d.name && (await syncWrap?.AddDBItemAt(d.db, d.name, before))) await reread();
      return;
    }
    if (!d.db || !d.name || !(await syncWrap?.AddDBItem(d.db, d.name, false, false))) return;
    await reread();
  };

  const onpasteitem = async (t: string) => {
    edited = undefined;
    text = await normalise(t);
  };
  const sharedRow = async (raw: string): Promise<PoolItem> => {
    const p = (await syncWrap?.ParseItem(raw))?.item;
    const s = p && summary(p);
    return {
      name: s?.name ?? raw.split('\n')[1] ?? raw,
      rarity: (s?.rarity ?? 'normal') as PoolItem['rarity'],
      base: s?.base ?? '',
      slot: s?.fits[0] ?? '—',
      scope: 'all builds',
      mods: [...(s?.implicits ?? []), ...(s?.explicits ?? [])].map((m) => m.text),
      impact: []
    };
  };
  const readShared = () =>
    Promise.all([
      syncWrap?.GetSharedItemSets().then((l) => (sharedSets = l)),
      syncWrap
        ?.GetSharedItems()
        .then((texts) => Promise.all(texts.map(async (raw) => ({ text: raw, row: await sharedRow(raw) }))))
        .then((l) => (sharedItems = l))
    ]).catch(logError);
  // SharedItemListControl.lua:55-72
  const onshareditem = async (op: { op: 'delete' | 'copy' | 'open'; index: number }) => {
    const t = sharedItems[op.index]?.text;
    if (t === undefined) return;
    if (op.op === 'copy') await navigator.clipboard?.writeText(t.replace(/\r?\n/g, '\r\n'));
    if (op.op === 'open') {
      edited = undefined;
      text = await normalise(t);
      pageView = 'equipped';
    }
    if (op.op === 'delete' && (await syncWrap?.DeleteSharedItem(op.index))) await readShared();
  };
  const ondropshared = async (d: ItemDrag) => {
    if (d.shared !== undefined) return;
    if (await syncWrap?.ShareItem(d.id ?? 0, d.db ?? '', d.name ?? '', Number.MAX_SAFE_INTEGER)) await readShared();
  };
  const onshared = async (op: { op: 'share'; id: string } | { op: 'import' | 'delete'; index: number } | { op: 'rename'; index: number; title: string }) => {
    if (op.op === 'share') await syncWrap?.ShareItemSet(Number(op.id));
    if (op.op === 'delete') await syncWrap?.DeleteSharedItemSet(op.index);
    if (op.op === 'rename') await syncWrap?.RenameSharedItemSet(op.index, op.title);
    if (op.op === 'import' && (await syncWrap?.ImportSharedItemSet(op.index))) await reread();
    await readShared();
  };
  $effect(() => {
    void $currentBuild;
    void untrack(() => readShared());
  });

  const editedText = (ed: Editing) => {
    const key = ed.key;
    const slot = key === undefined ? undefined : view?.equipped.find((e) => e.SlotName === key);
    const t = key === undefined ? undefined : (slot?.Text ?? view?.texts[key]);
    return t === undefined ? undefined : { id: slot?.ItemID ?? Number(key), name: slot?.Name ?? ed.item.name, text: unescapeXml(t) };
  };
  const onedittext = (ed: Editing) => {
    const e = editedText(ed);
    if (!e) return;
    edited = e.id > 0 ? { id: e.id, name: e.name } : undefined;
    text = e.text;
    pageView = 'equipped';
  };
  const onclone = (ed: Editing) => {
    const e = editedText(ed);
    if (!e) return;
    edited = undefined;
    text = e.text;
    pageView = 'equipped';
  };
  const onmodtable = (ed: Editing) => {
    const e = editedText(ed);
    if (!e) return;
    const same = e.text === text && display;
    onedittext(ed);
    if (same) customRequest = true;
    else customFor = e.text;
  };
  // ItemsTab.lua:991-993
  let editorModTable = $state<boolean>();
  $effect(() => {
    const e = editing && untrack(() => editedText(editing!));
    editorModTable = undefined;
    if (!e) return;
    void Promise.resolve(syncWrap?.GetDisplayItemCraft(e.text))
      .then((c) => {
        if (editing && untrack(() => editedText(editing!))?.text === e.text) editorModTable = !!c?.addCustom;
      })
      .catch(logError);
  });

  // ItemsTab.lua:1471-1484
  // stat differences
  const onkeydown = (e: KeyboardEvent) => {
    if (!(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey) return;
    const k = e.key.toLowerCase();
    if (k === 'd') {
      e.preventDefault();
      void Promise.resolve(syncWrap?.ToggleStatDifferences())
        .then((on) => (statDiffs = on ?? true))
        .catch(logError);
    } else if (k === 'f') {
      e.preventDefault();
      const focused = document.activeElement?.getAttribute('aria-label');
      dbKind = focused === 'Search uniques' || (dbKind === 'RARE' && focused !== 'Search templates') ? 'RARE' : 'UNIQUE';
      pageView = 'browse';
      const label = `Search ${dbKind === 'RARE' ? 'templates' : 'uniques'}`;
      const focus = (tries: number) => {
        const el = document.querySelector<HTMLInputElement>(`input[aria-label="${label}"]`);
        if (el) el.focus();
        else if (tries > 0) setTimeout(() => focus(tries - 1), 30);
      };
      void tick().then(() => focus(10));
    }
  };
</script>

<svelte:window {onkeydown} />
{#if view}
  <ItemsPage
    bind:layout
    data={{ activeSet: view.activeSet, items: view.items, sets: view.sets, impact: {}, jewels: view.jewels, stash: view.stash }}
    sets={view.sets}
    setId={view.activeSet}
    {flasks}
    jewels={view.jewels}
    abyssal={view.abyssal}
    extraSlots={view.extraSlots}
    {socketTip}
    added={[]}
    shared={sharedItems.map((s) => s.row)}
    browse={[]}
    affixes={AFFIXES}
    bind:view={pageView}
    bind:hover
    bind:editing
    {whatIf}
    craft={false}
    slotItems={view.slotItems}
    readonly={READONLY}
    paste={{ ...paste, message, editing: edited?.name }}
    bind:pasteText={text}
    onset={(id: string) => {
      onset(id).catch(logError);
    }}
    onsets={(op: ItemSetOp) => {
      onsets(op).catch(logError);
    }}
    onswap={(second: boolean) => {
      onswap(second).catch(logError);
    }}
    {onflask}
    onequip={(slot: string, id: number | null) => {
      onequip(slot, id).catch(logError);
    }}
    onsave={(_: ItemSummary, slot: string | undefined, equip: boolean) => {
      onsave(equip, slot).catch(logError);
    }}
    {onedittext}
    {editorModTable}
    {onmodtable}
    {onclone}
    bind:customRequest
    {oncancelpaste}
    onbuysimilar={pasteTrade && (() => (buying = true))}
    {ondelete}
    {onjewel}
    {display}
    ondisplayedit={(e: DisplayEdit) => {
      ondisplayedit(e).catch(logError);
    }}
    {displayCraft}
    {craftBases}
    {craftPreview}
    oncraftpreview={(c: CraftChoice) => {
      oncraftpreview(c).catch(logError);
    }}
    bind:craftFrom
    oncraftcreate={(c: CraftChoice) => {
      oncraftcreate(c).catch(logError);
    }}
    {stashFilters}
    {stashFilter}
    {sharedSets}
    {socketView}
    onshared={(op: Parameters<typeof onshared>[0]) => {
      onshared(op).catch(logError);
    }}
    specs={treeSpecs.labels}
    spec={treeSpecs.active}
    onspec={(i: number) => {
      hover = null;
      syncWrap
        ?.SetActiveTreeSpec(i)
        .then(() => {
          epoch++;
          return load($buildRevision, $outputs);
        })
        .catch(logError);
    }}
    onspecmanage={() => {
      treeManageRequest.set(true);
      void goto(`${appBase}/tree`);
    }}
    onstashfilter={(f: string) => {
      stashFilter = f;
      load($buildRevision, $outputs).catch(logError);
    }}
    ondropslot={(slot: string, d: ItemDrag) => {
      ondropslot(slot, d).catch(logError);
    }}
    ondropstash={(d: ItemDrag, before?: number) => {
      ondropstash(d, before).catch(logError);
    }}
    onpasteitem={(t: string) => {
      onpasteitem(t).catch(logError);
    }}
    onshareditem={(op: Parameters<typeof onshareditem>[0]) => {
      onshareditem(op).catch(logError);
    }}
    ondropshared={(d: ItemDrag) => {
      ondropshared(d).catch(logError);
    }}
    {onsort}
    {ontoggle}
    {oncopy}
    ondeleteunused={() => {
      syncWrap?.DeleteUnusedItems().then(reread).catch(logError);
    }}
    ondeleteall={() => {
      syncWrap?.DeleteAllItems().then(reread).catch(logError);
    }}>
    {#snippet itemDB()}
      <ItemDB
        bind:kind={dbKind}
        lists={dbLists[dbKind]}
        bind:filter={dbFilters[dbKind]}
        bind:sort={dbSort}
        rows={dbRows}
        busy={dbBusy}
        bind:hover={dbHover}
        tip={dbTipProps}
        note={dbNote}
        onadd={(name: string, equip: boolean, second: boolean) => {
          dbAdd(name, equip, second).catch(logError);
        }}
        onopen={(name: string) => {
          dbOpen(name).catch(logError);
        }}
        oncopy={(name: string) => {
          dbCopy(name).catch(logError);
        }}
        oncraft={(base: string) => (craftFrom = base)} />
    {/snippet}
  </ItemsPage>
{/if}
<BuySimilarDialog
  bind:open={buying}
  item={pasteTrade?.item}
  slot={pasteTrade?.slot}
  defences={pasteTrade?.defences}
  onsearch={(url: string) => window.open(url, '_blank', 'noopener')} />
