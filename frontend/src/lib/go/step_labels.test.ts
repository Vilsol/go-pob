import { describe, expect, it } from 'vitest';
import { activeSkillGroups, activeSpecNodes, configChangeLabel, itemTextName, skillsChangeLabel, treeChangeLabel, xmlAttrs } from './step_labels';

const tree = (active: number, ...specs: string[]) =>
  `<PathOfBuilding><Tree activeSpec="${active}">${specs.map((n, i) => `<Spec title="S${i}" nodes="${n}" treeVersion="3_29"><URL></URL></Spec>`).join('')}</Tree><TreeView searchStr=""></TreeView></PathOfBuilding>`;

const NODES: Record<number, { name: string; rank: number }> = {
  1: { name: 'Strength', rank: 0 },
  2: { name: 'Dexterity', rank: 0 },
  3: { name: 'Hasty Demise', rank: 3 },
  4: { name: 'Elemental Equilibrium', rank: 5 }
};
const info = (id: number) => NODES[id];

describe('tree labels', () => {
  it('reads the active spec’s nodes, not the first spec’s', () => {
    expect(activeSpecNodes(tree(2, '1,2', '3,4'))).toEqual([3, 4]);
    expect(activeSpecNodes(tree(1, '', '3'))).toEqual([]);
  });

  it('names the most notable node of a path and counts the rest', () => {
    expect(treeChangeLabel([], [1, 3, 2], info)).toBe('Allocated Hasty Demise (+2 path nodes)');
    expect(treeChangeLabel([1], [1, 2], info)).toBe('Allocated Dexterity');
    expect(treeChangeLabel([1, 2, 3, 4], [1], info)).toBe('Deallocated Elemental Equilibrium (+2 dependent nodes)');
    expect(treeChangeLabel([1, 2], [1, 3], info)).toBe('Allocated Hasty Demise, deallocated Dexterity');
  });

  it('counts nodes it cannot name, and says nothing when the nodes did not change', () => {
    expect(treeChangeLabel([], [70000, 70001], info)).toBe('Allocated 2 passives');
    expect(treeChangeLabel([1], [1], info)).toBeUndefined();
  });
});

const gem = (name: string, attrs = '') =>
  `<Gem level="20" ${attrs} nameSpec="${name}" gemId="Metadata/Items/Gems/${name.startsWith('Sup') ? 'SupportGem' : 'SkillGem'}${name}" quality="0" enabled="true"></Gem>`;
const skills = (...groups: string[]) =>
  `<PathOfBuilding><Skills activeSkillSet="2"><SkillSet id="1" title="Other"><Skill label="" enabled="true">${gem('Ignored')}</Skill></SkillSet><SkillSet id="2" title="Main">${groups.join('')}</SkillSet></Skills></PathOfBuilding>`;
const group = (gems: string[], attrs = 'label="" enabled="true"') => `<Skill ${attrs}>${gems.join('')}</Skill>`;

describe('skill labels', () => {
  it('reads the active skill set’s groups', () => {
    const gs = activeSkillGroups(skills(group([gem('Fireball'), gem('SupportCombustion')])));
    expect(gs).toHaveLength(1);
    expect(gs[0].gems.map((g) => g.nameSpec)).toEqual(['Fireball', 'SupportCombustion']);
  });

  it('names a gem added, removed, replaced, levelled and disabled', () => {
    const one = skills(group([gem('Fireball')]));
    expect(skillsChangeLabel(one, skills(group([gem('Fireball'), gem('SupportCombustion')])))).toBe('Added SupportCombustion to Fireball');
    expect(skillsChangeLabel(skills(group([gem('Fireball'), gem('SupportCombustion')])), one)).toBe('Removed SupportCombustion from Fireball');
    expect(skillsChangeLabel(one, skills(group([gem('Arc')])))).toBe('Replaced Fireball with Arc');
    expect(skillsChangeLabel(one, skills(group([gem('Fireball').replace('level="20"', 'level="21"')])))).toBe('Set Fireball to level 21');
    expect(skillsChangeLabel(one, skills(group([gem('Fireball').replace('enabled="true"', 'enabled="false"')])))).toBe('Disabled Fireball');
    expect(skillsChangeLabel(skills(group([])), skills(group([gem('Fireball')])))).toBe('Added Fireball to an empty socket group');
    const two = (x: string) => skills(group([gem('Fireball'), gem('SupportCombustion', x)]));
    expect(skillsChangeLabel(two(''), two('count="2"'))).toBe('Set SupportCombustion count to 2');
    const off = skills(group([gem('Fireball'), gem('SupportCombustion').replace('enabled="true"', 'enabled="false"')]));
    expect(skillsChangeLabel(two(''), off)).toBe('Disabled SupportCombustion in Fireball');
  });

  it('names a group added, removed, relabelled, socketed and moved', () => {
    const a = group([gem('Fireball')]);
    const b = group([gem('Arc')]);
    expect(skillsChangeLabel(skills(a), skills(a, group([])))).toBe('Added a socket group');
    expect(skillsChangeLabel(skills(a, b), skills(a))).toBe('Removed socket group Arc');
    expect(skillsChangeLabel(skills(a), skills(group([gem('Fireball')], 'label="Main" enabled="true"')))).toBe('Renamed Fireball to Main');
    expect(skillsChangeLabel(skills(a), skills(group([gem('Fireball')], 'label="" slot="Helmet" enabled="true"')))).toBe('Socketed Fireball in Helmet');
    expect(skillsChangeLabel(skills(a, b), skills(b, a))).toBe('Moved Fireball');
    expect(skillsChangeLabel(skills(a), skills(a))).toBeUndefined();
  });
});

describe('item and config labels', () => {
  it('names an item from its text', () => {
    expect(itemTextName('Item Class: Body Armours\nRarity: UNIQUE\nKaom’s Heart\nGlorious Plate')).toBe('Kaom’s Heart');
    expect(itemTextName('Rarity: RARE\r\nDoom Visor\r\nHubris Circlet')).toBe('Doom Visor');
    expect(itemTextName('')).toBe('an item');
  });

  it('names a config change from the option’s label and value', () => {
    expect(configChangeLabel('enemyLevel', 84, false, { label: 'Enemy Level:', type: 'count' })).toBe('Set Enemy Level to 84');
    expect(configChangeLabel('conditionStationary', true, false, { label: 'Are you stationary?', type: 'check' })).toBe('Turned on Are you stationary');
    expect(
      configChangeLabel('enemyIsBoss', 'Pinnacle', false, {
        label: 'Is the enemy a Boss?',
        type: 'list',
        list: [
          { label: 'No', value: 'None' },
          { label: 'Pinnacle Boss', value: 'Pinnacle' }
        ]
      })
    ).toBe('Set Is the enemy a Boss to Pinnacle Boss');
    expect(configChangeLabel('customMods', '', true, { label: 'Custom Modifiers:', type: 'text' })).toBe('Cleared Custom Modifiers');
  });

  it('reads attributes in any order and unescapes them', () => {
    expect(xmlAttrs('b="2" a="x &amp; y &#39;z&#39;"')).toEqual({ a: "x & y 'z'", b: '2' });
  });
});
