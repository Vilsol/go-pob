<script lang="ts">
  import Textarea from './Textarea.svelte';
  import Select from './Select.svelte';
  import Input from './Input.svelte';
  import Button from './Button.svelte';
  import Field from './Field.svelte';
  import Badge from './Badge.svelte';
  import InlineMessage from './InlineMessage.svelte';
  import SectionHeading from './SectionHeading.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Checkbox from './Checkbox.svelte';
  import { PARTY_CATEGORIES, PARTY_DESTINATIONS, summarize, type PartyCategory, type PartyEntry, type PartyMember } from './party';
  import { codeVerdict, urlVerdict, type Verdict } from './share';
  import { tone as toneClass, type Tone } from './tone';

  type Props = {
    buffers: Record<PartyCategory, string>;
    members: PartyMember[];
    from?: Record<string, string>;
    impact: [string, string, 'up' | 'down'][];
    summaries?: Partial<Record<PartyCategory, PartyEntry[]>>;
    stale?: boolean;
    importError?: string | null;
    importText?: string;
    destination?: string;
    append?: boolean;
    advanced?: boolean;
    disabled?: boolean;
    onimport?: (text: string, destination: string, append: boolean) => void;
    onclear?: (destination: string) => void;
    onrebuild?: () => void;
    onmember?: (id: string, active: boolean) => void;
    onremove?: (id: string) => void;
    onbuffer?: (cat: PartyCategory, text: string) => void;
  };
  let {
    buffers,
    members,
    from = {},
    impact,
    summaries = {},
    stale = false,
    importError = null,
    importText = $bindable(''),
    destination = $bindable('All'),
    append = $bindable(false),
    advanced = $bindable(false),
    disabled = $bindable(false),
    onimport,
    onclear,
    onrebuild,
    onmember,
    onremove,
    onbuffer
  }: Props = $props();

  const uid = $props.id();
  let verdict = $state<Verdict | null>(null);
  $effect(() => {
    const t = importText.trim();
    const url = t ? urlVerdict(t) : undefined;
    verdict = !t ? null : (url ?? { valid: false, text: 'Checking…' });
    if (!t || url) return;
    let live = true;
    void codeVerdict(t).then((v) => {
      if (live) verdict = v;
    });
    return () => (live = false);
  });

  const view = (c: PartyCategory) => summaries[c] ?? summarize(c, buffers[c]);
  const byId = $derived(Object.fromEntries(members.map((m) => [m.id, m])));
  const giver = (c: PartyCategory, name: string) => from[name] ?? from[`@${c}`];
  const gives = (id: string) =>
    PARTY_CATEGORIES.filter((c) => view(c.id).some((e) => giver(c.id, e.name) === id))
      .map((c) => c.label)
      .join(', ');
  const live = (c: PartyCategory, name: string) => !disabled && byId[giver(c, name)]?.active !== false;
</script>

