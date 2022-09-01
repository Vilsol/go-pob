export interface Coord {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Sprite {
  filename: string;
  w: number;
  h: number;
  coords: Coord;
}

export interface Sprites {
  background: { [key: string]: Sprite };
  normalActive: { [key: string]: Sprite };
  notableActive: { [key: string]: Sprite };
  keystoneActive: { [key: string]: Sprite };
  normalInactive: { [key: string]: Sprite };
  notableInactive: { [key: string]: Sprite };
  keystoneInactive: { [key: string]: Sprite };
  mastery: { [key: string]: Sprite };
  masteryConnected: { [key: string]: Sprite };
  masteryActiveSelected: { [key: string]: Sprite };
  masteryInactive: { [key: string]: Sprite };
  masteryActiveEffect: { [key: string]: Sprite };
  ascendancyBackground: { [key: string]: Sprite };
  ascendancy: { [key: string]: Sprite };
  startNode: { [key: string]: Sprite };
  groupBackground: { [key: string]: Sprite };
  frame: { [key: string]: Sprite };
  jewel: { [key: string]: Sprite };
  line: { [key: string]: Sprite };
  jewelRadius: { [key: string]: Sprite };
}

export interface ExpansionJewel {
  size: number;
  index: number;
  proxy: string;
  parent?: string;
}

export interface MasteryEffect {
  effect: number;
  stats: string[];
  reminderText?: string[];
}

export interface Points {
  totalPoints: number;
  ascendancyPoints: number;
}

export interface Node {
  skill?: number;
  name?: string;
  icon?: string;
  isNotable?: boolean;
  recipe?: string[];
  stats?: string[];
  group?: number;
  orbit?: number;
  orbitIndex?: number;
  out?: string[];
  in?: string[];
  reminderText?: string[];
  isMastery?: boolean;
  inactiveIcon?: string;
  activeIcon?: string;
  activeEffectImage?: string;
  masteryEffects?: MasteryEffect[];
  grantedStrength?: number;
  ascendancyName?: string;
  grantedDexterity?: number;
  isAscendancyStart?: boolean;
  isMultipleChoice?: boolean;
  grantedIntelligence?: number;
  isJewelSocket?: boolean;
  expansionJewel?: ExpansionJewel;
  grantedPassivePoints?: number;
  isKeystone?: boolean;
  flavourText?: string[];
  isProxy?: boolean;
  isMultipleChoiceOption?: boolean;
  isBlighted?: boolean;
  classStartIndex?: number;
}

export interface Group {
  x: number;
  y: number;
  orbits: number[];
  nodes: string[];
  isProxy?: boolean;
}

export interface ExtraImage {
  x: number;
  y: number;
  image: string;
}

export interface Classes {
  StrDexIntClass: number;
  StrClass: number;
  DexClass: number;
  IntClass: number;
  StrDexClass: number;
  StrIntClass: number;
  DexIntClass: number;
}

export interface CharacterAttributes {
  Strength: number;
  Dexterity: number;
  Intelligence: number;
}

export interface Constants {
  classes: Classes;
  characterAttributes: CharacterAttributes;
  PSSCentreInnerRadius: number;
  skillsPerOrbit: number[];
  orbitRadii: number[];
}

export interface FlavourTextRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Ascendancy {
  id: string;
  name: string;
  flavourText?: string;
  flavourTextColour?: string;
  flavourTextRect?: FlavourTextRect;
}

export interface Class {
  name: string;
  base_str: number;
  base_dex: number;
  base_int: number;
  ascendancies: Ascendancy[];
}

export interface Tree {
  tree: string;
  classes: Class[];
  groups: { [key: string]: Group };
  nodes: { [key: string]: Node };
  extraImages: { [key: string]: ExtraImage };
  jewelSlots: number[];
  min_x: number;
  min_y: number;
  max_x: number;
  max_y: number;
  constants: Constants;
  sprites: Sprites;
  imageZoomLevels: number[];
  points: Points;
}
