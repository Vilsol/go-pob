<script lang="ts">
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import type { TipProps } from './items';
  import type { Tone } from './tone';
  import { itemWikiName } from './wiki';

  // ItemSlotControl.lua:48-56
  type Option = { id: number; label: string; tone: Tone };
  type Props = {
    slot: string;
    options: Option[];
    equipped?: number;
    class?: string;
    onequip: (id: number | null) => void;
    ondelete?: () => void;
    tipFor?: (id: number) => TipProps | undefined;
    onhoveritem?: (id: number | null) => void;
  };
  let { slot, options, equipped, class: extra = '', onequip, ondelete, tipFor, onhoveritem }: Props = $props();
  const uid = $props.id();
  let open = $state(false);
  let hovered = $state<{ id: number; el: HTMLElement } | null>(null);
  const hover = (id: number) => (el: HTMLElement | null) => {
    hovered = el ? { id, el } : null;
    onhoveritem?.(el ? id : null);
  };
  $effect(() => {
    if (!open && hovered) {
      hovered = null;
      onhoveritem?.(null);
    }
  });
  const tip = $derived(hovered && open ? tipFor?.(hovered.id) : undefined);
  const wikiOf = (id: number) => {
    const t = open ? tipFor?.(id) : undefined;
    return t ? itemWikiName(t.rarity, t.item, t.base) || undefined : undefined;
  };

  const items = $derived([
    ...(equipped ? [{ label: 'Unequip', onselect: () => onequip(null) }] : []),
    ...options
      .filter((o) => o.id !== equipped)
      .map((o, i) => ({
        label: `Equip ${o.label}`,
        tone: o.tone,
        separator: i === 0 && !!equipped,
        onselect: () => onequip(o.id),
        onhover: tipFor ? hover(o.id) : undefined,
        // ItemSlotControl.lua:162-173
        wiki: wikiOf(o.id)
      })),
    ...(equipped && ondelete ? [{ label: 'Delete item…', separator: true, onselect: ondelete }] : [])
  ]);
</script>

<Menu label={slot} {items} bind:open class="top-full mt-1 max-h-80 w-64 overflow-y-auto">
  {#snippet trigger(a)}
    <Button {...a} size="2xs" variant="ghost" class={['min-w-6 px-1.5 font-mono', extra]} aria-label="Change {slot}" title="Equip or unequip"
      ><Icon name="more" size={14} /></Button>
  {/snippet}
</Menu>
{#if tip && hovered}
  <FloatingItemTip anchor={hovered.el} id="{uid}-tip" {...tip} />
{/if}
