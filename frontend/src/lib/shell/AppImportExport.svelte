<script lang="ts">
  import { resource } from 'runed';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { base } from '$app/paths';
  import { buildRevision, currentBuild, currentBuildPath, outputs, UITick } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import ImportExportPage from '$lib/design/ImportExportPage.svelte';
  import { downloadUrl, jsonVerdict, type ImportMode } from '$lib/design/share';
  import { REALMS, UNAVAILABLE } from './import_export';
  import { identityOf } from './identity';
  import { downloadBuild } from './link_import';
  import { unsavedName } from './build_name.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import PobText from '$lib/design/PobText.svelte';
  import CharacterImport from '$lib/design/CharacterImport.svelte';
  import { NEGATIVE, POSITIVE, REALM_LIST, type AccountCharacter, type ImportStage } from '$lib/design/account';
  import { downloadCharacter, fetchCharacterList, lastAccount, lastRealm, loadHistory, removeAccount, saveAccount } from './account_import';

  const exported = resource(
    () => [identityOf($outputs), $buildRevision] as const,
    async () => (syncWrap ? await syncWrap.ExportCode() : '')
  );
  const code = $derived(exported.loading ? '' : (exported.current ?? ''));

  // ImportTab.lua:506-509, 746-747
  let exportParty = $state(false);
  $effect(() => {
    void $outputs;
    void Promise.resolve($currentBuild?.ExportParty())
      .then((on) => (exportParty = !!on))
      .catch(logError);
  });
  const onexportparty = (on: boolean) => {
    $currentBuild
      ?.SetExportParty(on)
      .then(() => UITick('exportParty'))
      .catch(logError);
  };

  let confirmImport = $state<() => void>();
  let confirmOpen = $state(false);
  let importText = $state('');
  let mode = $state<ImportMode>('replace');
  let copied = $state(false);
  let importError = $state<string | null>(null);
  let imported = $state<string | null>(null);
  let fetching = $state(false);

  $effect(() => {
    void code;
    copied = false;
  });

  const oncopy = (text: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() => (copied = true))
      .catch(logError);
  };

  const importCode = (text: string, how: ImportMode, failed: (err: unknown) => void) => {
    if (how === 'compare') {
      syncWrap
        ?.LoadCompareCode(text)
        .then(() => goto(`${base}/compare`))
        .catch(failed);
      return;
    }
    // ImportTab.lua:663-666
    const run = () =>
      syncWrap
        ?.ImportCode(text)
        .then(() => {
          if (how === 'new') {
            currentBuildPath.current = '';
            unsavedName.imported = true;
          }
          return goto(`${base}/tree`);
        })
        .catch(failed);
    if (how === 'replace') {
      // ImportTab.lua:648-653
      confirmImport = run;
      return;
    }
    void run();
  };
  $effect(() => {
    if (confirmImport) confirmOpen = true;
  });

  const onimport = (text: string, how: ImportMode) => {
    importError = null;
    const failed = (err: unknown) => {
      importError = err instanceof Error ? err.message : String(err);
    };
    imported = null;
    // ImportTab.lua:701-705
    if (jsonVerdict(text)?.valid) {
      void (async () => {
        const items = await syncWrap?.ImportItemsAndSkills(text);
        if (!items?.ok) return void (importError = items?.text ?? null);
        const tree = await syncWrap?.ImportPassiveTreeAndJewels(text);
        if (!tree?.ok) return void (importError = tree?.text ?? null);
        imported = `${items.text} ${tree.text}`;
      })().catch(failed);
      return;
    }
    const link = downloadUrl(text);
    if (link) {
      fetching = true;
      downloadBuild(text)
        .then((raw) => importCode(raw, how, failed))
        .catch(failed)
        .finally(() => (fetching = false));
      return;
    }
    importCode(text, how, failed);
  };

  // ImportTab.lua:321-472, 813-1020
  let realm = $state(lastRealm());
  let account = $state(lastAccount());
  let accountName = '';
  let history = $state(loadHistory());
  let charStatus = $state('Idle');
  let stage = $state<ImportStage>('GETACCOUNTNAME');
  let characters = $state<AccountCharacter[]>([]);
  let lastLeague = $state<string>();
  let confirmTree = $state<() => void>();
  let confirmTreeOpen = $state(false);
  $effect(() => {
    if (confirmTree) confirmTreeOpen = true;
  });

  const onstart = async () => {
    stage = 'DOWNLOADCHARLIST';
    charStatus = 'Retrieving character list...';
    const r = await fetchCharacterList(realm, account);
    charStatus = r.status;
    if (!r.ok) {
      stage = 'GETACCOUNTNAME';
      return;
    }
    account = accountName = r.accountName;
    characters = r.characters;
    stage = 'SELECTCHAR';
    history = saveAccount(history, account, realm);
  };

  const importCharacter = async (
    c: AccountCharacter,
    endpoint: 'passives' | 'items',
    apply: (body: string) => Promise<{ ok: boolean; text: string } | undefined>
  ) => {
    stage = 'IMPORTING';
    charStatus = endpoint === 'passives' ? 'Retrieving character passive tree...' : 'Retrieving character items...';
    const d = await downloadCharacter(endpoint, realm, accountName, c);
    if (!d.ok) {
      stage = 'SELECTCHAR';
      charStatus = d.status;
      return;
    }
    lastLeague ??= c.league;
    const r = await apply(d.body);
    stage = 'SELECTCHAR';
    charStatus = r?.ok ? `${POSITIVE}${r.text}` : `${NEGATIVE}${r?.text ?? 'no build'}`;
  };

  const onimporttree = (c: AccountCharacter, deleteJewels: boolean) => {
    const run = () =>
      void importCharacter(c, 'passives', async (body) => syncWrap?.ImportAccountPassiveTree(JSON.stringify(c), body, deleteJewels)).catch(logError);
    // ImportTab.lua:433-440
    if (($outputs?.PointsUsed?.[0] ?? 0) > 0) confirmTree = run;
    else run();
  };
  const onimportitems = (c: AccountCharacter, o: { clearItems: boolean; clearSkills: boolean; ignoreWeaponSwap: boolean }) =>
    void importCharacter(c, 'items', async (body) =>
      syncWrap?.ImportAccountItems(JSON.stringify(c), body, o.clearItems, o.clearSkills, o.ignoreWeaponSwap)
    ).catch(logError);
