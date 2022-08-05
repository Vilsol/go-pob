<div class="w-[25vw] min-w-[370px] max-w-[400px] h-full border-r-2 border-white flex flex-col bg-neutral-900 full-page">
  <div class="flex flex-col gap-3 border-b-2 border-white flex-1 p-2 sidebar-stat-wrapper">
    <div class="flex flex-col gap-2">
      <div class="flex flex-row gap-1">
        <button class="container min-w-fit flex-1">Import/Export Build</button>
        <button class="container min-w-fit flex-1">Notes</button>
        <button class="container min-w-fit flex-1">Configuration</button>
      </div>
      <div class="flex flex-row gap-1">
        <button class="container min-w-fit flex-1">Tree</button>
        <button class="container min-w-fit flex-1">Skills</button>
        <button class="container min-w-fit flex-1">Items</button>
        <button class="container min-w-fit flex-1">Calcs</button>
      </div>
    </div>

    <div class="flex flex-col gap-1">
      <div>Main Skill:</div>
      <div class="container select-wrapper min-w-full">
        <!-- TODO Placeholder -->
        <select class="input w-full">
          <option>&lt;No skills added yet&gt;</option>
        </select>
      </div>
    </div>

    <div class="container min-w-full overflow-y-auto flex-1 flex flex-col gap-2.5 overflow-y-scroll">
      <div class="side-by-side-3">
        <div>Average Damage:</div>
        <div>1.9</div>

        <div>Attack Rate:</div>
        <div>1.20</div>

        <div>Hit Chance:</div>
        <div>99%</div>

        <div>Total DPS:</div>
        <div>2.3</div>

        <div>Culling DPS:</div>
        <div>0.0</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-red-600">Strength:</div>
        <div>20</div>

        <div class="text-green-500">Dexterity:</div>
        <div>20</div>

        <div class="text-blue-500">Intelligence:</div>
        <div>20</div>
      </div>

      <div class="side-by-side-3">
        <div>Effective Hit Pool:</div>
        <div>63</div>

        <div class="text-gray-400">Phys Max Hit:</div>
        <div>60</div>

        <div class="text-yellow-600">Elemental Max Hit:</div>
        <div>38</div>

        <div class="text-pink-600">Chaos Max Hit:</div>
        <div>38</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-red-600">Total Life:</div>
        <div>60</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-blue-500">Total Mana:</div>
        <div>50</div>

        <div class="text-blue-500">Mana Regen:</div>
        <div>0.9</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-cyan-300">Recoverable ES:</div>
        <div>0</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-green-400">Evasion rating:</div>
        <div>16</div>
      </div>

      <div class="side-by-side-3">
        <div>Phys Damage Reduction:</div>
        <div>0%</div>
      </div>

      <div class="side-by-side-3">
        <div class="text-amber-600">Fire Resistance:</div>
        <div class="text-red-600">-60%</div>

        <div class="text-blue-500">Cold Resistance:</div>
        <div class="text-red-600">-60%</div>

        <div class="text-yellow-400">Lightning Resistance:</div>
        <div class="text-red-600">-60%</div>

        <div class="text-pink-600">Chaos Resistance:</div>
        <div class="text-red-600">-60%</div>
      </div>

      <div class="side-by-side-3">
        <div>Movement Speed Modifier:</div>
        <div>+0%</div>
      </div>
    </div>
  </div>

  <div class="flex flex-row p-2 h-[84px]">
    <div class="flex flex-col flex-1 gap-2">
      <button class="container min-w-full flex-1">Options</button>
      <button class="container min-w-full flex-1">About</button>
    </div>
    <div class="flex flex-col flex-1 items-center">
      <span class="flex-1">go-pob</span>
      <span class="flex-1">Version: 0.0.1</span>
    </div>
  </div>
</div>

<style lang="postcss">
  .sidebar-stat-wrapper {
    max-height: calc(100% - 84px);
  }
</style>
