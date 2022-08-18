<script lang="ts">
  import NumberInput from '../../lib/components/NumberInput.svelte';
  import Input from '../../lib/components/Input.svelte';
  import { currentBuild } from '../../lib/global';
  import { writable } from 'svelte/store';
  import { syncWrap } from '../../lib/go/worker';
  import SelectItem from '../../lib/components/SelectItem.svelte';
  import SelectSelection from '../../lib/components/SelectSelection.svelte';
  import Select from 'svelte-select';
  import { GetSkillGems } from '../../lib/cache';
  import { onMount } from 'svelte';
  import { colorCodes } from '../../lib/display/colors';
  import type { SkillGroupUpdate } from '../../lib/custom_types';
  import type { pob } from '../../lib/types';

  let skillSetCount = 1;
  $: $currentBuild?.Skills.SkillSets.then((v) => (skillSetCount = v.length));

  let activeSkillSet = 1;
  $: $currentBuild?.Skills.ActiveSkillSet.then((v) => (activeSkillSet = v));

  let visualSocketGroup = 0;
  const mainSocketGroup = writable(-1);
  $: $currentBuild?.Build.MainSocketGroup.then((v) => mainSocketGroup.set(v - 1));
  mainSocketGroup.subscribe((value) => {
    value >= 0 && syncWrap.SetMainSocketGroup(value + 1);
  });

  let updatingLabel = true;
  const socketGroupLabel = writable('');
  $: $currentBuild?.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup]?.Label.then((v) => {
    updatingLabel = true;
    socketGroupLabel.set(v);
    updatingLabel = false;
  });
  socketGroupLabel.subscribe((data) => {
    if (updatingLabel || !$currentBuild) {
      return;
    }
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Label =
      data as unknown as Promise<string>;
  });

  const onRightClickSocketGroup = (i: number, event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    mainSocketGroup.set(i);
  };

  let skillGemList = [];

  const gemColorMap = {
    STR: colorCodes.STRENGTH,
    DEX: colorCodes.DEXTERITY,
    INT: colorCodes.INTELLIGENCE,
    NONE: colorCodes.NORMAL
  };

  onMount(() => {
    // TODO Sort gems
    GetSkillGems().then((all) => {
      skillGemList = all.map((g) => ({
        label: '^' + gemColorMap[g.GemType] + g.Base.Name,
        value: g.ID,
        data: g
      }));
    });
  });

  let updatingCurrentGemGroup = true;
  const currentGemGroup = writable<SkillGroupUpdate | undefined>();
  $: skillGemList.length > 0 &&
    new Promise(async () => {
      const skillGroup = $currentBuild?.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup];
      if (!skillGroup) {
        currentGemGroup.set(undefined);
        return;
      }
      const gems = [];
      if (skillGroup.Gems && (await skillGroup.Gems)) {
        const len = await skillGroup.Gems.length;
        for (let i = 0; i < len; i++) {
          const gem = skillGroup.Gems[i];
          const gemID = await gem.GemID;
          gems.push({
            GemListValue: skillGemList.find((g) => g.value === gemID),
            GemID: gemID,
            Quality: await gem.Quality,
            Enabled: await gem.Enabled,
            EnableGlobal1: await gem.EnableGlobal1,
            EnableGlobal2: await gem.EnableGlobal2,
            QualityID: await gem.QualityID,
            Count: await gem.Count,
            Level: await gem.Level,
            SkillPart: await gem.SkillPart,
            SkillPartCalcs: await gem.SkillPartCalcs,
            NameSpec: await gem.NameSpec,
            SkillID: await gem.SkillID,
            SkillMinionItemSet: await gem.SkillMinionItemSet,
            SkillMinion: await gem.SkillMinion
          });
        }
      }

      updatingCurrentGemGroup = true;
      currentGemGroup.set({
        Gems: gems,
        Enabled: await skillGroup.Enabled,
        IncludeInFullDPS: await skillGroup.IncludeInFullDPS,
        Slot: await skillGroup.Slot
      });
      updatingCurrentGemGroup = false;
    });

  currentGemGroup.subscribe(async (group) => {
    if (updatingCurrentGemGroup) {
      return;
    }

    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Slot = group.Slot as Promise<string>;
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Enabled =
      group.Enabled as Promise<boolean>;
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].IncludeInFullDPS =
      group.IncludeInFullDPS as Promise<boolean>;

    if (group.Gems) {
      await $currentBuild.SetSocketGroupGems(
        activeSkillSet - 1,
        visualSocketGroup,
        group.Gems.map((g) => ({
          ...g,
          GemID: (g as unknown).GemListValue.value
        }))
      );
    }

    syncWrap.Tick();
  });

  let socketGroupList: string[] = [];
  $: $currentBuild?.Skills.SkillSets[activeSkillSet - 1].Skills.then(async (skills) => {
    const finalList = [];
    for (let i = 0; i < skills.length; i++) {
      finalList.push(await $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[i].Label);
    }
    socketGroupList = finalList;
  });

  const removeGem = (i: number) => {
    $currentGemGroup.Gems.splice(i, 1);
    currentGemGroup.set($currentGemGroup);
  };

  const addGem = () => {
    const firstGem = skillGemList[0];
    $currentGemGroup.Gems.push({
      GemListValue: firstGem,
      Quality: 0,
      SkillPart: 1,
      EnableGlobal2: true,
      SkillPartCalcs: 1,
      QualityID: 'Default',
      GemID: 'Metadata/Items/Gems/SkillGemFireball',
      Enabled: true,
      Count: 1,
      EnableGlobal1: true,
      NameSpec: firstGem.value,
      Level: 1,
      SkillID: firstGem.value,
      SkillMinionItemSet: 1,
      SkillMinion: 'SummonedPhantasm'
    } as unknown as pob.Gem);
    currentGemGroup.set($currentGemGroup);
  };
