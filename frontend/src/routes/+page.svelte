<script>
  import ItemFrame from '$lib/components/game/ItemFrame.svelte';
  import { colorCodes } from '$lib/display/colors';
</script>

<h1 class="text-3xl font-bold">TODO</h1>
<br />
Here are some sample item frames.

<div class="p-10 flex flex-wrap gap-10">
  <ItemFrame
    sections={[
      {
        large: true,
        text: ['Hallowed Hybrid Flask']
      },
      {
        color: colorCodes.GRAY,
        text: [
          `Recovers ^${colorCodes.WHITE}1740^# Life over ^${colorCodes.WHITE}5^# Seconds`,
          `Recovers ^${colorCodes.WHITE}480^# Mana over ^${colorCodes.WHITE}5^# Seconds`,
          `Consumes ^${colorCodes.WHITE}20^# of ^${colorCodes.WHITE}40^# Charges on use`
        ]
      },
      {
        color: colorCodes.GRAY,
        text: [`Requires Level ^${colorCodes.WHITE}60`]
      }
    ]} />

  <ItemFrame
    color={colorCodes.MAGIC}
    sections={[
      {
        large: true,
        color: colorCodes.MAGIC,
        text: ['Eternal Life Flask']
      },
      {
        color: colorCodes.GRAY,
        text: [
          `Recovers ^${colorCodes.WHITE}2080^# Life over ^${colorCodes.WHITE}2^# Seconds`,
          `Consumes ^${colorCodes.WHITE}15^# of ^${colorCodes.WHITE}45^# Charges on use`
        ]
      },
      {
        color: colorCodes.GRAY,
        text: [`Requires Level ^${colorCodes.WHITE}65`]
      }
    ]} />

  <ItemFrame
    color={colorCodes.RARE}
    sections={[
      {
        large: true,
        color: colorCodes.RARE,
        text: ['Hate Caress', 'Dragonscale Gauntlets']
      },
      {
        color: colorCodes.GRAY,
        text: [
          `Armour: ^${colorCodes.WHITE}126`,
          `Evasion Rating: ^${colorCodes.WHITE}155`,
          `Sockets: ^${colorCodes.DEXTERITY}G^#=^${colorCodes.INTELLIGENCE}B^#=^${colorCodes.STRENGTH}R^#=^${colorCodes.STRENGTH}R`
        ]
      },
      {
        color: colorCodes.GRAY,
        text: [`Requires Level ^${colorCodes.WHITE}68^#, ^${colorCodes.WHITE}115^# Str, ^${colorCodes.WHITE}108^# Dex, ^${colorCodes.WHITE}146^# Int`]
      },
      {
        color: colorCodes.MAGIC,
        text: [
          `+29 to Evasion Rating`,
          `+53 to maximum Life`,
          `9% increased Rarity of Items found`,
          `+45% to Fire Resistance`,
          `+48% to Lightning Resistance`,
          `Gain 11 Life per Enemy Killed`
        ]
      }
    ]} />

  <ItemFrame
    color={colorCodes.UNIQUE}
    sections={[
      {
        large: true,
        color: colorCodes.UNIQUE,
        text: ['Beacon of Madness', 'Two-Toned Boots']
      },
      {
        color: colorCodes.GRAY,
        text: [
          `Armour: ^${colorCodes.WHITE}143`,
          `Evasion Rating: ^${colorCodes.WHITE}30`,
          `Sockets: ^${colorCodes.INTELLIGENCE}B^#=^${colorCodes.INTELLIGENCE}B^# ^${colorCodes.STRENGTH}R`
        ]
      },
      {
        color: colorCodes.GRAY,
        text: [`Requires Level ^${colorCodes.WHITE}70^#, ^${colorCodes.WHITE}62^# Str, ^${colorCodes.WHITE}62^# Int`]
      },
      {
        color: colorCodes.MAGIC,
        text: [`+9% to Fire and Lightning Resistances`]
      },
      {
        color: colorCodes.MAGIC,
        text: [
          `Grants Level 1 Embrace Madness Skill`,
          `30% increased Movement Speed`,
          `26% increased Effect of Non-Damaging Ailments`,
          `You have Igniting, Chilling and Shocking Conflux while affected by Glorious Madness`,
          `Immune to Elemental Ailments while affected by Glorious Madness`
        ]
      },
      {
        color: colorCodes.UNIQUE,
        italic: true,
        text: [`Nothing spreads as quickly as an idea.`]
      }
    ]} />
</div>
