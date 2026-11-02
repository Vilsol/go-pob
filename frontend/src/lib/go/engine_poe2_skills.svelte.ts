import { onMount } from 'svelte';
import { fromStore } from 'svelte/store';
import { poe2Build } from '$lib/global';
import type { GemOptions } from '$lib/design/skills';
import type { GemAttr, SkillGemFixture, SocketGroupFixture } from '$lib/design/fixtures';
import { populateGems2, TERM_ALIASES2 } from '$lib/design/gem_search';
import type { SkillsPageModel } from './engine';
import { poe2Error, poe2Run, poe2Worker } from './engine_poe2.svelte';
import type { Poe2Gem, Poe2SkillGroup, Poe2SkillsEdit } from './sync_worker2';

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

// Global.lua (PoB-PoE2): colorCodes.STRENGTH, DEXTERITY, INTELLIGENCE
const ATTR_BY_CODE: Record<string, GemAttr> = { '^xE05030': 'str', '^x70FF70': 'dex', '^x7070FF': 'int' };
const ATTR_BY_COLOR: Record<number, GemAttr> = { 1: 'str', 2: 'dex', 3: 'int' };

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

// SkillsTab.lua, SkillListControl.lua, GemSelectControl.lua (PoB-PoE2): the Skills page over the open PoE2 build. PoE2
// sockets no gems in items, so the page has no slot, socket or link controls; a group picks the weapon sets it is used
// in instead. Skill sets, gem sorting by DPS, copy and paste, and the what-if tooltips are not here yet
export const poe2SkillsPage = (): SkillsPageModel => {
  const build = fromStore(poe2Build);
  let groups = $state.raw<Poe2SkillGroup[]>();
  let options = $state<GemOptions>();
  let gems = $state.raw<Poe2Gem[]>([]);
  let selected = $state(0);

  // what the page shows, fetched again for every calculation; the last shows meanwhile
  $effect(() => {
    const at = build.current;
    if (!at) return;
    void poe2Worker()
      .then(async (w) => {
        const [g, o] = await Promise.all([w.SkillGroups(), w.GemOptions()]);
        if (build.current !== at) return;
        groups = g;
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

  const sets = $derived([
    {
      title: 'Default',
      groups: (groups ?? []).map((g): SocketGroupFixture & { display: string } => ({
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
          count: m.count
        }))
      }))
    }
  ]);

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
      return !!groups && !!options;
    },
    get sets() {
      return sets;
    },
    set: 0,
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
    deltas: {},
    slots: [],
    sortOn: false,
    sortStat: 'Combined DPS',
    setLocked: 'PoE2 skill sets are not switched here yet',
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
      void edit({ op: 'gem', group: i + 1, gem: g.gems.length + 1, id: def.id });
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
    optionLists: { levels: LEVELS2, supports: SUPPORTS2, sort: false, alt: false },
    editor: {
      slotSelect: false,
      sockets: false,
      groupCount: (i: number) => !!groups?.[i]?.source,
      deletable: (i: number) => !!groups?.[i] && !groups[i].source,
      gemLocked: (i: number, gem: number) => !!groups?.[i]?.gems[gem]?.generated,
      countShown: (i: number, gem: number) => !!groups?.[i]?.gems[gem]?.countShown,
      ongemcount: (i: number, gem: number, count: number) => {
        const m = groups?.[i]?.gems[gem];
        if (m && m.count !== count) void edit({ op: 'count', group: i + 1, gem: gem + 1, value: count });
      },
      weaponSets: (i: number) => {
        const g = groups?.[i];
        return g && { set1: g.set1, set2: g.set2, locked: g.weaponSetLocked };
      },
      searchTerms: TERM_ALIASES2
    },
    get revision() {
      return build.current;
    }
  };
};
