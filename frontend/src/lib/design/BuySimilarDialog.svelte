<script lang="ts">
  import Input from './Input.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import { LISTED, REALMS, isTradeUnique, tradeModLine, tradeSearchUrl, type TradeDefence, type TradeItem, type TradeMod, type TradeOptions } from './trade';

  // CompareBuySimilar.lua:281-520
  // CompareBuySimilar.lua:436-444
  // realms: the Realm dropdown's choices, which a single choice locks (CompareBuySimilar.lua:385-396, PoB-PoE2); url: the
  // game's own URL builder in place of PoE1's; mods: the mod rows (CompareBuySimilar.lua:459-516, PoB-PoE2)
  type Props = {
    open?: boolean;
    item?: TradeItem;
    slot?: string;
    defences?: TradeDefence[];
    realms?: [string, string][];
    mods?: TradeMod[];
    url?: (o: TradeOptions) => Promise<string | undefined>;
    onsearch: (url: string) => void;
  };
  let { open = $bindable(false), item, slot, defences = [], realms = REALMS, mods = [], url, onsearch }: Props = $props();
  let defenceRows = $state<{ on: boolean; min: string; max: string }[]>([]);
  $effect(() => {
    defenceRows = defences.map((d) => ({ on: false, min: String(Math.floor(d.value)), max: '' }));
  });
  let modRows = $state<{ on: boolean; min: string; max: string }[]>([]);
  $effect(() => {
    modRows = mods.map((m) => ({ on: false, min: m.min, max: '' }));
  });
  const modLabel = (m: TradeMod) => m.lines.map((l) => tradeModLine(l).text).join(' ');
  // a mod row's boxes take decimals and negatives, as tonumber reads them
  const dec = (s: string) => (/^-?\d+(\.\d*)?$|^-?\.\d+$/.test(s.trim()) ? Number(s) : undefined);

  let realm = $state(realms[0]?.[1] ?? 'pc');
  let league = $state('Standard');
  let listed = $state('securable');
  let baseType = $state(false);
  let ilvlMin = $state('');
  let ilvlMax = $state('');
  const num = (s: string) => (/^\d+$/.test(s.trim()) ? Number(s) : undefined);
  const unique = $derived(!!item && isTradeUnique(item));

  const options = $derived.by((): TradeOptions => {
    const ticked = defences.flatMap((d, i) =>
      defenceRows[i]?.on ? [[d.tradeKey, { min: num(defenceRows[i].min), max: num(defenceRows[i].max) }] as const] : []
    );
    return {
      realm,
      league,
      listed,
      baseType,
      ilvlMin: num(ilvlMin),
      ilvlMax: num(ilvlMax),
      defences: Object.fromEntries(ticked),
      mods: mods.map((m, i) => ({ checked: m.canSearch && !!modRows[i]?.on, min: dec(modRows[i]?.min ?? ''), max: dec(modRows[i]?.max ?? '') }))
    };
  });
  // CompareBuySimilar.lua:345-348, 517-524: every control rebuilds the URL, and Open URL waits for one; the search opens
  // within the click, as a browser blocks a tab opened after an await
  let built = $state<{ key: string; url: string }>();
  $effect(() => {
    const o = options;
    const key = JSON.stringify(o);
    if (!url || !open) return;
    let current = true;
    url(o)
      .then((u) => {
        if (current && u) built = { key, url: u };
      })
      .catch(() => {});
    return () => {
      current = false;
    };
  });
  const searchUrl = $derived.by(() => {
    if (!item) return undefined;
    if (!url) return tradeSearchUrl(item, slot, options);
    return built?.key === JSON.stringify(options) ? built.url : undefined;
  });
</script>

<Dialog bind:open title="Buy Similar" size="md">
  {#if item}
    <div class="flex flex-col gap-3 text-13">
      <div class="grid grid-cols-3 gap-2">
        <FloatingLabel label="Realm"
          ><Select options={realms.map(([label, value]) => ({ label, value }))} disabled={realms.length < 2} bind:value={realm} /></FloatingLabel>
        <FloatingLabel label="League"><Input aria-label="League" bind:value={league} /></FloatingLabel>
        <FloatingLabel label="Listed"><Select options={LISTED.map(([label, value]) => ({ label, value }))} bind:value={listed} /></FloatingLabel>
      </div>
      {#if unique}
        <p class="text-text">{item.title || item.name}</p>
      {:else}
        <p class="text-text-2">Category: {item.type || 'Item'}</p>
        <Checkbox compact bind:checked={baseType}>Use specific base: {item.baseName || 'Unknown'}</Checkbox>
        <div class="flex items-center gap-2">
          <span class="grow text-text-2">Item Level:</span>
          <Input class="w-20" aria-label="Item level min" placeholder="Min" inputmode="numeric" bind:value={ilvlMin} />
          <Input class="w-20" aria-label="Item level max" placeholder="Max" inputmode="numeric" bind:value={ilvlMax} />
        </div>
        {#each defences as d, i (d.tradeKey)}
          {#if defenceRows[i]}
            <div class="flex items-center gap-2">
              <span class="grow"><Checkbox compact bind:checked={defenceRows[i].on}>{d.label}</Checkbox></span>
              <Input class="w-20" aria-label="{d.label} min" placeholder="Min" inputmode="numeric" bind:value={defenceRows[i].min} />
              <Input class="w-20" aria-label="{d.label} max" placeholder="Max" inputmode="numeric" bind:value={defenceRows[i].max} />
            </div>
          {/if}
        {/each}
      {/if}
      {#if mods.length}
        <div class="flex flex-col gap-1" role="group" aria-label="Mods">
          {#each mods as m, i (i)}
            {#if modRows[i]}
              <div class="flex items-center gap-2" class:mt-3={i > 0 && mods[i - 1].type !== m.type}>
                <span class="grow">
                  <Checkbox compact disabled={!m.canSearch} bind:checked={modRows[i].on}>
                    <span class="flex flex-col">
                      {#each m.lines as line, j (j)}
                        {@const l = tradeModLine(line)}
                        <span style:color={l.color}>{l.text}</span>
                      {/each}
                    </span>
                  </Checkbox>
                </span>
                {#if m.hasBoxes}
                  <Input
                    class="w-20"
                    aria-label="{modLabel(m)} min"
                    placeholder="Min"
                    inputmode="decimal"
                    disabled={!m.canSearch}
                    bind:value={modRows[i].min} />
                  <Input
                    class="w-20"
                    aria-label="{modLabel(m)} max"
                    placeholder="Max"
                    inputmode="decimal"
                    disabled={!m.canSearch}
                    bind:value={modRows[i].max} />
                {/if}
              </div>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  {/if}
  {#snippet actions()}
    <DialogActions
      confirm="Search"
      disabled={!searchUrl}
      oncancel={() => (open = false)}
      onconfirm={() => {
        if (!searchUrl) return;
        onsearch(searchUrl);
        open = false;
      }} />
  {/snippet}
</Dialog>
