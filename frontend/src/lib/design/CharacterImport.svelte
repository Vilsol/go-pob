<script lang="ts">
  import SectionHeading from './SectionHeading.svelte';
  import { untrack } from 'svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Checkbox from './Checkbox.svelte';
  import PobText from './PobText.svelte';
  import { canStart, characterList, leaguesOf, missingDiscriminator, type AccountCharacter, type ImportStage, type Realm } from './account';

  // ImportTab.lua:321-472
  type ItemOpts = { clearSkills: boolean; clearItems: boolean; ignoreWeaponSwap: boolean };
  type Props = {
    realms: Realm[];
    realm?: Realm;
    account?: string;
    history?: string[];
    status?: string;
    stage?: ImportStage;
    characters?: AccountCharacter[];
    lastLeague?: string;
    onstart?: (realm: Realm, account: string) => void;
    onremoveaccount?: (account: string) => void;
    onimporttree?: (c: AccountCharacter, deleteJewels: boolean) => void;
    onimportitems?: (c: AccountCharacter, o: ItemOpts) => void;
    onselect?: (c: AccountCharacter | undefined) => void;
    selected?: string;
    onclose?: () => void;
  };
  let {
    realms,
    realm = $bindable(realms[0]),
    account = $bindable(''),
    history = [],
    status = 'Idle',
    stage = 'GETACCOUNTNAME',
    characters = [],
    lastLeague,
    onstart,
    onremoveaccount,
    onimporttree,
    onimportitems,
    onselect,
    selected,
    onclose
  }: Props = $props();

  const uid = $props.id();
  let picked = $state(untrack(() => selected ?? ''));
  let deleteJewels = $state(true);
  let items = $state<ItemOpts>({ clearSkills: true, clearItems: true, ignoreWeaponSwap: false });
  let historyPick = $state('');

  const leagues = $derived(leaguesOf(characters));
  // ImportTab.lua:992-1010
  const standardOf = (l: string) =>
    l.includes('Hardcore') ? 'Hardcore' : l.includes('HC SSF') ? 'SSF Hardcore' : l.includes('SSF') ? 'SSF Standard' : 'Standard';
  let league = $state<string | undefined>();
  let restore = untrack(() => characters.find((c) => c.name === selected)?.league);
  $effect(() => {
    const ls = leagues;
    const kept = restore && ls.includes(restore) ? restore : undefined;
    restore = undefined;
    league = kept ?? (lastLeague && ls.includes(lastLeague) ? lastLeague : lastLeague && ls.includes(standardOf(lastLeague)) ? standardOf(lastLeague) : ls[0]);
  });
  const list = $derived(characterList(realm.realmCode, league, characters));
  const chosen = $derived(list.find((e) => e.char.name === picked) ?? list[0]);
  $effect(() => onselect?.(chosen?.char));
  const selecting = $derived(stage === 'SELECTCHAR');
  const startOff = $derived(canStart(account) ? undefined : 'Enter the account name with its discriminator, e.g. Name#1234');
</script>

<section aria-label="Character import" class="panel col-span-2 flex flex-col gap-3 p-3.5">
  <SectionHeading size="lg">IMPORT CHARACTER BY ACCOUNT NAME</SectionHeading>
  <div role="status" class="text-12"><PobText text={`^7Character import status: ${status}`} /></div>
  {#if stage === 'GETACCOUNTNAME' || stage === 'DOWNLOADCHARLIST'}
    <span class="text-12 text-text-2">To start importing a character, enter the character's account name:</span>
    <div class="flex flex-wrap items-end gap-2.5">
      <FloatingLabel label="Realm"
        ><select class="select input-md" value={realm.id} onchange={(e) => (realm = realms.find((r) => r.id === e.currentTarget.value) ?? realms[0])}
          >{#each realms as r (r.id)}<option value={r.id}>{r.label}</option>{/each}</select
        ></FloatingLabel>
      <label class="field"
        >Account name <input
          class="input input-md w-55 px-2.5"
          bind:value={account}
          placeholder="AccountName#0000"
          aria-describedby="{uid}-discriminator"
          onkeydown={(e) => {
            if (e.key === 'Enter' && !startOff) onstart?.(realm, account);
          }} /></label>
      <button class="btn btn-md btn-primary" disabled={!!startOff || stage !== 'GETACCOUNTNAME'} title={startOff} onclick={() => onstart?.(realm, account)}
        >Start</button>
      {#if history.length > 0}
        <FloatingLabel label="Account history"
          ><select class="select input-md" bind:value={historyPick} onchange={() => (account = historyPick)}
            ><option value="" disabled>—</option>{#each history as a (a)}<option>{a}</option>{/each}</select
          ></FloatingLabel>
        <button
          class="btn btn-md"
          aria-label="Remove account from history"
          disabled={!historyPick}
          title={historyPick ? 'Removes account from the dropdown list' : 'Pick an account in the history first'}
          onclick={() => {
            onremoveaccount?.(historyPick);
            historyPick = '';
          }}>X</button>
      {/if}
    </div>
    <div id="{uid}-discriminator" class="text-12">
      {#if missingDiscriminator(account)}<PobText text="^1Missing discriminator e.g. #1234" />{/if}
    </div>
    <p class="note">Note: if the account name contains non-ASCII characters it must be pasted into the textbox, not typed manually.</p>
  {:else}
    <span class="text-12 text-text-2">Choose character to import data from:</span>
    <div class="flex flex-wrap items-end gap-2.5">
      <FloatingLabel label="League"
        ><select class="select input-md" value={league ?? ''} onchange={(e) => (league = e.currentTarget.value || undefined)}
          >{#each leagues as l (l)}<option value={l}>{l}</option>{/each}<option value="">All</option></select
        ></FloatingLabel>
      <FloatingLabel label="Character"
        ><select
          class="select input-md w-100"
          value={chosen?.char.name ?? ''}
          disabled={!selecting}
          title={selecting ? undefined : 'Importing…'}
          onchange={(e) => (picked = e.currentTarget.value)}
          >{#each list as e (e.char.name)}<option value={e.char.name}>{e.label} — {e.detail}</option>{/each}</select
        ></FloatingLabel>
    </div>
    <div class="flex flex-wrap items-center gap-3 text-13">
      <button
        class="btn btn-md btn-primary"
        disabled={!selecting || !chosen}
        title={selecting ? undefined : 'Importing…'}
        onclick={() => chosen && onimporttree?.(chosen.char, deleteJewels)}>Passive Tree and Jewels</button>
      <Checkbox compact bind:checked={deleteJewels} title="Delete all equipped jewels when importing.">Delete jewels</Checkbox>
    </div>
    <div class="flex flex-wrap items-center gap-3 text-13">
      <button
        class="btn btn-md btn-primary"
        disabled={!selecting || !chosen}
        title={selecting ? undefined : 'Importing…'}
        onclick={() => chosen && onimportitems?.(chosen.char, items)}>Items and Skills</button>
      <Checkbox compact bind:checked={items.clearSkills} title="Delete all existing skills when importing.">Delete skills</Checkbox>
      <Checkbox compact bind:checked={items.clearItems} title="Delete all equipped items when importing.">Delete equipment</Checkbox>
      <Checkbox compact bind:checked={items.ignoreWeaponSwap} title="Ignore items and skills in weapon swap.">Ignore weapon swap</Checkbox>
    </div>
    <p class="note">Tip: After you finish importing a character, make sure you update the bandit choice, as it can only be imported by logging in.</p>
    <button class="btn btn-md self-start" onclick={() => onclose?.()}>Close</button>
  {/if}
</section>
