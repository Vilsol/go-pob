<script lang="ts">
  import NumberInput from './NumberInput.svelte';
  import { currentBuild } from '$lib/global';
  import { writable } from 'svelte/store';
  import { syncWrap } from '../go/worker';

  let updatingCurrentClass = true;
  const currentClass = new writable<string | undefined>();
  $: $currentBuild?.Build?.ClassName.then((value) => {
    updatingCurrentClass = true;
    currentClass.set(value);
    updatingCurrentClass = false;
  });
  currentClass.subscribe((value) => {
    if (updatingCurrentClass || !value) {
      return;
    }
    syncWrap?.SetClass(value);
    syncWrap?.SetAscendancy('None');
    currentBuild.set($currentBuild);
  });

  let updatingCurrentAscendancy = true;
  const currentAscendancy = new writable<string | undefined>();
  $: $currentBuild?.Build?.AscendClassName.then((value) => {
    updatingCurrentAscendancy = true;
    currentAscendancy.set(value);
    updatingCurrentAscendancy = false;
  });
  currentAscendancy.subscribe((value) => {
    if (updatingCurrentAscendancy || !value) {
      return;
    }
    syncWrap?.SetAscendancy(value);
    currentBuild.set($currentBuild);
  });

  let updatingCurrentLevel = true;
  const currentLevel = new writable<number>(1);
  $: $currentBuild?.Build?.Level.then((value) => {
    updatingCurrentLevel = true;
    currentLevel.set(value);
    updatingCurrentLevel = false;
  });
  currentLevel.subscribe((value) => {
    if (updatingCurrentLevel) {
      return;
    }
    syncWrap?.SetLevel(value);
    currentBuild.set($currentBuild);
  });
</script>

<div class="flex flex-row w-screen border-b-2 border-white bg-neutral-800 min-h-[3em]">
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
    <NumberInput prefix="Level:" min={1} max={100} bind:value={$currentLevel} />

    <div class="container select-wrapper">
      <select class="input" bind:value={$currentClass}>
        <option value="Duelist">Duelist</option>
        <option value="Marauder">Marauder</option>
        <option value="Ranger">Ranger</option>
        <option value="Scion">Scion</option>
        <option value="Shadow">Shadow</option>
        <option value="Templar">Templar</option>
        <option value="Witch">Witch</option>
      </select>
    </div>

    <div class="container select-wrapper">
      <select class="input" bind:value={$currentAscendancy}>
        <option value="None">None</option>
        {#if $currentClass === 'Scion'}
          <option value="Ascendant">Ascendant</option>
        {:else if $currentClass === 'Marauder'}
          <option value="Juggernaut">Juggernaut</option>
          <option value="Berserker">Berserker</option>
          <option value="Chieftain">Chieftain</option>
        {:else if $currentClass === 'Ranger'}
          <option value="Raider">Raider</option>
          <option value="Deadeye">Deadeye</option>
          <option value="Pathfinder">Pathfinder</option>
        {:else if $currentClass === 'Witch'}
          <option value="Occultist">Occultist</option>
          <option value="Elementalist">Elementalist</option>
          <option value="Necromancer">Necromancer</option>
        {:else if $currentClass === 'Duelist'}
          <option value="Slayer">Slayer</option>
          <option value="Gladiator">Gladiator</option>
          <option value="Champion">Champion</option>
        {:else if $currentClass === 'Templar'}
          <option value="Inquisitor">Inquisitor</option>
          <option value="Hierophant">Hierophant</option>
          <option value="Guardian">Guardian</option>
        {:else if $currentClass === 'Shadow'}
          <option value="Assassin">Assassin</option>
          <option value="Trickster">Trickster</option>
          <option value="Saboteur">Saboteur</option>
        {/if}
      </select>
    </div>
  </div>
</div>
