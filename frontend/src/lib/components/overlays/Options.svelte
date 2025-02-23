<script lang="ts">
  import { fontScaling, zoomSensitivity, devMode } from '$lib/global.js';

  let {
    onclose
  }: {
    onclose: () => void;
  } = $props();
</script>

<div class="flex flex-col gap-4">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container">Application Options</legend>
    <div class="side-by-side-max-content w-full">
      <label for="options-font-scaling">Font Scaling:</label>
      <div class="flex flex-row gap-1">
        <input id="options-font-scaling" type="range" min="6" max="18" bind:value={$fontScaling} step="0.5" class="slider" />
        <span aria-hidden="true">{$fontScaling}pt</span>
      </div>
      <label for="options-zoom-sensitivity">Zoom Sensitivity:</label>
      <div class="flex flex-row gap-1">
        <input id="options-zoom-sensitivity" type="range" min="1" max="10" bind:value={$zoomSensitivity} step="0.1" class="slider" />
        <span aria-hidden="true">{$zoomSensitivity}</span>
      </div>
      <label for="options-dev-mode">Dev mode:</label>
      <div class="flex flex-row gap-1">
        <input id="options-dev-mode" type="checkbox" bind:checked={$devMode} />
      </div>
    </div>
  </fieldset>

  <div class="flex flex-row items-center justify-center">
    <button class="container" onclick={onclose}>Close</button>
  </div>
</div>
