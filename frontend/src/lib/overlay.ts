import { get, writable } from 'svelte/store';
import type { Component } from 'svelte';

export interface OverlayConfig<TCompProps = object> {
  component: Component<{ onclose: () => void } & TCompProps>;
  props?: TCompProps;
  backdropClose?: boolean;
}

export const overlays = writable<OverlayConfig<object>[]>([]);

export const openOverlay = <T = object>(newOverlay: OverlayConfig<T>) => overlays.set([...get(overlays), newOverlay]);
