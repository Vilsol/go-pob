export type GemOptions = {
  defaultLevel: string;
  defaultQuality: number;
  showSupport: string;
  showAlt: boolean;
  showLegacy: boolean;
};

/* SkillSetListControl.lua:14-110 */
export type SkillSetOp = { op: 'new'; title: string } | { op: 'copy' | 'rename'; index: number; title: string } | { op: 'delete'; index: number };

// SkillsTab.lua:157-201 (PoB-PoE2): a group's Set 1 and Set 2 boxes; valid, enabled and forcedBoth where the game checks them
export type WeaponSetBoxes = {
  set1: boolean;
  set2: boolean;
  locked: boolean;
  valid?: [boolean, boolean];
  enabled?: [boolean, boolean];
  forcedBoth?: boolean;
  defaultAttack?: boolean;
};

export const weaponSetTip = (w: WeaponSetBoxes, set: 1 | 2): string | undefined =>
  w.forcedBoth
    ? 'This skill reserves Spirit in all weapon sets and must be enabled in both sets.'
    : w.locked
      ? w.defaultAttack
        ? 'Default attack skills can only be used in the weapon set that grants them.'
        : "Skills granted by items in weapon slots can only be used in that item's weapon set."
      : w.valid?.[set - 1] === false
        ? `This skill cannot be used with the weapons equipped in Set ${set}.`
        : undefined;
