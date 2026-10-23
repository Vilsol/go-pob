export type Cells = [number, number];

/* the largest box in the item's inventory proportions that fits w×h; an unknown size is one cell */
export function artBox(cells: Cells | undefined, w: number, h: number) {
  const [cw, ch] = cells && cells[0] > 0 && cells[1] > 0 ? cells : [1, 1];
  const cell = Math.min(w / cw, h / ch);
  return { w: cell * cw, h: cell * ch, cell, cols: cw, rows: ch };
}

/* socket i's cell in the game's order: down one column on a 1-wide item, on a wider one in rows of two that snake (1 2 / 4 3 / 5 6) */
export function socketGrid(count: number, width: number) {
  const cols = width >= 2 && count > 1 ? 2 : 1;
  const rows = Math.ceil(count / cols);
  const at = Array.from({ length: count }, (_, i) => {
    const row = Math.floor(i / cols);
    const col = i % cols;
    return { col: cols === 2 && row % 2 === 1 ? 1 - col : col, row };
  });
  return { cols, rows, at };
}

export type SocketedGroup = { group: number; source?: string; gems: { index: number; name: string }[] };
export type SocketedGem = { group: number; gem: number; name: string };

// SkillsTab.lua:266-281: the slot's groups in list order, each gem taking the next socket; abyssal sockets hold no gem
export function placeGems(line: string, groups: SocketedGroup[]): (SocketedGem | undefined)[] {
  const colours = [...line].filter((ch) => ch !== '-' && ch !== ' ');
  const out: (SocketedGem | undefined)[] = colours.map(() => undefined);
  const free = colours.flatMap((c, i) => (c === 'A' ? [] : [i]));
  let next = 0;
  for (const g of groups) {
    // a group an item or a passive grants occupies no socket
    if (g.source) continue;
    for (const gem of g.gems) {
      if (next < free.length) out[free[next++]] = { group: g.group, gem: gem.index, name: gem.name };
    }
  }
  return out;
}
