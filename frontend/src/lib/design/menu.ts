import type { Tone } from './tone';

// ItemTools.lua:380-409
export type MenuItem = {
  label: string;
  tone?: Tone;
  separator?: boolean;
  caption?: string;
  current?: boolean;
  disabled?: boolean;
  onselect?: () => void;
  onhover?: (el: HTMLElement | null) => void;
  wiki?: string;
  note?: string;
};
