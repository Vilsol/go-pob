<script>
  import NumberInput from './NumberInput.svelte';
</script>

<div class="flex flex-row w-screen border-b-2 border-white bg-neutral-800 min-h-[50px]">
  <!-- Left Side -->
  <div class="flex flex-row justify-between border-r-2 border-white p-2 flex-1 items-center">
    <div class="flex flex-row gap-3 items-center">
      <button class="container">&lt;&lt; Back</button>
      <div class="flex flex-row items-center">
        <span>Current Build:</span>
        <!-- TODO Placeholder -->
        <div class="ml-2 container">Unnamed build</div>
      </div>
      <button class="container">Save</button>
      <button class="container">Save As</button>
    </div>

    <div class="container h-fit">
      <!-- TODO Placeholder -->
      <!-- TODO Tooltip -->
      <span>0 / 123</span>
      <span class="ml-3">0 / 8</span>
    </div>
  </div>

  <!-- Right Side -->
  <div class="flex flex-row p-2 flex-1 items-center gap-3">
    <NumberInput prefix="Level:" min={1} max={100} value={1} />

    <div class="container select-wrapper">
      <select class="input">
        <option>Duelist</option>
        <option>Marauder</option>
        <option>Ranger</option>
        <option>Scion</option>
        <option>Shadow</option>
        <option>Templar</option>
        <option>Witch</option>
      </select>
    </div>

    <div class="container select-wrapper">
      <select class="input">
        <option>None</option>
      </select>
    </div>
  </div>
</div>
