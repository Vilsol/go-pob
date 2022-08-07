<script lang="ts">
  import type { Outputs } from "../custom_types";
  import { outputs } from "../global";
  import { displayStats } from "../display/stats";
  import type { Stat } from "../display/stats";
  import { printf } from "fast-printf";
  import { colorCodes } from "../display/colors";

  interface Line {
    label: string;
    value: string;
    labelColor: string;
    valueColor: string;
  }

  // TODO Warnings

  // Formats "1234.56" -> "1,234.5"
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
      color = "#ffffff";
    }

    let valStr = printf("%" + (statData.fmt || "d"), val);
    valStr = formatNumSep(valStr);

    /*
      TODO overCapStatVal
      if overCapStatVal and overCapStatVal > 0 then
        valStr = valStr .. "^x808080" .. " (+" .. s_format("%d", overCapStatVal) .. "%)"
      end
     */

    return [valStr, color];
  };

  const prepareOutput = (out: Outputs): Line[][] => {
    if (!out) {
      return [];
    }

    const lines: Line[][] = [];
    displayStats.forEach(statGroup => {
      const group: Line[] = [];

      statGroup.forEach(stat => {
        // TODO Handle labelStat
        const statName = stat.stat;
        if (!(statName in out.Output)) {
          return;
        }

        const value = out.Output[statName];
        if (stat.condFunc) {
          if (!stat.condFunc(value, out.Output)) {
            return;
          }
        }

        if (statName === "SkillDPS") {
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

        if (stat.flag) {
          if (out.SkillFlags[stat.flag] !== true) {
            return;
          }
        }

        if (stat.warnFunc) {
          // TODO Warnings
        }

        const formatted = formatStat(stat, value, stat.overCapStat ? out.Output[stat.overCapStat] : undefined);
        group.push({
          label: stat.label,
          labelColor: stat.color || "#ffffff",
          value: formatted[0],
          valueColor: formatted[1]
        });
      });

      if (group.length > 0) {
        lines.push(group);
      }
    });

    return lines;
  };
</script>

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
      {#each prepareOutput($outputs) as outputGroup}
        <div class="side-by-side-3">
          {#each outputGroup as stat}
            <div style="color: {stat.labelColor}">{stat.label}:</div>
            <div style="color: {stat.valueColor}">{stat.value}</div>
          {/each}
        </div>
      {/each}
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
