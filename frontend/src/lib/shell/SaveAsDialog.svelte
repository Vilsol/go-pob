<script lang="ts">
  import Select from '$lib/design/Select.svelte';
  import InlineMessage from '$lib/design/InlineMessage.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Field from '$lib/design/Field.svelte';
  import Input from '$lib/design/Input.svelte';
  import Button from '$lib/design/Button.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import PromptDialog from './PromptDialog.svelte';
  import { syncWrap } from '$lib/go/worker';
  import { currentBuildPath } from '$lib/global.js';
  import { logError } from '$lib/utils';
  import { leaf } from '$lib/design/builds';
  import { loadTree, pathOf } from './build_list';
  import { saveErrorText } from './save_error';
  import { saveAsName } from './build_name.svelte';
  import type { SavedBuild } from '$lib/design/fixtures/app';

  type Props = { open?: boolean; onsaved?: () => void };
  let { open = $bindable(false), onsaved }: Props = $props();

  let name = $state('Unnamed build');
  let folder = $state('');
  let folders = $state<string[]>([]);
  let builds = $state<SavedBuild[]>([]);
  let asking = $state(false);
  let replacing = $state(false);
  let failed = $state<string>();

  const load = () =>
    loadTree((dir) => syncWrap.ListBuilds(dir), '')
      .then((t) => {
        folders = t.folders;
        builds = t.builds;
      })
      .catch(logError);

  $effect(() => {
    if (!open) return;
    name = saveAsName(currentBuildPath.current);
    void load();
  });

  const write = () => {
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

  // Build.lua:1384-1396
  const save = () => {
    if (!name) return;
    if (builds.some((b) => b.folder === folder && b.name === name)) replacing = true;
    else write();
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

<Dialog bind:open title="Save build as" size="md">
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
      <Select class="w-full" bind:value={folder}>
        <option value="">All builds</option>
        {#each folders as f (f)}<option value={f}>{'  '.repeat(f.split('/').length) + leaf(f)}</option>{/each}
      </Select>
    </Field>
    <Button size="md" title="Make a folder in {folder ? leaf(folder) : 'All builds'}" onclick={() => (asking = true)}>New folder</Button>
  </div>
  {#if failed}<InlineMessage kind="error" size="md" tag="p" class="whitespace-pre-line">{failed}</InlineMessage>{/if}
  {#snippet actions()}
    <DialogActions confirm="Save" disabled={!name} oncancel={() => (open = false)} onconfirm={save} />
  {/snippet}
</Dialog>

<Dialog bind:open={replacing} title="Replace “{name}”?">
  <span class="text-13 text-text-2"
    >{folder ? leaf(folder) : 'All builds'} already holds a build named {name}. It is replaced, and cannot be brought back.</span>
  {#snippet actions()}
    <DialogActions
      confirm="Replace"
      tone="danger"
      oncancel={() => (replacing = false)}
      onconfirm={() => {
        replacing = false;
        write();
      }} />
  {/snippet}
</Dialog>

<PromptDialog bind:open={asking} title="New folder" label="Folder name" confirm="Create" onconfirm={newFolder} />
