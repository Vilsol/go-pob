import type { StepDelta, StepKind, StepLabel } from './step_labels';

export type UndoStep = { action: string; page: string; kind?: StepKind; delta?: StepDelta[]; at: number };

export type UndoView = { steps: UndoStep[]; undone: number; max: number };

export type Snapshot = Uint32Array;

export class LineTable {
  private lines: string[] = [];
  private ids = new Map<string, number>();
  private compactedAt = 0;

  encode(xml: string): Snapshot {
    const parts = xml.split('\n');
    const out = new Uint32Array(parts.length);
    for (let i = 0; i < parts.length; i++) {
      const line = parts[i];
      let id = this.ids.get(line);
      if (id === undefined) {
        id = this.lines.length;
        this.lines.push(line);
        this.ids.set(line, id);
      }
      out[i] = id;
    }
    return out;
  }

  decode(snap: Snapshot): string {
    const parts = new Array<string>(snap.length);
    for (let i = 0; i < snap.length; i++) {
      parts[i] = this.lines[snap[i]];
    }
    return parts.join('\n');
  }

  get size() {
    return this.lines.length;
  }

  compact(live: Snapshot[]): Snapshot[] {
    if (this.lines.length <= 2 * this.compactedAt + 4096) {
      return live;
    }
    const remap = new Map<number, number>();
    const lines: string[] = [];
    const out = live.map((snap) => {
      const next = new Uint32Array(snap.length);
      for (let i = 0; i < snap.length; i++) {
        let id = remap.get(snap[i]);
        if (id === undefined) {
          id = lines.length;
          lines.push(this.lines[snap[i]]);
          remap.set(snap[i], id);
        }
        next[i] = id;
      }
      return next;
    });
    this.lines = lines;
    this.ids = new Map(lines.map((l, i) => [l, i]));
    this.compactedAt = lines.length;
    return out;
  }
}

export const sameSnapshot = (a: Snapshot, b: Snapshot) => {
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
};

export const COALESCE_SAME_MS = 1000;
export const COALESCE_ANY_MS = 300;

type Entry = UndoStep & { source: string; snap: Snapshot };

// a label is named from the state before the step, so a step that joins a burst renames itself
export type LabelOf = Omit<StepLabel, 'kind'> & { kind?: StepKind; delta?: StepDelta[] };
export type Describe = LabelOf | ((before: Snapshot) => LabelOf);

export class UndoHistory {
  private base?: Snapshot;
  private entries: Entry[] = [];
  private undone = 0;
  private joinable = false;

  constructor(
    readonly table = new LineTable(),
    readonly max = 50
  ) {}

  reset(snap: Snapshot) {
    this.base = snap;
    this.entries = [];
    this.undone = 0;
    this.joinable = false;
    this.compact();
  }

  clear() {
    this.base = undefined;
    this.entries = [];
    this.undone = 0;
    this.joinable = false;
  }

  get open() {
    return this.base !== undefined;
  }

  current(): Snapshot | undefined {
    return this.undone < this.entries.length ? this.entries[this.undone].snap : this.base;
  }

  record(source: string, label: Describe, snap: Snapshot, at: number, join = true): boolean {
    const now = this.current();
    if (!now || sameSnapshot(now, snap)) {
      return false;
    }
    if (this.undone > 0) {
      this.entries.splice(0, this.undone);
      this.undone = 0;
      this.joinable = false;
    }
    const top = this.entries[0];
    const joins = join && this.joinable && top !== undefined && (at - top.at < COALESCE_ANY_MS || (top.source === source && at - top.at < COALESCE_SAME_MS));
    if (joins) {
      top.snap = snap;
      top.at = at;
      if (top.source === source && typeof label === 'function') {
        Object.assign(top, label(this.entries[1]?.snap ?? this.base!));
      }
    } else {
      this.entries.unshift({ ...(typeof label === 'function' ? label(now) : label), source, snap, at });
      if (this.entries.length > this.max) {
        this.base = this.entries.pop()!.snap;
      }
    }
    this.joinable = join;
    this.compact();
    return true;
  }

  goTo(i: number): Snapshot | undefined {
    if (!this.base || i < 0 || i > this.entries.length) {
      return undefined;
    }
    this.undone = i;
    this.joinable = false;
    return this.current();
  }

  undo(): Snapshot | undefined {
    return this.undone < this.entries.length ? this.goTo(this.undone + 1) : undefined;
  }

  redo(): Snapshot | undefined {
    return this.undone > 0 ? this.goTo(this.undone - 1) : undefined;
  }