</script>

{#if confirmTree}<Dialog
    bind:open={confirmTreeOpen}
    title="Character Import"
    onclose={() => {
      confirmTree = undefined;
    }}>
    <p class="m-0 text-13">Importing the passive tree will overwrite your current tree.</p>
    {#snippet actions()}
      <Button
        variant="primary"
        onclick={() => {
          const run = confirmTree;
          confirmTreeOpen = false;
          confirmTree = undefined;
          run?.();
        }}>Import</Button>
      <Button
        onclick={() => {
          confirmTreeOpen = false;
          confirmTree = undefined;
        }}>Cancel</Button>
    {/snippet}
  </Dialog>{/if}

{#if confirmImport}<Dialog
    bind:open={confirmOpen}
    title="Build Import"
    onclose={() => {
      confirmImport = undefined;
    }}>
    <p class="m-0 text-13"><PobText text="^xFF9922Warning:^7 Importing to the current build will erase ALL existing data for this build." /></p>
    {#snippet actions()}
      <Button
        variant="primary"
        onclick={() => {
          const run = confirmImport;
          confirmOpen = false;
          confirmImport = undefined;
          run?.();
        }}>Import</Button>
      <Button
        onclick={() => {
          confirmOpen = false;
          confirmImport = undefined;
        }}>Cancel</Button>
    {/snippet}
  </Dialog>{/if}

<ImportExportPage
  {code}
  {copied}
  {importError}
  {imported}
  {fetching}
  realms={REALMS}
  unavailable={UNAVAILABLE}
  bind:importText
  bind:mode
  bind:exportParty
  {onexportparty}
  {oncopy}
  {onimport}
  focusCode={!!page.state.focusImport}>
  {#snippet characterImport()}
    <CharacterImport
      realms={REALM_LIST}
      bind:realm
      bind:account
      {history}
      status={charStatus}
      {stage}
      {characters}
      {lastLeague}
      onstart={() => void onstart().catch(logError)}
      onremoveaccount={(a: string) => (history = removeAccount(history, a))}
      {onimporttree}
      {onimportitems}
      onclose={() => {
        stage = 'GETACCOUNTNAME';
        charStatus = 'Idle';
      }} />
  {/snippet}
</ImportExportPage>
