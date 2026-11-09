<script module lang="ts">
  // the build a character import from this dialog opened; the character's other half goes into it
  let madeBuild: object | undefined;
</script>

<script lang="ts">
  import { resource } from 'runed';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { get } from 'svelte/store';
  import { untrack } from 'svelte';
  import { engines, game } from '$lib/go/engine';
  import { buildRevision, currentBuild, currentBuildPath, outputs, poe2Build, poe2Enabled, UITick } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import { addToast } from '$lib/design/toast';
  import ImportExportPage from '$lib/design/ImportExportPage.svelte';
  import { NOT_A_CODE, POE2_CHARACTER, POE2_CHARACTER_DISABLED, downloadUrl, importFailure, jsonVerdict, type ImportMode } from '$lib/design/share';
  import { REALMS, SITE_KEY, UNAVAILABLE, loadExportSite, saveExportSite } from './import_export';
  import { followOtherTabs } from '$lib/storage/tabs';
  import { identityOf } from './identity';
  import { beforeReplace, downloadBuild, shareBuild } from './link_import';
  import { openForeign, openPoe2Code } from '$lib/go/game_route';
  import { decodeCode, rootElement } from '$lib/go/game_detect';
  import { poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import { COMPARE_MIXED_POE1 } from './compare_types';
  import { opening, unsavedName } from './build_name.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import CharacterImport from '$lib/design/CharacterImport.svelte';
  import { REALM_LIST, type AccountCharacter } from '$lib/design/account';
  import { removeAccount } from './account_import';
  import { accountSession, importItems, importTree, resetAccountSession, startAccount } from './account_session.svelte';
  import { newBuild } from './build_list';
  import { compareLoaded, type ImportTab } from './import_modal.svelte';
  import AppImportExportPlanner from './AppImportExportPlanner.svelte';

  // tab: the side the import dialog shows; focus: the code field takes focus (Build.lua:1160-1162); ondone: an
  // import that opened a build or the Compare page
  type Props = { tab?: ImportTab; focus?: boolean; ondone?: () => void };
  let { tab = 'import', focus = false, ondone }: Props = $props();

  // ImportTab.lua:132-203 for the open build's game
  const share = $derived(engines[$game ?? 'poe1'].share);
  const exported = resource(
    () => [identityOf($outputs), $buildRevision, $poe2Build, share, tab] as const,
    async ([, , , s, t]) => (t === 'export' ? ((await s.code()) ?? '') : '')
  );
  const code = $derived(exported.loading ? '' : (exported.current ?? ''));

  // ImportTab.lua:506-509, 746-747; ImportTab.lua:138-141 (PoB-PoE2)
  let exportParty = $state(false);
  // a read that started before the latest one, or before a click, is stale
  let partyRead = 0;
  $effect(() => {
    void $outputs;
    // the PoE2 box shows only on the export side, and the worker is asked only there
    const build = $game === 'poe2' && $poe2Build;
    if (build && tab !== 'export') return;
    const at = ++partyRead;
    void Promise.resolve(build ? poe2Worker().then(async (w) => (await w.GetPartyExport())?.on) : $currentBuild?.ExportParty())
      .then((on) => {
        if (at === partyRead) exportParty = !!on;
      })
      .catch(logError);
  });
  const onexportparty = (on: boolean) => {
    partyRead++;
    if ($game === 'poe2') {
      void poe2Run((w) => w.SetExportParty(on));
      return;
    }
    $currentBuild
      ?.SetExportParty(on)
      .then(() => UITick('exportParty', on ? 'Turned on Export support' : 'Turned off Export support'))
      .catch(logError);
  };

  let importText = $state('');
  let mode = $state<ImportMode>('new');
  // an import never replaces the open build: it opens as a new one, or beside it on Compare
  const modes = $derived<ImportMode[]>($game ? ['new', 'compare'] : ['new']);
  $effect(() => {
    if (!modes.includes(mode)) mode = 'new';
  });
  // a character import never writes into a build it did not open: it opens a new one first, asked about as a code
  // import is (Build.lua:1421-1448); the account session survives, so the character's other half can follow
  const intoNewBuild = (run: () => Promise<unknown>) =>
    new Promise<void>((done, fail) => {
      const go = () => {
        const kept = { ...accountSession };
        Promise.resolve(newBuild())
          .then(() => {
            Object.assign(accountSession, kept);
            madeBuild = get(currentBuild);
            return run();
          })
          .then(() => done(), fail);
      };
      if (madeBuild && get(currentBuild) === madeBuild) run().then(() => done(), fail);
      else if (get(game) && beforeReplace.ask) beforeReplace.ask(go);
      else go();
    });
  let copied = $state(false);
  let importError = $state<string | null>(null);
  let imported = $state<string | null>(null);
  let fetching = $state(false);

  // ImportTab.lua:534-554
  const siteOf = (label: string | undefined) => share.sites.find((x) => x.label === label)?.label;
  let site = $state(untrack(() => siteOf(loadExportSite()) ?? share.sites[0].label));
  $effect(() => {
    if (!siteOf(site)) site = share.sites[0].label;
  });
  $effect(() => followOtherTabs(SITE_KEY, { set: (v?: string) => (site = siteOf(v) ?? site) }, loadExportSite));
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
        if (shared !== code) return;
        shareLink = l;
        copied = false;
        navigator.clipboard
          .writeText(l)
          .then(() => addToast(`Link copied\n${l}`))
          .catch(logError);
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

  // ImportTab.lua:286-294 (PoB-PoE2); a code whose XML is a PoE1 build is refused as the Compare page refuses it
  const loadCompare2 = async (text: string) => {
    const xml = await decodeCode(text);
    if (xml === undefined) throw new Error(NOT_A_CODE);
    if (rootElement(xml) === 'PathOfBuilding') throw new Error(COMPARE_MIXED_POE1);
    return (await poe2Worker()).LoadCompare(xml, 'Imported comparison');
  };

  const importCode = (text: string, how: ImportMode, failed: (err: unknown) => void) => {
    if (how === 'compare') {
      void (get(game) === 'poe2' ? loadCompare2(text) : syncWrap?.LoadCompareCode(text))
        ?.then(() => {
          compareLoaded.seq++;
          ondone?.();
          return goto(`${base}/compare`);
        })
        .catch(failed);
      return;
    }
    // ImportTab.lua:663-666
    const run = () => {
      const kept = opening.modFlag;
      opening.modFlag = true;
      const clear = () => syncWrap?.ClearBuild();
      return openPoe2Code(text, clear)
        .then((opened) => opened || syncWrap?.ImportCode(text).then((foreign) => openForeign(foreign, clear)))
        .then(
          (opened) => {
            ondone?.();
            if (opened) {
              currentBuildPath.current = '';
              return;
            }
            resetAccountSession();
            currentBuildPath.current = '';
            unsavedName.imported = true;
            return goto(`${base}/tree`);
          },
          (err: unknown) => {
            // a refused build leaves the open one as it was
            opening.modFlag = kept;
            throw err;
          }
        )
        .catch(failed);
    };
    // Build.lua:1421-1448: the open build's unsaved changes are asked about first, as leaving it does
    if (beforeReplace.ask) beforeReplace.ask(() => void run());
    else void run();
  };

  const onimport = (text: string, how: ImportMode) => {
    importError = null;
    const failed = (err: unknown) => {
      importError = importFailure(err);
    };
    imported = null;
    // ImportTab.lua:701-705
    const json = jsonVerdict(text);
    if (json?.game === 'poe2') return void (importError = $poe2Enabled ? POE2_CHARACTER : POE2_CHARACTER_DISABLED);
    if (json?.valid) {
      intoNewBuild(async () => {
        const items = await syncWrap?.ImportItemsAndSkills(text);
        if (!items?.ok) return void (importError = items?.text ?? null);
        const tree = await syncWrap?.ImportPassiveTreeAndJewels(text);
        if (!tree?.ok) return void (importError = tree?.text ?? null);
        imported = `${items.text} ${tree.text}`;
      }).catch(failed);
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
    phone="centred"
    bind:open={confirmTreeOpen}
    title="Character Import"
    onclose={() => {
      confirmTree = undefined;
    }}>
    <p class="m-0 text-13">Importing the passive tree will overwrite your current tree.</p>
    {#snippet actions()}
      <DialogActions
        confirm="Import"
        oncancel={() => {
          confirmTreeOpen = false;
          confirmTree = undefined;
        }}
        onconfirm={() => {
          const run = confirmTree;
          confirmTreeOpen = false;
          confirmTree = undefined;
          run?.();
        }} />
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
  sites={share.sites}
  bind:importText
  bind:mode
  {modes}
  show={tab}
  bind:exportParty
  {onexportparty}
  {oncopy}
  {onimport}
  focusCode={focus}>
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
      onimporttree={(c: AccountCharacter, deleteJewels: boolean) => {
        void intoNewBuild(() => Promise.resolve(importTree(c, deleteJewels, (run) => (confirmTree = run)))).catch(logError);
      }}
      onimportitems={(c: AccountCharacter, o: Parameters<typeof importItems>[1]) => {
        void intoNewBuild(() => Promise.resolve(importItems(c, o))).catch(logError);
      }}
      onclose={() => {
        s.stage = 'GETACCOUNTNAME';
        s.status = 'Idle';
      }} />
  {/snippet}
</ImportExportPage>
{#if tab === 'export' && $game === 'poe2'}<AppImportExportPlanner />{/if}