  view(): UndoView {
    return { steps: this.entries.map(({ action, page, kind, delta, at }) => ({ action, page, kind, delta, at })), undone: this.undone, max: this.max };
  }

  private compact() {
    const live = [...(this.base ? [this.base] : []), ...this.entries.map((e) => e.snap)];
    const next = this.table.compact(live);
    if (next === live) {
      return;
    }
    let k = 0;
    if (this.base) {
      this.base = next[k++];
    }
    for (const e of this.entries) {
      e.snap = next[k++];
    }
  }
}

/* ImportTab.lua:648-653 */
export const OPEN_SOURCES = new Set(['ImportCode', 'NewBuild', 'OpenBuild']);
// Build.lua:892-896
export const NOT_A_CHANGE = new Set(['setCalcTabElements', 'Undo', 'Redo', 'UndoTo', 'EstimatePlayerProgress']);

export const STEP_LABELS: Record<string, StepLabel> = {
  SetLevel: { action: 'Changed the character level', page: 'Tree', kind: 'tree' },
  SetClass: { action: 'Changed class', page: 'Tree', kind: 'tree' },
  SetAscendancy: { action: 'Changed ascendancy', page: 'Tree', kind: 'tree' },
  AllocateNode: { action: 'Allocated passives', page: 'Tree', kind: 'tree' },
  DeallocateNodes: { action: 'Deallocated passives', page: 'Tree', kind: 'tree' },
  SetActiveTreeSpec: { action: 'Switched the passive tree', page: 'Tree', kind: 'set' },
  NewTreeSpec: { action: 'Added a passive tree', page: 'Tree', kind: 'set' },
  CopyTreeSpec: { action: 'Copied a passive tree', page: 'Tree', kind: 'set' },
  RenameTreeSpec: { action: 'Renamed a passive tree', page: 'Tree', kind: 'set' },
  DeleteTreeSpec: { action: 'Deleted a passive tree', page: 'Tree', kind: 'set' },
  SelectMasteryEffect: { action: 'Chose a mastery effect', page: 'Tree', kind: 'tree' },
  SocketTreeJewel: { action: 'Socketed a jewel', page: 'Tree', kind: 'item' },
  RemoveTattoo: { action: 'Reset a tattooed node', page: 'Tree', kind: 'tree' },
  ChangeTreeAscendancy: { action: 'Changed ascendancy', page: 'Tree', kind: 'tree' },
  ChangeTreeClass: { action: 'Changed class', page: 'Tree', kind: 'tree' },
  ClickTreeAscendancyNode: { action: 'Changed ascendancy', page: 'Tree', kind: 'tree' },
  MoveTreeSpec: { action: 'Moved a passive tree', page: 'Tree', kind: 'set' },
  ImportTreeLink: { action: 'Imported a tree link', page: 'Tree', kind: 'import' },
  ConvertTreeSpec: { action: 'Converted a passive tree', page: 'Tree', kind: 'tree' },
  ConvertAllTreeSpecs: { action: 'Converted all passive trees', page: 'Tree', kind: 'tree' },
  ResetTreeNodes: { action: 'Reset the passive tree', page: 'Tree', kind: 'tree' },
  RemoveAllTreeTattoos: { action: 'Removed all tattoos', page: 'Tree', kind: 'tree' },
  SelectLoadout: { action: 'Switched the loadout', page: 'Loadout', kind: 'set' },
  NewLoadout: { action: 'Added a loadout', page: 'Loadout', kind: 'set' },
  'SetConfigOption: change': { action: 'Changed a config option', page: 'Config', kind: 'config' },
  'SetConfigOption: remove': { action: 'Reset a config option', page: 'Config', kind: 'config' },
  ResetConfigOptions: { action: 'Reset the config', page: 'Config', kind: 'config' },
  SetCustomModifierBlocks: { action: 'Edited custom modifiers', page: 'Config', kind: 'config' },
  SetActiveConfigSet: { action: 'Switched the config set', page: 'Config', kind: 'set' },
  'EditConfigSets: new': { action: 'Added a config set', page: 'Config', kind: 'set' },
  'EditConfigSets: copy': { action: 'Copied a config set', page: 'Config', kind: 'set' },
  'EditConfigSets: rename': { action: 'Renamed a config set', page: 'Config', kind: 'set' },
  'EditConfigSets: delete': { action: 'Deleted a config set', page: 'Config', kind: 'set' },
  'EditConfigSets: move': { action: 'Moved a config set', page: 'Config', kind: 'set' },
  CopyCompareConfig: { action: "Copied the compared build's config", page: 'Compare', kind: 'config' },
  UseCompareConfig: { action: 'Used a compared config option', page: 'Compare', kind: 'config' },
  CopyCompareSpec: { action: "Copied the compared build's tree", page: 'Compare', kind: 'tree' },
  CopyCompareItem: { action: "Copied the compared build's item", page: 'Compare', kind: 'item' },
  EquipItem: { action: 'Equipped an item', page: 'Items', kind: 'item' },
  SaveItem: { action: 'Added an item', page: 'Items', kind: 'item' },
  ReplaceItem: { action: 'Edited an item', page: 'Items', kind: 'item' },
  UnequipSlot: { action: 'Unequipped an item', page: 'Items', kind: 'item' },
  SetFlaskActive: { action: 'Toggled a flask', page: 'Items', kind: 'item' },
  SetActiveItemSet: { action: 'Switched the item set', page: 'Items', kind: 'set' },
  EquipItemID: { action: 'Equipped an item', page: 'Items', kind: 'item' },
  SetMainSocketGroup: { action: 'Changed the main skill', page: 'Skills', kind: 'gem' },
  currentGemGroup: { action: 'Edited a socket group', page: 'Skills', kind: 'gem' },
  socketGroupLabel: { action: 'Renamed a socket group', page: 'Skills', kind: 'gem' },
  socketGroups: { action: 'Added or removed socket groups', page: 'Skills', kind: 'gem' },
  MoveSocketGroup: { action: 'Moved a socket group', page: 'Skills', kind: 'gem' },
  SetActiveSkillSet: { action: 'Switched the skill set', page: 'Skills', kind: 'set' },
  NewSkillSet: { action: 'Added a skill set', page: 'Skills', kind: 'set' },
  CopySkillSet: { action: 'Copied a skill set', page: 'Skills', kind: 'set' },
  RenameSkillSet: { action: 'Renamed a skill set', page: 'Skills', kind: 'set' },
  DeleteSkillSet: { action: 'Deleted a skill set', page: 'Skills', kind: 'set' },
  PasteSocketGroup: { action: 'Pasted a socket group', page: 'Skills', kind: 'gem' },
  SetSpectres: { action: 'Changed the spectre library', page: 'Skills', kind: 'gem' },
  SetCalcsBuffMode: { action: 'Changed the calculation mode', page: 'Calcs', kind: 'calcs' },
  SetCalcsShowMinion: { action: 'Toggled minion stats', page: 'Calcs', kind: 'calcs' },
  SetCalcsSkillNumber: { action: 'Changed the socket group', page: 'Calcs', kind: 'calcs' },
  SetMainActiveSkillCalcs: { action: 'Changed the active skill', page: 'Calcs', kind: 'calcs' },
  SetGemSkillPartCalcs: { action: 'Changed the skill part', page: 'Calcs', kind: 'calcs' },
  SetGemSkillStageCountCalcs: { action: 'Changed the stage count', page: 'Calcs', kind: 'calcs' },
  SetGemSkillMineCountCalcs: { action: 'Changed the mine count', page: 'Calcs', kind: 'calcs' },
  SetGemSkillMinionCalcs: { action: 'Changed the minion', page: 'Calcs', kind: 'calcs' },
  SetGemSkillMinionItemSetCalcs: { action: 'Changed the minion', page: 'Calcs', kind: 'calcs' },
  SetGemSkillMinionSkillCalcs: { action: 'Changed the minion skill', page: 'Calcs', kind: 'calcs' },
  SetCharacterLevelAutoMode: { action: 'Switched the level mode', page: 'Tree', kind: 'tree' },
  SetMainActiveSkill: { action: 'Changed the main active skill', page: 'Sidebar', kind: 'gem' },
  SetGemSkillPart: { action: 'Changed the skill part', page: 'Sidebar', kind: 'gem' },
  SetGemSkillStageCount: { action: 'Changed the stage count', page: 'Sidebar', kind: 'gem' },
  SetGemSkillMineCount: { action: 'Changed the mine count', page: 'Sidebar', kind: 'gem' },
  SetGemSkillMinion: { action: 'Changed the minion', page: 'Sidebar', kind: 'gem' },
  SetGemSkillMinionItemSet: { action: 'Changed the minion', page: 'Sidebar', kind: 'gem' },
  SetGemSkillMinionSkill: { action: 'Changed the minion skill', page: 'Sidebar', kind: 'gem' },
  SetWeaponSet: { action: 'Switched the weapon set', page: 'Items', kind: 'set' },
  NewItemSet: { action: 'Added an item set', page: 'Items', kind: 'set' },
  CopyItemSet: { action: 'Copied an item set', page: 'Items', kind: 'set' },
  RenameItemSet: { action: 'Renamed an item set', page: 'Items', kind: 'set' },
  DeleteItemSet: { action: 'Deleted an item set', page: 'Items', kind: 'set' },
  MoveItemSet: { action: 'Moved an item set', page: 'Items', kind: 'set' },
  DeleteItem: { action: 'Deleted an item', page: 'Items', kind: 'item' },
  DeleteUnusedItems: { action: 'Deleted unused items', page: 'Items', kind: 'item' },
  DeleteAllItems: { action: 'Deleted all items', page: 'Items', kind: 'item' },
  SortItems: { action: 'Sorted the items', page: 'Items', kind: 'item' },
  ToggleEquipItem: { action: 'Equipped an item', page: 'Items', kind: 'item' },
  AddDBItem: { action: 'Added an item', page: 'Items', kind: 'item' },
  DropItemOnSlot: { action: 'Equipped an item', page: 'Items', kind: 'item' },
  EquipItemInSet: { action: 'Equipped an item on the minion', page: 'Sidebar', kind: 'item' },
  ImportSharedItemSet: { action: 'Imported a shared item set', page: 'Items', kind: 'set' },
  SetPartyBuffer: { action: 'Edited a party buffer', page: 'Party', kind: 'party' },
  SetPartyOptions: { action: 'Changed the party options', page: 'Party', kind: 'party' },
  ImportParty: { action: 'Imported a party member', page: 'Party', kind: 'party' },
  ClearParty: { action: 'Cleared the party', page: 'Party', kind: 'party' },
  DisablePartyEffects: { action: 'Disabled party effects', page: 'Party', kind: 'party' },
  RebuildParty: { action: 'Rebuilt the party', page: 'Party', kind: 'party' },
  ToggleStatDifferences: { action: 'Toggled stat differences', page: 'Items', kind: 'item' },
  AddSharedItem: { action: 'Added a shared item', page: 'Items', kind: 'item' },
  MoveItem: { action: 'Reordered the items', page: 'Items', kind: 'item' },
  AddTimelessResult: { action: 'Added a timeless jewel', page: 'Tree', kind: 'item' },
  SetPartyMemberActive: { action: 'Switched a party member', page: 'Party', kind: 'party' },
  RemovePartyMember: { action: 'Removed a party member', page: 'Party', kind: 'party' },
  ChangeSecondaryAscendancy: { action: 'Changed the secondary ascendancy', page: 'Tree', kind: 'tree' },
  OptimiseSockets: { action: 'Optimised sockets', page: 'Skills', kind: 'gem' },
  EquipCompareJewel: { action: "Socketed the compared build's jewel", page: 'Compare', kind: 'item' },
  exportParty: { action: 'Toggled Export support', page: 'Import / Export', kind: 'party' },
  AddTattoo: { action: 'Tattooed a node', page: 'Tree', kind: 'tree' },
  ImportItemsAndSkills: { action: 'Imported items and skills', page: 'Import / Export', kind: 'import' },
  ImportPassiveTreeAndJewels: { action: 'Imported the passive tree and jewels', page: 'Import / Export', kind: 'import' },
  ImportAccountPassiveTree: { action: 'Imported the passive tree and jewels', page: 'Import / Export', kind: 'import' },
  ImportAccountItems: { action: 'Imported items and skills', page: 'Import / Export', kind: 'import' }
};

// the steps named by what changed between the states before and after, rather than by the write's arguments
export const DIFF_LABELS: Record<string, 'tree' | 'skills'> = {
  AllocateNode: 'tree',
  DeallocateNodes: 'tree',
  currentGemGroup: 'skills',
  socketGroupLabel: 'skills',
  socketGroups: 'skills',
  PasteSocketGroup: 'skills',
  MoveSocketGroup: 'skills'
};

export const stepLabel = (source: string): StepLabel => STEP_LABELS[source] ?? { action: source, page: '', kind: 'other' };

export const UNRECORDED = 'Unrecorded';

const NOTES_SECTIONS = /<Notes>[\s\S]*?<\/Notes>|<Notes\/>|<RichNotes[\s\S]*?<\/RichNotes>|<RichNotes[^>]*\/>/g;

export const unrecordedLabel = (before: string, after: string): StepLabel => {
  const rest = (xml: string) => xml.replace(NOTES_SECTIONS, '');
  return rest(before) === rest(after) ? { action: 'Edited the notes', page: 'Notes', kind: 'notes' } : { action: 'Other changes', page: '', kind: 'other' };
};
