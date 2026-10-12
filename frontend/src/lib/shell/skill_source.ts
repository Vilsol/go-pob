// SkillsTab.lua:357-391, SkillsTab.lua:1185
export function itemSourceNote(activeColor: string, skillName: string, itemColor: string, itemName: string, noSupports = false) {
  const sourceName = `'${itemColor}${itemName}^7'`;
  let label = `^7This is a special group created for the '${activeColor}${skillName}^7' skill,
which is being provided by ${sourceName}.
You cannot delete this group, but it will disappear if you un-equip the item.`;
  if (!noSupports) {
    label += `\n\nYou cannot add support gems to this group, but support gems in
any other group socketed into ${sourceName}
will automatically apply to the skill.`;
  }
  return label;
}

// SkillsTab.lua:375-383
export function nodeSourceNote(activeColor: string, skillName: string, normalColor: string, nodeName: string, noSupports = false) {
  return itemSourceNote(activeColor, skillName, normalColor, nodeName, noSupports).replace('un-equip the item.', 'un-allocate the node.');
}

// SkillsTab.lua:363-369
export function explodeSourceNote(sources: { color: string; name: string }[]) {
  let label = `^7This is a special group created for the enemy explosion effect,
which comes from the following sources:`;
  for (const s of sources) label += `\n\t${s.color}${s.name || '???'}`;
  return label + '^7\nYou cannot delete this group, but it will disappear if you lose the above sources.';
}
