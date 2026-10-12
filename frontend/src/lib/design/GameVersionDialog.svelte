<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import PobText from './PobText.svelte';

  // Build.lua:1325-1350
  type Props = { open: boolean; version: string; onconvert: () => void; oncancel: () => void };
  let { open = $bindable(), version, onconvert, oncancel }: Props = $props();
  const NOTE = `^x80A080Info:^7 You are trying to load a build created for a version of Path of Exile that is
not supported by us. You will have to convert it to the current game version to load it.
To use a build newer than the current supported game version, you may have to update.
To use a build older than the current supported game version, we recommend loading it
in an older version of Path of Building Community instead.`;
  const WARNING = `^xFF9922Warning:^7 Converting a build to a different game version may have side effects.
For example, if the passive tree has changed, then some passives may be deallocated.
You should create a backup copy of the build before proceeding.`;
  let decided = false;
</script>

<Dialog
  bind:open
  title="Game Version"
  class="w-145 p-4"
  onclose={() => {
    if (!decided) oncancel();
    decided = false;
  }}>
  <div class="flex flex-col gap-3 text-13 whitespace-pre-line">
    <p class="m-0"><PobText text={NOTE} /></p>
    <p class="m-0"><PobText text={WARNING} /></p>
  </div>
  {#snippet actions()}
    <Button
      variant="primary"
      onclick={() => {
        decided = true;
        open = false;
        onconvert();
      }}>Convert to {version}</Button>
    <Button
      onclick={() => {
        decided = true;
        open = false;
        oncancel();
      }}>Cancel</Button>
  {/snippet}
</Dialog>
