<script lang="ts">
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { phoneMedia } from '$lib/design/mobile/mobile';
  import SkillsPanes from '$lib/design/mobile/SkillsPanes.svelte';
  import { onMount, untrack } from 'svelte';
  import SkillsPage from '$lib/design/SkillsPage.svelte';
  import type { GemOptions, SkillSetOp } from '$lib/design/skills';
  import type { GemAttr, SkillGemFixture, SocketGroupFixture } from '$lib/design/fixtures';
  import { buildRevision, currentBuild, outputs, UITick } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { GetSkillGems, type SkillGemCacheItem } from '$lib/cache';
  import { populateGems } from '$lib/design/gem_search';
  import { colorCodesX } from '$lib/display/colors';
  import { base } from '$app/paths';
  import { slotIcon } from '$lib/design/slot_icon';
  import { explodeSourceNote, itemSourceNote, nodeSourceNote } from './skill_source';
  import { logError } from '$lib/utils';
  import { send } from '$lib/type_utils';
  import type { pob } from '$lib/types';

  // THE LOAD PATH NEVER WRITES

  type Gem = { -readonly [K in keyof pob.Gem]: pob.Gem[K] };
  type Group = {
    label: string;
    display: string;
    enabled: boolean;
    inFull: boolean;
    slot: string;
    source: string;
    imbued: string;
    groupCount?: number;
    gems: Gem[];
  };
  type View = {
    setTitles: string[];
    setIds: number[];
    activeSet: number;
    groups: Group[];
    main: number;
  };

  let view = $state<View>();
  let selected = $state(0);

  let editSeq = 0;
  let loadSeq = 0;

  const load = async (build: NonNullable<typeof $currentBuild>) => {
    const seq = editSeq;
    const token = ++loadSeq;
    const skills = build.Skills;
    const activeSet = Math.max(0, (await build.ActiveSkillSetIndex()) ?? 0);
    const setCount = Math.max(1, (await skills.SkillSets.length) ?? 1);
    const set = skills.SkillSets[activeSet];
    const [titles, setIds, summaries, info] = await Promise.all([
      Promise.all(Array.from({ length: setCount }, (_, i) => skills.SkillSets[i].Title)),
      Promise.all(Array.from({ length: setCount }, (_, i) => skills.SkillSets[i].ID)),
      syncWrap?.GetSocketGroupSummaries(activeSet) ?? [],
      syncWrap?.GetBuildInfo()
    ]);
    const groups = await Promise.all(
      summaries.map(async (row, i) => {
        const g = set.Skills[i];
        const [label, slot, source, sourceAttr, gems, groupCount, imbued] = await Promise.all([
          g.Label,
          g.Slot,
          g.Source,
          g.SourceAttr,
          g.Gems,
          g.GroupCount,
          g.ImbuedSupport
        ]);
        return {
          label: label ?? '',
          display: row.label,
          enabled: row.enabled,
          inFull: row.fullDPS,
          slot: slot ?? '',
          source: sourceAttr || source || '',
          imbued: imbued ?? '',
          groupCount: groupCount ?? undefined,
          gems: gems ?? []
        };
      })
    );
    if (seq !== editSeq || token !== loadSeq) return;
    view = {
      setTitles: titles.map((t, i) => t || (i === 0 ? 'Default' : `Skill set ${i + 1}`)),
      setIds,
      activeSet,
      groups,
      main: (info?.mainSocketGroup ?? 0) - 1
    };
    selected = Math.min(selected, Math.max(0, groups.length - 1));
  };

  $effect(() => {
    const build = $currentBuild;
    if (!build) return;
    untrack(() => load(build)).catch(logError);
  });

  let gemList = $state<SkillGemCacheItem[]>([]);
  onMount(() => {
    GetSkillGems()
      .then((all) => (gemList = all))
      .catch(logError);
  });
  const ATTR: Record<string, GemAttr> = { STR: 'str', DEX: 'dex', INT: 'int', NONE: 'white' };
  const gemById = $derived(new Map(gemList.map((g) => [g.ID, g])));
  const gemByName = $derived.by(() => {
    const m = new Map<string, SkillGemCacheItem>();
    for (const g of gemList) if (g.Base?.Name && !m.has(g.Base.Name)) m.set(g.Base.Name, g);
    return m;
  });

  let socketsBySlot = $state<Record<string, string>>({});
  let itemBySlot = $state<Record<string, { name: string; rarity: string; type: string }>>({});
  $effect(() => {
    void $buildRevision;
    if (!$currentBuild || !syncWrap) return;
    const sw = syncWrap;
    untrack(async () => {
      const out: Record<string, string> = {};
      const names: Record<string, { name: string; rarity: string; type: string }> = {};
      for (const e of await sw.GetEquippedItems()) {
        if (e.Flask) continue;
        const { item } = await sw.ParseItem(e.Text);
        if (item?.Sockets) out[e.SlotName] = item.Sockets;
        if (item) names[e.SlotName] = { name: item.Name, rarity: item.Rarity, type: item.Type };
      }
      socketsBySlot = out;
      itemBySlot = names;
    }).catch(logError);
  });

  // SkillsTab.lua:1166-1181
  let gemErrors = $state<Record<string, string>>({});
  $effect(() => {
    const names = new Set((view?.groups ?? []).flatMap((g) => g.gems.filter((m) => !m.GemID && !m.SkillID && m.NameSpec.trim()).map((m) => m.NameSpec)));
    for (const name of names) {
      if (name in untrack(() => gemErrors)) continue;
      syncWrap
        ?.GemError('', '', name)
        .then((msg) => (gemErrors = { ...gemErrors, [name]: msg }))
        .catch(logError);
    }
  });

  const toFixture = (g: Group): SocketGroupFixture & { display: string } => ({
    label: g.label,
    display: g.display,
    slot: g.slot,
    enabled: g.enabled,
    inFull: g.inFull,
    count: g.groupCount ?? 1,
    sockets: socketsBySlot[g.slot] ?? '',
    gems: g.gems.map((m) => {
      const def = gemById.get(m.GemID);
      return {
        name: def?.Base?.Name || m.NameSpec || m.SkillID,
        level: m.Level,
        quality: m.Quality,
        enabled: m.Enabled,
        support: !!def?.Support,
        attr: (def && ATTR[def.GemType]) ?? 'white',
        count: m.Count ?? 1,
        error: !m.GemID && !m.SkillID ? gemErrors[m.NameSpec] : undefined,
        globals: def?.GlobalEffects?.length
          ? [
              { label: def.GlobalEffects[0] ?? '', on: m.EnableGlobal1 },
              { label: def.GlobalEffects[1] ?? '', on: m.EnableGlobal2 }
            ]
          : undefined
      };
    })
  });
  const sets = $derived(view ? view.setTitles.map((title, i) => ({ title, groups: i === view!.activeSet ? view!.groups.map(toFixture) : [] })) : []);

  const SLOT_NAMES = [
    'Weapon 1',
    'Weapon 2',
    'Weapon 1 Swap',
    'Weapon 2 Swap',
    'Helmet',
    'Body Armour',
    'Gloves',
    'Boots',
    'Amulet',
    'Ring 1',
    'Ring 2',
    'Ring 3',
    'Belt'
  ];
  let slotItems = $state<Record<string, string>>({});
  $effect(() => {
    void $buildRevision;
    syncWrap
      ?.GetSocketedInItemNames(SLOT_NAMES)
      .then((names) => (slotItems = names))
      .catch(logError);
  });
  const SLOTS = $derived([
    { value: 'None', label: 'None' },
    ...SLOT_NAMES.map((s) => ({ value: s, label: s.replace(/ Swap$/, ' (Swap)'), title: slotItems[s] ?? 'No item is equipped in this slot.' }))
  ]);

  const groupRef = (i: number) => {
    const build = $currentBuild;
    if (!build || !view) return undefined;
    return build.Skills.SkillSets[view.activeSet].Skills[i];
  };

  const writeGems = async (i: number) => {
    const build = $currentBuild;
    const g = view?.groups[i];
    if (!build || !view || !g) return;
    editSeq++;
    await build.SetSocketGroupGems(view.activeSet, i, $state.snapshot(g.gems));
    UITick('currentGemGroup');
  };

  const commitMainSocketGroup = (index: number) => {
    if (view) view.main = index;
    if (index >= 0) {
      syncWrap?.SetMainSocketGroup(index + 1).catch(logError);
    }
    currentBuild.set($currentBuild);
  };

  const ongroup = (i: number, patch: Partial<SocketGroupFixture>) => {
    const g = view?.groups[i];
    const ref = groupRef(i);
    if (!g || !ref) return;
    editSeq++;
    if (patch.label !== undefined && patch.label !== g.label) {
      g.label = patch.label;
      ref.Label = send(patch.label);
      UITick('socketGroupLabel');
    }
    let changed = false;
    if (patch.slot !== undefined && patch.slot !== g.slot && view) {
      g.slot = patch.slot;
      // SkillsTab.lua:168-184
      Promise.resolve($currentBuild?.SetSocketGroupSlot(view.activeSet, i, patch.slot))
        .then(() => UITick('currentGemGroup'))
        .catch(logError);
    }
    if (patch.enabled !== undefined && patch.enabled !== g.enabled) {
      g.enabled = patch.enabled;
      ref.Enabled = send(patch.enabled);
      changed = true;
    }
    // SkillsTab.lua:349-353
    if (patch.count !== undefined && patch.count !== (g.groupCount ?? 1)) {
      g.groupCount = patch.count;
      ref.GroupCount = send(patch.count);
      changed = true;
    }
    if (patch.inFull !== undefined && patch.inFull !== g.inFull) {
      g.inFull = patch.inFull;
      ref.IncludeInFullDPS = send(patch.inFull);
      changed = true;
    }
    if (changed) UITick('currentGemGroup');
  };

  const ongem = (i: number, gem: number, patch: Partial<SkillGemFixture>) => {
    const m = view?.groups[i]?.gems[gem];
    if (!m) return;
    let changed = false;
    if (patch.level !== undefined && patch.level !== m.Level) {
      m.Level = patch.level;
      changed = true;
    }
    if (patch.quality !== undefined && patch.quality !== m.Quality) {
      m.Quality = patch.quality;
      changed = true;
    }
    if (patch.enabled !== undefined && patch.enabled !== m.Enabled) {
      m.Enabled = patch.enabled;
      changed = true;
    }
    // SkillsTab.lua:1015-1020
    if (patch.globals) {
      const [g1, g2] = patch.globals;
      if (g1 && g1.on !== m.EnableGlobal1) {
        m.EnableGlobal1 = g1.on;
        changed = true;
      }
      if (g2 && g2.on !== m.EnableGlobal2) {
        m.EnableGlobal2 = g2.on;
        changed = true;
      }
    }
    if (changed) writeGems(i).catch(logError);
  };
  const ongemcount = (i: number, gem: number, count: number) => {
    const m = view?.groups[i]?.gems[gem];
    if (!m || m.Count === count) return;
    m.Count = count;
    writeGems(i).catch(logError);
  };

  let options = $state<GemOptions>();

  // SkillsTab.lua:818, GemSelectControl.lua:587-600
  // SkillsTab.lua:768-800
  const onaddgem = async (i: number, picked: SkillGemFixture) => {
    const def = gemByName.get(picked.name);
    const g = view?.groups[i];
    if (!def || !g || !syncWrap) return;
    const level = (await syncWrap.GemDefaultLevel(def.ID)) || 1;
    const gem = await syncWrap.DefaultGem(def.ID, def.Base.Name, def.ID, level, options?.defaultQuality ?? 0);
    g.gems.push(gem);
    await writeGems(i);
  };
  const onremovegem = (i: number, gem: number) => {
    const g = view?.groups[i];
    if (!g?.gems[gem]) return;
    g.gems.splice(gem, 1);
    writeGems(i).catch(logError);
  };
  const onreordergems = (from: number, to: number) => {
    const g = view?.groups[selected];
    if (!g || from === to) return;
    g.gems.splice(to, 0, ...g.gems.splice(from, 1));
    writeGems(selected).catch(logError);
  };

  // SkillsTab.lua:95-98
  const onset = (i: number) => {
    const id = view?.setIds[i];
    if (id === undefined) return;
    editSeq++;
    selected = 0;
    syncWrap?.SetActiveSkillSet(id).catch(logError);
  };

  const onsets = (op: SkillSetOp) => {
    const id = op.op === 'new' ? undefined : view?.setIds[op.index];
    if (op.op !== 'new' && id === undefined) return;
    editSeq++;
    const done = (() => {
      switch (op.op) {
        case 'new':
          return syncWrap?.NewSkillSet(op.title);
        case 'copy':
          return syncWrap?.CopySkillSet(id!, op.title);
        case 'rename':
          return syncWrap?.RenameSkillSet(id!, op.title);
        case 'delete':
          return syncWrap?.DeleteSkillSet(id!);
      }
    })();
    Promise.resolve(done).catch(logError);
  };

  // SkillListControl.lua:123-142
  const onreordergroups = (from: number, to: number) => {
    const v = view;
    if (!v || from === to || !v.groups[from] || !v.groups[to]) return;
    editSeq++;
    const follow = (i: number) => (i === from ? to : i > from && i <= to ? i - 1 : i < from && i >= to ? i + 1 : i);
    v.groups.splice(to, 0, ...v.groups.splice(from, 1));
    v.main = v.main < 0 ? v.main : follow(v.main);
    selected = follow(selected);
    syncWrap?.MoveSocketGroup(from, to).catch(logError);
  };

  const onnewgroup = async () => {
    editSeq++;
    await $currentBuild?.AddNewSocketGroup();
    if (view) selected = view.groups.length;
    currentBuild.set($currentBuild);
    UITick('socketGroups');
  };
  // SkillsTab.lua:667-713, SkillListControl.lua:148-152
  const oncopygroup = async (i: number) => {
    const build = $currentBuild;
    if (!build || view?.groups[i]?.source) return;
    const text = await build.CopySocketGroup(i);
    await navigator.clipboard?.writeText(text);
  };
  const onpastegroup = async (text: string) => {
    const build = $currentBuild;
    if (!build) return;
    editSeq++;
    const i = await build.PasteSocketGroup(text);
    if (i < 0) return;
    selected = i;
    currentBuild.set(build);
    UITick('PasteSocketGroup');
  };
  const ondeletegroup = async (i: number) => {
    editSeq++;
    await $currentBuild?.DeleteSocketGroup(i);
    selected = 0;
    currentBuild.set($currentBuild);
    UITick('socketGroups');
  };
  const ondeleteall = async () => {
    editSeq++;
    await $currentBuild?.DeleteAllSocketGroups();
    await $currentBuild?.AddNewSocketGroup();
    selected = 0;
    commitMainSocketGroup(0);
    UITick('socketGroups');
  };

  const SORT_FIELDS: Record<string, string> = {
    'Full DPS': 'FullDPS',
    'Combined DPS': 'CombinedDPS',
    'Hit DPS': 'TotalDPS',
    'Average Hit': 'AverageDamage',
    'DoT DPS': 'TotalDot',
    'Bleed DPS': 'BleedDPS',
    'Ignite DPS': 'IgniteDPS',
    'Poison DPS': 'TotalPoisonDPS',
    'Effective Hit Pool': 'TotalEHP'
  };
  const sortLabel = (field: string) => Object.entries(SORT_FIELDS).find(([, v]) => v === field)?.[0] ?? 'Combined DPS';

  let sortOn = $state(true);
  let sortStat = $state('Combined DPS');
  let sortOnSaved = true;
  let sortField = 'CombinedDPS';

  // GemSelectControl.lua:110-116
  let characterLevel = $state(1);
  $effect(() => {
    void $buildRevision;
    syncWrap
      ?.GetBuildInfo()
      .then((i) => {
        if (i) characterLevel = i.level;
      })
      .catch(logError);
  });
  const imbuedOptions = $derived(
    [...gemByName.values()]
      .filter((g) => g.Support)
      .map((g) => g.Base.Name.replace(/ Support$/, ''))
      .sort()
  );
  const catalogue = $derived(
    populateGems(
      [...gemByName.values()].map((g) => ({
        name: g.Base.Name,
        attr: ATTR[g.GemType] ?? 'white',
        support: !!g.Support,
        searchName: g.Name || g.Base.Name,
        tags: [...(g.Tags ?? [])],
        levelReq: g.LevelRequirement || 1,
        legacy: g.Legacy,
        hideFromGemList: g.HideFromGemList,
        legacyAwakened: g.LegacyAwakened
      })),
      {
        showSupport: options?.showSupport ?? 'ALL',
        matchLevel: options?.defaultLevel === 'characterLevel',
        characterLevel,
        showLegacy: options?.showLegacy ?? false
      }
    ).sort((a, b) => a.name.localeCompare(b.name))
  );

  let gemOptionsLoaded = $state(false);

  $effect(() => {
    const build = $currentBuild;
    if (!build) return;
    const skills = build.Skills;
    Promise.all([
      skills.SortGemsByDPS,
      skills.SortGemsByDPSField,
      syncWrap?.GetGemDefaultLevelMode(),
      skills.DefaultGemQuality,
      skills.ShowSupportGemTypes,
      skills.ShowAltQualityGems,
      skills.ShowLegacyGems
    ])
      .then(([byDPS, byDPSField, gemLevel, gemQuality, supportTypes, altQuality, showLegacy]) => {
        sortOn = byDPS;
        sortOnSaved = byDPS;
        sortField = byDPSField;
        sortStat = sortLabel(byDPSField);
        options = {
          defaultLevel: gemLevel ?? 'normalMaximum',
          defaultQuality: gemQuality ?? 0,
          showSupport: supportTypes || 'ALL',
          showAlt: altQuality,
          showLegacy
        };
        gemOptionsLoaded = true;
      })
      .catch(logError);
  });

  // GemSelectControl.lua:255-441
  let sortCache = $state.raw<{ canSupport: Record<string, boolean>; dps: Record<string, number>; progress?: number }>();
  let gemDeltas = $state.raw<Record<string, number>>({});
  let pickerRun = 0;
  let building = false;
  // GemSelectControl.lua:258-264
  const dropSortCache = () => {
    pickerRun++;
    building = false;
    sortCache = undefined;
    gemDeltas = {};
  };
  $effect(() => {
    void $outputs;
    dropSortCache();
  });
  const onsort = (on: boolean, stat: string) => {
    const build = $currentBuild;
    if (!gemOptionsLoaded || !build) return;
    if (on !== sortOnSaved) {
      sortOnSaved = on;
      build.SetSortGemsByDPS(on).catch(logError);
      dropSortCache();
    }
    sortOn = on;
    const field = SORT_FIELDS[stat];
    if (field && field !== sortField) {
      sortField = field;
      build.SetSortGemsByDPSField(field).catch(logError);
      dropSortCache();
    }
    sortStat = stat;
  };

  const onoption = (patch: Partial<GemOptions>) => {
    const build = $currentBuild;
    const o = options;
    if (!gemOptionsLoaded || !build || !o) return;
    // SkillsTab.lua:130-132
    if (patch.defaultLevel !== undefined && patch.defaultLevel !== o.defaultLevel) {
      o.defaultLevel = patch.defaultLevel;
      dropSortCache();
      build.SetDefaultGemLevelMode(patch.defaultLevel).catch(logError);
    }
    if (patch.defaultQuality !== undefined && patch.defaultQuality !== o.defaultQuality) {
      o.defaultQuality = patch.defaultQuality;
      dropSortCache();
      build.SetDefaultGemQuality(patch.defaultQuality).catch(logError);
    }
    if (patch.showSupport !== undefined && patch.showSupport !== o.showSupport) {
      o.showSupport = patch.showSupport;
      dropSortCache();
      build.SetShowSupportGemTypes(patch.showSupport).catch(logError);
    }
    if (patch.showLegacy !== undefined && patch.showLegacy !== o.showLegacy) {
      o.showLegacy = patch.showLegacy;
      dropSortCache();
      build.SetShowLegacyGems(patch.showLegacy).catch(logError);
    }
    if (patch.showAlt !== undefined && patch.showAlt !== o.showAlt) {
      o.showAlt = patch.showAlt;
      build.SetShowAltQualityGems(patch.showAlt).catch(logError);
    }
  };

  const onpicker = (group: number, open: boolean) => {
    if (!open) {
      if (building) {
        pickerRun++;
        building = false;
        sortCache = undefined;
      }
      return;
    }
    if (sortCache || building) return;
    const run = ++pickerRun;
    building = true;
    const list = catalogue.flatMap((g) => {
      const def = gemByName.get(g.name);
      return def ? [{ name: g.name, gem: { id: def.ID, name: def.Base.Name } }] : [];
    });
    const field = sortField;
    const by = sortOn;
    void (async () => {
      const sc = await syncWrap?.GemSortCache(
        group,
        list.map((x) => x.gem)
      );
      if (!sc || run !== pickerRun) return void (building = run === pickerRun ? false : building);
      const canSupport: Record<string, boolean> = {};
      list.forEach((x, i) => (canSupport[x.name] = sc.canSupport[i]));
      const dps: Record<string, number> = {};
      sortCache = { canSupport, dps, progress: by ? 0 : undefined };
      if (!by) return void (building = false);
      const pending = list.filter((_, i) => sc.pending[i]);
      if (!pending.length) {
        sortCache = { canSupport, dps };
        return void (building = false);
      }
      const deltas: Record<string, number> = {};
      for (let i = 0; i < pending.length; i += 8) {
        const chunk = pending.slice(i, i + 8);
        const r = await syncWrap?.GemSortDPS(
          group,
          chunk.map((x) => x.gem),
          field
        );
        if (!r || run !== pickerRun) return void (building = run === pickerRun ? false : building);
        if (i === 0) for (const x of list) dps[x.name] = r.base;
        chunk.forEach((x, j) => {
          dps[x.name] = r.dps[j];
          if (r.base) deltas[x.name] = Math.round(((r.dps[j] - r.base) / r.base) * 1000) / 10;
        });
        sortCache = { canSupport, dps: { ...dps }, progress: Math.floor(((i + chunk.length) / pending.length) * 100) };
        gemDeltas = { ...deltas };
      }
      building = false;
    })().catch(logError);
  };

  // SkillsTab.lua:1185-1193
  const GEM_COLOR: Record<string, string> = { str: colorCodesX.STRENGTH, dex: colorCodesX.DEXTERITY, int: colorCodesX.INTELLIGENCE };
  let supportPairs = $state<{ group: number; pairs: [number, number][] }>({ group: -1, pairs: [] });
  $effect(() => {
    void $outputs;
    const group = selected;
    void Promise.resolve(syncWrap?.GemSupportPairs(group))
      .then((pairs) => (supportPairs = { group, pairs: pairs ?? [] }))
      .catch(logError);
  });
  // SkillsTab.lua:363-379
  let sourceNames = $state<{ group: number; names: { Name: string; Rarity: string }[] }>({ group: -1, names: [] });
  $effect(() => {
    void $outputs;
    const group = selected;
    const set = view?.activeSet ?? 0;
    const source = view?.groups[group]?.source ?? '';
    if (source !== 'Explode' && !source.startsWith('Tree:')) return;
    void Promise.resolve(syncWrap?.GetSocketGroupSourceNames(set, group))
      .then((names) => (sourceNames = { group, names: names ?? [] }))
      .catch(logError);
  });
  // SkillsTab.lua:294-313
  let optimiseUseful = $state<{ group: number; useful: boolean }>({ group: -1, useful: false });
  $effect(() => {
    void $outputs;
    const group = selected;
    const slot = view?.groups[group]?.slot;
    if (!slot) {
      optimiseUseful = { group, useful: false };
      return;
    }
    void Promise.resolve(syncWrap?.OptimiseSocketsUseful(slot))
      .then((useful) => (optimiseUseful = { group, useful: !!useful }))
      .catch(logError);
  });
  const editor = {
    optimiseEnabled: (i: number) => optimiseUseful.group === i && optimiseUseful.useful,
    supporting: (i: number) => (supportPairs.group === i ? supportPairs.pairs : []),
    groupCount: (i: number) => !!view?.groups[i]?.source,
    gemsLocked: (i: number) => !!view?.groups[i]?.source,
    sourceNote: (i: number) => {
      const g = view?.groups[i];
      const rarityColor = (r: string) => (colorCodesX as Record<string, string>)[r] ?? colorCodesX.NORMAL;
      const names = sourceNames.group === i ? sourceNames.names : [];
      if (g?.source === 'Explode') return explodeSourceNote(names.map((n) => ({ color: rarityColor(n.Rarity), name: n.Name })));
      const item = g?.source.startsWith('Item:') ? itemBySlot[g.slot] : undefined;
      const gem = g && toFixture(g).gems[0];
      if (g?.source.startsWith('Tree:') && gem && names[0])
        return nodeSourceNote(GEM_COLOR[gem.attr] ?? colorCodesX.NORMAL, gem.name, colorCodesX.NORMAL, names[0].Name);
      if (!item || !gem) return undefined;
      const rarity = (colorCodesX as Record<string, string>)[item.rarity] ?? colorCodesX.NORMAL;
      return itemSourceNote(GEM_COLOR[gem.attr] ?? colorCodesX.NORMAL, gem.name, rarity, item.name);
    },
    ongemcount,
    maxLevel: (i: number, gem: number) => gemById.get(view?.groups[i]?.gems[gem]?.GemID ?? '')?.MaxLevel ?? 40,
    socketsLocked: 'Item sockets and links are edited on the Items page',
    // SkillsTab.lua:294-343
    imbued: (i: number) => {
      const g = view?.groups[i];
      if (!g || g.source) return undefined;
      const taken = view!.groups.some((o, j) => j !== i && o.slot && o.slot === g.slot && o.imbued);
      return { value: g.imbued, options: imbuedOptions, enabled: !!g.slot && (!taken || !!g.imbued) };
    },
    onimbued: (i: number, name: string) => {
      const g = view?.groups[i];
      if (!g || !view) return;
      editSeq++;
      g.imbued = name;
      Promise.resolve($currentBuild?.SetSocketGroupImbuedSupport(view.activeSet, i, name))
        .then(() => UITick('currentGemGroup'))
        .catch(logError);
    },
    onoptimise: (i: number) => {
      const slot = view?.groups[i]?.slot;
      if (slot) void syncWrap?.OptimiseSockets(slot).catch(logError);
    },
    slotLocked: (i: number) => (view?.groups[i]?.source ? 'This group comes from an item or a passive; its slot is fixed' : undefined),
    gemwhatif: async (i: number, name: string) => {
      const def = gemByName.get(name);
      return def ? syncWrap?.GemSelectCompare(i, def.ID, def.Base.Name) : undefined;
    },
    qualitywhatif: async (i: number, gem: number) => syncWrap?.GemQualityCompare(i, gem),
    enabledwhatif: async (i: number, gem: number) => syncWrap?.GemEnableCompare(i, gem),
    gemtip: async (i: number, kind: 'gem' | 'list' | 'quality', key: number | string) => {
      if (kind === 'list') {
        const def = gemByName.get(String(key));
        return def ? syncWrap?.GemListTooltip(def.ID, def.Base.Name) : undefined;
      }
      return kind === 'gem' ? syncWrap?.GemTooltip(i, Number(key)) : syncWrap?.GemQualityTooltip(i, Number(key));
    },
    get sortCache() {
      return sortCache;
    },
    onpicker,
    slottip: async (slot: string) => {
      const name = slotItems[slot];
      const t = name ? await syncWrap?.ItemSourceTooltip(name) : undefined;
      return t
        ? { influence: [...(t.Influence ?? [])], title: [...(t.Title ?? [])], lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })) }
        : undefined;
    }
  };
  const phone = phoneMedia();
