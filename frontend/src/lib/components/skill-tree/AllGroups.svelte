<script lang="ts">
  import { T } from '@threlte/core';
  import { PlaneGeometry } from 'three';
  import type { Tree } from '../../skill_tree/types';
  import {
    toCanvasCoords,
    drawnGroups,
    ascendancyGroupPositionOffsets,
    ascendancyGroups,
    classStartGroups,
    inverseSpritesOther,
    ascendancyStartGroups
  } from '../../skill_tree';
  import { loadSpriteTexture, relativeScale } from './common';

  interface Props {
    cdnBase: string;
    currentClass?: string;
    currentAscendancy?: string;
    skillTree: Tree;
  }

  let { cdnBase, currentClass, currentAscendancy, skillTree }: Props = $props();

  // Create geometries for different group types
  const groupGeometry = new PlaneGeometry(1, 1);
</script>

{#each Array.from(drawnGroups) as [groupId, group]}
  {@const posX = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.x) || 0) + group.x}
  {@const posY = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.y) || 0) + group.y}
  {@const canvasPos = toCanvasCoords(posX, posY)}

  {@const maxOrbit = Math.max(...group.orbits)}
  {#if groupId in classStartGroups}
    {@const isActiveClass = currentClass === skillTree.classes[classStartGroups[groupId]].name}
    {@const spriteName = isActiveClass ? `center${skillTree.classes[classStartGroups[groupId]].name.toLowerCase()}` : 'PSStartNodeBackgroundInactive'}

    {#await loadSpriteTexture(spriteName, cdnBase, inverseSpritesOther) then t}
      <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y, 0.001]} scale={relativeScale(t.sprite, [1, 1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent opacity={isActiveClass ? 1 : 0.5} />
      </T.Mesh>
    {/await}
  {:else if groupId in ascendancyGroups}
    {#if ascendancyStartGroups.has(groupId)}
      {#await loadSpriteTexture(`Classes${ascendancyGroups[groupId]}`, cdnBase, inverseSpritesOther) then t}
        <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y, 0.001]} scale={relativeScale(t.sprite, [1, 1, 1])}>
          <T.MeshBasicMaterial map={t.texture} transparent opacity={currentAscendancy === ascendancyGroups[groupId] ? 1 : 0.5} />
        </T.Mesh>
      {/await}
    {/if}
  {:else if maxOrbit === 1}
    {#await loadSpriteTexture('PSGroupBackground1', cdnBase, inverseSpritesOther) then t}
      <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y, 0.001]} scale={relativeScale(t.sprite, [1, 1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>
    {/await}
  {:else if maxOrbit === 2}
    {#await loadSpriteTexture('PSGroupBackground2', cdnBase, inverseSpritesOther) then t}
      <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y, 0.001]} scale={relativeScale(t.sprite, [1, 1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>
    {/await}
  {:else if maxOrbit === 3 || group.orbits.length > 1}
    {#await loadSpriteTexture('PSGroupBackground3', cdnBase, inverseSpritesOther) then t}
      <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y + 0.725, 0.001]} scale={relativeScale(t.sprite, [1, 1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>

      <T.Mesh geometry={groupGeometry} position={[canvasPos.x, -canvasPos.y - 0.7, 0.001]} scale={relativeScale(t.sprite, [-1, -1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>
    {/await}
  {/if}
{/each}
