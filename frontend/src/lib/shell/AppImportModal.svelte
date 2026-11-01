<script lang="ts">
  import Dialog from '$lib/design/Dialog.svelte';
  import Tabs from '$lib/design/Tabs.svelte';
  import { game } from '$lib/go/engine';
  import AppImportExport from './AppImportExport.svelte';
  import { importDialog, returnFocus, type ImportTab } from './import_modal.svelte';
  import { panelId, tabId } from '$lib/design/tab_ids';

  // the PoE2 calculator keeps no export state yet, and with no build open there is nothing to export
  const canExport = $derived($game === 'poe1');
  const tabs = $derived([{ id: 'import', label: 'Import' }, ...(canExport ? [{ id: 'export', label: 'Export' }] : [])]);
  const tab = $derived<ImportTab>(canExport ? importDialog.tab : 'import');
  const uid = $props.id();
</script>

<Dialog
  bind:open={importDialog.open}
  title={canExport ? 'Import / Export' : 'Import build'}
  sheet
  size="xl"
  phone="fullscreen"
  anchor="top"
  onclose={returnFocus}>
  {#snippet head()}
    {#if tabs.length > 1}<Tabs
        id={uid}
        variant="segmented"
        label="Import or export"
        {tabs}
        selected={tab}
        onchange={(id: string) => {
          importDialog.tab = id as ImportTab;
          importDialog.focus = false;
        }} />{/if}
  {/snippet}
  {#if importDialog.open}
    <div
      role={tabs.length > 1 ? 'tabpanel' : undefined}
      id={panelId(uid)}
      aria-labelledby={tabs.length > 1 ? tabId(uid, tab) : undefined}
      class="scroll flex min-h-0 flex-col gap-4 p-4 max-phone:p-3">
      <AppImportExport {tab} focus={importDialog.focus} ondone={() => (importDialog.open = false)} />
    </div>
  {/if}
</Dialog>
