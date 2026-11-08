import { onMount } from 'svelte';
import { fromStore } from 'svelte/store';
import { poe2Build } from '$lib/global';
import type { GemOptions, SkillSetOp } from '$lib/design/skills';
import type { GemAttr, SkillGemFixture, SocketGroupFixture } from '$lib/design/fixtures';
import { populateGems2, TERM_ALIASES2 } from '$lib/design/gem_search';
import type { SkillsPageModel } from './engine';
import { poe2Error, poe2Run, poe2Worker } from './engine_poe2.svelte';
import type { Poe2Gem, Poe2SkillGroup, Poe2SkillSets, Poe2SkillsEdit } from './sync_worker2';

// SkillsTab.lua:13-42 (PoB-PoE2)
const LEVELS2 = [
  { value: 'normalMaximum', label: 'Normal Maximum', description: 'All gems default to their highest valid non-corrupted gem level.' },
  {
    value: 'corruptedMaximum',
    label: 'Corrupted Maximum',
    description: 'Normal gems default to their highest valid corrupted gem level.\nAwakened gems default to their highest valid non-corrupted gem level.'
  },
  {
    value: 'characterLevel',
    label: 'Match Character Level',
    description:
      'All gems default to their highest valid non-corrupted gem level that your character meets the level requirement for.\nThis hides gems with a minimum level requirement above your character level, preventing them from showing up in the dropdown list.'
  }
];
const SUPPORTS2 = [
  { value: 'ALL', label: 'All' },
  { value: 'LINEAGE', label: 'Lineage' },
  { value: 'NORMAL', label: 'Non-Lineage' }
];

// SkillsTab.lua:44-49 (PoB-PoE2)
const CORRUPT2 = ['Not Corrupted', '+1 to Gem Level', 'Corrupted', '-1 to Gem Level'];

// Global.lua (PoB-PoE2): colorCodes.STRENGTH, DEXTERITY, INTELLIGENCE
const ATTR_BY_CODE: Record<string, GemAttr> = { '^xE05030': 'str', '^x70FF70': 'dex', '^x7070FF': 'int' };
const ATTR_BY_COLOR: Record<number, GemAttr> = { 1: 'str', 2: 'dex', 3: 'int' };

// SkillsTab.lua:51-61 (PoB-PoE2), less Full DPS, which the PoE2 calculator does not calculate yet
const SORT_FIELDS2: Record<string, string> = {
  'Combined DPS': 'CombinedDPS',
  'Hit DPS': 'TotalDPS',
  'Average Hit': 'AverageDamage',
  'DoT DPS': 'TotalDot',
  'Bleed DPS': 'BleedDPS',
  'Ignite DPS': 'IgniteDPS',
  'Poison DPS': 'TotalPoisonDPS',
  'Effective Hit Pool': 'TotalEHP'
};
const sortLabel2 = (field: string) => Object.entries(SORT_FIELDS2).find(([, v]) => v === field)?.[0] ?? 'Combined DPS';

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

