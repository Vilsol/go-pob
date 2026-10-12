<script lang="ts">
  import { onDestroy } from 'svelte';
  import { resource } from 'runed';
  import { buildRevision, currentBuild, outputs } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import PartyPage from '$lib/design/PartyPage.svelte';
  import { PARTY_CATEGORIES, type PartyCategory, type PartyEntry } from '$lib/design/party';
  import { linkCode } from './link_import';

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
    () => [$buildRevision, $currentBuild === undefined, $outputs] as const,
    async () => {
      if (!$currentBuild || !syncWrap) return undefined;
      const [view, members] = await Promise.all([syncWrap.GetParty(), syncWrap.GetPartyMembers()]);
      return view && { view, members };
    }
  );
  const impactRows = resource(
    () => [$buildRevision, $currentBuild === undefined, $outputs] as const,
    async () => ($currentBuild && syncWrap ? syncWrap.GetPartyImpact(IMPACT.map(([k]) => k)) : [])
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
  let loadedKey = -1;
  $effect(() => {
    const v = party.current?.view;
    if (!v || loadedKey === $buildRevision) return;
    loadedKey = $buildRevision;
    drafts = {};
    options = { destination: v.destination, append: v.append, advanced: v.advanced };
  });
  $effect(() => {
    const v = party.current?.view;
    const { destination, append, advanced } = options;
    if (!v || (v.destination === destination && v.append === append && v.advanced === advanced)) return;
    void syncWrap?.SetPartyOptions(destination, append, advanced).catch(logError);
  });

  const refetch = () => void party.refetch();

  let pending: { cat: PartyCategory; text: string } | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const flush = () => {
    clearTimeout(timer);
    const p = pending;
    pending = undefined;
    if (p) syncWrap?.SetPartyBuffer(GO_NAME[p.cat], p.text).then(refetch).catch(logError);
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
      await syncWrap?.ImportPartyMember(code, destination, append);
      drafts = {};
      refetch();
    } catch (err) {
      importError = err instanceof Error ? err.message : String(err);
    }
  };
  const onclear = (destination: string) => {
    flush();
    drafts = {};
    syncWrap?.ClearParty(destination).then(refetch).catch(logError);
  };
  const onrebuild = () => {
    flush();
    syncWrap?.RebuildParty().then(refetch).catch(logError);
  };
  const LABEL = Object.fromEntries(PARTY_CATEGORIES.map((c) => [GO_NAME[c.id], c.label]));
  const members = $derived((party.current?.members ?? []).map((m, i) => ({ ...m, id: String(i), gives: m.gives.map((g) => LABEL[g] ?? g).join(', ') })));
  const onmember = (id: string, active: boolean) => {
    flush();
    drafts = {};
    syncWrap?.SetPartyMemberActive(Number(id), active).then(refetch).catch(logError);
  };
  const onremove = (id: string) => {
    flush();
    drafts = {};
    syncWrap?.RemovePartyMember(Number(id)).then(refetch).catch(logError);
  };
  const disabled = $derived(party.current?.view.disabled ?? false);
  const setDisabled = (on: boolean) => {
    if (on === disabled) return;
    flush();
    (on ? syncWrap?.DisablePartyEffects() : syncWrap?.RebuildParty())?.then(refetch).catch(logError);
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
