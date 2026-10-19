// for a Tabs caller whose panel lives outside the tab strip: pass `id`, render the panel with these
export const tabId = (base: string, id: string) => `${base}-tab-${id}`;
export const panelId = (base: string) => `${base}-panel`;
