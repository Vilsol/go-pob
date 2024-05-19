<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { dump } from '$lib/type_utils';
  import { syncWrap } from '$lib/go/worker';

  const dispatch = createEventDispatcher();

  const close = () => {
    dispatch('close');
  };
</script>

<div class="flex flex-col gap-4">
  <div class='flex flex-row gap-4'>
    <fieldset class="border border-white bg-neutral-900">
      <legend class="container">Build Settings</legend>
      {#await syncWrap?.BuildInfo() then buildInfo}
        <table class='table-auto border-collapse border border-neutral-500'>
          <tbody>
            {#each buildInfo.Settings as dep}
              <tr>
                <td class='border border-neutral-600 p-2'>{dep.Key}</td>
                <td class='border border-neutral-600 p-2'>{dep.Value}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/await}
    </fieldset>

    <fieldset class="border border-white bg-neutral-900">
      <legend class="container">Dependencies</legend>
        {#await syncWrap?.BuildInfo() then buildInfo}
          <table class='table-auto border-collapse border border-neutral-500'>
            <tbody>
              {#each buildInfo.Deps as dep}
                <tr>
                  <td class='border border-neutral-600 p-2'>{dep.Path}</td>
                  <td class='border border-neutral-600 p-2'>{dep.Version}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/await}
    </fieldset>
  </div>

  <div class="flex flex-row items-center justify-center">
    <button class="container" on:click={close}>Close</button>
  </div>
</div>
