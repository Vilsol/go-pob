import type { WarningKind } from '$lib/display/sidebar';

// an issue's kind and the page that fixes it
export const issueKinds: Record<WarningKind, [string, string]> = {
  stat: ['Stat', 'Sidebar'],
  cost: ['Cost', 'Main skill'],
  skill: ['Skill', 'Skills'],
  item: ['Item', 'Items'],
  points: ['Passives', 'Tree']
};
