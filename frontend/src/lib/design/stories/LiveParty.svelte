<script lang="ts">
  import PartyPage from '../PartyPage.svelte';
  import type { BuildFixture } from '../fixtures';
  import { partySample } from '../fixtures/app';
  import { PARTY_CATEGORIES, destinationCategories, type PartyCategory, type PartyMember } from '../party';

  type Props = { fx: BuildFixture; advanced?: boolean; empty?: boolean };
  let { fx, advanced = false, empty = false }: Props = $props();

  const b = fx.compare.b;
  const stat = (key: string) => b.stats.flat().find((s) => s.key === `Player:${key}`)?.v ?? 0;
  const sample: Record<PartyCategory, string> = {
    ...(partySample.buffers as Record<PartyCategory, string>),
    PartyMemberStats: ['Life', 'Mana', 'EnergyShield'].map((k) => `${k}=${Math.round(stat(k))}`).join('\n')
  };
  const blank = Object.fromEntries(PARTY_CATEGORIES.map((c) => [c.id, ''])) as Record<PartyCategory, string>;

  let s = $state({
    buffers: empty ? { ...blank } : { ...sample },
    members: [
      { id: 'b', name: `${b.name} (imported)`, className: b.className, ascendancy: b.ascendClassName, level: b.level, active: true },
      { ...partySample.member, active: true }
    ] as PartyMember[],
    advanced,
    disabled: false,
    rebuilt: 0
  });
</script>

<PartyPage
  buffers={s.buffers}
  members={empty ? [] : s.members}
  from={partySample.from}
  impact={partySample.impact}
  bind:advanced={s.advanced}
  bind:disabled={s.disabled}
  onimport={(_t: string, dest: string, append: boolean) => {
    for (const c of destinationCategories(dest)) s.buffers[c] = append && s.buffers[c] ? `${s.buffers[c]}\n${sample[c]}` : sample[c];
  }}
  onclear={(dest: string) => {
    for (const c of destinationCategories(dest)) s.buffers[c] = '';
  }}
  onrebuild={() => s.rebuilt++}
  onmember={(id: string, active: boolean) => {
    const m = s.members.find((x) => x.id === id);
    if (m) m.active = active;
  }}
  onbuffer={(c: PartyCategory, text: string) => (s.buffers[c] = text)} />
