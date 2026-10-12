import { describe, expect, it } from 'vitest';
import { windowTitle } from './window_title';

// Main.lua:1750-1756, PassiveSpec.lua:2302-2308
describe('the window title', () => {
  it('names the build and its ascendancy (or class), with a bloodline after a plus', () => {
    expect(windowTitle(true, { name: 'Fireball', className: 'Witch', ascendClassName: 'Elementalist' })).toBe('Fireball (Elementalist) - go-pob');
    expect(windowTitle(true, { name: 'Leveling', className: 'Witch', ascendClassName: 'None' })).toBe('Leveling (Witch) - go-pob');
    expect(windowTitle(true, { name: 'B', className: 'Ranger', ascendClassName: 'Warden', secondaryAscendancy: 'Oshabi' })).toBe(
      'B (Warden + Oshabi) - go-pob'
    );
  });
  it('is the app name with the option off or no build open', () => {
    expect(windowTitle(false, { name: 'Fireball', className: 'Witch', ascendClassName: 'Elementalist' })).toBe('go-pob');
    expect(windowTitle(true)).toBe('go-pob');
  });
});
