export type PartyView = {
  buffers: Record<string, string>;
  summaries: Record<string, { group: string; name: string; value: string }[]>;
  destination: string;
  append: boolean;
  advanced: boolean;
  applied: boolean;
  disabled: boolean;
};

export type PartyMemberView = { name: string; className: string; ascendancy: string; level: number; active: boolean; gives: string[] };
