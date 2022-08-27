<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { fontScaling } from '$lib/global';

  const dispatch = createEventDispatcher();

  const close = () => {
    dispatch('close');
  };
</script>

<div class="flex flex-col gap-4">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container">Application Options</legend>
    <div class="side-by-side-max-content w-full">
      <span>Font Scaling:</span>
      <div class="flex flex-row gap-1">
        <input type="range" min="6" max="18" bind:value={$fontScaling} step="0.5" class="slider" />
        <span>{$fontScaling}pt</span>
      </div>
    </div>
  </fieldset>

  <div class="flex flex-row items-center justify-center">
    <button class="container" on:click={close}>Close</button>
  </div>
</div>
