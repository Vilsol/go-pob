<script lang="ts">
  import Tooltip from './Tooltip.svelte';
  import Field from './Field.svelte';
  import PobText from './PobText.svelte';
  import type { CalcsHeaderField, CalcsHeaderSide } from './compare';
  import type { calculator } from '$lib/types';

  // CompareTab.lua:4480-4608
  type Props = {
    sides: [CalcsHeaderSide, CalcsHeaderSide];
    oncontrol: (side: 0 | 1, field: CalcsHeaderField, value: number | string | boolean | undefined) => void;
    /* CompareCalcsHelpers.lua:355-394 */
    breakdown?: (side: 0 | 1, name: string) => Promise<calculator.CompareSkillBreakdownRow[]>;
  };
  let { sides, oncontrol, breakdown }: Props = $props();

  // CompareTab.lua:4556-4593, 4892-4898
  const BREAKDOWN: Record<string, string> = { 'Aura/Buff Skills': 'SkillBuffs', 'Curses/Debuffs': 'SkillDebuffs' };
  let panel = $state.raw<{ key: string; anchor: HTMLElement; rows?: calculator.CompareSkillBreakdownRow[] }>();
  const showPanel = async (side: 0 | 1, label: string, anchor: HTMLElement) => {
    const name = BREAKDOWN[label];
    if (!name || !breakdown) return;
    const key = `${side}:${label}`;
    panel = { key, anchor };
    const rows = await breakdown(side, name);
    if (panel?.key === key && rows.length) panel = { ...panel, rows };
  };

  // CompareTab.lua:483-488
  const MODES = [
    { label: 'Unbuffed', buffMode: 'UNBUFFED' },
    { label: 'Buffed', buffMode: 'BUFFED' },
    { label: 'In Combat', buffMode: 'COMBAT' },
    { label: 'Effective DPS', buffMode: 'EFFECTIVE' }
  ];
  const count = (text: string) => {
    const v = text.trim() === '' ? undefined : Number(text);
    return v === undefined || Number.isInteger(v) ? { ok: true, v } : { ok: false, v };
  };
  const TONE = ['text-code-evasion', 'text-code-orange'];
</script>

