<script lang="ts">
  import NumberInput from '../lib/components/NumberInput.svelte';
  import Input from '../lib/components/Input.svelte';
</script>

<div class="p-2 px-4">
  <div class="flex flex-row gap-4 flex-wrap">
    <!-- Left Side -->
    <div class="flex flex-col w-[450px] gap-2">
      <div class="flex flex-row items-center gap-2">
        <span>Skill set:</span>

        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Default</option>
          </select>
        </div>

        <button class="container">Manage...</button>
      </div>

      <div class="flex flex-row items-center gap-2">
        <span class="flex-1">Socket Groups:</span>

        <button class="container">New</button>
        <button class="container">Delete All</button>
        <button class="container" disabled>Delete</button>
      </div>

      <select class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[300px]" size="18">
        <option>Fireball (Active)</option>
      </select>

      <p class="text-sm">
        Usage Tips:<br />
        - You can copy/paste socket groups using Ctrl+C and Ctrl+V.<br />
        - Ctrl + Click to enable/disable socket groups.<br />
        - Ctrl + Right click to include/exclude in FullDPS calculations.<br />
        - Right click to set as the Main skill group.<br />
      </p>

      <fieldset class="border border-white bg-neutral-900 p-2">
        <legend class="container">Gem Options</legend>
        <div class="side-by-side-max-content">
          <span>Sort gems by DPS:</span>
          <div class="flex flex-row gap-2">
            <input type="checkbox" class="text-2xl" />
            <div class="container select-wrapper">
              <select class="input">
                <option>Full DPS</option>
                <option>Combined DPS</option>
                <option>Total DPS</option>
                <option>Average Hit</option>
                <option>DoT DPS</option>
                <option>Bleed DPS</option>
                <option>Ignite DPS</option>
                <option>Poison DPS</option>
              </select>
            </div>
          </div>

          <span>Match gems character level:</span>
          <div class="flex flex-row gap-2">
            <input type="checkbox" class="text-2xl" />
          </div>

          <span>Default gem level:</span>
          <NumberInput min={1} max={21} />

          <span>Default gem quality:</span>
          <NumberInput min={1} max={21} />

          <span>Show support gems:</span>
          <div class="flex flex-row gap-2">
            <div class="container select-wrapper">
              <select class="input">
                <option>All</option>
                <option>Non-Awakened</option>
                <option>Awakened</option>
              </select>
            </div>
          </div>

          <span>Show gem quality variants:</span>
          <div class="flex flex-row gap-2">
            <input type="checkbox" class="text-2xl" />
          </div>
        </div>
      </fieldset>
    </div>

    <!-- Right Side -->
    <div class="flex flex-col min-w-[500px] gap-2">
      <Input prefix="Label:" fullWidth={true} />

      <div class="flex flex-row w-full justify-between">
        <div class="flex flex-row items-center gap-2">
          <span>Socketed in:</span>
          <div class="container select-wrapper">
            <select class="input">
              <option>None</option>
            </select>
          </div>
        </div>

        <div class="flex flex-row items-center gap-2">
          <span>Enabled:</span>
          <input type="checkbox" class="text-2xl" />
        </div>

        <div class="flex flex-row items-center gap-2">
          <span>Include in Full DPS:</span>
          <input type="checkbox" class="text-2xl" />
        </div>
      </div>

      <div class="grid gem-grid gap-1 w-full">
        <div />
        <div>Gem Name</div>
        <div>Level</div>
        <div>Variant</div>
        <div>Quality</div>
        <div>Enabled</div>
        <div>Count</div>

        <button class="container font-bold">X</button>

        <div class="container select-wrapper min-w-full">
          <select class="input text-blue-500">
            <option>Fireball</option>
          </select>
        </div>

        <NumberInput min={1} max={21} value={1} />

        <div class="container select-wrapper">
          <select class="input">
            <option>Default</option>
          </select>
        </div>

        <NumberInput min={1} max={21} value={1} />

        <input type="checkbox" class="text-2xl" />

        <NumberInput min={1} max={99} value={1} />
      </div>
    </div>
  </div>
</div>

<style lang="postcss">
  .gem-grid {
    grid-template-columns: min-content 2fr 1fr 1fr 1fr 1fr 1fr;
  }
</style>
