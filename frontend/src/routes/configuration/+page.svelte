<script lang="ts">
  import NumberInput from '$lib/components/NumberInput.svelte';
  import Select from 'svelte-select';
  import SelectItem from '$lib/components/SelectItem.svelte';
  import SelectSelection from '$lib/components/SelectSelection.svelte';
  import type { ConfigSection } from '$lib/display/configurations';
  import { configurations } from '$lib/display/configurations';
  import { currentBuild } from '$lib/global';
  import type { pob } from '$lib/types';
  import { syncWrap } from '$lib/go/worker';
  import type { Remote } from 'comlink';
  import ColoredText from '$lib/components/common/ColoredText.svelte';
  import { logError } from '$lib/utils';

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const valueWatchers: Record<string, any> = {};

  // eslint-disable-next-line
  const filterSections = (list: ConfigSection[], _: Remote<pob.PathOfBuilding>): ConfigSection[] =>
    list
      .map((s) => ({
        ...s,
        variables: s.variables.filter((varData) => {
          if (varData.ifOption !== undefined) {
            // TODO self.input[varData.ifOption]
            return false;
          } else if (varData.ifCond !== undefined || varData.ifMinionCond !== undefined || varData.ifEnemyCond !== undefined) {
            /*
          TODO ifCond, ifMinionCond, ifEnemyCond
					local mainEnv = self.build.calcsTab.mainEnv
					if self.input[varData.var] then
						if varData.implyCondList then
							for _, implyCond in ipairs(varData.implyCondList) do
								if (implyCond and mainEnv.conditionsUsed[implyCond]) then
									return true
								end
							end
						end
						if (varData.implyCond and mainEnv.conditionsUsed[varData.implyCond]) or
						   (varData.implyMinionCond and mainEnv.minionConditionsUsed[varData.implyMinionCond]) or
						   (varData.implyEnemyCond and mainEnv.enemyConditionsUsed[varData.implyEnemyCond]) then
							return true
						end
					end
					if varData.ifCond then
						return mainEnv.conditionsUsed[varData.ifCond]
					elseif varData.ifMinionCond then
						return mainEnv.minionConditionsUsed[varData.ifMinionCond]
					else
						return mainEnv.enemyConditionsUsed[varData.ifEnemyCond]
					end
         */
            return false;
          } else if (varData.ifMult !== undefined || varData.ifEnemyMult !== undefined) {
            /*
          TODO ifMult, ifEnemyMult
					local mainEnv = self.build.calcsTab.mainEnv
					if self.input[varData.var] then
						if varData.implyCondList then
							for _, implyCond in ipairs(varData.implyCondList) do
								if (implyCond and mainEnv.conditionsUsed[implyCond]) then
									return true
								end
							end
						end
						if (varData.implyCond and mainEnv.conditionsUsed[varData.implyCond]) or
						   (varData.implyMinionCond and mainEnv.minionConditionsUsed[varData.implyMinionCond]) or
						   (varData.implyEnemyCond and mainEnv.enemyConditionsUsed[varData.implyEnemyCond]) then
							return true
						end
					end
					if varData.ifMult then
						return mainEnv.multipliersUsed[varData.ifMult]
					else
						return mainEnv.enemyMultipliersUsed[varData.ifEnemyMult]
					end
         */
            return false;
          } else if (varData.ifFlag) {
            /*
          TODO ifFlag
					local skillModList = self.build.calcsTab.mainEnv.player.mainSkill.skillModList
					local skillFlags = self.build.calcsTab.mainEnv.player.mainSkill.skillFlags
					-- Check both the skill mods for flags and flags that are set via calcPerform
					return skillFlags[varData.ifFlag] or skillModList:Flag(nil, varData.ifFlag)
         */
            return false;
          } else if (varData.ifSkill !== undefined || varData.ifSkillList !== undefined) {
            /*
          TODO ifSkill, ifSkillList
					if varData.ifSkillList then
						for _, skillName in ipairs(varData.ifSkillList) do
							if self.build.calcsTab.mainEnv.skillsUsed[skillName] then
								return true
							end
						end
					else
						return self.build.calcsTab.mainEnv.skillsUsed[varData.ifSkill]
					end
         */
            return false;
          } else if (varData.ifSkillFlag !== undefined) {
            /*
          TODO ifSkillFlag
          for _, activeSkill in ipairs(self.build.calcsTab.mainEnv.player.activeSkillList) do
            if activeSkill.skillFlags[varData.ifSkillFlag] then
              return true
            end
          end
					return false
         */
            return false;
          }

          if (!(varData.var in valueWatchers)) {
            let defaultState: unknown = varData.defaultState;
            if (varData.type === 'list') {
              defaultState = varData.list[0];
            }

            const privateName = '_' + varData.var;
            valueWatchers[privateName] = defaultState;

            // TODO Clean up this mess
            Object.defineProperty(valueWatchers, varData.var, {
              get() {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-return,@typescript-eslint/no-unsafe-member-access
                return this[privateName];
              },
              set(value) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                if (this[privateName] === value) {
                  return;
                }

                if (varData.type === 'list') {
                  // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                  if (this[privateName] === undefined || this[privateName].value !== value.value) {
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument,@typescript-eslint/no-unsafe-member-access
                    syncWrap?.SetConfigOption(varData.var, value.value).catch(logError);
                  }
                } else {
                  // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                  syncWrap?.SetConfigOption(varData.var, value).catch(logError);
                }

                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                this[privateName] = value;
              }
            });

            syncWrap
              ?.GetConfigOption(varData.var)
              .then((val) => {
                if (varData.type === 'list') {
                  const elem = varData.list.find((l) => l.value === val);
                  if (elem !== undefined) {
                    valueWatchers[varData.var] = elem;
                  }
                } else {
                  valueWatchers[varData.var] = val;
                }
              })
              .catch(logError);
          }

          return true;
        })
      }))
      .filter((s) => s.variables.length > 0);

  let hoveredItem: ConfigSection['variables'][number] | undefined = $state(undefined);

  let tooltipStyle = $state('');
  let tooltipElement = $state<HTMLElement>();

  const moveEvent = (event: MouseEvent) => {
    if (hoveredItem && hoveredItem.tooltip !== undefined) {
      let left = event.x;
      if (event.x > window.innerWidth / 2) {
        left -= (tooltipElement?.clientWidth || 200) + 15;
      } else {
        left += 15;
      }

      let top = event.y;
      if (event.y > window.innerHeight / 2) {
        top -= (tooltipElement?.clientHeight || 100) + 15;
      }

      tooltipStyle = `top: ${top}px; left: ${left}px`;
    } else {
      tooltipStyle = 'display: none';
    }
  };

  const sections = $derived($currentBuild ? filterSections(configurations, $currentBuild) : []);
