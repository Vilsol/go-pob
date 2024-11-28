<script lang="ts">
  import type { Outputs } from '../custom_types';
  import { outputs, currentBuild } from '../global';
  import { displayStats } from '../display/stats';
  import type { Stat } from '../display/stats';
  import { printf } from 'fast-printf';
  import { colorCodes } from '../display/colors';
  import { base } from '$app/paths';
  import { onMount } from 'svelte';
  import { GetSkillGems } from '../cache';
  import { exposition } from '../types';
  import { writable } from 'svelte/store';
  import { syncWrap } from '../go/worker';
  import { openOverlay } from '../overlay';
  import Options from './overlays/Options.svelte';
  import Version from '$lib/components/overlays/Version.svelte';
  import { logError } from '$lib/utils';

  interface Line {
    label: string;
    value: string;
    labelColor: string;
    valueColor: string;
  }

  // TODO Warnings

  // Formats "1234.56" -> "1,234.5"
  // eslint-disable-next-line
  const formatNumSep = (str: string): string => {
    /*
      TODO formatNumSep
      return string.gsub(str, "(-?%d+%.?%d+)", function(m)
          local x, y, minus, integer, fraction = m:find("(-?)(%d+)(%.?%d*)")
            if main.showThousandsSeparators then
                integer = integer:reverse():gsub("(%d%d%d)", "%1"..main.thousandsSeparator):reverse()
                -- There will be leading separators if the number of digits are divisible by 3
                -- This checks for their presence and removes them
                -- Don't use patterns here because thousandsSeparator can be a pattern control character, and will crash if used
                if main.thousandsSeparator ~= "" then
                    local thousandsSeparator = string.find(integer, main.thousandsSeparator, 1, 2)
                    if thousandsSeparator and thousandsSeparator == 1 then
                        integer = integer:sub(2)
                    end
                end
            else
                integer = integer:reverse():gsub("(%d%d%d)", "%1"):reverse()
            end
            return minus..integer..fraction:gsub("%.", main.decimalSeparator)
        end)
     */
    return str;
  };

  const formatStat = (statData: Stat, statVal: number, overCapStatVal?: number): [string, string] => {
    const val = statVal * (((statData.pc || statData.mod) && 100) || 1) - ((statData.mod && 100) || 0);
    let color = colorCodes.NEGATIVE;
    if (statVal >= 0) {
      color = '#ffffff';
    }

    let valStr = printf('%' + (statData.fmt || 'd'), val);
    valStr = formatNumSep(valStr);

    if (overCapStatVal && overCapStatVal > 0) {
      // TODO overCapStatVal
      // valStr = valStr .. "^x808080" .. " (+" .. s_format("%d", overCapStatVal) .. "%)"
    }

    return [valStr, color];
  };

  const prepareOutput = (out?: Outputs): Line[][] => {
    if (!out || !out.Output) {
      return [];
    }

    const lines: Line[][] = [];

    for (const statGroup of displayStats) {
      const group: Line[] = [];
      for (const stat of statGroup) {
        // TODO Handle labelStat
        const statName = stat.stat;
        if (!statName || !(statName in out.Output)) {
          continue;
        }

        const value = out.Output[statName];
        if (stat.condFunc) {
          if (!stat.condFunc(value, out.Output)) {
            continue;
          }
        }

        if (statName === 'SkillDPS') {
          /*
            TODO Special SkillDPS handling
						labelColor = colorCodes.CUSTOM
						table.sort(actor.output.SkillDPS, function(a,b) return (a.dps * a.count) > (b.dps * b.count) end)
						for _, skillData in ipairs(actor.output.SkillDPS) do
							local triggerStr = ""
							if skillData.trigger and skillData.trigger ~= "" then
								triggerStr = colorCodes.WARNING.." ("..skillData.trigger..")"..labelColor
							end
							local lhsString = labelColor..skillData.name..triggerStr..":"
							if skillData.count >= 2 then
								lhsString = labelColor..tostring(skillData.count).."x "..skillData.name..triggerStr..":"
							end
							t_insert(statBoxList, {
								height = 16,
								lhsString,
								self:FormatStat({fmt = "1.f"}, skillData.dps * skillData.count, overCapStatVal),
							})
							if skillData.skillPart then
								t_insert(statBoxList, {
									height = 14,
									align = "CENTER_X", x = 140,
									"^8"..skillData.skillPart,
								})
							end
							if skillData.source then
								t_insert(statBoxList, {
									height = 14,
									align = "CENTER_X", x = 140,
									colorCodes.WARNING.."from " ..skillData.source,
								})
							end
						end
           */
        }

        if (stat.flag && out.SkillFlags) {
          if (out.SkillFlags[stat.flag] !== true) {
            continue;
          }
        }

        if (stat.warnFunc) {
          // TODO Warnings
        }

        const formatted = formatStat(stat, value, stat.overCapStat ? out.Output[stat.overCapStat] : undefined);
        group.push({
          label: stat.label,
          labelColor: stat.color || '#ffffff',
          value: formatted[0],
          valueColor: formatted[1]
        });
      }

      if (group.length > 0) {
        lines.push(group);
      }
    }

    return lines;
  };

  const skillGemMapping = $state<Record<string, exposition.SkillGem>>({});
  onMount(() => {
    GetSkillGems()
      .then((all) => {
        all.forEach((g) => {
          skillGemMapping[g.ID] = g;
        });
      })
      .catch(logError);
  });

  let activeSkillSet = $state(1);
  $effect(() => {
    $currentBuild?.Skills?.ActiveSkillSet?.then((v) => {
      if (v < 1) {
        activeSkillSet = 1;
        return;
      }
      activeSkillSet = v;
    }).catch(logError);
  });
  $effect(() => {
    console.log('ACTIVE:', activeSkillSet);
  });

  let socketGroupList = $state<string[]>([]);
  $effect(() => {
    $currentBuild?.Skills?.SkillSets?.[activeSkillSet - 1]?.Skills?.then(async (skills: unknown[]) => {
      const finalList: string[] = [];
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

        if (label) {
          finalList.push(label);
        }
      }
      socketGroupList = finalList;
    }).catch(logError);
  });

  const mainSocketGroup = writable(-1);
  $effect(() => {
    $currentBuild?.Build?.MainSocketGroup?.then((v) => mainSocketGroup.set(v - 1)).catch(logError);
  });
  mainSocketGroup.subscribe((value) => {
    if (value >= 0) {
      syncWrap?.SetMainSocketGroup(value + 1).catch(logError);
    }
    currentBuild.set($currentBuild);
  });

  const openOptions = () => {
    openOverlay({
      component: Options,
      props: {}
    });
  };

  const openVersion = () => {
    openOverlay({
      component: Version,
      props: {}
    });
  };

  let collapsed = $state(false);
