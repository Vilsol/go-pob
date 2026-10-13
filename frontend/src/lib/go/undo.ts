export type UndoStep = { action: string; page: string; at: number };

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

  record(source: string, label: { action: string; page: string }, snap: Snapshot, at: number, join = true): boolean {
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
    } else {
      this.entries.unshift({ ...label, source, snap, at });
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
    return { steps: this.entries.map(({ action, page, at }) => ({ action, page, at })), undone: this.undone, max: this.max };
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

export const STEP_LABELS: Record<string, { action: string; page: string }> = {
  SetLevel: { action: 'change level', page: 'Tree' },
  SetClass: { action: 'change class', page: 'Tree' },
  SetAscendancy: { action: 'change ascendancy', page: 'Tree' },
  AllocateNode: { action: 'allocate passives', page: 'Tree' },
  DeallocateNodes: { action: 'deallocate passives', page: 'Tree' },
  SetActiveTreeSpec: { action: 'switch the passive tree', page: 'Tree' },
  NewTreeSpec: { action: 'add a passive tree', page: 'Tree' },
  CopyTreeSpec: { action: 'copy a passive tree', page: 'Tree' },
  RenameTreeSpec: { action: 'rename a passive tree', page: 'Tree' },
  DeleteTreeSpec: { action: 'delete a passive tree', page: 'Tree' },
  SelectMasteryEffect: { action: 'choose a mastery effect', page: 'Tree' },
  SocketTreeJewel: { action: 'socket a jewel', page: 'Tree' },
  RemoveTattoo: { action: 'reset a tattooed node', page: 'Tree' },
  ChangeTreeAscendancy: { action: 'change ascendancy', page: 'Tree' },
  MoveTreeSpec: { action: 'move a passive tree', page: 'Tree' },
  ImportTreeLink: { action: 'import a tree link', page: 'Tree' },
  ConvertTreeSpec: { action: 'convert a passive tree', page: 'Tree' },
  ConvertAllTreeSpecs: { action: 'convert all passive trees', page: 'Tree' },
  ResetTreeNodes: { action: 'reset the tree', page: 'Tree' },
  RemoveAllTreeTattoos: { action: 'remove all tattoos', page: 'Tree' },
  SelectLoadout: { action: 'switch the loadout', page: 'Loadout' },
  NewLoadout: { action: 'add a loadout', page: 'Loadout' },
  'SetConfigOption: change': { action: 'change a config option', page: 'Config' },
  'SetConfigOption: remove': { action: 'reset a config option', page: 'Config' },
  ResetConfigOptions: { action: 'reset the config', page: 'Config' },
  SetCustomModifierBlocks: { action: 'edit custom modifiers', page: 'Config' },
  SetActiveConfigSet: { action: 'switch the config set', page: 'Config' },
  CopyCompareConfig: { action: "copy the compared build's config", page: 'Compare' },
  UseCompareConfig: { action: 'use a compared config option', page: 'Compare' },
  CopyCompareSpec: { action: "copy the compared build's tree", page: 'Compare' },
  CopyCompareItem: { action: "copy the compared build's item", page: 'Compare' },
  EquipItem: { action: 'equip an item', page: 'Items' },
  SaveItem: { action: 'add an item', page: 'Items' },
  ReplaceItem: { action: 'edit an item', page: 'Items' },
  UnequipSlot: { action: 'unequip an item', page: 'Items' },
  SetFlaskActive: { action: 'toggle a flask', page: 'Items' },
  SetActiveItemSet: { action: 'switch the item set', page: 'Items' },
  EquipItemID: { action: 'equip an item', page: 'Items' },
  SetMainSocketGroup: { action: 'change the main skill', page: 'Skills' },
  currentGemGroup: { action: 'edit a socket group', page: 'Skills' },
  socketGroupLabel: { action: 'rename a socket group', page: 'Skills' },
  socketGroups: { action: 'add or remove socket groups', page: 'Skills' },
  MoveSocketGroup: { action: 'move a socket group', page: 'Skills' },
  SetActiveSkillSet: { action: 'switch the skill set', page: 'Skills' },
  NewSkillSet: { action: 'add a skill set', page: 'Skills' },
  CopySkillSet: { action: 'copy a skill set', page: 'Skills' },
  RenameSkillSet: { action: 'rename a skill set', page: 'Skills' },
  DeleteSkillSet: { action: 'delete a skill set', page: 'Skills' },
  PasteSocketGroup: { action: 'paste a socket group', page: 'Skills' },
  SetSpectres: { action: 'change the spectre library', page: 'Skills' },
  SetCalcsBuffMode: { action: 'change the calculation mode', page: 'Calcs' },
  SetCalcsShowMinion: { action: 'show minion stats', page: 'Calcs' },
  SetCalcsSkillNumber: { action: 'change the socket group', page: 'Calcs' },
  SetMainActiveSkillCalcs: { action: 'change the active skill', page: 'Calcs' },
  SetGemSkillPartCalcs: { action: 'change the skill part', page: 'Calcs' },
  SetGemSkillStageCountCalcs: { action: 'change the stage count', page: 'Calcs' },
  SetGemSkillMineCountCalcs: { action: 'change the mine count', page: 'Calcs' },
  SetGemSkillMinionCalcs: { action: 'change the minion', page: 'Calcs' },
  SetGemSkillMinionItemSetCalcs: { action: 'change the minion', page: 'Calcs' },
  SetGemSkillMinionSkillCalcs: { action: 'change the minion skill', page: 'Calcs' },
  SetCharacterLevelAutoMode: { action: 'switch the level mode', page: 'Tree' },
  SetMainActiveSkill: { action: 'change the main active skill', page: 'Sidebar' },
  SetGemSkillPart: { action: 'change the skill part', page: 'Sidebar' },
  SetGemSkillStageCount: { action: 'change the stage count', page: 'Sidebar' },
  SetGemSkillMineCount: { action: 'change the mine count', page: 'Sidebar' },
  SetGemSkillMinion: { action: 'change the minion', page: 'Sidebar' },
  SetGemSkillMinionItemSet: { action: 'change the minion', page: 'Sidebar' },
  SetGemSkillMinionSkill: { action: 'change the minion skill', page: 'Sidebar' },
  SetWeaponSet: { action: 'switch the weapon set', page: 'Items' },
  NewItemSet: { action: 'add an item set', page: 'Items' },
  CopyItemSet: { action: 'copy an item set', page: 'Items' },
  RenameItemSet: { action: 'rename an item set', page: 'Items' },
  DeleteItemSet: { action: 'delete an item set', page: 'Items' },
  MoveItemSet: { action: 'move an item set', page: 'Items' },
  DeleteItem: { action: 'delete an item', page: 'Items' },
  DeleteUnusedItems: { action: 'delete unused items', page: 'Items' },
  DeleteAllItems: { action: 'delete all items', page: 'Items' },
  SortItems: { action: 'sort the items', page: 'Items' },
  ToggleEquipItem: { action: 'equip an item', page: 'Items' },
  AddDBItem: { action: 'add an item', page: 'Items' },
  DropItemOnSlot: { action: 'equip an item', page: 'Items' },
  EquipItemInSet: { action: 'equip an item on the minion', page: 'Sidebar' },
  ImportSharedItemSet: { action: 'import a shared item set', page: 'Items' },
  SetPartyBuffer: { action: 'edit a party buffer', page: 'Party' },
  SetPartyOptions: { action: 'change the party options', page: 'Party' },
  ImportParty: { action: 'import a party member', page: 'Party' },
  ClearParty: { action: 'clear the party', page: 'Party' },
  DisablePartyEffects: { action: 'disable party effects', page: 'Party' },
  RebuildParty: { action: 'rebuild the party', page: 'Party' },
  ToggleStatDifferences: { action: 'toggle stat differences', page: 'Items' },
  AddSharedItem: { action: 'add a shared item', page: 'Items' },
  MoveItem: { action: 'reorder the items', page: 'Items' },
  AddTimelessResult: { action: 'add a timeless jewel', page: 'Tree' },
  SetPartyMemberActive: { action: 'switch a party member', page: 'Party' },
  RemovePartyMember: { action: 'remove a party member', page: 'Party' },
  ChangeSecondaryAscendancy: { action: 'change secondary ascendancy', page: 'Tree' },
  OptimiseSockets: { action: 'optimise sockets', page: 'Skills' },
  EquipCompareJewel: { action: "socket the compared build's jewel", page: 'Compare' },
  exportParty: { action: 'toggle Export support', page: 'Import / Export' },
  AddTattoo: { action: 'tattoo a node', page: 'Tree' },
  ImportItemsAndSkills: { action: 'import items and skills', page: 'Import / Export' },
  ImportPassiveTreeAndJewels: { action: 'import the passive tree and jewels', page: 'Import / Export' },
  ImportAccountPassiveTree: { action: 'import the passive tree and jewels', page: 'Import / Export' },
  ImportAccountItems: { action: 'import items and skills', page: 'Import / Export' }
};

export const stepLabel = (source: string) => STEP_LABELS[source] ?? { action: source, page: '' };

export const UNRECORDED = 'Unrecorded';

const NOTES_SECTIONS = /<Notes>[\s\S]*?<\/Notes>|<Notes\/>|<RichNotes[\s\S]*?<\/RichNotes>|<RichNotes[^>]*\/>/g;

export const unrecordedLabel = (before: string, after: string) => {
  const rest = (xml: string) => xml.replace(NOTES_SECTIONS, '');
  return rest(before) === rest(after) ? { action: 'edit notes', page: 'Notes' } : { action: 'other changes', page: '' };
};