<section aria-label="Party import" class="flex w-90 shrink-0 flex-col gap-2.5 max-phone:w-full">
  <p class="note m-0">
    To import a build it must be exported with “Export support” enabled on the Import / Export page. Auras with the highest effect take priority; your curses
    take priority over a support’s. All of these effects can be found on the Calcs page.
  </p>
  <Field label="Enter a build code/URL below:">
    <Input
      size="lg"
      class="bg-surface-1 text-13"
      bind:value={importText}
      placeholder="https://pobb.in/… or a build code"
      aria-describedby="{uid}-state" /></Field>
  <span id="{uid}-state" aria-live="polite" class={['toned min-h-4 text-12', toneClass(!verdict ? 'muted' : verdict.valid ? 'up' : 'down')]}
    >{verdict?.text ?? ''}</span>
  {#if importError}<InlineMessage kind="error" tag="span">{importError}</InlineMessage>{/if}
  <div class="flex flex-wrap items-center gap-1.5">
    <FloatingLabel label="Destination"
      ><Select size="md" bind:value={destination} title="Destination for Import / Clear">
        {#each PARTY_DESTINATIONS as d (d)}<option>{d}</option>{/each}
      </Select></FloatingLabel>
    <Button variant="primary" class="px-3 text-12" disabled={!verdict?.valid} onclick={() => onimport?.(importText.trim(), destination, append)}>Import</Button>
    <Checkbox compact bind:checked={append}>Append</Checkbox>
    <Button variant="ghost" class="px-3 text-12" onclick={() => onclear?.(destination)}>Clear</Button>
  </div>
  <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
    <Checkbox compact class="text-13" bind:checked={advanced}>Show Advanced Info</Checkbox>
    <Checkbox compact class="text-13" bind:checked={disabled}>Disable Party Effects</Checkbox>
    <Button
      size="sm"
      class="px-3"
      title="Reparse all the inputs in case they have been disabled or they have changed since loading the build or importing"
      onclick={onrebuild}>Rebuild All</Button>
  </div>
  {#if stale}<span class="note" role="status">A buffer changed since the effects were applied: Rebuild All applies it.</span>{/if}
  {#if members.length}
    <SectionHeading size="md" class="mt-1.5">MEMBERS</SectionHeading>
    {#each members as m (m.id)}
      <div class="panel flex flex-col gap-1 px-3 py-2.5" data-member={m.id} data-active={m.active}>
        <div class="flex justify-between gap-2">
          <span class="truncate font-semibold">{m.name}</span>
          <span class="flex shrink-0 items-center gap-2">
            <Checkbox compact checked={m.active} onchange={() => onmember?.(m.id, !m.active)} aria-label="{m.name} active">active</Checkbox>
            {#if onremove}<Button
                size="2xs"
                variant="ghost"
                aria-label="Remove {m.name}"
                title="Remove this member and what it gives"
                onclick={() => onremove(m.id)}>Remove</Button
              >{/if}
          </span>
        </div>
        <span class={['toned text-12', toneClass(m.className.toLowerCase() as Tone)]} data-pob-color>{m.className} · {m.ascendancy} · level {m.level}</span>
        <span class="note">{(m.gives ?? gives(m.id)) || 'nothing imported yet'}</span>
      </div>
    {/each}
  {/if}
  <div class="panel mt-auto flex flex-col gap-0.75 px-3 py-2.5 font-mono text-12" aria-label="Effect of the party" role="group">
    <span class="font-sans text-12 text-text-2">Effect of the party on this build</span>
    {#if disabled}
      <span class="text-text-3">Party effects disabled</span>
    {:else}
      {#each impact as [label, text, dir] (label)}<div class="flex justify-between gap-2">
          <span>{label}</span><span class={['toned', toneClass(dir)]}>{text}</span>
        </div>{/each}
    {/if}
  </div>
</section>

<section
  aria-label="Party effects"
  class="scroll grid min-w-0 flex-[1_1_0] grid-cols-3 content-start gap-3 max-phone:flex-none max-phone:grid-cols-1 max-phone:overflow-visible">
  {#each PARTY_CATEGORIES as c (c.id)}
    {@const entries = view(c.id)}
    <section class="panel flex flex-col" aria-label={c.label}>
      <h3 class="subhead flex justify-between border-b border-line px-3 py-2">
        {c.label.toUpperCase()}<span class="font-mono font-normal">{entries.length}</span>
      </h3>
      {#if advanced}
        <Textarea
          class="m-2 h-28 font-mono text-11"
          aria-label="{c.label} buffer"
          spellcheck="false"
          value={buffers[c.id]}
          oninput={(e: Event & { currentTarget: HTMLTextAreaElement }) => onbuffer?.(c.id, e.currentTarget.value)} />
      {:else if !entries.length}
        <span class="px-3 py-2 text-12 text-text-3">nothing imported</span>
      {:else}
        {#each entries as e, i (i)}
          {@const on = live(c.id, e.name)}
          {@const who = byId[giver(c.id, e.name)]}
          <div class={['flex items-center gap-2 border-b border-surface-2 px-3 py-1.75 text-12', !on && 'bg-inset']} data-entry={e.name} data-live={on}>
            <span class={['truncate font-semibold', on ? 'text-text' : 'text-text-3 line-through']}>{e.name}</span>
            {#if who}<Badge variant="outline" title="from {who.name}">{who.name.split(' ')[0]}</Badge>{/if}
            <span class={['ml-auto font-mono whitespace-nowrap', on ? 'text-text-2' : 'text-text-3']}>{e.value}</span>
          </div>
        {/each}
      {/if}
    </section>
  {/each}
</section>
