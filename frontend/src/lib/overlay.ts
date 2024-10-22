import { get, writable } from 'svelte/store';
import type { Component } from 'svelte';

export interface OverlayConfig {
  component: Component<{ onclose: () => void }>;
  props?: Record<string, unknown>;
  backdropClose?: boolean;
}

export const overlays = writable<OverlayConfig[]>([]);

export const openOverlay = (newOverlay: OverlayConfig) => overlays.set([...get(overlays), newOverlay]);
