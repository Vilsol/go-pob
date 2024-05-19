<script lang="ts">
  import NumberInput from '../../lib/components/NumberInput.svelte';
  import Input from '../../lib/components/Input.svelte';
  import { currentBuild, UITick } from '../../lib/global';
  import { writable } from 'svelte/store';
  import { syncWrap } from '../../lib/go/worker';
  import SelectItem from '../../lib/components/SelectItem.svelte';
  import SelectSelection from '../../lib/components/SelectSelection.svelte';
  import Select from 'svelte-select';
  import { GetSkillGems } from '../../lib/cache';
  import { onMount } from 'svelte';
  import { colorCodes, formatColors } from '../../lib/display/colors';
  import type { SkillGroupUpdate } from '../../lib/custom_types';
  import type { pob, exposition } from '../../lib/types';

  let skillSetCount = 1;
  $: $currentBuild?.Skills?.SkillSets?.then((v) => (skillSetCount = v?.length || 0));

  let activeSkillSet = 1;
  $: $currentBuild?.Skills?.ActiveSkillSet?.then((v) => (activeSkillSet = v));

  let visualSocketGroup = 0;
  const mainSocketGroup = writable(-1);
  $: $currentBuild?.Build?.MainSocketGroup?.then((v) => mainSocketGroup.set(v - 1));
  mainSocketGroup.subscribe((value) => {
    value >= 0 && syncWrap?.SetMainSocketGroup(value + 1);
    currentBuild.set($currentBuild);
  });

  let updatingLabel = true;
  const socketGroupLabel = writable('');
  $: $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1].Skills?.[visualSocketGroup]?.Label.then((v: string) => {
    updatingLabel = true;
    socketGroupLabel.set(v);
    updatingLabel = false;
  });
  socketGroupLabel.subscribe((data) => {
    if (updatingLabel || !$currentBuild) {
      return;
    }
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Label = data as Promise<string> & string;
  });

  const onRightClickSocketGroup = (i: number, event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    mainSocketGroup.set(i);
  };

  let skillGemList: { label: string; value: string }[] = [];
  const skillGemMapping: Record<string, exposition.SkillGem> = {};

  const gemColorMap: Record<string, string> = {
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

      all.forEach((g) => {
        skillGemMapping[g.ID] = g;
      });
    });
  });

  let updatingCurrentGemGroup = true;
  const currentGemGroup = writable<SkillGroupUpdate | undefined>();
  $: skillGemList.length > 0 &&
    new Promise(async () => {
      const skillGroup = $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.[visualSocketGroup];
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

  let lastGroup = '';
  currentGemGroup.subscribe(async (group) => {
    if (
      updatingCurrentGemGroup ||
      !group ||
      !$currentBuild ||
      !$currentBuild.Skills ||
      !$currentBuild.Skills.SkillSets ||
      !$currentBuild.Skills.SkillSets[activeSkillSet - 1] ||
      !$currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills ||
      !$currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup]
    ) {
      return;
    }

    // Prevent re-execution
    const newGroup = JSON.stringify(group);
    if (lastGroup === newGroup) {
      return;
    }
    lastGroup = newGroup;

    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Slot = group.Slot as Promise<string> & string;
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].Enabled = group.Enabled as Promise<boolean> & boolean;
    $currentBuild.Skills.SkillSets[activeSkillSet - 1].Skills[visualSocketGroup].IncludeInFullDPS = group.IncludeInFullDPS as Promise<boolean> & boolean;

    if (group.Gems) {
      await $currentBuild?.SetSocketGroupGems?.(
        activeSkillSet - 1,
        visualSocketGroup,
        group.Gems.map((g) => ({
          ...g,
          GemID: (g as unknown).GemListValue.value
        }))
      );
    }

    UITick('currentGemGroup');
  });

  let socketGroupList: {
    label: string;
    enabled: boolean;
    fullDPS: boolean;
  }[] = [];
  $: $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.then(async (skills: unknown[]) => {
    const finalList = [];
    for (let i = 0; i < skills.length; i++) {
      let label: string | undefined = await $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.[i].Label;
      if (label === '') {
        const allGems = $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.[i].Gems;
        if (await allGems) {
          for (let j = 0; j < (await allGems.length); j++) {
            const gem = skillGemMapping[await allGems[j].GemID];
            if (!gem || gem.Support) {
              continue;
            }

            if (label !== '') {
              label += ', ';
            }

            label += gem.Base.Name;
          }
        } else {
          label = '<No active skills>';
        }
      }

      finalList.push({
        label,
        enabled: await $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.[i].Enabled,
        fullDPS: await $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.[i].IncludeInFullDPS
      });
    }
    socketGroupList = finalList;
  });

  const removeGem = (i: number) => {
    $currentGemGroup?.Gems?.splice(i, 1);
    currentGemGroup.set($currentGemGroup);
  };

  const addGem = () => {
    const firstGem = skillGemList[0];
    $currentGemGroup?.Gems?.push({
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

  const registerProperty = (obj: object, publicKey: string, onUpdate: (value: unknown) => void) => {
    const privateKey = '_' + publicKey;
    Object.defineProperty(obj, publicKey, {
      get() {
        return this[privateKey];
      },
      set(value) {
        if (this[privateKey] !== value && $currentBuild) {
          onUpdate(value);
          this[privateKey] = value;
        }
      }
    });
  };

  // Populated with defaults
  const gemOptions = {
    _sortGemsByDPS: true,
    _sortGemsByDPSField: 'FullDPS',
    _matchGemLevelToCharacterLevel: false,
    _defaultGemLevel: 20,
    _defaultGemQuality: 0,
    _showSupportGemTypes: 'ALL',
    _showAltQualityGems: false
  };

  registerProperty(gemOptions, 'sortGemsByDPS', (value) => $currentBuild?.SetSortGemsByDPS(value.toString() === 'true'));
  registerProperty(gemOptions, 'sortGemsByDPSField', (value) => $currentBuild?.SetSortGemsByDPSField(value as string));
  registerProperty(gemOptions, 'matchGemLevelToCharacterLevel', (value) => $currentBuild?.SetMatchGemLevelToCharacterLevel(value.toString() === 'true'));
  registerProperty(gemOptions, 'defaultGemLevel', (value) => $currentBuild?.SetDefaultGemLevel(parseInt(value.toString())));
  registerProperty(gemOptions, 'defaultGemQuality', (value) => $currentBuild?.SetDefaultGemQuality(parseInt(value.toString())));
  registerProperty(gemOptions, 'showSupportGemTypes', (value) => $currentBuild?.SetShowSupportGemTypes(value as string));
  registerProperty(gemOptions, 'showAltQualityGems', (value) => $currentBuild?.SetShowAltQualityGems(value.toString() === 'true'));

  $: $currentBuild?.Skills?.then((skillData) => {
    gemOptions._sortGemsByDPS = skillData.SortGemsByDPS;
    gemOptions._sortGemsByDPSField = skillData.SortGemsByDPSField;
    gemOptions._matchGemLevelToCharacterLevel = skillData.MatchGemLevelToCharacterLevel;
    gemOptions._defaultGemLevel = skillData.DefaultGemLevel;
    gemOptions._defaultGemQuality = skillData.DefaultGemQuality;
    gemOptions._showSupportGemTypes = skillData.ShowSupportGemTypes;
    gemOptions._showAltQualityGems = skillData.ShowAltQualityGems;
  });

  const addNewSocketGroup = () => {
    $currentBuild?.AddNewSocketGroup();
    currentBuild.set($currentBuild);
  };

  const deleteSelectedSocketGroup = () => {
    $currentBuild?.DeleteSocketGroup(visualSocketGroup);
    visualSocketGroup = 0;
    currentBuild.set($currentBuild);
  };

  const deleteAllSocketGroups = () => {
    $currentBuild?.DeleteAllSocketGroups();
    $currentBuild?.AddNewSocketGroup();
    visualSocketGroup = 0;
    mainSocketGroup.set(0);
    currentBuild.set($currentBuild);
  };
</script>

{#if $currentBuild}
  <div class="p-2 px-4 w-full h-full overflow-y-auto">
    <div class="flex flex-row gap-4 max-xl:flex-wrap">
      <!-- Left Side -->
      <div class="flex flex-col min-w-[24em] gap-2">
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

          <button class="container" on:click={addNewSocketGroup}>New</button>
          <button class="container" disabled={socketGroupList.length <= 1} on:click={deleteAllSocketGroups}>Delete All</button>
          <button class="container" disabled={visualSocketGroup < 0 || socketGroupList.length <= 1} on:click={deleteSelectedSocketGroup}>Delete</button>
        </div>

        <select bind:value={visualSocketGroup} class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[19em]" size="18">
          {#each socketGroupList as group, i}
            <option value={i} on:contextmenu={(event) => onRightClickSocketGroup(i, event)}>
              {group.label}
              {!group.enabled ? ' (Disabled)' : ''}
              {$mainSocketGroup === i ? ' (Active)' : ''}
              {group.fullDPS ? ' (FullDPS)' : ''}
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
              <input type="checkbox" class="text-2xl" bind:checked={gemOptions.sortGemsByDPS} />
              <div class="container select-wrapper">
                <select class="input" bind:value={gemOptions.sortGemsByDPSField}>
                  <option value="FullDPS">Full DPS</option>
                  <option value="CombinedDPS">Combined DPS</option>
                  <option value="TotalDPS">Total DPS</option>
                  <option value="AverageDamage">Average Hit</option>
                  <option value="TotalDot">DoT DPS</option>
                  <option value="BleedDPS">Bleed DPS</option>
                  <option value="IgniteDPS">Ignite DPS</option>
                  <option value="TotalPoisonDPS">Poison DPS</option>
                </select>
              </div>
            </div>

            <span>Match gems character level:</span>
            <div class="flex flex-row gap-2">
              <input type="checkbox" class="text-2xl" bind:checked={gemOptions.matchGemLevelToCharacterLevel} />
            </div>

            <span>Default gem level:</span>
            <NumberInput min={1} max={40} bind:value={gemOptions.defaultGemLevel} />

            <span>Default gem quality:</span>
            <NumberInput min={0} max={30} bind:value={gemOptions.defaultGemQuality} />

            <span>Show support gems:</span>
            <div class="flex flex-row gap-2">
              <div class="container select-wrapper">
                <select class="input" bind:value={gemOptions.showSupportGemTypes}>
                  <option value="ALL">All</option>
                  <option value="NORMAL">Non-Awakened</option>
                  <option value="AWAKENED">Awakened</option>
                </select>
              </div>
            </div>

            <span>Show gem quality variants:</span>
            <div class="flex flex-row gap-2">
              <input type="checkbox" class="text-2xl" bind:checked={gemOptions.showAltQualityGems} />
            </div>
          </div>
        </fieldset>
      </div>

      <!-- Right Side -->
      {#if $currentGemGroup}
        <div class="flex flex-col min-w-[30em] gap-2">
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
                  clearable={false}
                  placeholder=""
                  showChevron={true}
                  listOffset={0}>
                  <div slot="selection" let:selection>
                    <SelectSelection item={selection}/>
                  </div>
                  <div slot="item" let:item>
                    <SelectItem {item}/>
                  </div>
                </Select>
              </div>

              <NumberInput min={1} max={gemGroup.GemListValue.data.MaxLevel} bind:value={gemGroup.Level} />

              <div class="container select-wrapper">
                <select class="input">
                  <option>Default</option>
                </select>
              </div>

              <NumberInput min={0} max={30} bind:value={gemGroup.Quality} />

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
