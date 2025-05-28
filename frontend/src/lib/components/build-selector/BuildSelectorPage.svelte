<script lang="ts">
  import InfoBox from '$lib/components/InfoBox.svelte';
  import FileBrowser from '$lib/components/FileBrowser.svelte';
  import { openOverlay } from '$lib/overlay';
  import TextDialog from '../overlays/TextDialog.svelte';
  import { syncWrap } from '$lib/go/worker';
  import { Emitter, logError } from '$lib/utils';
  import { storage } from '$lib/types';
  import ConfirmationDialog from '$lib/components/overlays/ConfirmationDialog.svelte';
  import { currentBuildPath } from '$lib/global';

  let path = $state('');
  let selected = $state<storage.DirEntry | undefined>(undefined);

  const refresh = new Emitter<void>();

  const newBuild = () => {
    void syncWrap.NewBuild();
    currentBuildPath.current = '';
  };

  const newFolder = () => {
    openOverlay({
      component: TextDialog,
      props: {
        title: 'New Folder',
        subtitle: 'Enter folder name:',
        confirm: 'Create',
        onconfirm: (name: string) => {
          void syncWrap
            .NewFolder(path + '/' + name)
            .then(() => {
              refresh.emit();
            })
            .catch(logError);
        }
      }
    });
  };

  const openBuild = (entry: storage.DirEntry) => {
    currentBuildPath.current = path + '/' + entry.Name;
    syncWrap.OpenBuild(path + '/' + entry.Name).catch(logError);
  };

  const open = () => {
    if (!selected) {
      return;
    }

    if (selected.Type === 'file') {
      openBuild(selected);
    } else {
      path += '/' + selected.Name;
    }

    selected = undefined;
  };

  const copy = () => {
    if (!selected) {
      return;
    }

    const from = path + '/' + selected.Name;

    openOverlay({
      component: TextDialog,
      props: {
        title: 'Copy',
        subtitle: 'Enter new name:',
        confirm: 'Copy',
        onconfirm: (name: string) => {
          void syncWrap
            .Copy(from, path + '/' + name)
            .then(() => {
              refresh.emit();
            })
            .catch(logError);
        }
      }
    });
  };

  const rename = () => {
    if (!selected) {
      return;
    }

    const from = path + '/' + selected.Name;

    openOverlay({
      component: TextDialog,
      props: {
        title: 'Rename',
        subtitle: 'Enter new name:',
        confirm: 'Rename',
        onconfirm: (name: string) => {
          void syncWrap
            .Rename(from, path + '/' + name)
            .then(() => {
              refresh.emit();
            })
            .catch(logError);
        }
      }
    });
  };

  const del = () => {
    if (!selected) {
      return;
    }

    const from = path + '/' + selected.Name;

    openOverlay({
      component: ConfirmationDialog,
      props: {
        title: 'Delete',
        subtitle: `Are you sure you want to delete this: ${selected.Name}`,
        confirm: 'Delete',
        onconfirm: () => {
          void syncWrap
            .Delete(from)
            .then(() => {
              refresh.emit();
            })
            .catch(logError);
        }
      }
    });
  };
</script>

<div class="flex flex-col gap-2 p-2 items-center max-h-full h-full">
  <div class="flex flex-row gap-2 justify-center">
    <button class="container" type="button" onclick={newBuild}>New Build</button>
    <button class="container" type="button" onclick={newFolder}>New Folder</button>
    <button class="container" type="button" disabled={!selected} onclick={open}>Open</button>
    <button class="container" type="button" disabled={!selected} onclick={copy}>Copy</button>
    <button class="container" type="button" disabled={!selected} onclick={rename}>Rename</button>
    <button class="container" type="button" disabled={!selected} onclick={del}>Delete</button>
  </div>

  <div class="w-[50vw] min-w-96 flex flex-col gap-2 max-h-full h-[calc(100%-50px)]">
    <FileBrowser bind:path bind:selected {refresh} {openBuild} />
  </div>
</div>

<div class="absolute left-0 bottom-0 border-t-2 border-r-2">
  <InfoBox />
</div>
