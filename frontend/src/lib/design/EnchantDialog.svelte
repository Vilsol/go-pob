<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import { enchantSkills, sortByValue, type CraftApi, type EnchantChoices } from './crafting';
  import type { DisplayEdit, TipProps } from './items';

  // ItemsTab.lua:2661-2870
  // ItemsTab.lua:2754-2814
  type Props = { open?: boolean; slot: number; craft: CraftApi; onedit: (edit: DisplayEdit) => void };
  let { open = $bindable(false), slot, craft, onedit }: Props = $props();

  let choices = $state.raw<EnchantChoices>();
  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let allSkills = $state(false);
  let skill = $state(0);
  let source = $state(0);
  let sort = $state('');
  let values = $state.raw<number[]>();
  let sorting = $state(false);
  let pick = $state(0);
  let preview = $state.raw<TipProps>();

  $effect(() => {
    if (!open) return;
    allSkills = false;
    skill = source = pick = 0;
    sort = '';
    values = undefined;
    void craft.enchantChoices().then((c) => (choices = c));
    void craft.sortStats().then((s) => (stats = s));
  });

  const skillNames = $derived(choices ? enchantSkills(choices, allSkills) : []);
  const sources = $derived(choices?.skills.find((s) => s.skill === (choices?.haveSkills ? skillNames[skill] : ''))?.sources ?? []);
  const lines = $derived(sources[Math.min(source, sources.length - 1)]?.lines ?? []);
  const order = $derived(sortByValue(sort ? values : undefined, lines.length));
  const line = $derived(lines[order[pick]]);

  $effect(() => {
    const stat = sort;
    const list = lines;
    values = undefined;
    pick = 0;
    if (!stat || !list.length) return;
    sorting = true;
    void craft.enchantSort(slot, list, stat).then((v) => {
      if (stat === sort && list === lines) values = v;
      sorting = false;
    });
  });
  $effect(() => {
    const l = line;
    if (!open || !l) {
      preview = undefined;
      return;
    }
    void craft.preview({ kind: 'enchant', index: slot, text: l }).then((p) => {
      if (l === line) preview = p;
    });
  });
</script>

<Dialog bind:open title="Enchant Item" class="w-150 p-4.5">
  <div class="flex flex-col gap-2.5 text-13">
    {#if choices?.haveSkills}
      <div class="flex items-end gap-3">
        <FloatingLabel label="Skill" class="grow"
          ><Select
            class="h-7.5"
            aria-label="Skill"
            options={skillNames.map((s, i) => ({ value: String(i), label: s }))}
            value={String(skill)}
            onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
              skill = Number(e.currentTarget.value);
              source = Math.min(source, Math.max(0, sources.length - 1));
            }} /></FloatingLabel>
        <Checkbox
          compact
          checked={allSkills || !choices.used.length}
          disabled={!choices.used.length}
          title="Show all skills, not just those used by this build."
          onchange={(e: Event & { currentTarget: HTMLInputElement }) => {
            allSkills = e.currentTarget.checked;
            skill = 0;
          }}>All skills</Checkbox>
      </div>
    {/if}
    <div class="grid grid-cols-2 gap-3">
      <FloatingLabel label="Source"
        ><Select
          class="h-7.5"
          aria-label="Source"
          options={sources.map((s, i) => ({ value: String(i), label: s.label }))}
          value={String(Math.min(source, Math.max(0, sources.length - 1)))}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (source = Number(e.currentTarget.value))} /></FloatingLabel>
      <FloatingLabel label="Sort by"
        ><Select
          class="h-7.5"
          aria-label="Sort by"
          options={[{ value: '', label: 'Default' }, ...stats.map((s) => ({ value: s.stat, label: s.label }))]}
          bind:value={sort} /></FloatingLabel>
    </div>
    <FloatingLabel label="Enchantment"
      ><Select
        class="h-7.5"
        aria-label="Enchantment"
        aria-busy={sorting}
        options={order.map((k, i) => ({ value: String(i), label: lines[k] }))}
        value={String(pick)}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (pick = Number(e.currentTarget.value))} /></FloatingLabel>
    {#if sorting}<span class="note">Sorting…</span>{/if}
    {#if preview}
      <div aria-label="Enchanted item" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <Button
      size="md"
      variant="primary"
      disabled={!line}
      onclick={() => {
        onedit({ kind: 'enchant', index: slot, text: line });
        open = false;
      }}>Enchant</Button>
    <Button
      size="md"
      onclick={() => {
        onedit({ kind: 'enchant', index: slot, value: 1 });
        open = false;
      }}>Remove</Button>
    <Button size="md" onclick={() => (open = false)}>Cancel</Button>
  {/snippet}
</Dialog>
