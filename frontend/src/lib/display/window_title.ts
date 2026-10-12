import { writable } from 'svelte/store';

/* Launch.lua:9 */
export const APP_NAME = 'go-pob';

const KEY = 'options:showTitlebarName';
const load = () => {
  try {
    return globalThis.localStorage?.getItem(KEY) !== 'false';
  } catch {
    return true;
  }
};

/* Main.lua:1080-1082, Main.lua:111 */
export const showTitlebarName = writable<boolean>(load());
showTitlebarName.subscribe((v) => {
  try {
    globalThis.localStorage?.setItem(KEY, String(v));
  } catch {
    // empty
  }
});

/* Main.lua:1750-1756, PassiveSpec.lua:2302-2308 */
export const windowTitle = (show: boolean, build?: { name: string; className: string; ascendClassName: string; secondaryAscendancy?: string }): string => {
  if (!show || !build) return APP_NAME;
  let classText = !build.ascendClassName || build.ascendClassName === 'None' ? build.className : build.ascendClassName;
  if (build.secondaryAscendancy) classText += ' + ' + build.secondaryAscendancy;
  return `${build.name} (${classText}) - ${APP_NAME}`;
};
