import type { IconName } from './Icon.svelte';
import type { StepDelta, StepKind } from '$lib/go/step_labels';

export const STEP_ICONS: Record<StepKind, IconName> = {
  tree: 'tree',
  item: 'armour',
  gem: 'gem',
  config: 'sliders',
  import: 'import-export',
  set: 'layers',
  party: 'party',
  calcs: 'calc',
  notes: 'notes',
  other: 'more'
};

export const STEP_KINDS: Record<StepKind, string> = {
  tree: 'Passive tree',
  item: 'Items',
  gem: 'Skills',
  config: 'Configuration',
  import: 'Import',
  set: 'Set switch',
  party: 'Party',
  calcs: 'Calcs',
  notes: 'Notes',
  other: 'Other'
};

export const DELTA_SHOWN = 8;

// a screen reader's one-line version of the stat change
export const deltaText = (delta: StepDelta[] | undefined) =>
  delta === undefined
    ? ''
    : delta.length
      ? delta.map((d) => `${d.section === 'Minion' ? 'Minion ' : ''}${d.label} ${d.value}`).join(', ')
      : 'No headline stat changed';
