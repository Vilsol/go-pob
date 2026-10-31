import type { Tree2 } from './scene2';

const node = (id: number, x: number, y: number, extra: Partial<Tree2['nodes'][number]> = {}): Tree2['nodes'][number] => ({
  id,
  x,
  y,
  angle: 0,
  group: 1,
  orbit: 0,
  type: 'Normal',
  name: `n${id}`,
  stats: [],
  connections: [],
  ...extra
});

// a group-orbit arc (2-3), an orbit-radius arc (3-4), a line (4-5), a class start, an OnlyImage, an ascendancy start
// and a radius socket with data.jewelRadius's first five radii (Data.lua:668-675, times PassiveTreeJewelDistanceMultiplier)
export const tinyTree = (): Tree2 => ({
  orbitRadii: [0, 82, 162, 335],
  groups: [
    { id: 1, x: 0, y: 0 },
    { id: 2, x: 500, y: 0 },
    { id: 3, x: 1000, y: 0 }
  ],
  minX: -1000,
  minY: -1000,
  maxX: 1000,
  maxY: 1000,
  nodes: [
    node(1, 0, 0, { type: 'ClassStart', classesStart: ['Witch'], connections: [{ id: 2, orbit: 0 }] }),
    node(2, 0, -82, { orbit: 1, angle: 0, connections: [{ id: 3, orbit: 0 }] }),
    node(3, 82, 0, { orbit: 1, angle: Math.PI / 2, connections: [{ id: 4, orbit: 3 }] }),
    node(4, 500, 0, { group: 2, type: 'Notable', name: 'Notable Four', stats: ['10% increased Life'], connections: [{ id: 5, orbit: 0 }] }),
    node(5, 1000, 0, { group: 3, isAttribute: true, name: 'Attribute' }),
    node(6, 500, 300, { group: 2, type: 'OnlyImage' }),
    node(7, 0, 600, { group: 4, type: 'AscendClassStart', isAscendancyStart: true, ascendancy: 'Lich' }),
    node(8, 500, -400, { group: 5, type: 'Socket', name: 'Jewel Socket', nodesInRadius: [[4], [4, 3], [4, 3, 2], [4, 3, 2, 5], [5]] })
  ],
  jewelRadius: [
    { inner: 0, outer: 1200, col: '#bb6600' },
    { inner: 0, outer: 1380, col: '#66ffcc' },
    { inner: 0, outer: 1560, col: '#2222cc' },
    { inner: 0, outer: 1800, col: '#c100ff' },
    { inner: 780, outer: 1140, col: '#d35400' }
  ]
});
