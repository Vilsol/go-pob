<script lang="ts">
  import type { Node } from '../../skill_tree/types';
  import { allInverseSpritesheets, calculateNodePos, drawnNodes, SpritesheetType } from '../../skill_tree';
  import { onMount } from 'svelte';
  import { Container, Sprite } from 'pixi.js';

  interface Props {
    hoverPath: number[];
    hoveredNode?: Node;
    activeNodes: number[];
    parentContainer: Container;
    onClick: (node: Node) => void;
  }

  let { hoverPath, activeNodes, parentContainer, hoveredNode = $bindable(), onClick }: Props = $props();

  const container = new Container();

  let hoverSet = $derived(new Set(hoverPath));
  let activeSet = $derived(new Set(activeNodes));

  onMount(() => {
    parentContainer.addChild(container);

    drawnNodes.keys().forEach((nNodeId) => {
      const node: Node = drawnNodes.get(nNodeId)!;
      const position = calculateNodePos(node);

      if (node.isAscendancyStart) {
        const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS].AscendancyMiddle.textures.AscendancyMiddle);
        g.position = position;
        g.anchor.set(0.5, 0.5);
        container.addChild(g);
      } else if (node.isKeystone) {
        const g = new Sprite(
          allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!]
        );
        g.position = position;
        g.anchor.set(0.5, 0.5);
        container.addChild(g);

        $effect(() => {
          g.texture = allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!];
        });

        const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'KeystoneFrameAllocated' : 'KeystoneFrameUnallocated';
        const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
        g2.position = position;
        g2.anchor.set(0.5, 0.5);
        container.addChild(g2);

        $effect(() => {
          const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'KeystoneFrameAllocated' : 'KeystoneFrameUnallocated';
          g2.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
        });

        g2.on('mouseover', () => (hoveredNode = node));
        g2.on('mouseout', () => (hoveredNode = undefined));
        g2.on('mouseup', () => onClick(node));

        g2.eventMode = 'static';
        g2.cursor = 'pointer';
      } else if (node.isNotable) {
        if (node.ascendancyName) {
          const g = new Sprite(
            allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!]
          );
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            g.texture = allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!];
          });

          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'AscendancyFrameLargeAllocated' : 'AscendancyFrameLargeNormal';
          const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g2.position = position;
          g2.anchor.set(0.5, 0.5);
          container.addChild(g2);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'AscendancyFrameLargeAllocated' : 'AscendancyFrameLargeNormal';
            g2.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g2.on('mouseover', () => (hoveredNode = node));
          g2.on('mouseout', () => (hoveredNode = undefined));
          g2.on('mouseup', () => onClick(node));

          g2.eventMode = 'static';
          g2.cursor = 'pointer';
        } else {
          const g = new Sprite(
            allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!]
          );
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            g.texture = allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!];
          });

          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'NotableFrameAllocated' : 'NotableFrameUnallocated';
          const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g2.position = position;
          g2.anchor.set(0.5, 0.5);
          container.addChild(g2);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'NotableFrameAllocated' : 'NotableFrameUnallocated';
            g2.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g2.on('mouseover', () => (hoveredNode = node));
          g2.on('mouseout', () => (hoveredNode = undefined));
          g2.on('mouseup', () => onClick(node));

          g2.eventMode = 'static';
          g2.cursor = 'pointer';
        }
      } else if (node.isJewelSocket) {
        if (node.expansionJewel) {
          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'JewelSocketAltActive' : 'JewelSocketAltNormal';
          const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'JewelSocketAltActive' : 'JewelSocketAltNormal';
            g.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g.on('mouseover', () => (hoveredNode = node));
          g.on('mouseout', () => (hoveredNode = undefined));
          g.on('mouseup', () => onClick(node));

          g.eventMode = 'static';
          g.cursor = 'pointer';
        } else {
          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'JewelFrameAllocated' : 'JewelFrameUnallocated';
          const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'JewelFrameAllocated' : 'JewelFrameUnallocated';
            g.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g.on('mouseover', () => (hoveredNode = node));
          g.on('mouseout', () => (hoveredNode = undefined));
          g.on('mouseup', () => onClick(node));

          g.eventMode = 'static';
          g.cursor = 'pointer';
        }
      } else if (node.isMastery) {
        const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? node.activeIcon! : node.inactiveIcon!;
        const source = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE;
        const g = new Sprite(allInverseSpritesheets[source][name].textures[name]);
        g.position = position;
        g.anchor.set(0.5, 0.5);
        container.addChild(g);

        $effect(() => {
          const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? node.activeIcon! : node.inactiveIcon!;
          const newSource = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE;
          g.texture = allInverseSpritesheets?.[newSource]?.[newName]?.textures[newName];
        });

        g.on('mouseover', () => (hoveredNode = node));
        g.on('mouseout', () => (hoveredNode = undefined));
        g.on('mouseup', () => onClick(node));

        g.eventMode = 'static';
        g.cursor = 'pointer';
      } else {
        if (node.ascendancyName) {
          const g = new Sprite(
            allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!]
          );
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            g.texture = allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!];
          });

          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'AscendancyFrameSmallAllocated' : 'AscendancyFrameSmallNormal';
          const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g2.position = position;
          g2.anchor.set(0.5, 0.5);
          container.addChild(g2);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'AscendancyFrameSmallAllocated' : 'AscendancyFrameSmallNormal';
            g2.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g2.on('mouseover', () => (hoveredNode = node));
          g2.on('mouseout', () => (hoveredNode = undefined));
          g2.on('mouseup', () => onClick(node));

          g2.eventMode = 'static';
          g2.cursor = 'pointer';
        } else {
          if ('classStartIndex' in node) {
            return;
          }

          const g = new Sprite(
            allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!]
          );
          g.position = position;
          g.anchor.set(0.5, 0.5);
          container.addChild(g);

          $effect(() => {
            g.texture = allInverseSpritesheets[activeSet.has(nNodeId) ? SpritesheetType.ACTIVE : SpritesheetType.INACTIVE][node.icon!].textures[node.icon!];
          });

          const name = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'PSSkillFrameActive' : 'PSSkillFrame';
          const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
          g2.position = position;
          g2.anchor.set(0.5, 0.5);
          container.addChild(g2);

          $effect(() => {
            const newName = activeSet.has(nNodeId) || hoverSet.has(nNodeId) ? 'PSSkillFrameActive' : 'PSSkillFrame';
            g2.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
          });

          g2.on('mouseover', () => (hoveredNode = node));
          g2.on('mouseout', () => (hoveredNode = undefined));
          g2.on('mouseup', () => onClick(node));

          g2.eventMode = 'static';
          g2.cursor = 'pointer';
        }
      }

      return () => {
        container.destroy({
          children: true
        });
      };
    });
  });
</script>
