<script lang="ts">
  import Input from './Input.svelte';
  import Select from './Select.svelte';
  import Textarea from './Textarea.svelte';
  import Button from './Button.svelte';
  import Field from './Field.svelte';
  import InlineMessage from './InlineMessage.svelte';
  import SectionHeading from './SectionHeading.svelte';
  import type { Snippet } from 'svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Checkbox from './Checkbox.svelte';
  import ChoiceCard from './ChoiceCard.svelte';
  import type { Character } from './fixtures/app';
  import { EXPORT_SITES, codeVerdict, jsonVerdict, urlVerdict, type ImportMode, type Verdict } from './share';
  import { tone as toneClass, type Tone } from './tone';

  type CharOpts = { deleteJewels: boolean; deleteSkills: boolean; deleteEquipment: boolean; ignoreSwap: boolean };
  type Unavailable = { share?: string; exportParty?: string; links?: string; characters?: string; modes?: Partial<Record<ImportMode, string>> };
  type Props = {
    code: string;
    link?: string | null;
    site?: string;
    exportParty?: boolean;
    importText?: string;
    mode?: ImportMode;
    realms: string[];
    account?: string;
    realm?: string;
    characters?: Character[] | null;
    unavailable?: Unavailable;
    copied?: boolean;
    importError?: string | null;
    /* ImportTab.lua:1489 */
    imported?: string | null;
    /* ImportTab.lua:709-714 */
    fetching?: boolean;
    /* ImportTab.lua:544 */
    sharing?: boolean;
    /* ImportTab.lua:548 */
    shareError?: string | null;
    oncopy?: (code: string) => void;
    onshare?: (site: string) => void;
    onimport?: (text: string, mode: ImportMode) => void;
    onfetch?: (account: string, realm: string) => void;
    onimporttree?: (c: Character, o: CharOpts) => void;
    onimportitems?: (c: Character, o: CharOpts) => void;
    /* ImportTab.lua:506-509 */
    onexportparty?: (on: boolean) => void;
    /* Build.lua:1160-1162 */
    focusCode?: boolean;
    characterImport?: Snippet;
  };
  let {
    code,
    link = null,
    site = $bindable(EXPORT_SITES[0].label),
    exportParty = $bindable(false),
    importText = $bindable(''),
    mode = $bindable('new'),
    realms,
    account = $bindable(''),
    realm = $bindable(realms[0]),
    characters = null,
    unavailable = {},
    copied = false,
    importError = null,
    imported = null,
    fetching = false,
    sharing = false,
    shareError = null,
    oncopy,
    onshare,
    onimport,
    onfetch,
    onimporttree,
    onimportitems,
    onexportparty,
    focusCode = false,
    characterImport
  }: Props = $props();
  let codeInput = $state<HTMLInputElement>();
  $effect(() => {
    if (focusCode) codeInput?.focus();
  });

  const MODES: [ImportMode, string, string][] = [
    ['new', 'Import as a new build', 'opens it as a new build in the build list'],
    ['replace', 'Import to this build', 'overwrites the tree, items, skills and config of this build'],
    ['compare', 'Import as a comparison', 'opens it on the Compare page next to this build']
  ];
  const HINT = 'Paste a code or a link from pobb.in, PoeNinja, pob.codes, Maxroll, Pastebin.com or poedb.tw, or a character JSON';

  let verdict = $state<Verdict | null>(null);
  $effect(() => {
    const t = importText.trim();
    const known = t ? urlVerdict(t) : undefined;
    const url = known && unavailable.links ? { valid: false, text: `${known.text}, but ${unavailable.links}`, site: known.site } : known;
    const json = t && !url ? jsonVerdict(t) : undefined;
    verdict = !t ? null : (url ?? json ?? { valid: false, text: 'Checking…' });
    if (!t || url || json) return;
    let live = true;
    void codeVerdict(t).then((v) => {
      if (live) verdict = v;
    });
    return () => (live = false);
  });
  const verdictTone = $derived<Tone>(!verdict ? 'muted' : verdict.valid ? 'up' : verdict.text === 'Checking…' ? 'muted' : 'down');
  const modeOff = (m: ImportMode) => unavailable.modes?.[m];
  const canImport = $derived(!!verdict?.valid && !fetching && !modeOff(mode));

  const uid = $props.id();
  let opts = $state<CharOpts>({ deleteJewels: true, deleteSkills: true, deleteEquipment: true, ignoreSwap: false });
  let league = $state('');
  let picked = $state(0);
  const leagues = $derived([...new Set((characters ?? []).map((c) => c.league))]);
  const inLeague = $derived((characters ?? []).filter((c) => c.league === (league || leagues[0])));
  const chosen = $derived(inLeague[picked] ?? inLeague[0]);
