<script lang="ts">
  import Input from '../../lib/components/Input.svelte';
  import { syncWrap } from '../../lib/go/worker';
  import { sampleBuildCode } from '../../lib/global';

  // TODO Set to empty in prod
  let importCode = sampleBuildCode;

  const importBuildFromCode = () => {
    syncWrap
      ?.ImportCode(importCode)
      .catch((err) => {
        // TODO Error notification
        console.error(err);
      })
      .then(() => {
        syncWrap?.Tick('importBuildFromCode');
      });
  };
</script>

<div class="p-2 px-4 h-full flex flex-col flex-wrap gap-4 w-full overflow-x-auto min-w-[40em]">
  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Character Import</legend>
    <div class="flex flex-col gap-3">
      <span>Character import status: Idle</span>

      <div class="flex flex-col gap-1">
        <span>To start importing a character, enter the character's account name:</span>

        <div class="flex flex-row gap-1">
          <div class="container select-wrapper">
            <select class="input">
              <option>PC</option>
              <option>Xbox</option>
              <option>PS4</option>
              <option>Garena</option>
              <option>Tencent</option>
            </select>
          </div>

          <Input />

          <button class="container">Start</button>

          <div class="flex flex-row gap-2 flex-1">
            <div class="container select-wrapper min-w-full">
              <select class="input" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2 mt-4">
    <legend class="container">Build Sharing</legend>
    <div class="flex flex-col gap-3">
      <div class="flex flex-col gap-1">
        <div class="flex flex-row gap-2 items-center">
          <span>Generate a code to share this build with other Path of Building users:</span>
          <button class="container">Generate</button>
        </div>

        <div class="flex flex-row gap-2 items-center">
          <Input placeholder="Code" disabled={true} />

          <button class="container" disabled>Copy</button>

          <div class="container select-wrapper disabled">
            <select class="input" disabled>
              <option>Pastebin.com</option>
              <option>PoeNinja</option>
              <option>pobb.in</option>
            </select>
          </div>

          <button class="container" disabled>Share</button>
        </div>

        <span>Note: this code can be very long; you can 'Share' to shrink it.</span>
      </div>

      <div class="flex flex-col gap-1">
        <span>To import a build, enter URL or code here:</span>

        <Input fullWidth={true} bind:value={importCode} />

        <div class="flex flex-row gap-1">
          <div class="container select-wrapper disabled">
            <select class="input" disabled>
              <option>Import to a new build</option>
            </select>
          </div>

          <button class="container" disabled={importCode === ''} on:click={importBuildFromCode}>Import</button>
        </div>
      </div>
    </div>
  </fieldset>
</div>
