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
      syncWrap
        ?.ImportItemsAndSkills(text)
        .then((r) => {
          if (r.ok) imported = r.text;
          else importError = r.text;
        })
        .catch(failed);
      return;
    }
    const link = downloadUrl(text);
    if (link) {
      fetching = true;
      downloadBuild(link.site, link.url)
        .then((raw) => importCode(raw, how, failed))
        .catch(failed)
        .finally(() => (fetching = false));
      return;
    }
    importCode(text, how, failed);
  };
</script>

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
  focusCode={!!page.state.focusImport} />
