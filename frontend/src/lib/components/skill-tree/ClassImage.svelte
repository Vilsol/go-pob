<script lang="ts">
  import { T } from '@threlte/core';
  import { toCanvasCoords } from '../../skill_tree';
  import type { Tree } from '$lib/skill_tree/types';
  import { relativeScale } from '$lib/components/skill-tree/common';
  import { useTexture } from '@threlte/extras';
  import type { Texture } from 'three';

  interface Props {
    cdnBase: string;
    currentClass?: string;
    skillTree: Tree;
  }

  let { cdnBase, currentClass, skillTree }: Props = $props();

  const getImage = async (url: string): Promise<{ texture: Texture; image: HTMLImageElement }> => {
    const texture = await useTexture(url);
    return {
      texture,
      image: texture.image
    };
  };
</script>

{#if currentClass}
  {@const classIndex = skillTree.classes.findIndex((c) => c.name === currentClass)}
  {#if classIndex in skillTree.extraImages}
    {@const img = skillTree.extraImages[classIndex]}

    {#await getImage(cdnBase + '/raw/' + img.image) then t}
      {@const canvasPos = toCanvasCoords(img.x, img.y)}
      <T.Mesh position={[canvasPos.x, -canvasPos.y, 0]} scale={relativeScale({ w: t.image.width, h: t.image.height }, [0.5, 0.5, 1])}>
        <T.PlaneGeometry />
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>
    {/await}
  {/if}
{/if}
