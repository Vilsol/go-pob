<script lang="ts">
  import { Canvas } from '@threlte/core';
  import SkillTree from './SkillTree.svelte';
  import type { Tree } from '$lib/skill_tree/types';
  import * as THREE from 'three';
  import { Pane, Checkbox } from 'svelte-tweakpane-ui';

  let parentContainer = $state<HTMLElement>();

  interface Props {
    skillTree: Tree;
    skillTreeVersion: string;
  }

  let { skillTree, skillTreeVersion }: Props = $props();

  let groupsEnabled = $state(true);
  let connectionsEnabled = $state(true);
  let nodesEnabled = $state(true);
</script>

<div class="w-full h-full max-w-full max-h-full overflow-hidden relative" style="background: #080c11; touch-action: none" bind:this={parentContainer}>
  <Canvas shadows={false} toneMapping={THREE.NoToneMapping}>
    <SkillTree {skillTree} {skillTreeVersion} {parentContainer} {groupsEnabled} {connectionsEnabled} {nodesEnabled} />
  </Canvas>

  <div class="absolute top-0 right-0">
    <Pane title="Toggles" position="inline">
      <Checkbox label="Groups" bind:value={groupsEnabled} />
      <Checkbox label="Connections" bind:value={connectionsEnabled} />
      <Checkbox label="Nodes" bind:value={nodesEnabled} />
    </Pane>
  </div>
</div>
