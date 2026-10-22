<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import ItemIcon from './ItemIcon.svelte';
  import GemIcon from './GemIcon.svelte';

  const { Story } = defineMeta({ title: 'Spec/ItemIcon', component: ItemIcon });

  // stands in for the CDN's art, so the stories never reach the network
  const art = (fill: string) =>
    `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2 4"><rect x=".5" y=".2" width="1" height="3.6" rx=".3" fill="${fill}"/></svg>`)}`;
</script>

<Story name="Loaded" asChild>
  <div class="flex items-end gap-3">
    <ItemIcon name="Headhunter" base="Leather Belt" rarity="unique" size={48} src={art('#af6025')} />
    <ItemIcon name="Doom Grip" base="Leather Belt" rarity="rare" src={art('#d8c75a')} />
    <GemIcon name="Fireball" src={art('#5a7fd8')} />
  </div>
</Story>

<Story name="Missing" asChild>
  <div class="flex items-end gap-3">
    <ItemIcon name="Headhunter" base="Leather Belt" rarity="unique" size={48} />
    <ItemIcon name="Doom Grip" base="Leather Belt" rarity="rare" />
    <ItemIcon name="Crimson Jewel" base="Crimson Jewel" rarity="magic" size={24} />
    <GemIcon name="Added Fire Damage" />
  </div>
</Story>

<Story name="Offline" asChild>
  <div class="flex items-end gap-3">
    <ItemIcon name="Headhunter" base="Leather Belt" rarity="unique" size={48} src="/offline/Art/2DItems/Belts/Headhunter.png" />
    <span class="flex items-center gap-1.5 text-12"><GemIcon name="Fireball" decorative src="/offline/Art/2DItems/Gems/Fireball.png" />Fireball</span>
  </div>
</Story>