</script>

{#if view && options}
  <SkillsPanes on={phone.current}>
    <SkillsPage
      {sets}
      set={view.activeSet}
      {selected}
      mainGroup={view.main}
      {catalogue}
      deltas={gemDeltas}
      slots={SLOTS}
      bind:sortOn
      bind:sortStat
      {onset}
      {onsets}
      {onreordergroups}
      oncopygroup={(i: number) => void oncopygroup(i).catch(logError)}
      grouptip={async (i: number) => syncWrap?.SocketGroupTooltip(i)}
      groupicon={(slot: string) => {
        const file = slotIcon(slot, Object.fromEntries(Object.entries(itemBySlot).map(([s, it]) => [s, it.type])));
        return file && `${base}/slot-icons/${file}`;
      }}
      onpastegroup={(text: string) => void onpastegroup(text).catch(logError)}
      onselect={(i: number) => (selected = i)}
      {ongroup}
      {ongem}
      onaddgem={(i: number, gem: SkillGemFixture) => void onaddgem(i, gem).catch(logError)}
      {onremovegem}
      onnewgroup={() => void onnewgroup().catch(logError)}
      {onreordergems}
      onmain={commitMainSocketGroup}
      ondeletegroup={(i: number) => void ondeletegroup(i).catch(logError)}
      ondeleteall={() => void ondeleteall().catch(logError)}
      {onsort}
      {options}
      {onoption}
      {editor}
      revision={$outputs} />
  </SkillsPanes>
{:else}
  <EmptyState busy text="Loading the skills…" />
{/if}
