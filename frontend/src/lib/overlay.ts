import { get, writable } from 'svelte/store';
import type { SvelteComponent } from 'svelte';

export interface OverlayConfig {
  component: SvelteComponent;
  props?: Record<string, unknown>;
  backdropClose?: boolean;
}

export const overlays = writable<OverlayConfig[]>([]);

export const openOverlay = (newOverlay: OverlayConfig) => overlays.set([...get(overlays), newOverlay]);
