// @vitest-environment jsdom

import { describe, expect, it, afterEach, vi } from 'vitest';
import { render, cleanup, waitFor } from '@testing-library/svelte';
import ItemIcon from './ItemIcon.svelte';
import { ART_CDN, FLASK_SPRITE, GEM_SPRITE, artPath, artUrl, setArtResolver, spriteOf, type ArtQuery } from './item_art';

afterEach(() => {
  cleanup();
  setArtResolver(undefined);
});

const q = (Name: string, Base = '', Unique = false, Gem = false): ArtQuery => ({ Name, Base, Unique, Gem });

describe('artPath', () => {
  it('asks the resolver once per task for every lookup, and remembers answers', async () => {
    const resolver = vi.fn((qs: ArtQuery[]) => Promise.resolve(qs.map((x) => `Art/${x.Name}`)));
    setArtResolver(resolver);
    const got = await Promise.all([artPath(q('A')), artPath(q('B')), artPath(q('A'))]);
    expect(got).toEqual(['Art/A', 'Art/B', 'Art/A']);
    expect(resolver).toHaveBeenCalledTimes(1);
    expect(resolver.mock.calls[0][0].map((x) => x.Name)).toEqual(['A', 'B']);
    expect(await artPath(q('B'))).toBe('Art/B');
    expect(resolver).toHaveBeenCalledTimes(1);
  });

  it('keys a unique, a base and a gem of one name apart', async () => {
    const resolver = vi.fn((qs: ArtQuery[]) => Promise.resolve(qs.map((x) => (x.Gem ? 'gem' : x.Unique ? 'unique' : 'base'))));
    setArtResolver(resolver);
    expect(await Promise.all([artPath(q('X', 'B')), artPath(q('X', 'B', true)), artPath(q('X', '', false, true))])).toEqual(['base', 'unique', 'gem']);
  });

  it('answers empty when the resolver fails, and asks again next time', async () => {
    const resolver = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(['Art/A']);
    setArtResolver(resolver);
    expect(await artPath(q('A'))).toBe('');
    expect(await artPath(q('A'))).toBe('Art/A');
    expect(resolver).toHaveBeenCalledTimes(2);
  });

  it('maps an art path to the CDN', () => {
    expect(artUrl('Art/2DItems/Belts/Headhunter')).toBe(`${ART_CDN}Art/2DItems/Belts/Headhunter.png`);
  });
});

describe('ItemIcon', () => {
  it('shows the text fallback, named, when the art is unknown', async () => {
    setArtResolver((qs) => Promise.resolve(qs.map(() => '')));
    const { container, getByRole } = render(ItemIcon, { name: 'Doom Grip', base: 'Leather Belt', rarity: 'rare', size: 40 });
    await waitFor(() => expect(container.querySelector('[data-item-icon]')?.getAttribute('data-item-icon')).toBe('fallback'));
    expect(container.querySelector('img')).toBeNull();
    expect(getByRole('img', { name: 'Doom Grip' }).dataset.initials).toBe('LB');
    expect(container.textContent?.trim()).toBe('');
    const box = container.querySelector('[data-item-icon]') as HTMLElement;
    expect([box.style.width, box.style.height]).toEqual(['40px', '40px']);
  });

  it('sizes in rem when asked, so it scales with the root font like a rem-sized box around it', () => {
    setArtResolver((qs) => Promise.resolve(qs.map(() => '')));
    const { container } = render(ItemIcon, { name: 'Doom Grip', base: 'Leather Belt', size: 40, unit: 'rem' });
    const box = container.querySelector('[data-item-icon]') as HTMLElement;
    expect([box.style.width, box.style.height]).toEqual(['2.5rem', '2.5rem']);
    expect((box.firstElementChild as HTMLElement).style.fontSize).toBe('0.875rem');
  });

  it("names the image after the item and uses a unique's own name for its fallback", async () => {
    setArtResolver((qs) => Promise.resolve(qs.map(() => 'Art/2DItems/Belts/Headhunter')));
    const { container } = render(ItemIcon, { name: 'Headhunter', base: 'Leather Belt', rarity: 'unique' });
    await waitFor(() => expect(container.querySelector('img')).not.toBeNull());
    const img = container.querySelector('img') as HTMLImageElement;
    expect(img.getAttribute('src')).toBe(`${ART_CDN}Art/2DItems/Belts/Headhunter.png`);
    expect(img.alt).toBe('Headhunter');
    expect(img.getAttribute('loading')).toBe('lazy');
    expect((container.querySelector('[data-item-icon] > span') as HTMLElement).dataset.initials).toBe('H');
  });

  it('falls back to text when the image fails to load', async () => {
    const { container, getByRole } = render(ItemIcon, { name: 'Fireball', gem: true, src: '/missing.png' });
    const img = container.querySelector('img') as HTMLImageElement;
    img.dispatchEvent(new Event('error'));
    await waitFor(() => expect(container.querySelector('img')).toBeNull());
    expect(getByRole('img', { name: 'Fireball' }).dataset.initials).toBe('F');
  });

  it('is hidden from assistive tech when decorative', async () => {
    const { container } = render(ItemIcon, { name: 'Fireball', gem: true, decorative: true, src: '/x.png' });
    expect(container.querySelector('[data-item-icon]')?.getAttribute('aria-hidden')).toBe('true');
    expect((container.querySelector('img') as HTMLImageElement).alt).toBe('');
    container.querySelector('img')?.dispatchEvent(new Event('load'));
    await waitFor(() => expect(container.querySelector('[data-item-icon]')?.getAttribute('data-item-icon')).toBe('loaded'));
  });
});

