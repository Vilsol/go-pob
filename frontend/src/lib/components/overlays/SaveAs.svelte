<script lang="ts">
  import Input from '$lib/components/Input.svelte';
  import FileBrowser from '$lib/components/FileBrowser.svelte';
  import { storage } from '$lib/types';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';

  interface Props {
    onclose: () => void;
  }

  let { onclose }: Props = $props();

  let name = $state('Unnamed build');
  let path = $state('');
  let selected = $state<storage.DirEntry | undefined>(undefined);

  const onSave = () => {
    syncWrap
      .SaveBuildAs(path + '/' + name)
      .then(() => {
        onclose();
      })
      .catch(logError);
  };
</script>

<div class="flex flex-col gap-2">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw] flex flex-col gap-2">
    <legend class="container">Save</legend>
    <div class="flex flex-row justify-center">
      <p>Enter new build name:</p>
    </div>

    <Input fullWidth={true} bind:value={name} />

    <div class="w-[50vw] min-w-96 flex flex-col gap-2 max-h-full">
      <FileBrowser bind:path bind:selected hideSearch={true} hideSort={true} hideBuilds={true} showNewFolder={true} />
    </div>

    <div class="flex flex-row justify-center gap-2">
      <button class="container" onclick={onSave} disabled={!name}>Save</button>
      <button class="container" onclick={onclose}>Cancel</button>
    </div>
  </fieldset>
</div>