<div role="group" aria-label="Calcs skill controls" class="grid grid-cols-2 gap-4 border-b border-line pb-2 text-12">
  {#each sides as s, i (i)}
    {@const side = i as 0 | 1}
    {@const c = s.controls}
    <div class="flex min-w-0 flex-col gap-1" aria-label="{s.name} calcs controls" role="group">
      <span class={['text-14 font-semibold', TONE[i]]}>{s.name}</span>
      <Field label="Socket Group:" inline>
        <select
          class="select input-xs max-w-60"
          aria-label="Socket Group"
          value={String(Math.max(0, Math.min(s.skillNumber, s.groups.length) - 1))}
          onchange={(e) => oncontrol(side, 'group', Number(e.currentTarget.value) + 1)}
          >{#each s.groups.length ? s.groups : ['<No skills added yet>'] as g, k (k)}<option value={String(k)}>{g}</option>{/each}</select
        ></Field>
      {#if c?.Skills?.length}
        <Field label="Active Skill:" inline>
          <select
            class="select input-xs max-w-60"
            aria-label="Active Skill"
            disabled={!c.SkillEnabled}
            value={String(c.MainActiveSkill - 1)}
            onchange={(e) => oncontrol(side, 'MainActiveSkill', Number(e.currentTarget.value) + 1)}
            >{#each c.Skills as l, k (k)}<option value={String(k)}>{l}</option>{/each}</select
          ></Field>
      {/if}
      {#if c?.PartShown}
        <Field label="Skill Part:" inline>
          <select
            class="select input-xs max-w-60"
            aria-label="Skill Part"
            value={String(c.SkillPart - 1)}
            onchange={(e) => oncontrol(side, 'SkillPart', Number(e.currentTarget.value) + 1)}
            >{#each c.Parts ?? [] as l, k (k)}<option value={String(k)}>{l}</option>{/each}</select
          ></Field>
      {/if}
      {#each [{ shown: c?.StageShown, field: 'StageCount' as const, label: 'Stages', value: c?.StageCount }, { shown: c?.MineShown, field: 'MineCount' as const, label: 'Mines', value: c?.MineCount }] as box (box.field)}
        {#if box.shown}
          <Field label={`${box.label}:`} inline>
            <input
              class="input input-xs w-13"
              inputmode="numeric"
              aria-label={box.label}
              value={box.value}
              onchange={(e) => {
                const r = count(e.currentTarget.value);
                if (r.ok) oncontrol(side, box.field, r.v);
              }} /></Field>
        {/if}
      {/each}
      {#if s.showMinion !== undefined}
        <label class="field field-inline"
          ><input
            class="check"
            type="checkbox"
            aria-label="Show Minion Stats"
            checked={s.showMinion}
            onchange={(e) => oncontrol(side, 'showMinion', e.currentTarget.checked)} />Show Minion Stats</label>
      {/if}
      {#if c?.MinionShown}
        <Field label="Minion:" inline>
          <select
            class="select input-xs max-w-60"
            aria-label="Minion"
            disabled={!c.MinionEnabled}
            value={String(c.Minion - 1)}
            onchange={(e) => oncontrol(side, 'Minion', Number(e.currentTarget.value) + 1)}
            >{#each c.Minions ?? [] as o, k (k)}<option value={String(k)}>{o.Label}</option>{/each}</select
          ></Field>
      {/if}
      {#if c?.MinionSkillShown}
        <Field label="Minion Skill:" inline>
          <select
            class="select input-xs max-w-60"
            aria-label="Minion Skill"
            disabled={!c.MinionSkillEnabled}
            value={String(c.MinionSkill - 1)}
            onchange={(e) => oncontrol(side, 'MinionSkill', Number(e.currentTarget.value) + 1)}
            >{#each c.MinionSkills ?? [] as l, k (k)}<option value={String(k)}>{l}</option>{/each}</select
          ></Field>
      {/if}
      <Field label="Calc Mode:" inline>
        <select class="select input-xs" aria-label="Calc Mode" value={s.buffMode} onchange={(e) => oncontrol(side, 'mode', e.currentTarget.value)}
          >{#each MODES as m (m.buffMode)}<option value={m.buffMode}>{m.label}</option>{/each}</select
        ></Field>
      {#each s.info as l (l.label)}
        {#if l.text && breakdown && BREAKDOWN[l.label]}
          <span
            role="button"
            tabindex="0"
            class="break-words hover:bg-code-highlight-bg"
            aria-label={l.label}
            aria-describedby={panel?.key === `${side}:${l.label}` && panel.rows ? 'compare-skill-breakdown' : undefined}
            onmouseenter={(e) => void showPanel(side, l.label, e.currentTarget)}
            onfocus={(e) => void showPanel(side, l.label, e.currentTarget)}
            onmouseleave={() => (panel = undefined)}
            onblur={() => (panel = undefined)}><PobText text="^7{l.label}: {l.text}" /></span>
        {:else if l.text}<span class="break-words" aria-label={l.label}><PobText text="^7{l.label}: {l.text}" /></span>{/if}
      {/each}
    </div>
  {/each}
</div>
{#if panel?.rows}
  <Tooltip id="compare-skill-breakdown" placement="beside" anchor={panel.anchor} size="none" class="border-2 border-code-highlight px-1 py-0.5">
    <table>
      <thead
        ><tr class="text-14"
          >{#each ['Value', 'Stat', 'Source', 'Source Name'] as h (h)}<th class="px-1.5 text-left font-normal">{h}</th>{/each}</tr
        ></thead>
      <tbody>
        {#each panel.rows as r, i (i)}
          <tr class="border-t border-line">
            <td class="px-1.5">{r.Value}</td><td class="px-1.5">{r.Name}</td><td class="px-1.5">{r.Source}</td><td class="px-1.5"
              ><PobText text={r.SourceName} /></td>
          </tr>
        {/each}
      </tbody>
    </table>
  </Tooltip>
{/if}
