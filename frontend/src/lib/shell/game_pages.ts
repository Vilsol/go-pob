import type { Component } from 'svelte';
import type { Game } from '$lib/go/engine';

// 'route': the route's own page; a loader: another body for that game; absent: the game has no page there yet
type Body = 'route' | (() => Promise<{ default: Component<{ route: string }> }>);
export type GamePage = { label: string } & Partial<Record<Game, Body>>;

const poe2Page = () => import('./Poe2Page.svelte');

// every page route, in the top bar's order, and the body each game shows on it
export const GAME_PAGES: Record<string, GamePage> = {
  tree: { label: 'Tree', poe1: 'route', poe2: poe2Page },
  skills: { label: 'Skills', poe1: 'route', poe2: 'route' },
  items: { label: 'Items', poe1: 'route', poe2: 'route' },
  calcs: { label: 'Calcs', poe1: 'route', poe2: 'route' },
  configuration: { label: 'Config', poe1: 'route', poe2: 'route' },
  notes: { label: 'Notes', poe1: 'route', poe2: 'route' },
  // the PoE2 calculator keeps no party or compare state yet
  party: { label: 'Party', poe1: 'route' },
  compare: { label: 'Compare', poe1: 'route' },
  // a dialog, not a page: the top bar leaves it out, and the phone frame's More links its address, which opens it
  import: { label: 'Import / Export', poe1: 'route', poe2: 'route' }
};

export const hasPage = (game: Game, route: string) => !!GAME_PAGES[route]?.[game];