</script>

<svelte:window onmousemove={moveEvent} />

<div class="p-2 px-4 h-full flex flex-col flex-wrap gap-4 w-full overflow-x-auto">
  {#each sections as section}
    <fieldset class="border border-white bg-neutral-900 p-2">
      <legend class="container">{section.name}</legend>
      <div class="side-by-side-max-content">
        {#each section.variables as v}
          <div><label for={v.var}><ColoredText text={v.label} /></label></div>
          <div
            class="w-full"
            onmouseover={() => (hoveredItem = v)}
            onfocus={() => (hoveredItem = v)}
            onmouseleave={() => (hoveredItem = undefined)}
            role="contentinfo">
            {#if v.type === 'list'}
              <div class="themed min-w-full">
                <Select items={v.list} bind:value={valueWatchers[v.var]} clearable={false} placeholder="" showChevron={true} listOffset={0} id={v.var}>
                  <!-- TODO CONVERT TO SVELTE 5 -->
                  <div slot="selection" let:selection>
                    <SelectSelection item={selection} />
                  </div>
                  <div slot="item" let:item>
                    <SelectItem {item} />
                  </div>
                </Select>
              </div>
            {:else if v.type === 'count' || v.type === 'integer' || v.type === 'countAllowZero'}
              <NumberInput min={v.type === 'countAllowZero' ? 0 : 1} fullWidth={true} id={v.var} bind:value={valueWatchers[v.var]} />
            {:else if v.type === 'check'}
              <input type="checkbox" class="text-2xl" id={v.var} bind:checked={valueWatchers[v.var]} />
            {/if}
          </div>
        {/each}
      </div>
    </fieldset>
  {/each}
</div>

<div class="absolute pointer-events-none border-amber-800 border-4 p-2 bg-black" style={tooltipStyle} bind:this={tooltipElement}>
  {#if hoveredItem !== undefined && hoveredItem.tooltip !== undefined}
    <ColoredText text={hoveredItem.tooltip.trim().replaceAll('\n', '<br/>')} />
  {/if}
</div>