// SkillsTab.lua, SkillListControl.lua, GemSelectControl.lua (PoB-PoE2): the Skills page over the open PoE2 build. PoE2
// sockets no gems in items, so the page has no slot, socket or link controls; a group picks the weapon sets it is used
// in instead. The what-if tooltips are not here yet
export const poe2SkillsPage = (): SkillsPageModel => {
  const build = fromStore(poe2Build);
  let groups = $state.raw<Poe2SkillGroup[]>();
  let skillSets = $state.raw<Poe2SkillSets>();
  let options = $state<GemOptions>();
  let gems = $state.raw<Poe2Gem[]>([]);
  let selected = $state(0);
  let sortOn = $state(true);
  let sortStat = $state('Combined DPS');

  // what the page shows, fetched again for every calculation; the last shows meanwhile
  $effect(() => {
    const at = build.current;
    if (!at) return;
    void poe2Worker()
      .then(async (w) => {
        const [g, o, s] = await Promise.all([w.SkillGroups(), w.GemOptions(), w.SkillSets()]);
        if (build.current !== at) return;
        groups = g;
        skillSets = s;
        sortOn = o.SortGemsByDPS;
        sortStat = sortLabel2(o.SortGemsByDPSField);
        options = {
          defaultLevel: o.DefaultGemLevel,
          defaultQuality: o.DefaultGemQuality,
          showSupport: o.ShowSupportGemTypes,
          showAlt: false,
          showLegacy: o.ShowLegacyGems
        };
        selected = Math.min(selected, Math.max(0, g.length - 1));
      })
      .catch(fail);
  });
  onMount(() => {
    void poe2Worker()
      .then((w) => w.GemList())
      .then((list) => (gems = list))
      .catch(fail);
  });

  const edit = (e: Poe2SkillsEdit) => poe2Run((w) => w.EditSkills(e));

  const activeSet = $derived(Math.max(0, skillSets?.sets.findIndex((s) => s.id === skillSets?.active) ?? 0));
  const activeGroups = $derived(
    (groups ?? []).map((g): SocketGroupFixture & { display: string } => ({
      label: g.userLabel,
      display: g.label,
      slot: '',
      enabled: g.enabled,
      inFull: g.includeInFullDPS,
      count: g.groupCount,
      sockets: '',
      set1: g.set1,
      set2: g.set2,
      gems: g.gems.map((m) => ({
        name: m.name,
        level: m.level,
        quality: m.quality,
        enabled: m.enabled,
        support: m.support,
        attr: ATTR_BY_CODE[m.color] ?? 'white',
        count: m.count,
        error: m.error || undefined
      }))
    }))
  );
  const sets = $derived((skillSets?.sets ?? [{ id: 1, title: 'Default' }]).map((s, i) => ({ title: s.title, groups: i === activeSet ? activeGroups : [] })));

  // SkillsTab.lua:85-88 (PoB-PoE2)
  const onset = (i: number) => {
    const id = skillSets?.sets[i]?.id;
    if (id === undefined || id === skillSets?.active) return;
    selected = 0;
    void edit({ op: 'activeSet', set: id });
  };
  // SkillSetListControl.lua:48-132, SkillsSetService.lua:17-46 (PoB-PoE2)
  const onsets = (op: SkillSetOp) => {
    if (op.op === 'new') {
      selected = 0;
      void edit({ op: 'newSet', title: op.title });
      return;
    }
    const id = skillSets?.sets[op.index]?.id;
    if (id === undefined) return;
    if (op.op === 'rename') void edit({ op: 'renameSet', set: id, title: op.title });
    else {
      selected = 0;
      void edit(op.op === 'copy' ? { op: 'copySet', set: id, title: op.title } : { op: 'deleteSet', set: id });
    }
  };

  // GemSelectControl.lua:104-133 (PoB-PoE2)
  const catalogue = $derived(
    populateGems2(
      gems.map((g) => ({
        id: g.id,
        name: g.name,
        attr: ATTR_BY_COLOR[g.color] ?? 'white',
        support: g.support,
        searchName: g.name,
        tags: g.tags,
        levelReq: g.levelReq,
        legacy: g.legacy,
        isLineage: g.isLineage
      })),
      {
        showSupport: options?.showSupport ?? 'ALL',
        matchLevel: options?.defaultLevel === 'characterLevel',
        characterLevel: build.current?.level ?? 1,
        showLegacy: options?.showLegacy ?? false
      }
    ).sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
  );
  const gemByName = $derived(new Map(gems.map((g) => [g.name, g])));

  // GemSelectControl.lua:238-432 (PoB-PoE2): the list's sort cache, made when it drops down and dropped with every
  // calculation; the DPS pass runs in chunks so the list shows its progress
  let sortCache = $state.raw<{ canSupport: Record<string, boolean>; dps: Record<string, number>; progress?: number }>();
  let gemDeltas = $state.raw<Record<string, number>>({});
  let pickerRun = 0;
  let building = false;
  const dropSortCache = () => {
    pickerRun++;
    building = false;
    sortCache = undefined;
    gemDeltas = {};
  };
  $effect(() => {
    void build.current;
    dropSortCache();
  });
  const saveGemOptions = (patch: { SortGemsByDPS?: boolean; SortGemsByDPSField?: string }) =>
    void poe2Worker()
      .then(async (w) => w.SetGemOptions({ ...(await w.GemOptions()), ...patch }))
      .catch(fail);
  // SkillsTab.lua:111-121 (PoB-PoE2)
  const onsort = (on: boolean, stat: string) => {
    if (on !== sortOn) {
      saveGemOptions({ SortGemsByDPS: on });
      dropSortCache();
    }
    sortOn = on;
    const field = SORT_FIELDS2[stat];
    if (field && stat !== sortStat) {
      saveGemOptions({ SortGemsByDPSField: field });
      dropSortCache();
    }
    sortStat = stat;
  };
  const onpicker = (i: number, open: boolean) => {
    if (!open) {
      if (building) dropSortCache();
      return;
    }
    if (sortCache || building) return;
    const run = ++pickerRun;
    building = true;
    const list = catalogue.flatMap((g) => {
      const def = gemByName.get(g.name);
      return def ? [{ name: g.name, id: def.id }] : [];
    });
    const field = SORT_FIELDS2[sortStat] ?? 'CombinedDPS';
    const by = sortOn;
    const stale = () => run !== pickerRun;
    void poe2Worker()
      .then(async (w) => {
        const sc = await w.GemSortCache(
          i + 1,
          list.map((x) => x.id)
        );
        if (stale()) return;
        const canSupport: Record<string, boolean> = {};
        list.forEach((x, j) => (canSupport[x.name] = sc.canSupport[j]));
        const dps: Record<string, number> = {};
        const pending = list.filter((_, j) => sc.pending[j]);
        sortCache = { canSupport, dps, progress: by && pending.length ? 0 : undefined };
        if (!by || !pending.length) return;
        const deltas: Record<string, number> = {};
        for (let k = 0; k < pending.length; k += 8) {
          const chunk = pending.slice(k, k + 8);
          const r = await w.GemSortDPS(
            i + 1,
            chunk.map((x) => x.id),
            field
          );
          if (stale()) return;
          if (k === 0) for (const x of list) dps[x.name] = r.base;
          chunk.forEach((x, j) => {
            dps[x.name] = r.dps[j];
            if (r.base) deltas[x.name] = Math.round(((r.dps[j] - r.base) / r.base) * 1000) / 10;
          });
          sortCache = { canSupport, dps: { ...dps }, progress: Math.floor(((k + chunk.length) / pending.length) * 100) };
          gemDeltas = { ...deltas };
        }
      })
      .catch(fail)
      .finally(() => {
        if (!stale()) building = false;
      });
  };

  // SkillsTab.lua:151-233 (PoB-PoE2)
  const ongroup = (i: number, patch: Partial<SocketGroupFixture>) => {
    const g = groups?.[i];
    if (!g) return;
    const group = i + 1;
    if (patch.label !== undefined && patch.label !== g.userLabel) void edit({ op: 'label', group, label: patch.label });
    if (patch.enabled !== undefined && patch.enabled !== g.enabled) void edit({ op: 'enabled', group, state: patch.enabled });
    if (patch.inFull !== undefined && patch.inFull !== g.includeInFullDPS) void edit({ op: 'fullDPS', group, state: patch.inFull });
    if (patch.count !== undefined && patch.count !== g.groupCount) void edit({ op: 'groupCount', group, count: patch.count });
    if (patch.set1 !== undefined && patch.set1 !== g.set1) void edit({ op: 'weaponSet', group, set: 1, state: patch.set1 });
    if (patch.set2 !== undefined && patch.set2 !== g.set2) void edit({ op: 'weaponSet', group, set: 2, state: patch.set2 });
  };
  // SkillsTab.lua:883-1047 (PoB-PoE2)
  const ongem = (i: number, gem: number, patch: Partial<SkillGemFixture>) => {
    const m = groups?.[i]?.gems[gem];
    if (!m) return;
    const at = { group: i + 1, gem: gem + 1 };
    if (patch.level !== undefined && patch.level !== m.level) void edit({ op: 'level', ...at, value: patch.level });
    if (patch.quality !== undefined && patch.quality !== m.quality) void edit({ op: 'quality', ...at, value: patch.quality });
    if (patch.enabled !== undefined && patch.enabled !== m.enabled) void edit({ op: 'gemEnabled', ...at, state: patch.enabled });
  };

  return {
    get ready() {
      return !!groups && !!options && !!skillSets;
    },
    get sets() {
      return sets;
    },
    get set() {
      return activeSet;
    },
    onset,
    onsets,
    get selected() {
      return selected;
    },
    get mainGroup() {
      const v = build.current as { mainSocketGroup?: number } | undefined;
      return (v?.mainSocketGroup ?? 1) - 1;
    },
    get catalogue() {
      return catalogue;
    },
    get deltas() {
      return gemDeltas;
    },
    slots: [],
    get sortOn() {
      return sortOn;
    },
    set sortOn(v: boolean) {
      sortOn = v;
    },
    get sortStat() {
      return sortStat;
    },
    set sortStat(v: string) {
      sortStat = v;
    },
    onsort,
    // SkillListControl.lua:108-127 (PoB-PoE2)
    onreordergroups: (from: number, to: number) => {
      if (from === to) return;
      selected = selected === from ? to : selected > from && selected <= to ? selected - 1 : selected < from && selected >= to ? selected + 1 : selected;
      void edit({ op: 'moveGroup', from: from + 1, to: to + 1 });
    },
    onselect: (i: number) => (selected = i),
    ongroup,
    ongem,
    // SkillsTab.lua:815-875 (PoB-PoE2): the slot past the last gem
    onaddgem: (i: number, picked: SkillGemFixture) => {
      const g = groups?.[i];
      const def = gemByName.get(picked.name);
      if (!g || !def) return;
      void edit({ op: 'addGem', group: i + 1, id: def.id });
    },
    // SkillsTab.lua:773-792 (PoB-PoE2)
    onremovegem: (i: number, gem: number) => void edit({ op: 'removeGem', group: i + 1, gem: gem + 1 }),
    // SkillListControl.lua:58-71 (PoB-PoE2)
    onnewgroup: () => {
      const n = groups?.length ?? 0;
      void edit({ op: 'newGroup' }).then((v) => {
        if (v) selected = n;
      });
    },
    // SkillListControl.lua:133-137, SkillsTab.lua:639-657 (PoB-PoE2)
    oncopygroup: (i: number) => {
      if (groups?.[i]?.source) return;
      void poe2Worker()
        .then((w) => w.CopySocketGroup(i + 1))
        .then((text) => (text ? navigator.clipboard?.writeText(text) : undefined))
        .catch(fail);
    },
    // SkillsTab.lua:674-762 (PoB-PoE2): a group with a gem goes after the last
    onpastegroup: (text: string) => {
      const n = groups?.length ?? 0;
      void edit({ op: 'paste', text }).then((v) => {
        if (v && v.socketGroups.length > n) selected = n;
      });
    },
    // SkillsTab.lua:594-599 (PoB-PoE2)
    onundo: () => void poe2Run((w) => w.SkillsUndo(false)),
    onredo: () => void poe2Run((w) => w.SkillsUndo(true)),
    // SkillListControl.lua:194-198 (PoB-PoE2)
    onmain: (i: number) => void poe2Run((w) => w.SetMainSocketGroup(i + 1)),
    // SkillListControl.lua:139-173 (PoB-PoE2)
    ondeletegroup: (i: number) => {
      selected = 0;
      void edit({ op: 'deleteGroup', group: i + 1 });
    },
    // SkillListControl.lua:45-54 (PoB-PoE2)
    ondeleteall: () => {
      selected = 0;
      void edit({ op: 'deleteAll' });
    },
    get options() {
      return options;
    },
    // SkillsTab.lua:114-140 (PoB-PoE2): the options steer the gem list and new gems, not the calculation
    onoption: (patch: Partial<GemOptions>) => {
      const o = options;
      if (!o) return;
      Object.assign(o, patch);
      void poe2Worker()
        .then(async (w) => {
          const saved = await w.GemOptions();
          await w.SetGemOptions({
            ...saved,
            DefaultGemLevel: o.defaultLevel,
            DefaultGemQuality: o.defaultQuality,
            ShowSupportGemTypes: o.showSupport,
            ShowLegacyGems: o.showLegacy
          });
        })
        .catch(fail);
    },
    optionLists: { levels: LEVELS2, supports: SUPPORTS2, sort: true, alt: false, sorts: Object.keys(SORT_FIELDS2) },
    editor: {
      slotSelect: false,
      sockets: false,
      groupCount: (i: number) => !!groups?.[i]?.source,
      deletable: (i: number) => !!groups?.[i] && !groups[i].source,
      gemLocked: (i: number, gem: number) => !!groups?.[i]?.gems[gem]?.generated,
      countShown: (i: number, gem: number) => !!groups?.[i]?.gems[gem]?.countShown,
      // SkillListControl.lua:45-54, 139-173 (PoB-PoE2)
      confirmDelete: (i: number | 'all') =>
        i === 'all'
          ? 'Are you sure you want to delete all socket groups in this build?'
          : groups?.[i]?.gems.length
            ? `Are you sure you want to delete '${groups[i].label}'?`
            : undefined,
      corrupt: (i: number, gem: number) => {
        const m = groups?.[i]?.gems[gem];
        if (!m?.corruptShown) return undefined;
        return {
          options: CORRUPT2,
          value: m.corruptIndex,
          onchange: (option: number) => {
            if (option !== m.corruptIndex) void edit({ op: 'corrupt', group: i + 1, gem: gem + 1, value: option });
          }
        };
      },
      ongemcount: (i: number, gem: number, count: number) => {
        const m = groups?.[i]?.gems[gem];
        if (m && m.count !== count) void edit({ op: 'count', group: i + 1, gem: gem + 1, value: count });
      },
      weaponSets: (i: number) => {
        const g = groups?.[i];
        return (
          g && {
            set1: g.set1,
            set2: g.set2,
            locked: g.weaponSetLocked,
            valid: g.weaponSetValid,
            enabled: g.weaponSetEnabled,
            forcedBoth: g.forcedBoth,
            defaultAttack: g.source === 'Default Attack'
          }
        );
      },
      searchTerms: TERM_ALIASES2,
      get sortCache() {
        return sortCache;
      },
      onpicker
    },
    get revision() {
      return build.current;
    }
  };
};
