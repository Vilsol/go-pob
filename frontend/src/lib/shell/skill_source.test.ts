import { describe, expect, it } from 'vitest';
import { explodeSourceNote, itemSourceNote, nodeSourceNote } from './skill_source';

// SkillsTab.lua:371-388
describe('itemSourceNote', () => {
  it("is PoB's note for an item-granted skill", () => {
    expect(itemSourceNote('^x7070FF', 'Summon Spectral Wolf', '^xAF6025', "Ahn's Contempt")).toBe(
      "^7This is a special group created for the '^x7070FFSummon Spectral Wolf^7' skill,\nwhich is being provided by '^xAF6025Ahn's Contempt^7'.\nYou cannot delete this group, but it will disappear if you un-equip the item.\n\nYou cannot add support gems to this group, but support gems in\nany other group socketed into '^xAF6025Ahn's Contempt^7'\nwill automatically apply to the skill."
    );
  });
});

// SkillsTab.lua:363-383
describe('nodeSourceNote and explodeSourceNote', () => {
  it("is PoB's note for a node-granted skill", () => {
    expect(nodeSourceNote('^xC8C8C8', 'Death Wish', '^xC8C8C8', "Hinekora, Death's Fury", true)).toBe(
      "^7This is a special group created for the '^xC8C8C8Death Wish^7' skill,\nwhich is being provided by '^xC8C8C8Hinekora, Death's Fury^7'.\nYou cannot delete this group, but it will disappear if you un-allocate the node."
    );
  });
  it('lists the explosion sources', () => {
    expect(
      explodeSourceNote([
        { color: '^xAF6025', name: "Inpulsa's Broken Heart" },
        { color: '^xC8C8C8', name: '' }
      ])
    ).toBe(
      "^7This is a special group created for the enemy explosion effect,\nwhich comes from the following sources:\n\t^xAF6025Inpulsa's Broken Heart\n\t^xC8C8C8???^7\nYou cannot delete this group, but it will disappear if you lose the above sources."
    );
  });
});
