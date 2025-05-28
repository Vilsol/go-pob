<script lang="ts">
  import Input from '$lib/components/Input.svelte';
  import { colorCodes } from '$lib/display/colors';
  import ItemFrame from '$lib/components/game/ItemFrame.svelte';
  import { syncWrap } from '$lib/go/worker';

  const baseSlots = [
    'Weapon 1',
    'Weapon 2',
    'Helmet',
    'Body Armour',
    'Gloves',
    'Boots',
    'Amulet',
    'Ring 1',
    'Ring 2',
    'Belt',
    'Flask 1',
    'Flask 2',
    'Flask 3',
    'Flask 4',
    'Flask 5'
  ];
</script>

<div class="p-2 px-4 w-full h-full overflow-y-auto">
  {#await syncWrap.GetItems() then items}
    {JSON.stringify(items)}
  {/await}

  <div class="flex flex-row gap-4">
    <div class="side-by-side-max-content min-w-[25em] max-w-[25em] h-fit">
      <span>Item set:</span>
      <div class="flex flex-row gap-1">
        <div class="container select-wrapper">
          <select class="input" disabled>
            <option>Default</option>
          </select>
        </div>
        <button class="container">Manage...</button>
      </div>

      <span><!-- Placeholder --></span>
      <div class="flex flex-row justify-between items-center gap-4 min-w-full">
        <span>Equipped items:</span>
        <div class="flex flex-row items-center gap-1">
          <span>Weapon set:</span>
          <button class="container">I</button>
          <button class="container">II</button>
        </div>
      </div>

      {#each baseSlots as slot}
        <div class="flex flex-row justify-end items-center gap-1">
          <span>{slot}:</span>
          {#if slot.startsWith('Flask')}
            <input type="checkbox" />
          {/if}
        </div>
        <div class="container select-wrapper min-w-full">
          <select class="input" disabled>
            <option>None</option>
          </select>
        </div>
      {/each}
    </div>

    <div class="min-w-[25em] max-w-[25em] flex flex-col gap-2 h-fit">
      <div class="flex flex-row items-center gap-1">
        <span>All items:</span>
        <button class="container">Sort</button>
        <button class="container" disabled>Delete Unused</button>
        <button class="container" disabled>Delete All</button>
        <button class="container" disabled>Delete</button>
      </div>

      <div>
        <select class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[19em]" size="18">
          <!-- TODO Tip about usage -->
        </select>
      </div>

      <div class="flex flex-row gap-1">
        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any Slot</option>
            <!-- TODO All Slots -->
          </select>
        </div>

        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any Type</option>
            <!-- TODO All Types -->
          </select>
        </div>
      </div>

      <div class="flex flex-row gap-1">
        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Sort by Name</option>
            <!-- TODO All Sort options -->
          </select>
        </div>

        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any league</option>
            <!-- TODO All Leagues -->
          </select>
        </div>
      </div>

      <div class="flex flex-row gap-1">
        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any Requirements</option>
            <!-- TODO All requirement options -->
          </select>
        </div>
      </div>

      <div class="flex flex-row gap-1">
        <div class="flex-1">
          <Input prefix="Search:" fullWidth={true} />
        </div>

        <div class="container select-wrapper min-w-fit">
          <select class="input">
            <option>Anywhere</option>
            <!-- TODO All Leagues -->
          </select>
        </div>
      </div>

      <div>
        <select class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[19em]" size="12">
          <!-- TODO Fill with unique items -->
        </select>
      </div>

      <div class="flex flex-row gap-1">
        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any Slot</option>
            <!-- TODO All Slots -->
          </select>
        </div>

        <div class="container select-wrapper flex-1">
          <select class="input">
            <option>Any Type</option>
            <!-- TODO All Types -->
          </select>
        </div>
      </div>

      <div class="flex flex-row gap-1">
        <div class="flex-1">
          <Input prefix="Search:" fullWidth={true} />
        </div>

        <div class="container select-wrapper min-w-fit">
          <select class="input">
            <option>Anywhere</option>
            <!-- TODO All Leagues -->
          </select>
        </div>
      </div>

      <div>
        <select class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[19em]" size="12">
          <!-- TODO Fill with rare items -->
        </select>
      </div>
    </div>

    <div class="min-w-[25em] max-w-[25em] flex flex-col gap-2 h-fit">
      <div class="flex flex-row gap-1 w-full">
        <button class="container flex-1">Craft Item...</button>
        <button class="container flex-1">Create custom...</button>
      </div>

      <div>
        Double-click an item from one of the lists,<br />
        or copy and paste an item from in game<br />
        (hover over the item and Ctrl+C) to view or edit<br />
        the item and add it to your build. You can<br />
        also clone an item within Path of Building by<br />
        copying and pasting it with Ctrl+C and Ctrl+V.<br />
        <br />
        You can Control + Click an item to equip it, or<br />
        drag it onto the slot. This will also add it to<br />
        your build if it's from the unique/template list.<br />
        If there's 2 slots an item can go in,<br />
        holding Shift will put it in the second.<br />
      </div>

      <div class="flex flex-row justify-between items-center">
        <span>Shared items:</span>
        <button class="container" disabled>Delete</button>
      </div>

      <div>
        <select class="bg-black border w-full border-neutral-400 flex-1 select-many max-h-[19em]" size="18">
          <!-- TODO Tip about shared items -->
        </select>
      </div>
    </div>
  </div>

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
</div>