</script>

{#if collapsed}
  <div class="h-full flex flex-col full-page relative">
    <button class="absolute -right-3 top-1/2 cursor-pointer font-bold" onclick={() => (collapsed = false)}>&gt;</button>
  </div>
{:else}
  <div class="w-[25vw] min-w-[17em] max-w-[25em] h-full border-r-2 border-white flex flex-col bg-neutral-900 full-page relative">
    <div class="flex flex-col gap-3 border-b-2 border-white flex-1 p-2 sidebar-stat-wrapper">
      <div class="flex flex-col gap-2">
        <div class="flex flex-row gap-1">
          <a href="{base}/import" class="container min-w-fit flex-1 text-center">Import/Export Build</a>
          <a href="{base}/notes" class="container min-w-fit flex-1 text-center">Notes</a>
          <a href="{base}/configuration" class="container min-w-fit flex-1 text-center">Config</a>
        </div>
        <div class="flex flex-row gap-1">
          <a href="{base}/tree" class="container min-w-fit flex-1 text-center">Tree</a>
          <a href="{base}/skills" class="container min-w-fit flex-1 text-center">Skills</a>
          <a href="{base}/items" class="container min-w-fit flex-1 text-center">Items</a>
          <a href="{base}/calcs" class="container min-w-fit flex-1 text-center">Calcs</a>
        </div>
      </div>

      <div class="flex flex-col gap-1">
        <div>Main Skill:</div>
        <div class="container select-wrapper min-w-full">
          <select class="input w-full" bind:value={$mainSocketGroup}>
            {#if socketGroupList.length === 0}
              <option>&lt;No skills added yet&gt;</option>
            {:else}
              {#each socketGroupList as group, i}
                <option value={i}>{group}</option>
              {/each}
            {/if}
          </select>
        </div>
      </div>

      <!-- TODO Skill Parts -->

      <div class="container min-w-full overflow-y-auto flex-1 flex flex-col gap-2.5 overflow-y-scroll">
        {#each prepareOutput($outputs) as outputGroup}
          <div class="side-by-side-sidebar">
            {#each outputGroup as stat}
              <div style="color: {stat.labelColor}">{stat.label}:</div>
              <div style="color: {stat.valueColor}">{stat.value}</div>
            {/each}
          </div>
        {/each}
      </div>
    </div>

    <div class="flex flex-row p-2">
      <div class="flex flex-col flex-1 gap-2">
        <button class="container min-w-full flex-1" onclick={openOptions}>Options</button>
        <button class="container min-w-full flex-1">About</button>
      </div>
      <div class="flex flex-col flex-1 items-center">
        <span class="flex-1 flex place-items-center">go-pob</span>
        <button class="flex-1 flex place-items-center cursor-pointer" onclick={openVersion}>
          {#await syncWrap?.BuildInfo() then buildInfo}
            Version: {buildInfo?.Main?.Version}
          {/await}
        </button>
      </div>
    </div>

    <button class="absolute -right-3.5 top-1/2 cursor-pointer font-bold" onclick={() => (collapsed = true)}>&lt;</button>
  </div>
{/if}

<style lang="postcss">
  .sidebar-stat-wrapper {
    max-height: calc(100% - 4.75em);
  }

  .side-by-side-sidebar {
    @apply grid gap-x-1 gap-y-0 items-center;
    grid-template-columns: 65% 35%;

    & div {
      @apply leading-snug;
    }

    & > *:nth-child(odd) {
      @apply text-right;
    }

    & > *:nth-child(even) {
      @apply text-left justify-self-start self-center;
    }
  }
</style>