describe('spriteOf', () => {
  it('cuts only the sheet shapes identified for flasks and gems', () => {
    expect(spriteOf('flask', 234, 156)).toBe(FLASK_SPRITE);
    expect(spriteOf('gem', 234, 78)).toBe(GEM_SPRITE);
    expect(spriteOf('gem', 78, 78)).toBeUndefined();
    expect(spriteOf('flask', 78, 156)).toBeUndefined();
    expect(spriteOf(undefined, 234, 156)).toBeUndefined();
    expect(spriteOf(undefined, 234, 78)).toBeUndefined();
    expect(spriteOf('gem', 234, 156)).toBeUndefined();
    expect(spriteOf('flask', 234, 78)).toBeUndefined();
    expect(spriteOf('flask', 0, 0)).toBeUndefined();
    expect(spriteOf('gem', 0, 0)).toBeUndefined();
  });
});

describe('ItemIcon sprite sheets', () => {
  const loadAs = (img: HTMLImageElement, w: number, h: number) => {
    Object.defineProperty(img, 'naturalWidth', { value: w });
    Object.defineProperty(img, 'naturalHeight', { value: h });
    img.dispatchEvent(new Event('load'));
  };
  const layers = (c: HTMLElement) =>
    [...c.querySelectorAll<HTMLElement>('[data-sprite-frame]')].map((l) => ({
      frame: l.dataset.spriteFrame,
      width: l.style.width,
      height: l.style.height,
      left: (l.firstElementChild as HTMLElement).style.left
    }));

  it('stacks a flask sheet back, liquid, front, each cropped to one 1x2 frame', async () => {
    const { container } = render(ItemIcon, { name: 'Divine Life Flask', base: 'Divine Life Flask', src: '/Art/2DItems/Flasks/lifeflask10.png' });
    loadAs(container.querySelector('img') as HTMLImageElement, 234, 156);
    await waitFor(() => expect(layers(container)).toHaveLength(3));
    expect(layers(container)).toEqual([
      { frame: '1', width: '50%', height: '100%', left: '-100%' },
      { frame: '2', width: '50%', height: '100%', left: '-200%' },
      { frame: '0', width: '50%', height: '100%', left: '0%' }
    ]);
    const sheet = container.querySelector('img') as HTMLImageElement;
    expect([sheet.alt, sheet.classList.contains('opacity-0')]).toEqual(['Divine Life Flask', true]);
    expect([...container.querySelectorAll('[data-sprite-frame] img')].every((i) => (i as HTMLImageElement).alt === '')).toBe(true);
  });

  it('stacks a gem sheet gem then symbol, and shows a single-frame gem whole', async () => {
    const { container } = render(ItemIcon, { name: 'Fireball', gem: true, src: '/g.png' });
    loadAs(container.querySelector('img') as HTMLImageElement, 234, 78);
    await waitFor(() =>
      expect(layers(container).map((l) => [l.frame, l.width, l.height])).toEqual([
        ['2', '100%', '100%'],
        ['0', '100%', '100%']
      ])
    );
    cleanup();
    const single = render(ItemIcon, { name: 'Added Fire Damage Support', gem: true, src: '/s.png' }).container;
    loadAs(single.querySelector('img') as HTMLImageElement, 78, 78);
    await waitFor(() => expect(single.querySelector('[data-item-icon]')?.getAttribute('data-item-icon')).toBe('loaded'));
    expect(layers(single)).toEqual([]);
    expect(single.querySelector('img')?.classList.contains('opacity-0')).toBe(false);
  });

  it('shows a sheet-shaped image whole when it is not flask or gem art', async () => {
    const { container } = render(ItemIcon, { name: 'Doom Grip', base: 'Leather Belt', rarity: 'rare', src: '/Art/2DItems/Belts/x.png' });
    loadAs(container.querySelector('img') as HTMLImageElement, 234, 156);
    await waitFor(() => expect(container.querySelector('[data-item-icon]')?.getAttribute('data-item-icon')).toBe('loaded'));
    expect(layers(container)).toEqual([]);
  });
});