</script>

{#if $currentBuild}
  <div class="p-2 px-4">
    <div class="flex flex-row gap-4 flex-wrap">
      <!-- Left Side -->
      <div class="flex flex-col w-[450px] gap-2">
        <div class="flex flex-row items-center gap-2">
          <span>Skill set:</span>

          <div class="container select-wrapper flex-1">
            <select class="input" disabled={skillSetCount === 1}>
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

        <select
          bind:value={visualSocketGroup}
          class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[300px]"
          size="18">
          {#each socketGroupList as label, i}
            <option value={i} on:contextmenu={(event) => onRightClickSocketGroup(i, event)}>
              {label}{$mainSocketGroup === i ? ' (Active)' : ''}
            </option>
          {/each}
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
      {#if $currentGemGroup}
        <div class="flex flex-col min-w-[500px] gap-2">
          <Input prefix="Label:" fullWidth={true} bind:value={$socketGroupLabel} />

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
              <input type="checkbox" class="text-2xl" bind:checked={$currentGemGroup.Enabled} />
            </div>

            <div class="flex flex-row items-center gap-2">
              <span>Include in Full DPS:</span>
              <input type="checkbox" class="text-2xl" bind:checked={$currentGemGroup.IncludeInFullDPS} />
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

            {#each $currentGemGroup.Gems as gemGroup, i}
              <button class="container font-bold" on:click={() => removeGem(i)}>X</button>

              <div class="min-w-full themed">
                <Select
                  bind:value={gemGroup.GemListValue}
                  items={skillGemList}
                  isClearable={false}
                  placeholder=""
                  showIndicator={true}
                  Item={SelectItem}
                  Selection={SelectSelection}
                  listOffset={0} />
              </div>

              <NumberInput min={1} max={gemGroup.GemListValue.data.MaxLevel} bind:value={gemGroup.Level} />

              <div class="container select-wrapper">
                <select class="input">
                  <option>Default</option>
                </select>
              </div>

              <NumberInput min={0} max={21} bind:value={gemGroup.Quality} />

              <input type="checkbox" class="text-2xl" bind:checked={gemGroup.Enabled} />

              <NumberInput min={1} max={99} bind:value={gemGroup.Count} />
            {/each}

            <div class="col-span-7 w-full mt-2">
              <button class="container font-bold min-w-full" on:click={() => addGem()}>Add Skill Gem</button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style lang="postcss">
  .gem-grid {
    grid-template-columns: min-content 4fr 1fr 1fr 1fr 1fr 1fr;
  }
</style>
