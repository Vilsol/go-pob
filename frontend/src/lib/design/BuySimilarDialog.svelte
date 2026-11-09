<script lang="ts">
  import Input from './Input.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import { LISTED, REALMS, isTradeUnique, tradeSearchUrl, type TradeDefence, type TradeItem, type TradeOptions } from './trade';

  // CompareBuySimilar.lua:281-520
  // CompareBuySimilar.lua:436-444
  // realms: the Realm dropdown's choices, which a single choice locks (CompareBuySimilar.lua:385-396, PoB-PoE2); url: the
  // game's own URL builder in place of PoE1's
  type Props = {
    open?: boolean;
    item?: TradeItem;
    slot?: string;
    defences?: TradeDefence[];
    realms?: [string, string][];
    url?: (o: TradeOptions) => Promise<string | undefined>;
    onsearch: (url: string) => void;
  };
  let { open = $bindable(false), item, slot, defences = [], realms = REALMS, url, onsearch }: Props = $props();
  let defenceRows = $state<{ on: boolean; min: string; max: string }[]>([]);
  $effect(() => {
    defenceRows = defences.map((d) => ({ on: false, min: String(Math.floor(d.value)), max: '' }));
  });

  let realm = $state(realms[0]?.[1] ?? 'pc');
  let league = $state('Standard');
  let listed = $state('securable');
  let baseType = $state(false);
  let ilvlMin = $state('');
  let ilvlMax = $state('');
  const num = (s: string) => (/^\d+$/.test(s.trim()) ? Number(s) : undefined);
  const unique = $derived(!!item && isTradeUnique(item));
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
    </div>
  {/if}
  {#snippet actions()}
    <DialogActions
      confirm="Search"
      disabled={!item}
      oncancel={() => (open = false)}
      onconfirm={() => {
        if (!item) return;
        const ticked = defences.flatMap((d, i) =>
          defenceRows[i]?.on ? [[d.tradeKey, { min: num(defenceRows[i].min), max: num(defenceRows[i].max) }] as const] : []
        );
        const o: TradeOptions = { realm, league, listed, baseType, ilvlMin: num(ilvlMin), ilvlMax: num(ilvlMax), defences: Object.fromEntries(ticked) };
        if (!url) {
          onsearch(tradeSearchUrl(item, slot, o));
          open = false;
          return;
        }
        void url(o).then((u) => {
          if (u) onsearch(u);
          open = false;
        });
      }} />
  {/snippet}
</Dialog>
