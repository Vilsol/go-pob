<script lang="ts">
  import type { Tree } from '../../skill_tree/types';
  import {
    toCanvasCoords,
    drawnGroups,
    ascendancyGroupPositionOffsets,
    ascendancyGroups,
    classStartGroups,
    ascendancyStartGroups,
    allInverseSpritesheets,
    SpritesheetType
  } from '../../skill_tree';
  import { Container, Graphics, Sprite } from 'pixi.js';
  import { onMount } from 'svelte';

  interface Props {
    currentClass?: string;
    currentAscendancy?: string;
    skillTree: Tree;
    parentContainer: Container;
  }

  let { currentClass, skillTree, parentContainer, currentAscendancy }: Props = $props();

  const container = new Container();

  onMount(() => {
    parentContainer.addChild(container);

    drawnGroups.forEach((group, groupId) => {
      const posX = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.x) || 0) + group.x;
      const posY = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.y) || 0) + group.y;
      const canvasPos = toCanvasCoords(posX, posY);

      const maxOrbit = Math.max(...group.orbits);

      if (groupId in classStartGroups) {
        const active = currentClass === skillTree.classes[classStartGroups[groupId]].name;
        const name = active ? 'center' + skillTree.classes[classStartGroups[groupId]].name.toLowerCase() : 'PSStartNodeBackgroundInactive';

        const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS][name].textures[name]);
        g.position = canvasPos;
        g.anchor.set(0.5, 0.5);
        g.alpha = active ? 1 : 0.5;
        container.addChild(g);

        $effect(() => {
          const newActive = currentClass === skillTree.classes[classStartGroups[groupId]].name;
          const newName = newActive ? 'center' + skillTree.classes[classStartGroups[groupId]].name.toLowerCase() : 'PSStartNodeBackgroundInactive';
          g.alpha = newActive ? 1 : 0.5;
          g.texture = allInverseSpritesheets[SpritesheetType.OTHERS][newName].textures[newName];
        });
      } else if (groupId in ascendancyGroups) {
        if (ascendancyStartGroups.has(groupId)) {
          const name = 'Classes' + ascendancyGroups[groupId];
          const sheet = allInverseSpritesheets[SpritesheetType.OTHERS][name];

          const g = new Sprite(sheet.textures[name]);
          g.position.set(canvasPos.x, canvasPos.y);
          g.anchor.set(0.5, 0.5);

          let mask = new Graphics().circle(0, 0, sheet.data.frames[name].frame.w / 2).fill(0xffffff);

          $effect(() => {
            g.alpha = currentAscendancy === ascendancyGroups[groupId] ? 1 : 0.5;
            g.updateCacheTexture();
          });

          g.mask = mask;
          g.addChild(mask);

          g.cacheAsTexture(true);

          container.addChild(g);
        }
      } else if (maxOrbit == 1) {
        const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS].PSGroupBackground1.textures.PSGroupBackground1);
        g.position = canvasPos;
        g.anchor.set(0.5, 0.5);
        container.addChild(g);
      } else if (maxOrbit == 2) {
        const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS].PSGroupBackground2.textures.PSGroupBackground2);
        g.position = canvasPos;
        g.anchor.set(0.5, 0.5);
        container.addChild(g);
      } else if (maxOrbit == 3 || group.orbits.length > 1) {
        const g = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS].PSGroupBackground3.textures.PSGroupBackground3);
        g.position = canvasPos;
        g.anchor.set(0.5, 1);
        container.addChild(g);

        const g2 = new Sprite(allInverseSpritesheets[SpritesheetType.OTHERS].PSGroupBackground3.textures.PSGroupBackground3);
        g2.position = canvasPos;
        g2.scale = -1;
        g2.anchor.set(0.5, 1);
        container.addChild(g2);
      }
    });

    return () => {
      container.destroy({
        children: true
      });
    };
  });
</script>
