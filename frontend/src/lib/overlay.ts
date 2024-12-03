import { get, writable } from 'svelte/store';
import type { Component } from 'svelte';

export interface OverlayConfig<TCompProps = {}> {
  component: Component<{ onclose: () => void } & TCompProps>;
  props?: TCompProps;
  backdropClose?: boolean;
}

export const overlays = writable<OverlayConfig<any>[]>([]);

export const openOverlay = <T = {}>(newOverlay: OverlayConfig<T>) => overlays.set([...get(overlays), newOverlay]);
