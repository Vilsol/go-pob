<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { backendLoaded, buildRevision, devMode, outputs } from '$lib/global.js';
  import { calcSections } from '$lib/calcs/calc_sections';
  import { calcElements, calcsView } from '$lib/display/calcs';
  import { syncWrap } from '$lib/go/worker';
  import MiniTreeView from '$lib/components/skill-tree/MiniTreeView.svelte';
  import CalcsPage from '$lib/design/CalcsPage.svelte';
  import { MainGroup } from '$lib/shell/main_group.svelte';
  import { type MainSkillField, writeMainSkill } from '$lib/shell/main_skill';
  import { logError } from '$lib/utils';
  import SpectreLibrary from '$lib/design/SpectreLibrary.svelte';
  import type { SpectreRow } from '$lib/design/spectres';
  import type { calculator } from '$lib/types';

  const main = new MainGroup();
  // CalcsTab.lua:452-455, CalcsTab.lua:40-119
  const tab = $derived($outputs?.CalcsTab);
  const calcs = $derived(tab?.Breakdown ? calcsView(calcSections, tab) : undefined);

  // CalcsTab.lua:12-17
  const MODES: Record<string, string> = { Unbuffed: 'UNBUFFED', Buffed: 'BUFFED', 'In Combat': 'COMBAT', 'Effective DPS': 'EFFECTIVE' };
  let inputs = $state<{ buffMode: string; showMinion: boolean; skillNumber: number }>({ buffMode: 'EFFECTIVE', showMinion: false, skillNumber: 1 });
  $effect(() => {
    void $outputs;
    void Promise.resolve(syncWrap?.GetCalcsTabInputs())
      .then((i) => {
        if (i) inputs = i;
      })
      .catch(logError);
  });
  // CalcsTab.lua:135-137
  const lists = $derived.by(() => {
    const s = tab?.OutputStrings ?? {};
    const rank = ['UNBUFFED', 'BUFFED', 'COMBAT', 'EFFECTIVE'].indexOf(inputs.buffMode);
    return [
      { label: 'Aura and Buff Skills', text: s.BuffList ?? '', from: 1, bd: 'SkillBuffs' },
      { label: 'Combat Buffs', text: s.CombatList ?? '', from: 2 },
      { label: 'Curses and Debuffs', text: s.CurseList ?? '', from: 3, bd: 'SkillDebuffs' }
    ].filter((l) => rank >= l.from);
  });

  // CalcsTab.lua:174-189, CalcSectionControl.lua:23-39
  // CalcsTab.lua:208-216
  const subsectionId = (label: string) => label.replace(/[^A-Za-z0-9]/g, '');
  const sectionKeys = calcSections.map((s) => [s[1], subsectionId(s[4][0]?.label ?? '')] as [string, string]);
  let collapsed = $state<Record<string, boolean>>();
  $effect(() => {
    void $buildRevision;
    void Promise.resolve(syncWrap?.GetCalcsSectionsCollapsed(sectionKeys))
      .then((saved) => {
        if (saved) collapsed = Object.fromEntries(calcSections.map((s, i) => [s[1], saved[i] ?? !!s[4][0]?.defaultCollapsed]));
      })
      .catch(logError);
  });
  const oncollapse = (id: string, value: boolean) => {
    const key = sectionKeys.find(([k]) => k === id);
    if (key) void syncWrap?.SetCalcsSectionCollapsed(key[0], key[1], value).catch(logError);
  };

  const modeLabel = $derived(Object.entries(MODES).find(([, v]) => v === inputs.buffMode)?.[0] ?? 'Effective DPS');

  // CalcsTab.lua:106-110, Build.lua:1453-1482
  let spectresOpen = $state(false);
  let library = $state.raw<SpectreRow[]>([]);
  let spectres = $state.raw<SpectreRow[]>([]);
  const row = (r: calculator.SpectreRow): SpectreRow => ({ id: r.ID, name: r.Name, skills: [...(r.Skills ?? [])], tooltip: [...(r.Tooltip ?? [])] });
  const openSpectres = () => {
    syncWrap
      ?.GetSpectreLibrary()
      .then((l) => {
        library = l.library.map(row);
        spectres = l.build.map(row);
        spectresOpen = true;
      })
      .catch(logError);
  };

  let destroyed = false;
  onDestroy(() => {
    destroyed = true;
    void syncWrap?.clearCalcTabElements();
  });

  onMount(() => {
    void backendLoaded.then(() => {
      if (!destroyed) void syncWrap?.setCalcTabElements(calcElements(calcSections));
    });
  });
</script>

{#if calcs}
  <CalcsPage
    {calcs}
    socketGroups={main.groups.length ? main.groups : ['<No skills added yet>']}
    mainSocketGroup={Math.max(0, Math.min(inputs.skillNumber, main.groups.length) - 1)}
    controls={tab?.MainSkill}
    mode={modeLabel}
    {lists}
    {collapsed}
    {oncollapse}
    showMinion={$outputs?.SkillFlags?.haveMinion ? inputs.showMinion : undefined}
    onshowminion={(show: boolean) => {
      inputs.showMinion = show;
      void syncWrap?.SetCalcsShowMinion(show);
    }}
    onmode={(label: string) => {
      const buffMode = MODES[label];
      if (!buffMode || buffMode === inputs.buffMode) return;
      inputs.buffMode = buffMode;
      void syncWrap?.SetCalcsBuffMode(buffMode);
    }}
    onmaingroup={(i: number) => {
      inputs.skillNumber = i + 1;
      void syncWrap?.SetCalcsSkillNumber(i + 1);
    }}
    oncontrol={(field: MainSkillField, v: number | undefined) => writeMainSkill(tab?.MainSkill, field, v, true)}
    onspectres={openSpectres}
    grouptip={async (i: number) => syncWrap?.SocketGroupTooltip(i)}
    modrows={async (sd: unknown) => syncWrap?.CalcsModSection(sd as calculator.ModSectionData)}
    listrows={async (bd: string) => syncWrap?.CalcsBreakdownModList(bd)}
    moddebug={$devMode ? async () => (await syncWrap?.GetModDebugDump()) ?? '' : undefined}
    itemtip={async (name: string) => {
      const t = await syncWrap?.ItemSourceTooltip(name);
      return t
        ? { influence: [...(t.Influence ?? [])], title: [...(t.Title ?? [])], lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })) }
        : undefined;
    }}>
    {#snippet nodeview(id: number)}<MiniTreeView nodeId={id} />{/snippet}
  </CalcsPage>
{:else}
  <p class="note p-5" role="status">Calculating…</p>
{/if}
<SpectreLibrary bind:open={spectresOpen} {library} {spectres} onsave={(ids: string[]) => syncWrap?.SetSpectres(ids).catch(logError)} />
