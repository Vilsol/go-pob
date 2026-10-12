// ItemListControl.lua:242-258

export type UsageContext = {
  sets: { id: string; title: string; slots: { slot: string; item: number }[] }[];
  activeSet: string;
  specs: { title: string; sockets: { node: number; item: number }[]; alloc: Set<number> }[];
  activeSpec: number;
};

// src/Classes/ItemListControl.lua:198-219
const socketedIn = (id: number, ctx: UsageContext): string | undefined => {
  let equipTree: string | undefined;
  let matchActive = false;
  for (let specId = ctx.specs.length - 1; specId >= 0; specId--) {
    const spec = ctx.specs[specId];
    for (const s of spec.sockets) {
      if (s.item === id && spec.alloc.has(s.node)) {
        if (specId === ctx.activeSpec || matchActive) {
          equipTree = undefined;
          matchActive = true;
        } else equipTree = spec.title || 'Default';
      }
    }
  }
  return equipTree;
};

// src/Classes/ItemListControl.lua:221-240
const abyssIn = (id: number, ctx: UsageContext): string | undefined => {
  let equipSet: string | undefined;
  let matchActive = false;
  for (const set of ctx.sets) {
    for (const s of set.slots) {
      if (s.item === id) {
        if (set.id === ctx.activeSet || matchActive) {
          equipSet = undefined;
          matchActive = true;
        } else equipSet = set.title || 'Default';
      }
    }
  }
  return equipSet;
};

// src/Classes/ItemListControl.lua:242-258
export const itemUsage = (id: number, kind: { jewel: boolean; abyss: boolean }, ctx: UsageContext): string => {
  const used = (kind.abyss ? abyssIn(id, ctx) : undefined) ?? (kind.jewel ? socketedIn(id, ctx) : undefined);
  if (used) return `Used in '${used}'`;
  const active = ctx.sets.find((s) => s.id === ctx.activeSet);
  const activeSpec = ctx.specs[ctx.activeSpec];
  if (active?.slots.some((s) => s.item === id)) return '';
  if (activeSpec?.sockets.some((s) => s.item === id && activeSpec.alloc.has(s.node))) return '';
  const other = ctx.sets.find((s) => s.id !== ctx.activeSet && s.slots.some((x) => x.item === id));
  return other ? `Used in '${other.title || 'Default'}'` : 'Unused';
};
