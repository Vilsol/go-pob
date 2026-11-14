import xml from './socket-group-colour-labels.xml?raw';
import { spectreBuild, type BuildFixture, type SkillsFixture, type SocketGroupFixture } from './index';

// a build whose skill set titles and socket group labels use colour codes for headings and blank spacer rows
export const colourLabelXml = xml;

export const colourLabelSkills = (text = xml): SkillsFixture & { mainSocketGroup: number } => {
  const doc = new DOMParser().parseFromString(text, 'application/xml');
  const skills = doc.querySelector('Skills')!;
  const sets = [...skills.querySelectorAll('SkillSet')].map((set) => ({
    title: set.getAttribute('title') ?? '',
    groups: [...set.querySelectorAll('Skill')].map(
      (s): SocketGroupFixture => ({
        label: s.getAttribute('label') ?? '',
        slot: s.getAttribute('slot') ?? '',
        enabled: s.getAttribute('enabled') !== 'false',
        inFull: s.getAttribute('includeInFullDPS') === 'true',
        count: 1,
        sockets: '',
        gems: [...s.querySelectorAll('Gem')].map((g) => ({
          name: g.getAttribute('nameSpec') ?? '',
          level: Number(g.getAttribute('level') ?? 1),
          quality: Number(g.getAttribute('quality') ?? 0),
          enabled: g.getAttribute('enabled') !== 'false',
          support: (g.getAttribute('skillId') ?? '').startsWith('Support'),
          attr: 'white',
          count: 1
        }))
      })
    )
  }));
  const main = Number(doc.querySelector('Build')?.getAttribute('mainSocketGroup') ?? 1);
  return { activeSet: Number(skills.getAttribute('activeSkillSet') ?? 1) - 1, sets, mainSocketGroup: main - 1 };
};

export const colourLabelBuild = ((): BuildFixture => {
  const { mainSocketGroup, ...skills } = colourLabelSkills();
  return { ...spectreBuild, skills, mainSocketGroup };
})();
