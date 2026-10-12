// SkillsTab.lua:91

export type GemLevelMode = 'normalMaximum' | 'corruptedMaximum' | 'awakenedMaximum' | 'characterLevel' | 'levelOne';

/* src/Classes/SkillsTab.lua:28-56 */
export const GEM_LEVEL_MODES: ReadonlyArray<{ value: GemLevelMode; label: string }> = [
  { value: 'normalMaximum', label: 'Normal Maximum' },
  { value: 'corruptedMaximum', label: 'Corrupted Maximum' },
  { value: 'awakenedMaximum', label: 'Awakened Maximum' },
  { value: 'characterLevel', label: 'Match Character Level' },
  { value: 'levelOne', label: 'Level 1' }
];

const MODE_VALUES: ReadonlySet<string> = new Set(GEM_LEVEL_MODES.map((m) => m.value));

export type GemLevelSetting = { kind: 'mode'; mode: GemLevelMode } | { kind: 'level'; level: number };

export const isGemLevelMode = (raw: string): raw is GemLevelMode => MODE_VALUES.has(raw);

/* SkillsTab.lua:91 */
export function parseGemLevel(raw: string | undefined | null): GemLevelSetting {
  if (raw === undefined || raw === null || raw === '') {
    return { kind: 'mode', mode: 'normalMaximum' };
  }

  if (isGemLevelMode(raw)) {
    return { kind: 'mode', mode: raw };
  }

  const n = Number.parseInt(raw, 10);
  if (Number.isFinite(n)) {
    return { kind: 'level', level: n };
  }

  return { kind: 'mode', mode: 'normalMaximum' };
}

export function formatGemLevel(setting: GemLevelSetting): string {
  return setting.kind === 'mode' ? setting.mode : String(setting.level);
}
