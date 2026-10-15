<script lang="ts">
  import { resource } from 'runed';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { base } from '$app/paths';
  import { buildRevision, currentBuild, currentBuildPath, outputs, UITick } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import ImportExportPage from '$lib/design/ImportExportPage.svelte';
  import { EXPORT_SITES, downloadUrl, jsonVerdict, type ImportMode } from '$lib/design/share';
  import { REALMS, UNAVAILABLE, loadExportSite, saveExportSite } from './import_export';
  import { identityOf } from './identity';
  import { downloadBuild, shareBuild } from './link_import';
  import { opening, unsavedName } from './build_name.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import PobText from '$lib/design/PobText.svelte';
  import CharacterImport from '$lib/design/CharacterImport.svelte';
  import { REALM_LIST, type AccountCharacter } from '$lib/design/account';
  import { removeAccount } from './account_import';
  import { accountSession, importItems, importTree, resetAccountSession, startAccount } from './account_session.svelte';

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

  // ImportTab.lua:534-554
  let site = $state((EXPORT_SITES.find((x) => x.label === loadExportSite()) ?? EXPORT_SITES[0]).label);
  let shareLink = $state<string | null>(null);
  let sharing = $state(false);
  let shareError = $state<string | null>(null);
  const onshare = (to: string) => {
    saveExportSite(to);
    sharing = true;
    shareError = null;
    shareLink = null;
    const shared = code;
    shareBuild(shared, to)
      .then((l) => {
        if (shared === code) shareLink = l;
      })
      .catch((err: unknown) => (shareError = err instanceof Error ? err.message : String(err)))
      .finally(() => (sharing = false));
  };

  $effect(() => {
    void code;
    copied = false;
    shareLink = null;
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
    const run = () => {
      opening.modFlag = true;
      return syncWrap
        ?.ImportCode(text)
        .then(() => {
          resetAccountSession();
          if (how === 'new') {
            currentBuildPath.current = '';
            unsavedName.imported = true;
          }
          return goto(`${base}/tree`);
        })
        .catch(failed);
    };
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
  const s = accountSession;
  let confirmTree = $state<() => void>();
  let confirmTreeOpen = $state(false);
  $effect(() => {
    if (confirmTree) confirmTreeOpen = true;
  });
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
  link={shareLink}
  {sharing}
  {shareError}
  bind:site
  {onshare}
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
      bind:realm={s.realm}
      bind:account={s.account}
      history={s.history}
      status={s.status}
      stage={s.stage}
      characters={s.characters}
      lastLeague={s.lastLeague}
      onstart={() => void startAccount().catch(logError)}
      onremoveaccount={(a: string) => (s.history = removeAccount(s.history, a))}
      selected={s.selected?.name}
      onselect={(c?: AccountCharacter) => (s.selected = c)}
      onimporttree={(c: AccountCharacter, deleteJewels: boolean) => importTree(c, deleteJewels, (run) => (confirmTree = run))}
      onimportitems={importItems}
      onclose={() => {
        s.stage = 'GETACCOUNTNAME';
        s.status = 'Idle';
      }} />
  {/snippet}
</ImportExportPage>
