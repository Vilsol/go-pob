<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import ItemIcon from './ItemIcon.svelte';
  import GemIcon from './GemIcon.svelte';
  import { expect, waitFor } from 'storybook/test';

  const { Story } = defineMeta({ title: 'Spec/ItemIcon', component: ItemIcon });

  // stands in for the CDN's art, so the stories never reach the network
  const art = (fill: string) =>
    `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2 4"><rect x=".5" y=".2" width="1" height="3.6" rx=".3" fill="${fill}"/></svg>`)}`;

  // synthetic sheets with the CDN's pixel sizes: 78px frames side by side, transparent around each shape
  const svg = (w: number, h: number, body: string) =>
    `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">${body}</svg>`)}`;
  const flaskSheet = svg(
    234,
    156,
    '<rect x="24" y="4" width="30" height="40" fill="#c9a227"/><rect x="88" y="30" width="58" height="122" rx="10" fill="#3a4a8c"/><rect x="166" y="60" width="58" height="90" rx="8" fill="#c0392b"/>'
  );
  const flaskFrame = svg(78, 156, '<rect x="10" y="20" width="58" height="130" rx="10" fill="#3a4a8c"/>');
  const gemSheet = svg(234, 78, '<circle cx="39" cy="20" r="12" fill="#c9a227"/><path d="M195 6 228 39 195 72 162 39z" fill="#2e9e44"/>');
  const gemSingle = svg(78, 78, '<path d="M39 6 72 39 39 72 6 39z" fill="#5a7fd8"/>');

  const icon = (root: HTMLElement, label: string) => root.querySelector<HTMLElement>(`[data-case="${label}"] [data-item-icon]`)!;

  /* each layer is one frame-sized window centred in the box, showing the sheet shifted by whole frames */
  const frames = (box: HTMLElement) => {
    const b = box.getBoundingClientRect();
    return [...box.querySelectorAll<HTMLElement>('[data-sprite-frame]')].map((l) => {
      const r = l.getBoundingClientRect();
      const img = l.querySelector('img')!.getBoundingClientRect();
      return {
        frame: Number(l.dataset.spriteFrame),
        w: r.width / b.width,
        h: r.height / b.height,
        centred: Math.abs(r.left + r.width / 2 - (b.left + b.width / 2)) < 0.5 && Math.abs(r.top + r.height / 2 - (b.top + b.height / 2)) < 0.5,
        shift: Math.round(((img.left - r.left) / r.width) * 100) / 100 + 0,
        sheet: Math.round((img.width / r.width) * 100) / 100
      };
    });
  };
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

<Story
  name="Sprite sheets"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await waitFor(() => expect([...canvasElement.querySelectorAll('[data-item-icon]')].every((b) => b.getAttribute('data-item-icon') === 'loaded')).toBe(true));
    for (const size of ['48', '20']) {
      const flask = frames(icon(canvasElement, `flask-${size}`));
      await expect(flask.map((f) => f.frame)).toEqual([1, 2, 0]);
      for (const f of flask) {
        await expect([f.w, f.h, f.centred, f.sheet, f.shift]).toEqual([0.5, 1, true, 3, 0 - f.frame]);
      }
      const gem = frames(icon(canvasElement, `gem-${size}`));
      await expect(gem.map((f) => f.frame)).toEqual([2, 0]);
      for (const f of gem) await expect([f.w, f.h, f.centred, f.sheet, f.shift]).toEqual([1, 1, true, 3, 0 - f.frame]);
      for (const whole of [`support-${size}`, `frame-${size}`, `belt-${size}`]) {
        const box = icon(canvasElement, whole);
        await expect(frames(box)).toEqual([]);
        await expect(box.querySelector('img')).not.toHaveClass('opacity-0');
      }
    }
    await expect(icon(canvasElement, 'flask-48').getBoundingClientRect().width).toBe(48);
    await expect(icon(canvasElement, 'flask-48').querySelector(':scope > img')).toHaveAttribute('alt', 'Life Flask');
  }}>
  <div class="flex flex-col gap-3">
    {#each [48, 20] as size (size)}
      <div class="flex items-end gap-3">
        <span data-case="flask-{size}"><ItemIcon name="Life Flask" base="Life Flask" {size} src={flaskSheet} /></span>
        <span data-case="gem-{size}"><GemIcon name="Fireball" {size} src={gemSheet} /></span>
        <span data-case="support-{size}"><GemIcon name="Added Fire Damage Support" {size} src={gemSingle} /></span>
        <span data-case="frame-{size}"><ItemIcon name="Life Flask" base="Life Flask" {size} src={flaskFrame} /></span>
        <span data-case="belt-{size}"><ItemIcon name="Leather Belt" base="Leather Belt" {size} src={flaskSheet} /></span>
      </div>
    {/each}
  </div>
</Story>
