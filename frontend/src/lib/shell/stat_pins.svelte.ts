// the Sidebar's pinned stats, shared by the desktop rail and the phone stats strip, so a pin outlives the shell switch
export const statPins = $state<{ ids: string[] }>({ ids: [] });