</script>

<section aria-label="Build sharing" class="panel flex flex-col gap-2.5 p-3.5">
  <SectionHeading size="lg">BUILD SHARING</SectionHeading>
  <Textarea readonly aria-label="Build code" class="h-27 p-2 font-mono text-11 break-all text-text-3" value={code} />
  <div class="flex flex-wrap items-center gap-2">
    <Button size="md" class="px-3" disabled={!code} onclick={() => oncopy?.(code)}>Copy code</Button>
    {#if copied}<InlineMessage kind="success" tag="span" class="whitespace-nowrap">✓ Copied</InlineMessage>{/if}
    <FloatingLabel label="Share via"
      ><Select size="md" bind:value={site} disabled={!!unavailable.share} title={unavailable.share}>
        {#each EXPORT_SITES as s (s.label)}<option value={s.label}>{s.label}</option>{/each}
      </Select></FloatingLabel>
    <Button size="md" variant="primary" disabled={!code || sharing || !!unavailable.share} title={unavailable.share} onclick={() => onshare?.(site)}
      >{sharing ? 'Creating link...' : 'Share'}</Button>
  </div>
  {#if shareError}<InlineMessage kind="error">Error creating link: {shareError}</InlineMessage>{/if}
  {#if link}
    <div class="flex items-center gap-2 text-12">
      <span class="whitespace-nowrap text-better">✓ Link created</span><Input
        readonly
        size="sm"
        class="grow font-mono text-12"
        value={link}
        aria-label="Share link" />
    </div>
  {/if}
  <Checkbox
    compact
    bind:checked={exportParty}
    disabled={!!unavailable.exportParty}
    title={unavailable.exportParty}
    onchange={(e: Event & { currentTarget: HTMLInputElement }) => onexportparty?.(e.currentTarget.checked)}
    >Export support: include data for party play (auras, curses, link skills)</Checkbox>
</section>

<section aria-label="Build import" class="panel flex flex-col gap-2.5 p-3.5">
  <SectionHeading size="lg">BUILD IMPORT</SectionHeading>
  <Field label="To import a build, enter URL or code here:">
    <Input
      size="lg"
      class="font-mono text-13"
      bind:ref={codeInput}
      bind:value={importText}
      aria-describedby="{uid}-verdict"
      onkeydown={(e: KeyboardEvent & { currentTarget: HTMLInputElement }) => {
        // ImportTab.lua:671-675
        if (e.key === 'Enter' && canImport) onimport?.(importText.trim(), mode);
      }} /></Field>
  <div id="{uid}-verdict" aria-live="polite" class={['toned text-12', toneClass(verdictTone)]}>{verdict?.text ?? HINT}</div>
  <div role="radiogroup" aria-label="Import as" class="flex flex-col gap-1.5 text-13">
    {#each MODES as [id, label, hint] (id)}
      <label class="flex items-start gap-2"
        ><input type="radio" name="{uid}-mode" class="radio mt-0.5" value={id} bind:group={mode} disabled={!!modeOff(id)} title={modeOff(id)} /><span
          class="flex flex-col"><span>{label}</span><span class="note">{modeOff(id) ?? hint}</span></span
        ></label>
    {/each}
  </div>
  <Button size="md" variant="primary" class="self-start" disabled={!canImport} onclick={() => onimport?.(importText.trim(), mode)}
    >{fetching ? 'Retrieving paste..' : 'Import'}</Button>
  {#if importError}<InlineMessage kind="error">Import failed: {importError}</InlineMessage>{/if}
  {#if imported}<InlineMessage kind="success">{imported}</InlineMessage>{/if}
</section>

{#if characterImport}{@render characterImport()}{:else}<section aria-label="Character import" class="panel col-span-2 flex flex-col gap-3 p-3.5">
    <div class="flex items-baseline justify-between">
      <SectionHeading size="lg">CHARACTER IMPORT</SectionHeading>
      <span class="note"
        >Character import status: <span class={['toned', toneClass(characters ? 'up' : 'muted')]}
          >{characters ? `Fetched ${characters.length} characters` : unavailable.characters ? 'Not available' : 'Not fetched'}</span
        ></span>
    </div>
    {#if unavailable.characters}<p class="note">{unavailable.characters}</p>{/if}
    <div class="flex items-end gap-2.5 max-phone:flex-wrap">
      <label class="field"
        >Account name <Input
          size="md"
          class="w-55 px-2.5"
          bind:value={account}
          placeholder="AccountName#0000"
          disabled={!!unavailable.characters}
          title={unavailable.characters} /></label>
      <FloatingLabel label="Realm"
        ><Select size="md" bind:value={realm} disabled={!!unavailable.characters} title={unavailable.characters}
          >{#each realms as r (r)}<option>{r}</option>{/each}</Select
        ></FloatingLabel>
      <Button size="md" disabled={!account.trim() || !!unavailable.characters} title={unavailable.characters} onclick={() => onfetch?.(account.trim(), realm)}
        >Fetch characters</Button>
    </div>
    {#if characters}
      <div class="flex items-start gap-4 max-phone:flex-col max-phone:items-stretch">
        <div class="flex w-105 shrink-0 flex-col gap-2 max-phone:w-auto">
          <FloatingLabel label="League"
            ><Select
              size="md"
              value={league || leagues[0]}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
                league = e.currentTarget.value;
                picked = 0;
              }}
              >{#each leagues as l (l)}<option>{l}</option>{/each}</Select
            ></FloatingLabel>
          <span class="text-12 text-text-2">Choose character to import data from:</span>
          <div role="group" aria-label="Characters" class="flex flex-col gap-2">
            {#each inLeague as c, i (c.name)}
              <ChoiceCard pressed={c === chosen} class="flex-row items-center justify-between px-2.5 py-2" onclick={() => (picked = i)}
                ><span class="font-semibold">{c.name}</span><span class={['toned text-12', toneClass(c.className.toLowerCase() as Tone)]} data-pob-color
                  >{c.ascendancy} · level {c.level}</span
                ></ChoiceCard>
            {/each}
          </div>
        </div>
        <div class="grid min-w-0 grow grid-cols-2 gap-3 max-phone:grid-cols-1">
          <div class="flex flex-col gap-2 rounded-md border border-line bg-ground p-3 text-13">
            <h3 class="text-13 font-bold">Passive Tree and Jewels</h3>
            <span class="text-12 text-worse">Importing the passive tree will overwrite your current tree.</span>
            <Checkbox compact bind:checked={opts.deleteJewels}>Delete jewels: delete all equipped jewels when importing</Checkbox>
            <Button variant="primary" class="self-start px-3 text-12" onclick={() => chosen && onimporttree?.(chosen, opts)}>Import tree</Button>
          </div>
          <div class="flex flex-col gap-2 rounded-md border border-line bg-ground p-3 text-13">
            <h3 class="text-13 font-bold">Items and Skills</h3>
            <Checkbox compact bind:checked={opts.deleteSkills}>Delete skills: delete all existing skills</Checkbox>
            <Checkbox compact bind:checked={opts.deleteEquipment}>Delete equipment: delete all equipped items</Checkbox>
            <Checkbox compact bind:checked={opts.ignoreSwap}>Ignore weapon swap: skip items and skills in weapon swap</Checkbox>
            <Button variant="primary" class="self-start px-3 text-12" onclick={() => chosen && onimportitems?.(chosen, opts)}>Import items and skills</Button>
          </div>
        </div>
      </div>
    {/if}
  </section>{/if}
