<script lang="ts">
  import Dialog from '$lib/design/Dialog.svelte';
  import Field from '$lib/design/Field.svelte';
  import Input from '$lib/design/Input.svelte';
  import Button from '$lib/design/Button.svelte';
  import PromptDialog from './PromptDialog.svelte';
  import { syncWrap } from '$lib/go/worker';
  import { currentBuildPath } from '$lib/global.js';
  import { logError } from '$lib/utils';
  import { leaf } from '$lib/design/builds';
  import { loadTree, pathOf } from './build_list';
  import { saveErrorText } from './save_error';

  type Props = { open?: boolean; onsaved?: () => void };
  let { open = $bindable(false), onsaved }: Props = $props();

  let name = $state('Unnamed build');
  let folder = $state('');
  let folders = $state<string[]>([]);
  let asking = $state(false);
  let failed = $state<string>();

  const load = () =>
    loadTree((dir) => syncWrap.ListBuilds(dir), '')
      .then((t) => (folders = t.folders))
      .catch(logError);

  $effect(() => {
    if (open) void load();
  });

  const save = () => {
    if (!name) return;
    const target = pathOf(folder, name);
    failed = undefined;
    syncWrap
      .SaveBuildAs(target)
      .then(() => {
        currentBuildPath.current = target;
        open = false;
        onsaved?.();
      })
      .catch((e: Error) => {
        logError(e);
        failed = saveErrorText(target);
      });
  };

  const newFolder = (n: string) => {
    const made = folder ? `${folder}/${n}` : n;
    syncWrap
      .NewFolder(pathOf(folder, n))
      .then(load)
      .then(() => (folder = made))
      .catch(logError);
  };
</script>

<Dialog bind:open title="Save build as" class="w-110">
  <Field label="Build name">
    <Input
      bind:value={name}
      onkeydown={(e: KeyboardEvent) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          save();
        }
      }} />
  </Field>
  <div class="flex items-end gap-2">
    <Field label="Folder" class="min-w-0 grow">
      <select class="select w-full" bind:value={folder}>
        <option value="">All builds</option>
        {#each folders as f (f)}<option value={f}>{'  '.repeat(f.split('/').length) + leaf(f)}</option>{/each}
      </select>
    </Field>
    <Button size="md" title="Make a folder in {folder ? leaf(folder) : 'All builds'}" onclick={() => (asking = true)}>New folder</Button>
  </div>
  {#if failed}<p role="alert" class="m-0 text-13 whitespace-pre-line text-worse">{failed}</p>{/if}
  {#snippet actions()}
    <Button size="md" variant="ghost" class="px-3" onclick={() => (open = false)}>Cancel</Button>
    <Button size="md" variant="primary" class="px-4" disabled={!name} onclick={save}>Save</Button>
  {/snippet}
</Dialog>

<PromptDialog bind:open={asking} title="New folder" label="Folder name" confirm="Create" onconfirm={newFolder} />
