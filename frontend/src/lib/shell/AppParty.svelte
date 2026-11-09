<script lang="ts">
  import { onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { resource } from 'runed';
  import { buildRevision, currentBuild, outputs, poe2Build, poe2Edited, poe2Session } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { game } from '$lib/go/engine';
  import { poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import { decodeCode, rootElement } from '$lib/go/game_detect';
  import { logError } from '$lib/utils';
  import PartyPage from '$lib/design/PartyPage.svelte';
  import { PARTY_CATEGORIES, type PartyCategory, type PartyEntry } from '$lib/design/party';
  import { linkCode } from './link_import';
  import { NOT_A_CODE, importFailure } from '$lib/design/share';
  import type { PartyMemberView, PartyView } from '$lib/go/party_view';

  // PartyTab.lua: the open build's party tab, whichever game it is; both workers name their party calls alike
  type PartyApi = {
    GetParty(): PartyView | undefined | Promise<PartyView | undefined>;
    GetPartyMembers(): PartyMemberView[] | Promise<PartyMemberView[]>;
    GetPartyImpact(stats: string[]): Promise<{ stat: string; with: number; without: number }[]>;
    SetPartyBuffer(name: string, text: string): Promise<unknown>;
    SetPartyOptions(destination: string, append: boolean, advanced: boolean): Promise<unknown>;
    ImportPartyMember(code: string, destination: string, append: boolean): Promise<unknown>;
    ClearParty(destination: string): Promise<unknown>;
    DisablePartyEffects(): Promise<unknown>;
    RebuildParty(): Promise<unknown>;
    SetPartyMemberActive(i: number, active: boolean): Promise<unknown>;
    RemovePartyMember(i: number): Promise<unknown>;
  };
  const poe2 = get(game) === 'poe2';
  const POE1_CODE = 'A Path of Exile 1 build cannot join a Path of Exile 2 party.';
  const poe2Api: PartyApi = {
    GetParty: async () => (await poe2Worker()).GetParty(),
    GetPartyMembers: async () => (await poe2Worker()).GetPartyMembers(),
    GetPartyImpact: async (stats) => (await poe2Worker()).GetPartyImpact(stats),
    SetPartyBuffer: (name, text) => poe2Run((w) => w.SetPartyBuffer(name, text)),
    SetPartyOptions: (destination, append, advanced) => poe2Run((w) => w.SetPartyOptions(destination, append, advanced)),
    // PartyTab.lua:162-203 (PoB-PoE2): the code's XML must be a PoE2 build
    ImportPartyMember: async (code, destination, append) => {
      const xml = await decodeCode(code);
      if (xml === undefined) throw new Error(NOT_A_CODE);
      if (rootElement(xml) === 'PathOfBuilding') throw new Error(POE1_CODE);
      poe2Edited(await (await poe2Worker()).ImportPartyMember(xml, destination, append));
    },
    ClearParty: (destination) => poe2Run((w) => w.ClearParty(destination)),
    DisablePartyEffects: () => poe2Run((w) => w.DisablePartyEffects()),
    RebuildParty: () => poe2Run((w) => w.RebuildParty()),
    SetPartyMemberActive: (i, active) => poe2Run((w) => w.SetPartyMemberActive(i, active)),
    RemovePartyMember: (i) => poe2Run((w) => w.RemovePartyMember(i))
  };
  const api = (): PartyApi | undefined => (poe2 ? poe2Api : (syncWrap as PartyApi | undefined));
  const open = () => (poe2 ? !!$poe2Build : !!$currentBuild);

  const GO_NAME: Record<PartyCategory, string> = {
    PartyMemberStats: 'PartyMemberStats',
    Aura: 'Aura',
    Curse: 'Curse',
    Warcry: 'Warcry Skills',
    Link: 'Link Skills',
    EnemyConditions: 'EnemyConditions',
    EnemyMods: 'EnemyMods'
  };
  const IMPACT: [string, string][] = [
    ['CombinedDPS', 'Combined DPS'],
    ['Life', 'Life'],
    ['EnergyShield', 'Energy Shield'],
    ['Mana', 'Mana'],
    ['Armour', 'Armour'],
    ['Evasion', 'Evasion'],
    ['EffectiveMovementSpeedMod', 'Movement Speed']
  ];

  const party = resource(
    () => [$buildRevision, $currentBuild === undefined, $outputs, $poe2Build] as const,
    async () => {
      const a = api();
      if (!open() || !a) return undefined;
      const [view, members] = await Promise.all([a.GetParty(), a.GetPartyMembers()]);
      return view && { view, members };
    }
  );
  const impactRows = resource(
    () => [$buildRevision, $currentBuild === undefined, $outputs, $poe2Build] as const,
    async () => {
      const a = api();
      return open() && a ? a.GetPartyImpact(IMPACT.map(([k]) => k)) : [];
    }
  );

  let drafts = $state<Partial<Record<PartyCategory, string>>>({});
  const buffers = $derived(
    Object.fromEntries(PARTY_CATEGORIES.map((c) => [c.id, drafts[c.id] ?? party.current?.view.buffers[GO_NAME[c.id]] ?? ''])) as Record<PartyCategory, string>
  );
  // PartyTab.lua:892-975
  const summaries = $derived.by(() => {
    const out: Partial<Record<PartyCategory, PartyEntry[]>> = {};
    const s = party.current?.view.summaries ?? {};
    for (const c of PARTY_CATEGORIES) {
      const list = s[GO_NAME[c.id]];
      if (!list) continue;
      out[c.id] = list.map((e) =>
        e.group
          ? { name: `${e.group}: ${e.name}`, value: e.value }
          : { name: e.name, value: c.id === 'EnemyMods' ? e.value.trim().replace(/,$/, '') : `${e.value}% effect` }
      );
    }
    return out;
  });
  const impact = $derived(
    (impactRows.current ?? [])
      .filter((r) => Math.abs(r.with - r.without) > 1e-9)
      .map((r): [string, string, 'up' | 'down'] => {
        const d = r.with - r.without;
        const label = IMPACT.find(([k]) => k === r.stat)?.[1] ?? r.stat;
        const text = `${d > 0 ? '+' : ''}${Math.abs(d) < 10 ? d.toFixed(2) : Math.round(d).toLocaleString('en-US')}`;
        return [label, text, d > 0 ? 'up' : 'down'];
      })
  );

  let importText = $state('');
  let importError = $state<string | null>(null);
  let options = $state({ destination: 'All', append: false, advanced: false });
  let loadedKey: unknown = -1;
  $effect(() => {
    const v = party.current?.view;
    const key = poe2 ? $poe2Session : $buildRevision;
    if (!v || loadedKey === key) return;
    loadedKey = key;
    drafts = {};
    options = { destination: v.destination, append: v.append, advanced: v.advanced };
  });
  $effect(() => {
    const v = party.current?.view;
    const { destination, append, advanced } = options;
    if (!v || (v.destination === destination && v.append === append && v.advanced === advanced)) return;
    void api()?.SetPartyOptions(destination, append, advanced).catch(logError);
  });

  const refetch = () => void party.refetch();

  let pending: { cat: PartyCategory; text: string } | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const flush = () => {
    clearTimeout(timer);
    const p = pending;
    pending = undefined;
    if (p) api()?.SetPartyBuffer(GO_NAME[p.cat], p.text).then(refetch).catch(logError);
  };
  onDestroy(flush);
  const onbuffer = (cat: PartyCategory, text: string) => {
    if (pending && pending.cat !== cat) flush();
    drafts[cat] = text;
    pending = { cat, text };
    clearTimeout(timer);
    timer = setTimeout(flush, 400);
  };

  const onimport = async (text: string, destination: string, append: boolean) => {
    importError = null;
    flush();
    try {
      const code = await linkCode(text);
      await api()?.ImportPartyMember(code, destination, append);
      drafts = {};
      refetch();
    } catch (err) {
      importError = importFailure(err);
    }
  };
  const onclear = (destination: string) => {
    flush();
    drafts = {};
    api()?.ClearParty(destination).then(refetch).catch(logError);
  };
  const onrebuild = () => {
    flush();
    api()?.RebuildParty().then(refetch).catch(logError);
  };
  const LABEL = Object.fromEntries(PARTY_CATEGORIES.map((c) => [GO_NAME[c.id], c.label]));
  const members = $derived((party.current?.members ?? []).map((m, i) => ({ ...m, id: String(i), gives: m.gives.map((g) => LABEL[g] ?? g).join(', ') })));
  const onmember = (id: string, active: boolean) => {
    flush();
    drafts = {};
    api()?.SetPartyMemberActive(Number(id), active).then(refetch).catch(logError);
  };
  const onremove = (id: string) => {
    flush();
    drafts = {};
    api()?.RemovePartyMember(Number(id)).then(refetch).catch(logError);
  };
  const disabled = $derived(party.current?.view.disabled ?? false);
  const setDisabled = (on: boolean) => {
    if (on === disabled) return;
    flush();
    (on ? api()?.DisablePartyEffects() : api()?.RebuildParty())?.then(refetch).catch(logError);
  };
</script>

{#if party.current}
  <PartyPage
    {buffers}
    {summaries}
    {members}
    {impact}
    stale={!party.current.view.applied && !disabled}
    {importError}
    bind:importText
    bind:destination={options.destination}
    bind:append={options.append}
    bind:advanced={options.advanced}
    bind:disabled={() => disabled, setDisabled}
    {onimport}
    {onclear}
    {onrebuild}
    {onmember}
    {onremove}
    {onbuffer} />
{/if}
