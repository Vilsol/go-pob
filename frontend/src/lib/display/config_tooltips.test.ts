// ConfigTab.lua:334-349, ConfigOptions.lua:11-72
import { describe, expect, it } from 'vitest';
import { configurations, optionTooltip } from './configurations';

const variable = (key: string) => configurations.flatMap((s) => s.variables).find((v) => v.var === key)!;

describe('config option tooltips', () => {
  it('shows the bandit benefit of the chosen bandit', () => {
    expect(optionTooltip(variable('bandit'), 'Oak')).toBe('^8+40 to Maximum ^xE05030Life');
    expect(optionTooltip(variable('bandit'), 'None')).toBe('^8Grants 1 Passive Skill Point');
  });

  it("shows the chosen god's souls, and nothing for Nothing", () => {
    expect(optionTooltip(variable('pantheonMinorGod'), 'Tukohama')).toBe(
      "^8Soul of Tukohama\n^63% additional Physical Damage Reduction per second you've been stationary, up to a maximum of 9%\n\n^8Tahsin, Warmaker\n^6Regenerate 2% of Life per second while stationary"
    );
    expect(optionTooltip(variable('pantheonMajorGod'), 'None')).toBeUndefined();
  });

  it('fills the life tooltips from the main output', () => {
    expect(optionTooltip(variable('conditionFullLife'), false, { FullLifePercentage: 90 })).toBe(
      'You can be considered to be on Full ^xE05030Life ^7if you have at least 90% ^xE05030Life ^7left.\nYou will automatically be considered to be on Full ^xE05030Life ^7if you have Chaos Inoculation,\nbut you can use this option to force it if necessary.'
    );
    expect(optionTooltip(variable('conditionLowLife'), false, { LowLifePercentage: 35 })).toBe(
      'You will automatically be considered to be on Low ^xE05030Life ^7if you have at least 65% ^xE05030Life ^7reserved\nbut you can use this option to force it if necessary.'
    );
  });

  it('keeps a static tooltip as it is', () => {
    const v = configurations.flatMap((s) => s.variables).find((x) => x.var === 'conditionStationary')!;
    expect(optionTooltip(v, 0)).toBe('Applies mods that use `while stationary` and `per / every second while stationary`');
  });
});
