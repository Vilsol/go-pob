package calculator

import (
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type CompiledList[T any] struct {
	Regex *regexp.Regexp
	Value T
}

/*
local function firstToUpper(str)
	return (str:gsub("^%l", string.upper))
end
*/

type ConquerorType struct {
	ID   string
	Type string
}

var conquerorList = map[string]ConquerorType{
	"xibaqua":  {ID: "1", Type: "vaal"},
	"zerphi":   {ID: "2", Type: "vaal"},
	"doryani":  {ID: "3", Type: "vaal"},
	"ahuana":   {ID: "2_v2", Type: "vaal"},
	"deshret":  {ID: "1", Type: "maraketh"},
	"asenath":  {ID: "2", Type: "maraketh"},
	"nasima":   {ID: "3", Type: "maraketh"},
	"balbala":  {ID: "1_v2", Type: "maraketh"},
	"cadiro":   {ID: "1", Type: "eternal"},
	"victario": {ID: "2", Type: "eternal"},
	"chitus":   {ID: "3", Type: "eternal"},
	"caspiro":  {ID: "3_v2", Type: "eternal"},
	"kaom":     {ID: "1", Type: "karui"},
	"rakiata":  {ID: "2", Type: "karui"},
	"kiloava":  {ID: "3", Type: "karui"},
	"akoya":    {ID: "3_v2", Type: "karui"},
	"venarius": {ID: "1", Type: "templar"},
	"dominus":  {ID: "2", Type: "templar"},
	"avarius":  {ID: "3", Type: "templar"},
	"maxarius": {ID: "1_v2", Type: "templar"},
}

// List of modifier forms
var formListCompiled map[string]CompiledList[string]
var formList = map[string]string{
	`^(\d+)% increased`:                                        "INC",
	`^(\d+)% faster`:                                           "INC",
	`^(\d+)% reduced`:                                          "RED",
	`^(\d+)% slower`:                                           "RED",
	`^(\d+)% more`:                                             "MORE",
	`^(\d+)% less`:                                             "LESS",
	`^([\+\-][\d\.]+)%?`:                                       "BASE",
	`^([\+\-][\d\.]+)%? to`:                                    "BASE",
	`^([\+\-]?[\d\.]+)%? of`:                                   "BASE",
	`^([\+\-][\d\.]+)%? base`:                                  "BASE",
	`^([\+\-]?[\d\.]+)%? additional`:                           "BASE",
	`(\d+) additional hits?`:                                   "BASE",
	`^you gain ([\d\.]+)`:                                      "BASE",
	`^gains? ([\d\.]+)% of`:                                    "BASE",
	`^([\+\-]?\d+)% chance`:                                    "CHANCE",
	`^([\+\-]?\d+)% additional chance`:                         "CHANCE",
	`penetrates? (\d+)%`:                                       "PEN",
	`penetrates (\d+)% of`:                                     "PEN",
	`penetrates (\d+)% of enemy`:                               "PEN",
	`^([\d\.]+) (\S+?) regenerated per second`:                 "REGENFLAT",
	`^([\d\.]+)% (?:of )?(\S+?) regenerated per second`:        "REGENPERCENT",
	`^regenerate ([\d\.]+) (\S+?) per second`:                  "REGENFLAT",
	`^regenerate ([\d\.]+)% (?:of |of your )(\S+?) per second`: "REGENPERCENT",
	`^you regenerate ([\d\.]+)% of (\S+?) per second`:          "REGENPERCENT",
	`^([\d\.]+) (\w+) damage taken per second`:                 "DEGEN",
	`^([\d\.]+) (\w+) damage per second`:                       "DEGEN",
	`(\d+) to (\d+) added (\w+) damage`:                        "DMG",
	`(\d+)\-(\d+) added (\w+) damage`:                          "DMG",
	`(\d+) to (\d+) additional (\w+) damage`:                   "DMG",
	`(\d+)\-(\d+) additional (\w+) damage`:                     "DMG",
	`^(\d+) to (\d+) (\w+) damage`:                             "DMG",
	`adds (\d+) to (\d+) (\w+) damage`:                         "DMG",
	`adds (\d+)\-(\d+) (\w+) damage`:                           "DMG",
	`adds (\d+) to (\d+) (\w+) damage to attacks`:              "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) damage to attacks`:                "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) attack damage`:                  "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) attack damage`:                    "DMGATTACKS",
	`(\d+) to (\d+) added attack (\w+) damage`:                 "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) damage to spells`:               "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) damage to spells`:                 "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) spell damage`:                   "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) spell damage`:                     "DMGSPELLS",
	`(\d+) to (\d+) added spell (\w+) damage`:                  "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) damage to attacks and spells`:   "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to attacks and spells`:     "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to spells and attacks`:   "DMGBOTH", // o_O
	`adds (\d+)\-(\d+) (\w+) damage to spells and attacks`:     "DMGBOTH", // o_O
	`adds (\d+) to (\d+) (\w+) damage to hits`:                 "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to hits`:                   "DMGBOTH",
	`^you have `:                                               "FLAG",
	`^you are `:                                                "FLAG",
	`^are `:                                                    "FLAG",
}

type modNameListType struct {
	names             []string
	tag               mod.Tag
	tagList           []mod.Tag
	flags             mod.MFlag
	addToMinion       bool
	keywordFlags      mod.KeywordFlag
	addToSkill        mod.Tag
	addToMinionTag    mod.Tag
	applyToEnemy      bool
	fn                func(caps []string) modNameListType
	modSuffix         string
	addToAura         bool
	newAuraOnlyAllies bool
	newAura           bool
	affectedByAura    bool
}

// Map of modifier names
var modNameListCompiled map[string]CompiledList[modNameListType]
var modNameList = map[string]modNameListType{
	// Attributes
	"strength":                   {names: []string{"Str"}},
	"dexterity":                  {names: []string{"Dex"}},
	"intelligence":               {names: []string{"Int"}},
	"omniscience":                {names: []string{"Omni"}},
	"strength and dexterity":     {names: []string{"Str", "Dex", "StrDex"}},
	"strength and intelligence":  {names: []string{"Str", "Int", "StrInt"}},
	"dexterity and intelligence": {names: []string{"Dex", "Int", "DexInt"}},
	"attributes":                 {names: []string{"Str", "Dex", "Int", "All"}},
	"all attributes":             {names: []string{"Str", "Dex", "Int", "All"}},
	"devotion":                   {names: []string{"Devotion"}},

	// Life/mana
	"life":                                  {names: []string{"Life"}},
	"maximum life":                          {names: []string{"Life"}},
	"life regeneration rate":                {names: []string{"LifeRegen"}},
	"mana":                                  {names: []string{"Mana"}},
	"maximum mana":                          {names: []string{"Mana"}},
	"mana regeneration":                     {names: []string{"ManaRegen"}},
	"mana regeneration rate":                {names: []string{"ManaRegen"}},
	"mana cost":                             {names: []string{"ManaCost"}},
	"mana cost of":                          {names: []string{"ManaCost"}},
	"mana cost of skills":                   {names: []string{"ManaCost"}},
	"mana cost of attacks":                  {names: []string{"ManaCost"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"total cost":                            {names: []string{"Cost"}},
	"total mana cost":                       {names: []string{"ManaCost"}},
	"total mana cost of skills":             {names: []string{"ManaCost"}},
	"life cost of skills":                   {names: []string{"LifeCost"}},
	"rage cost of skills":                   {names: []string{"RageCost"}},
	"cost of":                               {names: []string{"Cost"}},
	"cost of skills":                        {names: []string{"Cost"}},
	"mana reserved":                         {names: []string{"ManaReserved"}},
	"mana reservation":                      {names: []string{"ManaReserved"}},
	"mana reservation of skills":            {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"mana reservation efficiency of skills": {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency of skills": {names: []string{"LifeReservationEfficiency"}},
	"reservation of skills":                 {names: []string{"Reserved"}},
	"mana reservation if cast as an aura":   {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation if cast as an aura":        {names: []string{"Reserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation":                           {names: []string{"Reserved"}},
	"reservation efficiency":                {names: []string{"ReservationEfficiency"}},
	"reservation efficiency of skills":      {names: []string{"ReservationEfficiency"}},
	"mana reservation efficiency":           {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency":           {names: []string{"LifeReservationEfficiency"}},

	// Primary defences
	"maximum energy shield":              {names: []string{"EnergyShield"}},
	"energy shield recharge rate":        {names: []string{"EnergyShieldRecharge"}},
	"start of energy shield recharge":    {names: []string{"EnergyShieldRechargeFaster"}},
	"restoration of ward":                {names: []string{"WardRechargeFaster"}},
	"armour":                             {names: []string{"Armour"}},
	"evasion":                            {names: []string{"Evasion"}},
	"evasion rating":                     {names: []string{"Evasion"}},
	"energy shield":                      {names: []string{"EnergyShield"}},
	"ward":                               {names: []string{"Ward"}},
	"armour and evasion":                 {names: []string{"ArmourAndEvasion"}},
	"armour and evasion rating":          {names: []string{"ArmourAndEvasion"}},
	"evasion rating and armour":          {names: []string{"ArmourAndEvasion"}},
	"armour and energy shield":           {names: []string{"ArmourAndEnergyShield"}},
	"evasion rating and energy shield":   {names: []string{"EvasionAndEnergyShield"}},
	"evasion and energy shield":          {names: []string{"EvasionAndEnergyShield"}},
	"armour, evasion and energy shield":  {names: []string{"Defences"}},
	"defences":                           {names: []string{"Defences"}},
	"to evade":                           {names: []string{"EvadeChance"}},
	"chance to evade":                    {names: []string{"EvadeChance"}},
	"to evade attacks":                   {names: []string{"EvadeChance"}},
	"to evade attack hits":               {names: []string{"EvadeChance"}},
	"chance to evade attacks":            {names: []string{"EvadeChance"}},
	"chance to evade attack hits":        {names: []string{"EvadeChance"}},
	"chance to evade projectile attacks": {names: []string{"ProjectileEvadeChance"}},
	"chance to evade melee attacks":      {names: []string{"MeleeEvadeChance"}},

	// Resistances
	"physical damage reduction":                                   {names: []string{"PhysicalDamageReduction"}},
	"physical damage reduction from hits":                         {names: []string{"PhysicalDamageReductionWhenHit"}},
	"fire resistance":                                             {names: []string{"FireResist"}},
	"maximum fire resistance":                                     {names: []string{"FireResistMax"}},
	"cold resistance":                                             {names: []string{"ColdResist"}},
	"maximum cold resistance":                                     {names: []string{"ColdResistMax"}},
	"lightning resistance":                                        {names: []string{"LightningResist"}},
	"maximum lightning resistance":                                {names: []string{"LightningResistMax"}},
	"chaos resistance":                                            {names: []string{"ChaosResist"}},
	"maximum chaos resistance":                                    {names: []string{"ChaosResistMax"}},
	"fire and cold resistances":                                   {names: []string{"FireResist", "ColdResist"}},
	"fire and lightning resistances":                              {names: []string{"FireResist", "LightningResist"}},
	"cold and lightning resistances":                              {names: []string{"ColdResist", "LightningResist"}},
	"elemental resistance":                                        {names: []string{"ElementalResist"}},
	"elemental resistances":                                       {names: []string{"ElementalResist"}},
	"all elemental resistances":                                   {names: []string{"ElementalResist"}},
	"all resistances":                                             {names: []string{"ElementalResist", "ChaosResist"}},
	"all maximum elemental resistances":                           {names: []string{"ElementalResistMax"}},
	"all maximum resistances":                                     {names: []string{"ElementalResistMax", "ChaosResistMax"}},
	"all elemental resistances and maximum elemental resistances": {names: []string{"ElementalResist", "ElementalResistMax"}},
	"fire and chaos resistances":                                  {names: []string{"FireResist", "ChaosResist"}},
	"cold and chaos resistances":                                  {names: []string{"ColdResist", "ChaosResist"}},
	"lightning and chaos resistances":                             {names: []string{"LightningResist", "ChaosResist"}},

	// Damage taken
	"damage taken":                           {names: []string{"DamageTaken"}},
	"damage taken when hit":                  {names: []string{"DamageTakenWhenHit"}},
	"damage over time taken":                 {names: []string{"DamageTakenOverTime"}},
	"damage taken from damage over time":     {names: []string{"DamageTakenOverTime"}},
	"attack damage taken":                    {names: []string{"AttackDamageTaken"}},
	"spell damage taken":                     {names: []string{"SpellDamageTaken"}},
	"physical damage taken":                  {names: []string{"PhysicalDamageTaken"}},
	"physical damage from hits taken":        {names: []string{"PhysicalDamageTaken"}},
	"physical damage taken when hit":         {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from hits":        {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from attacks":     {names: []string{"PhysicalDamageTakenFromAttacks"}},
	"physical damage taken over time":        {names: []string{"PhysicalDamageTakenOverTime"}},
	"physical damage over time damage taken": {names: []string{"PhysicalDamageTakenOverTime"}},
	"reflected physical damage taken":        {names: []string{"PhysicalReflectedDamageTaken"}},
	"lightning damage taken":                 {names: []string{"LightningDamageTaken"}},
	"lightning damage from hits taken":       {names: []string{"LightningDamageTaken"}},
	"lightning damage taken when hit":        {names: []string{"LightningDamageTakenWhenHit"}},
	"lightning damage taken from attacks":    {names: []string{"LightningDamageTakenFromAttacks"}},
	"lightning damage taken over time":       {names: []string{"LightningDamageTakenOverTime"}},
	"cold damage taken":                      {names: []string{"ColdDamageTaken"}},
	"cold damage from hits taken":            {names: []string{"ColdDamageTaken"}},
	"cold damage taken when hit":             {names: []string{"ColdDamageTakenWhenHit"}},
	"cold damage taken from attacks":         {names: []string{"ColdDamageTakenFromAttacks"}},
	"cold damage taken over time":            {names: []string{"ColdDamageTakenOverTime"}},
	"fire damage taken":                      {names: []string{"FireDamageTaken"}},
	"fire damage from hits taken":            {names: []string{"FireDamageTaken"}},
	"fire damage taken when hit":             {names: []string{"FireDamageTakenWhenHit"}},
	"fire damage taken from attacks":         {names: []string{"FireDamageTakenFromAttacks"}},
	"fire damage taken over time":            {names: []string{"FireDamageTakenOverTime"}},
	"chaos damage taken":                     {names: []string{"ChaosDamageTaken"}},
	"chaos damage from hits taken":           {names: []string{"ChaosDamageTaken"}},
	"chaos damage taken when hit":            {names: []string{"ChaosDamageTakenWhenHit"}},
	"chaos damage taken from attacks":        {names: []string{"ChaosDamageTakenFromAttacks"}},
	"chaos damage taken over time":           {names: []string{"ChaosDamageTakenOverTime"}},
	"chaos damage over time taken":           {names: []string{"ChaosDamageTakenOverTime"}},
	"elemental damage taken":                 {names: []string{"ElementalDamageTaken"}},
	"elemental damage from hits taken":       {names: []string{"ElementalDamageTaken"}},
	"elemental damage taken when hit":        {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken from hits":       {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken over time":       {names: []string{"ElementalDamageTakenOverTime"}},
	"cold and lightning damage taken":        {names: []string{"ColdDamageTaken", "LightningDamageTaken"}},
	"fire and lightning damage taken":        {names: []string{"FireDamageTaken", "LightningDamageTaken"}},
	"fire and cold damage taken":             {names: []string{"FireDamageTaken", "ColdDamageTaken"}},
	"physical and chaos damage taken":        {names: []string{"PhysicalDamageTaken", "ChaosDamageTaken"}},
	"reflected elemental damage taken":       {names: []string{"ElementalReflectedDamageTaken"}},

	// Other defences
	"to dodge attacks":                            {names: []string{"AttackDodgeChance"}},
	"to dodge attack hits":                        {names: []string{"AttackDodgeChance"}},
	"to dodge spells":                             {names: []string{"SpellDodgeChance"}},
	"to dodge spell hits":                         {names: []string{"SpellDodgeChance"}},
	"to dodge spell damage":                       {names: []string{"SpellDodgeChance"}},
	"to dodge attacks and spells":                 {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attacks and spell damage":           {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack and spell hits":              {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack or spell hits":               {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to suppress spell damage":                    {names: []string{"SpellSuppressionChance"}},
	"amount of suppressed spell damage prevented": {names: []string{"SpellSuppressionEffect"}},
	"to block":                                        {names: []string{"BlockChance"}},
	"to block attacks":                                {names: []string{"BlockChance"}},
	"to block attack damage":                          {names: []string{"BlockChance"}},
	"block chance":                                    {names: []string{"BlockChance"}},
	"block chance with staves":                        {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"to block with staves":                            {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"block chance against projectiles":                {names: []string{"ProjectileBlockChance"}},
	"to block projectile attack damage":               {names: []string{"ProjectileBlockChance"}},
	"spell block chance":                              {names: []string{"SpellBlockChance"}},
	"to block spells":                                 {names: []string{"SpellBlockChance"}},
	"to block spell damage":                           {names: []string{"SpellBlockChance"}},
	"chance to block attacks and spells":              {names: []string{"BlockChance", "SpellBlockChance"}},
	"to block attack and spell damage":                {names: []string{"BlockChance", "SpellBlockChance"}},
	"maximum block chance":                            {names: []string{"BlockChanceMax"}},
	"maximum chance to block attack damage":           {names: []string{"BlockChanceMax"}},
	"maximum chance to block spell damage":            {names: []string{"SpellBlockChanceMax"}},
	"life gained when you block":                      {names: []string{"LifeOnBlock"}},
	"mana gained when you block":                      {names: []string{"ManaOnBlock"}},
	"maximum chance to dodge spell hits":              {names: []string{"SpellDodgeChanceMax"}},
	"to avoid physical damage from hits":              {names: []string{"AvoidPhysicalDamageChance"}},
	"to avoid fire damage when hit":                   {names: []string{"AvoidFireDamageChance"}},
	"to avoid fire damage from hits":                  {names: []string{"AvoidFireDamageChance"}},
	"to avoid cold damage when hit":                   {names: []string{"AvoidColdDamageChance"}},
	"to avoid cold damage from hits":                  {names: []string{"AvoidColdDamageChance"}},
	"to avoid lightning damage when hit":              {names: []string{"AvoidLightningDamageChance"}},
	"to avoid lightning damage from hits":             {names: []string{"AvoidLightningDamageChance"}},
	"to avoid elemental damage when hit":              {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid elemental damage from hits":             {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid projectiles":                            {names: []string{"AvoidProjectilesChance"}},
	"to avoid being stunned":                          {names: []string{"AvoidStun"}},
	"to avoid interruption from stuns while casting":  {names: []string{"AvoidInteruptStun"}},
	"to avoid being shocked":                          {names: []string{"AvoidShock"}},
	"to avoid being frozen":                           {names: []string{"AvoidFreeze"}},
	"to avoid being chilled":                          {names: []string{"AvoidChill"}},
	"to avoid being ignited":                          {names: []string{"AvoidIgnite"}},
	"to avoid elemental ailments":                     {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidIgnite", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	"to avoid elemental status ailments":              {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidIgnite", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	"to avoid bleeding":                               {names: []string{"AvoidBleed"}},
	"to avoid being poisoned":                         {names: []string{"AvoidPoison"}},
	"damage is taken from mana before life":           {names: []string{"DamageTakenFromManaBeforeLife"}},
	"lightning damage is taken from mana before life": {names: []string{"LightningDamageTakenFromManaBeforeLife"}},
	"damage taken from mana before life":              {names: []string{"DamageTakenFromManaBeforeLife"}},
	"effect of curses on you":                         {names: []string{"CurseEffectOnSelf"}},
	"effect of curses on them":                        {names: []string{"CurseEffectOnSelf"}},
	"life recovery rate":                              {names: []string{"LifeRecoveryRate"}},
	"mana recovery rate":                              {names: []string{"ManaRecoveryRate"}},
	"energy shield recovery rate":                     {names: []string{"EnergyShieldRecoveryRate"}},
	"energy shield regeneration rate":                 {names: []string{"EnergyShieldRegen"}},
	"recovery rate of life, mana and energy shield":   {names: []string{"LifeRecoveryRate", "ManaRecoveryRate", "EnergyShieldRecoveryRate"}},
	"recovery rate of life and energy shield":         {names: []string{"LifeRecoveryRate", "EnergyShieldRecoveryRate"}},
	"maximum life, mana and global energy shield":     {names: []string{"Life", "Mana", "EnergyShield"}, tag: mod.Global()},
	"non-chaos damage taken bypasses energy shield":   {names: []string{"PhysicalEnergyShieldBypass", "LightningEnergyShieldBypass", "ColdEnergyShieldBypass", "FireEnergyShieldBypass"}},

	// Stun/knockback modifiers
	"stun recovery":                {names: []string{"StunRecovery"}},
	"stun and block recovery":      {names: []string{"StunRecovery"}},
	"block and stun recovery":      {names: []string{"StunRecovery"}},
	"stun threshold":               {names: []string{"StunThreshold"}},
	"block recovery":               {names: []string{"BlockRecovery"}},
	"enemy stun threshold":         {names: []string{"EnemyStunThreshold"}},
	"stun duration on enemies":     {names: []string{"EnemyStunDuration"}},
	"stun duration":                {names: []string{"EnemyStunDuration"}},
	"to knock enemies back on hit": {names: []string{"EnemyKnockbackChance"}},
	"knockback distance":           {names: []string{"EnemyKnockbackDistance"}},

	// Auras/curses/buffs
	"aura effect":                                                {names: []string{"AuraEffect"}},
	"effect of non-curse auras you cast":                         {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	"effect of non-curse auras from your skills":                 {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	"effect of non-curse auras from your skills on your minions": {names: []string{"AuraEffectOnSelf"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}, addToMinion: true},
	"effect of non-curse auras":                                  {names: []string{"AuraEffect"}, tag: mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)},
	"effect of your curses":                                      {names: []string{"CurseEffect"}},
	"effect of auras on you":                                     {names: []string{"AuraEffectOnSelf"}},
	"effect of auras on your minions":                            {names: []string{"AuraEffectOnSelf"}, addToMinion: true},
	"effect of auras from mines":                                 {names: []string{"AuraEffect"}, keywordFlags: mod.KeywordFlagMine},
	"effect of consecrated ground you create":                    {names: []string{"ConsecratedGroundEffect"}},
	"curse effect":                                               {names: []string{"CurseEffect"}},
	"effect of curses applied by bane":                           {names: []string{"CurseEffect"}, tag: mod.Condition("AppliedByBane")},
	"effect of your marks":                                       {names: []string{"CurseEffect"}, tag: mod.SkillType(string(data.SkillTypeMark))},
	"effect of arcane surge on you":                              {names: []string{"ArcaneSurgeEffect"}},
	"curse duration":                                             {names: []string{"Duration"}, keywordFlags: mod.KeywordFlagCurse},
	"hex duration":                                               {names: []string{"Duration"}, tag: mod.SkillType(string(data.SkillTypeHex))},
	"radius of auras":                                            {names: []string{"AreaOfEffect"}, keywordFlags: mod.KeywordFlagAura},
	"radius of curses":                                           {names: []string{"AreaOfEffect"}, keywordFlags: mod.KeywordFlagCurse},
	"buff effect":                                                {names: []string{"BuffEffect"}},
	"effect of buffs on you":                                     {names: []string{"BuffEffectOnSelf"}},
	"effect of buffs granted by your golems":                     {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeGolem))},
	"effect of buffs granted by socketed golem skills":           {names: []string{"BuffEffect"}, addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"effect of the buff granted by your stone golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Stone Golem")},
	"effect of the buff granted by your lightning golems":        {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Lightning Golem")},
	"effect of the buff granted by your ice golems":              {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Ice Golem")},
	"effect of the buff granted by your flame golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Flame Golem")},
	"effect of the buff granted by your chaos golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Chaos Golem")},
	"effect of the buff granted by your carrion golems":          {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Carrion Golem")},
	"effect of offering spells":                                  {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")},
	"effect of offerings":                                        {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")},
	"effect of heralds on you":                                   {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of herald buffs on you":                              {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of buffs granted by your active ancestor totems":     {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of buffs your ancestor totems grant ":                {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of withered":                                         {names: []string{"WitherEffect"}},
	"warcry effect":                                              {names: []string{"BuffEffect"}, keywordFlags: mod.KeywordFlagWarcry},
	"aspect of the avian buff effect":                            {names: []string{"BuffEffect"}, tag: mod.SkillName("Aspect of the Avian")},
	"maximum rage":                                               {names: []string{"MaximumRage"}},
	"maximum fortification":                                      {names: []string{"MaximumFortification"}},
	"fortification":                                              {names: []string{"FortificationStacks"}},

	// Charges
	"maximum power charge":                                {names: []string{"PowerChargesMax"}},
	"maximum power charges":                               {names: []string{"PowerChargesMax"}},
	"minimum power charge":                                {names: []string{"PowerChargesMin"}},
	"minimum power charges":                               {names: []string{"PowerChargesMin"}},
	"power charge duration":                               {names: []string{"PowerChargesDuration"}},
	"maximum frenzy charge":                               {names: []string{"FrenzyChargesMax"}},
	"maximum frenzy charges":                              {names: []string{"FrenzyChargesMax"}},
	"minimum frenzy charge":                               {names: []string{"FrenzyChargesMin"}},
	"minimum frenzy charges":                              {names: []string{"FrenzyChargesMin"}},
	"frenzy charge duration":                              {names: []string{"FrenzyChargesDuration"}},
	"maximum endurance charge":                            {names: []string{"EnduranceChargesMax"}},
	"maximum endurance charges":                           {names: []string{"EnduranceChargesMax"}},
	"minimum endurance charge":                            {names: []string{"EnduranceChargesMin"}},
	"minimum endurance charges":                           {names: []string{"EnduranceChargesMin"}},
	"minimum endurance, frenzy and power charges":         {names: []string{"PowerChargesMin", "FrenzyChargesMin", "EnduranceChargesMin"}},
	"endurance charge duration":                           {names: []string{"EnduranceChargesDuration"}},
	"maximum frenzy charges and maximum power charges":    {names: []string{"FrenzyChargesMax", "PowerChargesMax"}},
	"maximum power charges and maximum endurance charges": {names: []string{"PowerChargesMax", "EnduranceChargesMax"}},
	"endurance, frenzy and power charge duration":         {names: []string{"PowerChargesDuration", "FrenzyChargesDuration", "EnduranceChargesDuration"}},
	"maximum siphoning charge":                            {names: []string{"SiphoningChargesMax"}},
	"maximum siphoning charges":                           {names: []string{"SiphoningChargesMax"}},
	"maximum challenger charges":                          {names: []string{"ChallengerChargesMax"}},
	"maximum blitz charges":                               {names: []string{"BlitzChargesMax"}},
	"maximum number of crab barriers":                     {names: []string{"CrabBarriersMax"}},
	"maximum blood charges":                               {names: []string{"BloodChargesMax"}},
	"charge duration":                                     {names: []string{"ChargeDuration"}},

	// On hit/kill/leech effects
	"life gained on kill":                                               {names: []string{"LifeOnKill"}},
	"mana gained on kill":                                               {names: []string{"ManaOnKill"}},
	"life gained for each enemy hit":                                    {names: []string{"LifeOnHit"}},
	"life gained for each enemy hit by attacks":                         {names: []string{"LifeOnHit"}, flags: mod.MFlagAttack},
	"life gained for each enemy hit by your attacks":                    {names: []string{"LifeOnHit"}, flags: mod.MFlagAttack},
	"life gained for each enemy hit by spells":                          {names: []string{"LifeOnHit"}, flags: mod.MFlagSpell},
	"life gained for each enemy hit by your spells":                     {names: []string{"LifeOnHit"}, flags: mod.MFlagSpell},
	"mana gained for each enemy hit by attacks":                         {names: []string{"ManaOnHit"}, flags: mod.MFlagAttack},
	"mana gained for each enemy hit by your attacks":                    {names: []string{"ManaOnHit"}, flags: mod.MFlagAttack},
	"energy shield gained for each enemy hit":                           {names: []string{"EnergyShieldOnHit"}},
	"energy shield gained for each enemy hit by attacks":                {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagAttack},
	"energy shield gained for each enemy hit by your attacks":           {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagAttack},
	"life and mana gained for each enemy hit":                           {names: []string{"LifeOnHit", "ManaOnHit"}, flags: mod.MFlagAttack},
	"damage as life":                                                    {names: []string{"DamageLifeLeech"}},
	"life leeched per second":                                           {names: []string{"LifeLeechRate"}},
	"mana leeched per second":                                           {names: []string{"ManaLeechRate"}},
	"total recovery per second from life leech":                         {names: []string{"LifeLeechRate"}},
	"recovery per second from life leech":                               {names: []string{"LifeLeechRate"}},
	"total recovery per second from energy shield leech":                {names: []string{"EnergyShieldLeechRate"}},
	"recovery per second from energy shield leech":                      {names: []string{"EnergyShieldLeechRate"}},
	"total recovery per second from mana leech":                         {names: []string{"ManaLeechRate"}},
	"recovery per second from mana leech":                               {names: []string{"ManaLeechRate"}},
	"total recovery per second from life, mana, or energy shield leech": {names: []string{"LifeLeechRate", "ManaLeechRate", "EnergyShieldLeechRate"}},
	"maximum recovery per life leech":                                   {names: []string{"MaxLifeLeechInstance"}},
	"maximum recovery per energy shield leech":                          {names: []string{"MaxEnergyShieldLeechInstance"}},
	"maximum recovery per mana leech":                                   {names: []string{"MaxManaLeechInstance"}},
	"maximum total recovery per second from life leech":                 {names: []string{"MaxLifeLeechRate"}},
	"maximum total life recovery per second from leech":                 {names: []string{"MaxLifeLeechRate"}},
	"maximum total recovery per second from energy shield leech":        {names: []string{"MaxEnergyShieldLeechRate"}},
	"maximum total energy shield recovery per second from leech":        {names: []string{"MaxEnergyShieldLeechRate"}},
	"maximum total recovery per second from mana leech":                 {names: []string{"MaxManaLeechRate"}},
	"maximum total mana recovery per second from leech":                 {names: []string{"MaxManaLeechRate"}},
	"to impale enemies on hit":                                          {names: []string{"ImpaleChance"}},
	"to impale on spell hit":                                            {names: []string{"ImpaleChance"}, flags: mod.MFlagSpell},
	"impale effect":                                                     {names: []string{"ImpaleEffect"}},
	"effect of impales you inflict":                                     {names: []string{"ImpaleEffect"}},

	// Projectile modifiers
	"projectile":       {names: []string{"ProjectileCount"}},
	"projectiles":      {names: []string{"ProjectileCount"}},
	"projectile speed": {names: []string{"ProjectileSpeed"}},
	"arrow speed":      {names: []string{"ProjectileSpeed"}, flags: mod.MFlagBow},

	// Totem/trap/mine/brand modifiers
	"totem placement speed":                      {names: []string{"TotemPlacementSpeed"}},
	"totem life":                                 {names: []string{"TotemLife"}},
	"totem duration":                             {names: []string{"TotemDuration"}},
	"maximum number of summoned totems":          {names: []string{"ActiveTotemLimit"}},
	"maximum number of summoned totems.":         {names: []string{"ActiveTotemLimit"}}, // Mark plz
	"maximum number of summoned ballista totems": {names: []string{"ActiveBallistaLimit"}, tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	"trap throwing speed":                        {names: []string{"TrapThrowingSpeed"}},
	"trap and mine throwing speed":               {names: []string{"TrapThrowingSpeed", "MineLayingSpeed"}},
	"trap trigger area of effect":                {names: []string{"TrapTriggerAreaOfEffect"}},
	"trap duration":                              {names: []string{"TrapDuration"}},
	"cooldown recovery speed for throwing traps": {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"cooldown recovery rate for throwing traps":  {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"mine laying speed":                          {names: []string{"MineLayingSpeed"}},
	"mine throwing speed":                        {names: []string{"MineLayingSpeed"}},
	"mine detonation area of effect":             {names: []string{"MineDetonationAreaOfEffect"}},
	"mine duration":                              {names: []string{"MineDuration"}},
	"activation frequency":                       {names: []string{"BrandActivationFrequency"}},
	"brand activation frequency":                 {names: []string{"BrandActivationFrequency"}},
	"brand attachment range":                     {names: []string{"BrandAttachmentRange"}},

	// Minion modifiers
	"maximum number of skeletons":               {names: []string{"ActiveSkeletonLimit"}},
	"maximum number of zombies":                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of raised zombies":          {names: []string{"ActiveZombieLimit"}},
	"number of zombies allowed":                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of spectres":                {names: []string{"ActiveSpectreLimit"}},
	"maximum number of golems":                  {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned golems":         {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned raging spirits": {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of raging spirits":          {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of summoned phantasms":      {names: []string{"ActivePhantasmLimit"}},
	"maximum number of summoned holy relics":    {names: []string{"ActiveHolyRelicLimit"}},
	"minion duration":                           {names: []string{"Duration"}, tag: mod.SkillType(string(data.SkillTypeCreatesMinion))},
	"skeleton duration":                         {names: []string{"Duration"}, tag: mod.SkillName("Summon Skeleton")},
	"sentinel of dominance duration":            {names: []string{"Duration"}, tag: mod.SkillName("Dominating Blow")},

	// Other skill modifiers
	"radius":                                {names: []string{"AreaOfEffect"}},
	"radius of area skills":                 {names: []string{"AreaOfEffect"}},
	"area of effect radius":                 {names: []string{"AreaOfEffect"}},
	"area of effect":                        {names: []string{"AreaOfEffect"}},
	"area of effect of skills":              {names: []string{"AreaOfEffect"}},
	"area of effect of area skills":         {names: []string{"AreaOfEffect"}},
	"aspect of the spider area of effect":   {names: []string{"AreaOfEffect"}, tag: mod.SkillName("Aspect of the Spider")},
	"firestorm explosion area of effect":    {names: []string{"AreaOfEffectSecondary"}, tag: mod.SkillName("Firestorm")},
	"duration":                              {names: []string{"Duration"}},
	"skill effect duration":                 {names: []string{"Duration"}},
	"chaos skill effect duration":           {names: []string{"Duration"}, keywordFlags: mod.KeywordFlagChaos},
	"aspect of the spider debuff duration":  {names: []string{"Duration"}, tag: mod.SkillName("Aspect of the Spider")},
	"fire trap burning ground duration":     {names: []string{"Duration"}, tag: mod.SkillName("Fire Trap")},
	"cooldown recovery":                     {names: []string{"CooldownRecovery"}},
	"cooldown recovery speed":               {names: []string{"CooldownRecovery"}},
	"cooldown recovery rate":                {names: []string{"CooldownRecovery"}},
	"weapon range":                          {names: []string{"WeaponRange"}},
	"melee range":                           {names: []string{"MeleeWeaponRange"}},
	"melee weapon range":                    {names: []string{"MeleeWeaponRange"}},
	"melee weapon and unarmed range":        {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee weapon and unarmed attack range": {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee strike range":                    {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"to deal double damage":                 {names: []string{"DoubleDamageChance"}},

	// Buffs
	"onslaught effect":          {names: []string{"OnslaughtEffect"}},
	"adrenaline duration":       {names: []string{"AdrenalineDuration"}},
	"effect of tailwind on you": {names: []string{"TailwindEffectOnSelf"}},
	"elusive effect":            {names: []string{"ElusiveEffect"}},
	"effect of elusive on you":  {names: []string{"ElusiveEffect"}},
	"effect of infusion":        {names: []string{"InfusionEffect"}},

	// Basic damage types
	"damage":           {names: []string{"Damage"}},
	"physical damage":  {names: []string{"PhysicalDamage"}},
	"lightning damage": {names: []string{"LightningDamage"}},
	"cold damage":      {names: []string{"ColdDamage"}},
	"fire damage":      {names: []string{"FireDamage"}},
	"chaos damage":     {names: []string{"ChaosDamage"}},
	"non-chaos damage": {names: []string{"NonChaosDamage"}},
	"elemental damage": {names: []string{"ElementalDamage"}},

	// Other damage forms
	"attack damage":                        {names: []string{"Damage"}, flags: mod.MFlagAttack},
	"attack physical damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"physical attack damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"minimum physical attack damage":       {names: []string{"MinPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"maximum physical attack damage":       {names: []string{"MaxPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"physical weapon damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"physical damage with weapons":         {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"melee damage":                         {names: []string{"Damage"}, flags: mod.MFlagMelee},
	"physical melee damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"melee physical damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"projectile damage":                    {names: []string{"Damage"}, flags: mod.MFlagProjectile},
	"projectile attack damage":             {names: []string{"Damage"}, flags: mod.MFlagProjectile | mod.MFlagAttack},
	"bow damage":                           {names: []string{"Damage"}, flags: mod.MFlagBow | mod.MFlagHit},
	"damage with arrow hits":               {names: []string{"Damage"}, flags: mod.MFlagBow | mod.MFlagHit},
	"wand damage":                          {names: []string{"Damage"}, flags: mod.MFlagWand | mod.MFlagHit},
	"wand physical damage":                 {names: []string{"PhysicalDamage"}, flags: mod.MFlagWand | mod.MFlagHit},
	"claw physical damage":                 {names: []string{"PhysicalDamage"}, flags: mod.MFlagClaw | mod.MFlagHit},
	"sword physical damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagSword | mod.MFlagHit},
	"damage over time":                     {names: []string{"Damage"}, flags: mod.MFlagDot},
	"physical damage over time":            {names: []string{"PhysicalDamage"}, keywordFlags: mod.KeywordFlagPhysicalDot},
	"cold damage over time":                {names: []string{"ColdDamage"}, keywordFlags: mod.KeywordFlagColdDot},
	"chaos damage over time":               {names: []string{"ChaosDamage"}, keywordFlags: mod.KeywordFlagChaosDot},
	"burning damage":                       {names: []string{"FireDamage"}, keywordFlags: mod.KeywordFlagFireDot},
	"damage with ignite":                   {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagIgnite},
	"damage with ignites":                  {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagIgnite},
	"incinerate damage for each stage":     {names: []string{"Damage"}, tagList: []mod.Tag{mod.Multiplier("IncinerateStage").Base(0), mod.SkillName("Incinerate")}},
	"physical damage over time multiplier": {names: []string{"PhysicalDotMultiplier"}},
	"fire damage over time multiplier":     {names: []string{"FireDotMultiplier"}},
	"cold damage over time multiplier":     {names: []string{"ColdDotMultiplier"}},
	"chaos damage over time multiplier":    {names: []string{"ChaosDotMultiplier"}},
	"damage over time multiplier":          {names: []string{"DotMultiplier"}},

	// Crit/accuracy/speed modifiers
	"critical strike chance":            {names: []string{"CritChance"}},
	"attack critical strike chance":     {names: []string{"CritChance"}, flags: mod.MFlagAttack},
	"critical strike multiplier":        {names: []string{"CritMultiplier"}},
	"attack critical strike multiplier": {names: []string{"CritMultiplier"}, flags: mod.MFlagAttack},
	"accuracy":                          {names: []string{"Accuracy"}},
	"accuracy rating":                   {names: []string{"Accuracy"}},
	"minion accuracy rating":            {names: []string{"Accuracy"}, addToMinion: true},
	"attack speed":                      {names: []string{"Speed"}, flags: mod.MFlagAttack},
	"cast speed":                        {names: []string{"Speed"}, flags: mod.MFlagCast},
	"warcry speed":                      {names: []string{"WarcrySpeed"}, keywordFlags: mod.KeywordFlagWarcry},
	"attack and cast speed":             {names: []string{"Speed"}},

	// Elemental ailments
	"to shock":                                    {names: []string{"EnemyShockChance"}},
	"shock chance":                                {names: []string{"EnemyShockChance"}},
	"to freeze":                                   {names: []string{"EnemyFreezeChance"}},
	"freeze chance":                               {names: []string{"EnemyFreezeChance"}},
	"to ignite":                                   {names: []string{"EnemyIgniteChance"}},
	"ignite chance":                               {names: []string{"EnemyIgniteChance"}},
	"to freeze, shock and ignite":                 {names: []string{"EnemyFreezeChance", "EnemyShockChance", "EnemyIgniteChance"}},
	"to scorch enemies":                           {names: []string{"EnemyScorchChance"}},
	"to inflict brittle":                          {names: []string{"EnemyBrittleChance"}},
	"to sap enemies":                              {names: []string{"EnemySapChance"}},
	"effect of scorch":                            {names: []string{"EnemyScorchEffect"}},
	"effect of sap":                               {names: []string{"EnemySapEffect"}},
	"effect of brittle":                           {names: []string{"EnemyBrittleEffect"}},
	"effect of shock":                             {names: []string{"EnemyShockEffect"}},
	"effect of shock on you":                      {names: []string{"SelfShockEffect"}},
	"effect of shock you inflict":                 {names: []string{"EnemyShockEffect"}},
	"effect of lightning ailments":                {names: []string{"EnemyShockEffect", "EnemySapEffect"}},
	"effect of chill":                             {names: []string{"EnemyChillEffect"}},
	"effect of chill and shock on you":            {names: []string{"SelfChillEffect", "SelfShockEffect"}},
	"chill effect":                                {names: []string{"EnemyChillEffect"}},
	"effect of chill you inflict":                 {names: []string{"EnemyChillEffect"}},
	"effect of cold ailments":                     {names: []string{"EnemyChillEffect", "EnemyBrittleEffect"}},
	"effect of chill on you":                      {names: []string{"SelfChillEffect"}},
	"effect of non-damaging ailments":             {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"effect of non-damaging ailments you inflict": {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"shock duration":                              {names: []string{"EnemyShockDuration"}},
	"shock duration on you":                       {names: []string{"SelfShockDuration"}},
	"duration of lightning ailments":              {names: []string{"EnemyShockDuration", "EnemySapDuration"}},
	"freeze duration":                             {names: []string{"EnemyFreezeDuration"}},
	"freeze duration on you":                      {names: []string{"SelfFreezeDuration"}},
	"chill duration":                              {names: []string{"EnemyChillDuration"}},
	"chill duration on you":                       {names: []string{"SelfChillDuration"}},
	"duration of cold ailments":                   {names: []string{"EnemyFreezeDuration", "EnemyChillDuration", "EnemyBrittleDuration"}},
	"ignite duration":                             {names: []string{"EnemyIgniteDuration"}},
	"ignite duration on you":                      {names: []string{"SelfIgniteDuration"}},
	"duration of ignite on you":                   {names: []string{"SelfIgniteDuration"}},
	"duration of elemental ailments":              {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of elemental ailments on you":       {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"duration of elemental status ailments":       {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments":                        {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments on you":                 {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfPoisonDuration", "SelfBleedDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"elemental ailment duration on you":           {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"duration of ailments you inflict":            {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments inflicted":              {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},

	// Other ailments
	"to poison":                       {names: []string{"PoisonChance"}},
	"to cause poison":                 {names: []string{"PoisonChance"}},
	"to poison on hit":                {names: []string{"PoisonChance"}},
	"poison duration":                 {names: []string{"EnemyPoisonDuration"}},
	"duration of poisons you inflict": {names: []string{"EnemyPoisonDuration"}},
	"to cause bleeding":               {names: []string{"BleedChance"}},
	"to cause bleeding on hit":        {names: []string{"BleedChance"}},
	"to inflict bleeding":             {names: []string{"BleedChance"}},
	"to inflict bleeding on hit":      {names: []string{"BleedChance"}},
	"bleed duration":                  {names: []string{"EnemyBleedDuration"}},
	"bleeding duration":               {names: []string{"EnemyBleedDuration"}},

	// Misc modifiers
	"movement speed":                        {names: []string{"MovementSpeed"}},
	"attack, cast and movement speed":       {names: []string{"Speed", "MovementSpeed"}},
	"action speed":                          {names: []string{"ActionSpeed"}},
	"light radius":                          {names: []string{"LightRadius"}},
	"rarity of items found":                 {names: []string{"LootRarity"}},
	"quantity of items found":               {names: []string{"LootQuantity"}},
	"item quantity":                         {names: []string{"LootQuantity"}},
	"strength requirement":                  {names: []string{"StrRequirement"}},
	"dexterity requirement":                 {names: []string{"DexRequirement"}},
	"intelligence requirement":              {names: []string{"IntRequirement"}},
	"omni requirement":                      {names: []string{"OmniRequirement"}},
	"strength and intelligence requirement": {names: []string{"StrRequirement", "IntRequirement"}},
	"attribute requirements":                {names: []string{"StrRequirement", "DexRequirement", "IntRequirement"}},
	"effect of socketed jewels":             {names: []string{"SocketedJewelEffect"}},
	"effect of socketed abyss jewels":       {names: []string{"SocketedJewelEffect"}},
	"to inflict fire exposure on hit":       {names: []string{"FireExposureChance"}},
	"to apply fire exposure on hit":         {names: []string{"FireExposureChance"}},
	"to inflict cold exposure on hit":       {names: []string{"ColdExposureChance"}},
	"to apply cold exposure on hit":         {names: []string{"ColdExposureChance"}},
	"to inflict lightning exposure on hit":  {names: []string{"LightningExposureChance"}},
	"to apply lightning exposure on hit":    {names: []string{"LightningExposureChance"}},

	// Flask modifiers
	"effect":                             {names: []string{"FlaskEffect"}},
	"effect of flasks":                   {names: []string{"FlaskEffect"}},
	"effect of flasks on you":            {names: []string{"FlaskEffect"}},
	"amount recovered":                   {names: []string{"FlaskRecovery"}},
	"life recovered":                     {names: []string{"FlaskRecovery"}},
	"life recovery from flasks used":     {names: []string{"FlaskLifeRecovery"}},
	"mana recovered":                     {names: []string{"FlaskRecovery"}},
	"life recovery from flasks":          {names: []string{"FlaskLifeRecovery"}},
	"mana recovery from flasks":          {names: []string{"FlaskManaRecovery"}},
	"life and mana recovery from flasks": {names: []string{"FlaskLifeRecovery", "FlaskManaRecovery"}},
	"flask effect duration":              {names: []string{"FlaskDuration"}},
	"recovery speed":                     {names: []string{"FlaskRecoveryRate"}},
	"recovery rate":                      {names: []string{"FlaskRecoveryRate"}},
	"flask recovery rate":                {names: []string{"FlaskRecoveryRate"}},
	"flask recovery speed":               {names: []string{"FlaskRecoveryRate"}},
	"flask life recovery rate":           {names: []string{"FlaskLifeRecoveryRate"}},
	"flask mana recovery rate":           {names: []string{"FlaskManaRecoveryRate"}},
	"extra charges":                      {names: []string{"FlaskCharges"}},
	"maximum charges":                    {names: []string{"FlaskCharges"}},
	"charges used":                       {names: []string{"FlaskChargesUsed"}},
	"charges per use":                    {names: []string{"FlaskChargesUsed"}},
	"flask charges used":                 {names: []string{"FlaskChargesUsed"}},
	"flask charges gained":               {names: []string{"FlaskChargesGained"}},
	"charge recovery":                    {names: []string{"FlaskChargeRecovery"}},
	"impales you inflict last":           {names: []string{"ImpaleStacksMax"}},
}

// List of modifier flags
var modFlagListCompiled map[string]CompiledList[modNameListType]
var modFlagList = map[string]modNameListType{
	// Weapon types
	"with axes":                      {flags: mod.MFlagAxe | mod.MFlagHit},
	"to axe attacks":                 {flags: mod.MFlagAxe | mod.MFlagHit},
	"with axe attacks":               {flags: mod.MFlagAxe | mod.MFlagHit},
	"with axes or swords":            {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagAxe | mod.MFlagSword)},
	"with bows":                      {flags: mod.MFlagBow | mod.MFlagHit},
	"to bow attacks":                 {flags: mod.MFlagBow | mod.MFlagHit},
	"with bow attacks":               {flags: mod.MFlagBow | mod.MFlagHit},
	"with claws":                     {flags: mod.MFlagClaw | mod.MFlagHit},
	"with claws or daggers":          {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger)},
	"to claw attacks":                {flags: mod.MFlagClaw | mod.MFlagHit},
	"with claw attacks":              {flags: mod.MFlagClaw | mod.MFlagHit},
	"dealt with claws":               {flags: mod.MFlagClaw | mod.MFlagHit},
	"with daggers":                   {flags: mod.MFlagDagger | mod.MFlagHit},
	"to dagger attacks":              {flags: mod.MFlagDagger | mod.MFlagHit},
	"with dagger attacks":            {flags: mod.MFlagDagger | mod.MFlagHit},
	"with maces":                     {flags: mod.MFlagMace | mod.MFlagHit},
	"to mace attacks":                {flags: mod.MFlagMace | mod.MFlagHit},
	"with mace attacks":              {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces and sceptres":        {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces or sceptres":         {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces, sceptres or staves": {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagMace | mod.MFlagStaff)},
	"to mace and sceptre attacks":    {flags: mod.MFlagMace | mod.MFlagHit},
	"to mace or sceptre attacks":     {flags: mod.MFlagMace | mod.MFlagHit},
	"with mace or sceptre attacks":   {flags: mod.MFlagMace | mod.MFlagHit},
	"with staves":                    {flags: mod.MFlagStaff | mod.MFlagHit},
	"to staff attacks":               {flags: mod.MFlagStaff | mod.MFlagHit},
	"with staff attacks":             {flags: mod.MFlagStaff | mod.MFlagHit},
	"with swords":                    {flags: mod.MFlagSword | mod.MFlagHit},
	"to sword attacks":               {flags: mod.MFlagSword | mod.MFlagHit},
	"with sword attacks":             {flags: mod.MFlagSword | mod.MFlagHit},
	"with wands":                     {flags: mod.MFlagWand | mod.MFlagHit},
	"to wand attacks":                {flags: mod.MFlagWand | mod.MFlagHit},
	"with wand attacks":              {flags: mod.MFlagWand | mod.MFlagHit},
	"unarmed":                        {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"unarmed melee":                  {flags: mod.MFlagUnarmed | mod.MFlagMelee | mod.MFlagHit},
	"with unarmed attacks":           {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"with unarmed melee attacks":     {flags: mod.MFlagUnarmed | mod.MFlagMelee},
	"to unarmed attacks":             {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"to unarmed melee hits":          {flags: mod.MFlagUnarmed | mod.MFlagMelee | mod.MFlagHit},
	"with one handed weapons":        {flags: mod.MFlagWeapon1H | mod.MFlagHit},
	"with one handed melee weapons":  {flags: mod.MFlagWeapon1H | mod.MFlagWeaponMelee | mod.MFlagHit},
	"with two handed weapons":        {flags: mod.MFlagWeapon2H | mod.MFlagHit},
	"with two handed melee weapons":  {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee | mod.MFlagHit},
	"with ranged weapons":            {flags: mod.MFlagWeaponRanged | mod.MFlagHit},

	// Skill types
	"spell":                            {flags: mod.MFlagSpell},
	"with spells":                      {flags: mod.MFlagSpell},
	"with spell damage":                {flags: mod.MFlagSpell},
	"for spells":                       {flags: mod.MFlagSpell},
	"for spell damage":                 {flags: mod.MFlagSpell},
	"with attacks":                     {keywordFlags: mod.KeywordFlagAttack},
	"with attack skills":               {keywordFlags: mod.KeywordFlagAttack},
	"for attacks":                      {flags: mod.MFlagAttack},
	"for attack damage":                {flags: mod.MFlagAttack},
	"weapon":                           {flags: mod.MFlagWeapon},
	"with weapons":                     {flags: mod.MFlagWeapon},
	"melee":                            {flags: mod.MFlagMelee},
	"with melee attacks":               {flags: mod.MFlagMelee},
	"with melee critical strikes":      {flags: mod.MFlagMelee, tag: mod.Condition("CriticalStrike")},
	"with melee skills":                {flags: mod.MFlagMelee},
	"with bow skills":                  {keywordFlags: mod.KeywordFlagBow},
	"on melee hit":                     {flags: mod.MFlagMelee},
	"with hits":                        {keywordFlags: mod.KeywordFlagHit},
	"with hits against nearby enemies": {keywordFlags: mod.KeywordFlagHit},
	"with hits and ailments":           {keywordFlags: mod.KeywordFlagHit | mod.KeywordFlagAilment},
	"with ailments":                    {flags: mod.MFlagAilment},
	"with ailments from attack skills": {flags: mod.MFlagAilment, keywordFlags: mod.KeywordFlagAttack},
	"with poison":                      {keywordFlags: mod.KeywordFlagPoison},
	"with bleeding":                    {keywordFlags: mod.KeywordFlagBleed},
	"for ailments":                     {flags: mod.MFlagAilment},
	"for poison":                       {keywordFlags: mod.KeywordFlagPoison | mod.KeywordFlagMatchAll},
	"for bleeding":                     {keywordFlags: mod.KeywordFlagBleed},
	"for ignite":                       {keywordFlags: mod.KeywordFlagIgnite},
	"area":                             {flags: mod.MFlagArea},
	"mine":                             {keywordFlags: mod.KeywordFlagMine},
	"with mines":                       {keywordFlags: mod.KeywordFlagMine},
	"trap":                             {keywordFlags: mod.KeywordFlagTrap},
	"with traps":                       {keywordFlags: mod.KeywordFlagTrap},
	"for traps":                        {keywordFlags: mod.KeywordFlagTrap},
	"that place mines or throw traps":  {keywordFlags: mod.KeywordFlagMine | mod.KeywordFlagTrap},
	"that throw mines":                 {keywordFlags: mod.KeywordFlagMine},
	"that throw traps":                 {keywordFlags: mod.KeywordFlagTrap},
	"brand":                            {tag: mod.SkillType(string(data.SkillTypeBrand))},
	"totem":                            {keywordFlags: mod.KeywordFlagTotem},
	"with totem skills":                {keywordFlags: mod.KeywordFlagTotem},
	"for skills used by totems":        {keywordFlags: mod.KeywordFlagTotem},
	"totem skills that cast an aura":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"aura skills that summon totems":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"of aura skills":                   {tag: mod.SkillType(string(data.SkillTypeAura))},
	"of curse skills":                  {keywordFlags: mod.KeywordFlagCurse},
	"with curse skills":                {keywordFlags: mod.KeywordFlagCurse},
	"of curse aura skills":             {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagCurse},
	"of curse auras":                   {keywordFlags: mod.KeywordFlagCurse | mod.KeywordFlagAura | mod.KeywordFlagMatchAll},
	"of hex skills":                    {tag: mod.SkillType(string(data.SkillTypeHex))},
	"with hex skills":                  {tag: mod.SkillType(string(data.SkillTypeHex))},
	"of herald skills":                 {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with herald skills":               {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with hits from herald skills":     {tag: mod.SkillType(string(data.SkillTypeHerald)), keywordFlags: mod.KeywordFlagHit},
	"minion skills":                    {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"of minion skills":                 {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"for curses":                       {keywordFlags: mod.KeywordFlagCurse},
	"for hexes":                        {tag: mod.SkillType(string(data.SkillTypeHex))},
	"warcry":                           {keywordFlags: mod.KeywordFlagWarcry},
	"vaal":                             {keywordFlags: mod.KeywordFlagVaal},
	"vaal skill":                       {keywordFlags: mod.KeywordFlagVaal},
	"with vaal skills":                 {keywordFlags: mod.KeywordFlagVaal},
	"with non-vaal skills":             {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	"with movement skills":             {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills":               {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills used":          {keywordFlags: mod.KeywordFlagMovement},
	"of travel skills":                 {tag: mod.SkillType(string(data.SkillTypeTravel))},
	"of banner skills":                 {tag: mod.SkillType(string(data.SkillTypeBanner))},
	"with lightning skills":            {keywordFlags: mod.KeywordFlagLightning},
	"with cold skills":                 {keywordFlags: mod.KeywordFlagCold},
	"with fire skills":                 {keywordFlags: mod.KeywordFlagFire},
	"with elemental skills":            {keywordFlags: mod.KeywordFlagLightning | mod.KeywordFlagCold | mod.KeywordFlagFire},
	"with chaos skills":                {keywordFlags: mod.KeywordFlagChaos},
	"with physical skills":             {tag: mod.SkillType(string(data.SkillTypePhysical))}, // TODO Verify: https://canary.discord.com/channels/676181894152454150/677968792164368401/1016528721588523159
	"with channelling skills":          {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"channelling skills":               {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"with brand skills":                {tag: mod.SkillType(string(data.SkillTypeBrand))},
	"for stance skills":                {tag: mod.SkillType(string(data.SkillTypeStance))},
	"of stance skills":                 {tag: mod.SkillType(string(data.SkillTypeStance))},
	"with skills that cost life":       {tag: mod.StatThreshold("LifeCost", 1)},
	"minion":                           {addToMinion: true},
	"zombie":                           {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	"raised zombie":                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	"skeleton":                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeleton")},
	"spectre":                          {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	"raised spectre":                   {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	"golem":                            {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	"chaos golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	"flame golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	"increased flame golem":            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	"ice golem":                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	"lightning golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	"stone golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	"animated guardian":                {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian")},

	// Other
	"global":                          {tag: mod.Global()},
	"from equipped shield":            {tag: mod.SlotName("Weapon 2")},
	"from equipped gloves and boots":  {tag: mod.SlotName("Gloves", "Boots")},
	"from equipped helmet and gloves": {tag: mod.SlotName("Helmet", "Gloves")},
	"from equipped helmet and boots":  {tag: mod.SlotName("Helmet", "Boots")},
	"from body armour":                {tag: mod.SlotName("Body Armour")},
	"from your body armour":           {tag: mod.SlotName("Body Armour")},
}

// List of modifier flags/tags that appear at the start of a line
var preFlagListCompiled map[string]CompiledList[modNameListType]
var preFlagList = map[string]modNameListType{
	// Weapon types
	`^axe attacks [hd][ae][va][el] `:                           {flags: mod.MFlagAxe},
	`^axe or sword attacks [hd][ae][va][el] `:                  {tag: mod.ModFlagOr(mod.MFlagAxe | mod.MFlagSword)},
	`^bow attacks [hd][ae][va][el] `:                           {flags: mod.MFlagBow},
	`^claw attacks [hd][ae][va][el] `:                          {flags: mod.MFlagClaw},
	`^claw or dagger attacks [hd][ae][va][el] `:                {tag: mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger)},
	`^dagger attacks [hd][ae][va][el] `:                        {flags: mod.MFlagDagger},
	`^mace or sceptre attacks [hd][ae][va][el] `:               {flags: mod.MFlagMace},
	`^mace, sceptre or staff attacks [hd][ae][va][el] `:        {tag: mod.ModFlagOr(mod.MFlagMace | mod.MFlagStaff)},
	`^staff attacks [hd][ae][va][el] `:                         {flags: mod.MFlagStaff},
	`^sword attacks [hd][ae][va][el] `:                         {flags: mod.MFlagSword},
	`^wand attacks [hd][ae][va][el] `:                          {flags: mod.MFlagWand},
	`^unarmed attacks [hd][ae][va][el] `:                       {flags: mod.MFlagUnarmed},
	`^attacks with one handed weapons [hd][ae][va][el] `:       {flags: mod.MFlagWeapon1H},
	`^attacks with two handed weapons [hd][ae][va][el] `:       {flags: mod.MFlagWeapon2H},
	`^attacks with melee weapons [hd][ae][va][el] `:            {flags: mod.MFlagWeaponMelee},
	`^attacks with one handed melee weapons [hd][ae][va][el] `: {flags: mod.MFlagWeapon1H | mod.MFlagWeaponMelee},
	`^attacks with two handed melee weapons [hd][ae][va][el] `: {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee},
	`^attacks with ranged weapons [hd][ae][va][el] `:           {flags: mod.MFlagWeaponRanged},

	// Damage types
	`^attack damage `:         {flags: mod.MFlagAttack},
	`^hits deal `:             {keywordFlags: mod.KeywordFlagHit},
	`^critical strikes deal `: {tag: mod.Condition("CriticalStrike")},
	`^poisons you inflict with critical strikes have `: {keywordFlags: mod.KeywordFlagPoison | mod.KeywordFlagMatchAll, tag: mod.Condition("CriticalStrike")},

	// Add to minion
	`^minions `:                                                      {addToMinion: true},
	`^minions [hd][ae][va][el] `:                                     {addToMinion: true},
	`^minions leech `:                                                {addToMinion: true},
	`^minions' attacks deal `:                                        {addToMinion: true, flags: mod.MFlagAttack},
	`^golems [hd][ae][va][el] `:                                      {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	`^summoned golems [hd][ae][va][el] `:                             {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	`^golem skills have `:                                            {tag: mod.SkillType(string(data.SkillTypeGolem))},
	`^zombies [hd][ae][va][el] `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	`^raised zombies [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	`^skeletons [hd][ae][va][el] `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeleton")},
	`^raging spirits [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit")},
	`^summoned raging spirits [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit")},
	`^spectres [hd][ae][va][el] `:                                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	`^chaos golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	`^summoned chaos golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	`^flame golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	`^summoned flame golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	`^ice golems [hd][ae][va][el] `:                                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	`^summoned ice golems [hd][ae][va][el] `:                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	`^lightning golems [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	`^summoned lightning golems [hd][ae][va][el] `:                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	`^stone golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	`^summoned stone golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	`^summoned carrion golems [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem")},
	`^summoned skitterbots [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem")},
	`^blink arrow and blink arrow clones [hd][ae][va][el] `:          {addToMinion: true, addToMinionTag: mod.SkillName("Blink Arrow")},
	`^mirror arrow and mirror arrow clones [hd][ae][va][el] `:        {addToMinion: true, addToMinionTag: mod.SkillName("Mirror Arrow")},
	`^animated weapons [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Animate Weapon")},
	`^animated guardians? deals? `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian")},
	`^summoned holy relics [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Holy Relic")},
	`^summoned reaper [dh][ea][as]l?s? `:                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Reaper")},
	`^herald skills [hd][ae][va][el] `:                               {tag: mod.SkillType(string(data.SkillTypeHerald))},
	`^agony crawler deals `:                                          {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^summoned agony crawler fires `:                                 {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^sentinels of purity deal `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels of absolution have `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels have `:                                      {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity", "Dominating Blow", "Absolution")},
	`^raised zombies' slam attack has `:                              {addToMinion: true, tag: mod.SkillId("ZombieSlam")},
	`^raised spectres, raised zombies, and summoned skeletons have `: {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre", "Raise Zombie", "Summon Skeleton")},

	// Totem/trap/mine
	`^attacks used by totems have `:            {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagTotem},
	`^spells cast by totems [hd][ae][va][el] `: {flags: mod.MFlagSpell, keywordFlags: mod.KeywordFlagTotem},
	`^trap and mine damage `:                   {keywordFlags: mod.KeywordFlagTrap | mod.KeywordFlagMine},
	`^skills used by traps [hd][ae][va][el] `:  {keywordFlags: mod.KeywordFlagTrap},
	`^skills used by mines [hd][ae][va][el] `:  {keywordFlags: mod.KeywordFlagMine},

	// Local damage
	`^attacks with this weapon `:                  {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^attacks with this weapon [hd][ae][va][el] `: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^hits with this weapon [hd][ae][va][el] `:    {flags: mod.MFlagHit, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},

	// Skill types
	`^attacks [hd][ae][va][el] `:                  {flags: mod.MFlagAttack},
	`^attack skills [hd][ae][va][el] `:            {keywordFlags: mod.KeywordFlagAttack},
	`^spells [hd][ae][va][el] a? ?`:               {flags: mod.MFlagSpell},
	`^spell skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagSpell},
	`^projectile attack skills [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^projectiles from attacks [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^arrows [hd][ae][va][el] `:                   {keywordFlags: mod.KeywordFlagBow},
	`^bow skills [hdf][aei][var][el] `:            {keywordFlags: mod.KeywordFlagBow},
	`^projectiles [hdf][aei][var][el] `:           {flags: mod.MFlagProjectile},
	`^melee attacks have `:                        {flags: mod.MFlagMelee},
	`^movement attack skills have `:               {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagMovement},
	`^travel skills have `:                        {tag: mod.SkillType(string(data.SkillTypeTravel))},
	`^lightning skills [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning},
	`^lightning spells [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning, flags: mod.MFlagSpell},
	`^cold skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold},
	`^cold spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold, flags: mod.MFlagSpell},
	`^fire skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire},
	`^fire spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire, flags: mod.MFlagSpell},
	`^chaos skills [hd][ae][va][el] a? ?`:         {keywordFlags: mod.KeywordFlagChaos},
	`^vaal skills [hd][ae][va][el] `:              {keywordFlags: mod.KeywordFlagVaal},
	`^brand skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagBrand},
	`^channelling skills [hd][ae][va][el] `:       {tag: mod.SkillType(string(data.SkillTypeChannel))},
	`^curse skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagCurse},
	`^hex skills [hd][ae][va][el] `:               {tag: mod.SkillType(string(data.SkillTypeHex))},
	`^mark skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeMark))},
	`^melee skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeMelee))},
	`^guard skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeGuard))},
	`^nova spells [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeNova))},
	`^area skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeArea))},
	`^aura skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeAura))},
	`^prismatic skills [hd][ae][va][el] `:         {tag: mod.SkillType(string(data.SkillTypeRandomElement))},
	`^warcry skills have `:                        {tag: mod.SkillType(string(data.SkillTypeWarcry))},
	`^non\-curse aura skills have `:               {tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	`^non\-channelling skills have `:              {tag: mod.SkillType(string(data.SkillTypeChannel)).Neg(true)},
	`^non\-vaal skills deal `:                     {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	`^skills [hdfg][aei][vari][eln] `:             {},

	// Slot specific
	`^left ring slot: `:                                   {tag: mod.SlotNumber(1)},
	`^right ring slot: `:                                  {tag: mod.SlotNumber(2)},
	`^socketed gems [hgd][ae][via][enl] `:                 {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed skills [hgd][ae][via][enl] `:               {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed attacks [hgd][ae][via][enl] `:              {addToSkill: mod.SocketedIn("{SlotName}").Keyword("attack")},
	`^socketed spells [hgd][ae][via][enl] `:               {addToSkill: mod.SocketedIn("{SlotName}").Keyword("spell")},
	`^socketed curse gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("curse")},
	`^socketed melee gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("melee")},
	`^socketed golem gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed golem skills [hgd][ae][via][enl] `:         {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed golem skills have minions `:                {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed vaal skills [hgd][ae][via][enl] `:          {addToSkill: mod.SocketedIn("{SlotName}").Keyword("vaal")},
	`^socketed projectile spells [hgdf][aei][viar][enl] `: {addToSkill: mod.SocketedIn("{SlotName}"), tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeProjectile)), mod.SkillType(string(data.SkillTypeSpell))}},

	// Enemy modifiers
	`^enemies withered by you [th]a[vk]e `: {tag: mod.Multiplier("MultiplierThreshold").Base(1), applyToEnemy: true},
	`^enemies (\w+) by you take `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{
				tag:          mod.Condition(utils.Capital(caps[0])),
				applyToEnemy: true,
				modSuffix:    "Taken",
			}
		},
	},
	`^enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(utils.Capital(caps[0])), applyToEnemy: true}
		},
	},
	`^hits against enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ActorCondition("enemy", utils.Capital(caps[0]))}
		},
	},
	`^enemies shocked or frozen by you take `:                     {tag: mod.Condition("Shocked", "Frozen"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies affected by your spider's webs [thd][ae][avk][el] `: {tag: mod.MultiplierThreshold("Spider's WebStack").Threshold(1), applyToEnemy: true},
	`^enemies you curse take `:                                    {tag: mod.Condition("Cursed"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies you curse ?h?a?v?e? `:                               {tag: mod.Condition("Cursed"), applyToEnemy: true},
	`^nearby enemies take `:                                       {modSuffix: "Taken", applyToEnemy: true},
	`^nearby enemies have `:                                       {applyToEnemy: true},
	`^nearby enemies deal `:                                       {applyToEnemy: true},
	`^nearby enemies `:                                            {applyToEnemy: true},
	`^enemies near your totems deal `:                             {applyToEnemy: true},

	// Other
	`^your flasks grant `:                         {},
	`^when hit, `:                                 {},
	`^you and allies [hgd][ae][via][enl] `:        {},
	`^auras from your skills grant `:              {addToAura: true},
	`^you and nearby allies `:                     {newAura: true},
	`^you and nearby allies [hgd][ae][via][enl] `: {newAura: true},
	`^nearby allies [hgd][ae][via][enl] `:         {newAura: true, newAuraOnlyAllies: true},
	`^you and allies affected by auras from your skills [hgd][ae][via][enl] `: {affectedByAura: true},
	`^take `:      {modSuffix: "Taken"},
	`^marauder: `: {tag: mod.Condition("ConnectedToMarauderStart")},
	`^duelist: `:  {tag: mod.Condition("ConnectedToDuelistStart")},
	`^ranger: `:   {tag: mod.Condition("ConnectedToRangerStart")},
	`^shadow: `:   {tag: mod.Condition("ConnectedToShadowStart")},
	`^witch: `:    {tag: mod.Condition("ConnectedToWitchStart")},
	`^templar: `:  {tag: mod.Condition("ConnectedToTemplarStart")},
	`^scion: `:    {tag: mod.Condition("ConnectedToScionStart")},
	`^skills supported by spellslinger have `:                               {tag: mod.Condition("SupportedBySpellslinger")},
	`^skills that have dealt a critical strike in the past 8 seconds deal `: {tag: mod.Condition("CritInPast8Sec")},
	`^blink arrow and mirror arrow have `:                                   {tag: mod.SkillName("Blink Arrow", "Mirror Arrow")},
	`attacks with energy blades `:                                           {flags: mod.MFlagAttack, tag: mod.Condition("EnergyBladeActive")},

	// While in the presence of...
	`^while a unique enemy is in your presence, `:        {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`^while a pinnacle atlas boss is in your presence, `: {tag: mod.ActorCondition("enemy", "PinnacleBoss")},
}

// List of modifier tags
var modTagListCompiled map[string]CompiledList[modNameListType]
var modTagList = map[string]modNameListType{
	`on enemies`:                       {},
	`while active`:                     {},
	` on critical strike`:              {tag: mod.Condition("CriticalStrike")},
	`from critical strikes`:            {tag: mod.Condition("CriticalStrike")},
	`with critical strikes`:            {tag: mod.Condition("CriticalStrike")},
	`while affected by auras you cast`: {affectedByAura: true},
	`for you and nearby allies`:        {newAura: true},

	// Multipliers
	`per power charge`:      {tag: mod.Multiplier("PowerCharge").Base(0)},
	`per frenzy charge`:     {tag: mod.Multiplier("FrenzyCharge").Base(0)},
	`per endurance charge`:  {tag: mod.Multiplier("EnduranceCharge").Base(0)},
	`per siphoning charge`:  {tag: mod.Multiplier("SiphoningCharge").Base(0)},
	`per challenger charge`: {tag: mod.Multiplier("ChallengerCharge").Base(0)},
	`per gale force`:        {tag: mod.Multiplier("GaleForce").Base(0)},
	`per intensity`:         {tag: mod.Multiplier("Intensity").Base(0)},
	`per brand`:             {tag: mod.Multiplier("ActiveBrand").Base(0)},
	`per brand, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ActiveBrand").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per blitz charge`: {tag: mod.Multiplier("BlitzCharge").Base(0)},
	`per ghost shroud`: {tag: mod.Multiplier("GhostShroud").Base(0)},
	`per crab barrier`: {tag: mod.Multiplier("CrabBarrier").Base(0)},
	`per rage`:         {tag: mod.Multiplier("Rage").Base(0)},
	`per (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Rage").Base(0).Div(utils.Float(caps[0]))}
		},
	},
	`per level`: {tag: mod.Multiplier("Level").Base(0)},
	`per (\d+) player levels`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Level").Base(0).Div(utils.Float(caps[0]))}
		},
	},
	`per defiance`:                                         {tag: mod.Multiplier("Defiance").Base(0)},
	`for each equipped normal item`:                        {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each normal item equipped`:                        {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each normal item you have equipped`:               {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each equipped magic item`:                         {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each magic item equipped`:                         {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each magic item you have equipped`:                {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each equipped rare item`:                          {tag: mod.Multiplier("RareItem").Base(0)},
	`for each rare item equipped`:                          {tag: mod.Multiplier("RareItem").Base(0)},
	`for each rare item you have equipped`:                 {tag: mod.Multiplier("RareItem").Base(0)},
	`for each equipped unique item`:                        {tag: mod.Multiplier("UniqueItem").Base(0)},
	`for each unique item equipped`:                        {tag: mod.Multiplier("UniqueItem").Base(0)},
	`for each unique item you have equipped`:               {tag: mod.Multiplier("UniqueItem").Base(0)},
	`per elder item equipped`:                              {tag: mod.Multiplier("ElderItem").Base(0)},
	`per shaper item equipped`:                             {tag: mod.Multiplier("ShaperItem").Base(0)},
	`per elder or shaper item equipped`:                    {tag: mod.Multiplier("ShaperOrElderItem").Base(0)},
	`for each corrupted item equipped`:                     {tag: mod.Multiplier("CorruptedItem").Base(0)},
	`for each equipped corrupted item`:                     {tag: mod.Multiplier("CorruptedItem").Base(0)},
	`for each uncorrupted item equipped`:                   {tag: mod.Multiplier("NonCorruptedItem").Base(0)},
	`per abyssa?l? jewel affecting you`:                    {tag: mod.Multiplier("AbyssJewel").Base(0)},
	`for each herald s?k?i?l?l? ?affecting you`:            {tag: mod.Multiplier("Herald").Base(0)},
	`for each of your aura or herald skills affecting you`: {tag: mod.Multiplier("Herald", "AuraAffectingSelf")},
	`for each type of abyssa?l? jewel affecting you`:       {tag: mod.Multiplier("AbyssJewelType").Base(0)},
	// TODO "per (.+) eye jewel affecting you, up to a maximum of %+?(%d+)%%": function(type, _, num) return { tag = { type = "Multiplier", var = (type:gsub("^%l", string.upper)) .. "EyeJewel", limit = tonumber(num), limitTotal = true } } end,
	`per sextant affecting the area`: {tag: mod.Multiplier("Sextant").Base(0)},
	`per buff on you`:                {tag: mod.Multiplier("BuffOnSelf").Base(0)},
	`per curse on enemy`:             {tag: mod.Multiplier("CurseOnEnemy").Base(0)},
	`for each curse on enemy`:        {tag: mod.Multiplier("CurseOnEnemy").Base(0)},
	`per curse on you`:               {tag: mod.Multiplier("CurseOnSelf").Base(0)},
	`per poison on you`:              {tag: mod.Multiplier("PoisonStack").Base(0)},
	`for each poison on you`:         {tag: mod.Multiplier("PoisonStack").Base(0)},
	`for each poison on you up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`per poison on you, up to (\d+) per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each poison you have inflicted recently`: {tag: mod.Multiplier("PoisonAppliedRecently").Base(0)},
	`for each poison you have inflicted recently, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonAppliedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each shocked enemy you've killed recently`: {tag: mod.Multiplier("ShockedEnemyKilledRecently").Base(0)},
	`per enemy killed recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`per (\d+) rampage kills`: {
		fn: func(caps []string) modNameListType {
			num := utils.Float(caps[0])
			return modNameListType{tag: mod.Multiplier("Rampage").Div(num).Limit(1000 / num).LimitTotal(true)}
		}},
	`per minion, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SummonedMinion").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each enemy you or your minions have killed recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByMinionsRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each enemy you or your minions have killed recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByMinionsRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each (\d+) total mana y?o?u? ?h?a?v?e? ?spent recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaSpentRecently").Div(utils.Float(caps[0]))}
		}},
	// TODO "for each (%d+) total mana you have spent recently, up to (%d+)%%": function(num, _, limit) return { tag: { type = "Multiplier", var = "ManaSpentRecently", div = num, limit = tonumber(limit), limitTotal = true } } }},
	// TODO "per (%d+) mana spent recently, up to (%d+)%%": function(num, _, limit) return { tag: { type = "Multiplier", var = "ManaSpentRecently", div = num, limit = tonumber(limit), limitTotal = true } } }},
	`for each time you've blocked in the past 10 seconds`: {tag: mod.Multiplier("BlockedPast10Sec")},
	`per enemy killed by you or your totems recently`:     {tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByTotemsRecently")},
	`per nearby enemy, up to \+?(\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`to you and allies`:                    {},
	`per red socket`:                       {tag: mod.Multiplier("RedSocketIn{SlotName}").Base(0)},
	`per green socket on main hand weapon`: {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per green socket on`:                  {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per red socket on main hand weapon`:   {tag: mod.Multiplier("RedSocketInWeapon 1").Base(0)},
	`per green socket`:                     {tag: mod.Multiplier("GreenSocketIn{SlotName}").Base(0)},
	`per blue socket`:                      {tag: mod.Multiplier("BlueSocketIn{SlotName}").Base(0)},
	`per white socket`:                     {tag: mod.Multiplier("WhiteSocketIn{SlotName}").Base(0)},
	`for each impale on enemy`:             {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	`per animated weapon`:                  {tag: mod.Multiplier("AnimatedWeapon").Actor("parent")},
	`per grasping vine`:                    {tag: mod.Multiplier("GraspingVinesCount").Base(0)},
	`per fragile regrowth`:                 {tag: mod.Multiplier("FragileRegrowthCount").Base(0)},
	`per allocated mastery passive skill`:  {tag: mod.Multiplier("AllocatedMastery").Base(0)},
	`per allocated notable passive skill`:  {tag: mod.Multiplier("AllocatedNotable").Base(0)},

	// Per stat
	`per (\d+)% of maximum mana they reserve`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaReservedPercent")}
		}},
	`per (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Str")}
		}},
	`per (\d+) dexterity`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Dex")}
		}},
	`per (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Int")}
		}},
	`per (\d+) omniscience`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Omni")}
		}},
	`per (\d+) total attributes`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Str", "Dex", "Int")}
		}},
	`per (\d+) of your lowest attribute`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestAttribute")}
		}},
	`per (\d+) reserved life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LifeReserved")}
		}},
	`per (\d+) unreserved maximum mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaUnreserved")}
		}},
	// TODO "per (%d+) unreserved maximum mana, up to (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "ManaUnreserved", div = num, limit = tonumber(limit), limitTotal = true } } }},
	`per (\d+) armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Armour")}
		}},
	`per (\d+) evasion rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Evasion")}
		}},
	// TODO "per (%d+) evasion rating, up to (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "Evasion", div = num, limit = tonumber(limit), limitTotal = true } } }},
	`per (\d+) maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShield")}
		}},
	`per (\d+) maximum life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life")}
		}},
	`per (\d+) of maximum life or maximum mana, whichever is lower`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestOfMaximumLifeAndMaximumMana")}
		}},
	`per (\d+) player maximum life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life").Actor("parent")}
		}},
	`per (\d+) maximum mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Mana")}
		}},
	// TODO "per (%d+) maximum mana, up to (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "Mana", div = num, limit = tonumber(limit), limitTotal = true } } }},
	// TODO "per (%d+) maximum mana, up to a maximum of (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "Mana", div = num, limit = tonumber(limit), limitTotal = true } } }},
	`per (\d+) accuracy rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Accuracy")}
		}},
	`per (\d+)% block chance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "BlockChance")}
		}},
	`per (\d+)% chance to block on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ShieldBlockChance")}
		}},
	`per (\d+)% chance to block attack damage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "BlockChance")}
		}},
	`per (\d+)% chance to block spell damage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "SpellBlockChance")}
		}},
	`per (\d+) of the lowest of armour and evasion rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestOfArmourAndEvasion")}
		}},
	`per (\d+) maximum energy shield on helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		}},
	`per (\d+) evasion rating on body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		}},
	`per (\d+) armour on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2")}
		}},
	`per (\d+) armour or evasion rating on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2", "EvasionOnWeapon 2")}
		}},
	`per (\d+) evasion rating on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnWeapon 2")}
		}},
	`per (\d+) maximum energy shield on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2")}
		}},
	`per (\d+) maximum energy shield on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2")}
		}},
	`per (\d+) evasion on boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		}},
	`per (\d+) armour on gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnGloves")}
		}},
	`per (\d+)% chaos resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ChaosResist")}
		}},
	`per (\d+)% cold resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ColdResistOver75")}
		}},
	`per (\d+)% lightning resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LightningResistOver75")}
		}},
	`per (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Devotion")}
		}},
	// TODO "per (%d+)% missing fire resistance, up to a maximum of (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "MissingFireResist", div = num, globalLimit = tonumber(limit), globalLimitKey = "ReplicaNebulisFire" } } }},
	// TODO "per (%d+)%% missing cold resistance, up to a maximum of (%d+)%%": function(num, _, limit) return { tag: { type = "PerStat", stat = "MissingColdResist", div = num, globalLimit = tonumber(limit), globalLimitKey = "ReplicaNebulisCold" } } }},
	`per endurance, frenzy or power charge`: {tag: mod.PerStat(0, "TotalCharges")},
	`per fortification`:                     {tag: mod.PerStat(0, "FortificationStacks")},
	`per totem`:                             {tag: mod.PerStat(0, "TotemsSummoned")},
	`per summoned totem`:                    {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each summoned totem`:               {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each time they have chained`:       {tag: mod.PerStat(0, "Chain")},
	`for each time it has chained`:          {tag: mod.PerStat(0, "Chain")},
	`for each summoned golem`:               {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`for each golem you have summoned`:      {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned golem`:                    {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned sentinel of purity`:       {tag: mod.PerStat(0, "ActiveSentinelOfPurityLimit")},
	`per summoned skeleton`:                 {tag: mod.PerStat(0, "ActiveSkeletonLimit")},
	`per skeleton you own`:                  {tag: mod.PerStat(0, "ActiveSkeletonLimit").Actor("parent")},
	`per summoned raging spirit`:            {tag: mod.PerStat(0, "ActiveRagingSpiritLimit")},
	`for each raised zombie`:                {tag: mod.PerStat(0, "ActiveZombieLimit")},
	`per zombie you own`:                    {tag: mod.PerStat(0, "ActiveZombieLimit").Actor("parent")},
	`per raised spectre`:                    {tag: mod.PerStat(0, "ActiveSpectreLimit")},
	`per spectre you own`:                   {tag: mod.PerStat(0, "ActiveSpectreLimit").Actor("parent")},
	`for each remaining chain`:              {tag: mod.PerStat(0, "ChainRemaining")},
	`for each enemy pierced`:                {tag: mod.PerStat(0, "PiercedCount")},

	// Stat conditions
	`with (\d+) or more strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		}},
	`with at least (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		}},
	`w?h?i[lf]e? you have at least (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		}},
	`w?h?i[lf]e? you have at least (\d+) dexterity`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Dex", utils.Float(caps[0]))}
		}},
	`w?h?i[lf]e? you have at least (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", utils.Float(caps[0]))}
		}},
	`at least (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", utils.Float(caps[0]))}
		}},
	`if dexterity is higher than intelligence`: {tag: mod.Condition("DexHigherThanInt")},
	`if strength is higher than intelligence`:  {tag: mod.Condition("StrHigherThanInt")},
	`w?h?i[lf]e? you have at least (\d+) maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("EnergyShield", utils.Float(caps[0]))}
		}},
	`against targets they pierce`: {tag: mod.StatThreshold("PierceCount", 1)},
	`against pierced targets`:     {tag: mod.StatThreshold("PierceCount", 1)},
	`to targets they pierce`:      {tag: mod.StatThreshold("PierceCount", 1)},
	`w?h?i[lf]e? you have at least (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Devotion", utils.Float(caps[0]))}
		}},
	`while you have at least (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Rage").Threshold(utils.Float(caps[0]))}
		}},

	// Slot conditions
	`when in main hand`:  {tag: mod.SlotNumber(1)},
	`when in off hand`:   {tag: mod.SlotNumber(2)},
	`in main hand`:       {tag: mod.InSlot(1)},
	`in off hand`:        {tag: mod.InSlot(2)},
	`w?i?t?h? main hand`: {tagList: []mod.Tag{mod.Condition("MainHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`w?i?t?h? off hand`:  {tagList: []mod.Tag{mod.Condition("OffHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`[fi]?[rn]?[of]?[ml]?[ i]?[hc]?[it]?[te]?[sd]? ? with this weapon`: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`if your other ring is a shaper item`:                              {tag: mod.Condition("ShaperItemInRing {OtherSlotNum}")},
	`if your other ring is an elder item`:                              {tag: mod.Condition("ElderItemInRing {OtherSlotNum}")},
	// TODO "if you have a (%a+) (%a+) in (%a+) slot": function(_, rarity, item, slot) return { tag: { type = "Condition", var = rarity:gsub("^%l", string.upper).."ItemIn"..item:gsub("^%l", string.upper).." "..(slot == "right" and 2 or slot == "left" and 1) } } }},
	`of skills supported by spellslinger`: {tag: mod.Condition("SupportedBySpellslinger")},
	// TODO Equipment conditions
	// TODO "while holding a (%w+)": function (_, gear) return {
	//	tag: { type = "Condition", varList = { "Using"..firstToUpper(gear) } }
	//} }},
	// TODO "while holding a (%w+) or (%w+)": function (_, g1, g2) return {
	//	tag: { type = "Condition", varList = { "Using"..firstToUpper(g1), "Using"..firstToUpper(g2) } }
	//} }},
	`while your off hand is empty`:              {tag: mod.Condition("OffHandIsEmpty")},
	`with shields`:                              {tag: mod.Condition("UsingShield")},
	`while dual wielding`:                       {tag: mod.Condition("DualWielding")},
	`while dual wielding claws`:                 {tag: mod.Condition("DualWieldingClaws")},
	`while dual wielding or holding a shield`:   {tag: mod.Condition("DualWielding", "UsingShield")},
	`while wielding an axe`:                     {tag: mod.Condition("UsingAxe")},
	`while wielding an axe or sword`:            {tag: mod.Condition("UsingAxe", "UsingSword")},
	`while wielding a bow`:                      {tag: mod.Condition("UsingBow")},
	`while wielding a claw`:                     {tag: mod.Condition("UsingClaw")},
	`while wielding a dagger`:                   {tag: mod.Condition("UsingDagger")},
	`while wielding a claw or dagger`:           {tag: mod.Condition("UsingClaw", "UsingDagger")},
	`while wielding a mace`:                     {tag: mod.Condition("UsingMace")},
	`while wielding a mace or sceptre`:          {tag: mod.Condition("UsingMace")},
	`while wielding a mace, sceptre or staff`:   {tag: mod.Condition("UsingMace", "UsingStaff")},
	`while wielding a staff`:                    {tag: mod.Condition("UsingStaff")},
	`while wielding a sword`:                    {tag: mod.Condition("UsingSword")},
	`while wielding a melee weapon`:             {tag: mod.Condition("UsingMeleeWeapon")},
	`while wielding a one handed weapon`:        {tag: mod.Condition("UsingOneHandedWeapon")},
	`while wielding a two handed weapon`:        {tag: mod.Condition("UsingTwoHandedWeapon")},
	`while wielding a two handed melee weapon`:  {tagList: []mod.Tag{mod.Condition("UsingTwoHandedWeapon"), mod.Condition("UsingMeleeWeapon")}},
	`while wielding a wand`:                     {tag: mod.Condition("UsingWand")},
	`while wielding two different weapon types`: {tag: mod.Condition("WieldingDifferentWeaponTypes")},
	`while unarmed`:                             {tag: mod.Condition("Unarmed")},
	`while you are unencumbered`:                {tag: mod.Condition("Unencumbered")},
	`equipped bow`:                              {tag: mod.Condition("UsingBow")},
	`with a normal item equipped`:               {tag: mod.MultiplierThreshold("NormalItem").Threshold(1)},
	`with a magic item equipped`:                {tag: mod.MultiplierThreshold("MagicItem").Threshold(1)},
	`with a rare item equipped`:                 {tag: mod.MultiplierThreshold("RareItem").Threshold(1)},
	`with a unique item equipped`:               {tag: mod.MultiplierThreshold("UniqueItem").Threshold(1)},
	`if you wear no corrupted items`:            {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if no worn items are corrupted`:            {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if no equipped items are corrupted`:        {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if all worn items are corrupted`:           {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
	`if all equipped items are corrupted`:       {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
	`if equipped shield has at least (\d+)% chance to block`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("ShieldBlockChance", utils.Float(caps[0]))}
		},
	},
	`if you have (\d+) primordial items socketed or equipped`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PrimordialItem").Threshold(utils.Float(caps[0]))}
		},
	},

	// Player status conditions
	`wh[ie][ln]e? on low life`:                         {tag: mod.Condition("LowLife")},
	`wh[ie][ln]e? not on low life`:                     {tag: mod.Condition("LowLife").Neg(true)},
	`wh[ie][ln]e? on low mana`:                         {tag: mod.Condition("LowMana")},
	`wh[ie][ln]e? not on low mana`:                     {tag: mod.Condition("LowMana").Neg(true)},
	`wh[ie][ln]e? on full life`:                        {tag: mod.Condition("FullLife")},
	`wh[ie][ln]e? not on full life`:                    {tag: mod.Condition("FullLife").Neg(true)},
	`wh[ie][ln]e? no life is reserved`:                 {tag: mod.StatThreshold("LifeReserved", 0).Upper(true)},
	`wh[ie][ln]e? no mana is reserved`:                 {tag: mod.StatThreshold("ManaReserved", 0).Upper(true)},
	`wh[ie][ln]e? on full energy shield`:               {tag: mod.Condition("FullEnergyShield")},
	`wh[ie][ln]e? not on full energy shield`:           {tag: mod.Condition("FullEnergyShield").Neg(true)},
	`wh[ie][ln]e? you have energy shield`:              {tag: mod.Condition("HaveEnergyShield")},
	`wh[ie][ln]e? you have no energy shield`:           {tag: mod.Condition("HaveEnergyShield").Neg(true)},
	`if you have energy shield`:                        {tag: mod.Condition("HaveEnergyShield")},
	`while stationary`:                                 {tag: mod.Condition("Stationary")},
	`while moving`:                                     {tag: mod.Condition("Moving")},
	`while channelling`:                                {tag: mod.Condition("Channelling")},
	`if you've been channelling for at least 1 second`: {tag: mod.Condition("Channelling")},
	`if you've inflicted exposure recently`:            {tag: mod.Condition("AppliedExposureRecently")},
	`while you have no power charges`:                  {tag: mod.StatThreshold("PowerCharges", 0).Upper(true)},
	`while you have no frenzy charges`:                 {tag: mod.StatThreshold("FrenzyCharges", 0).Upper(true)},
	`while you have no endurance charges`:              {tag: mod.StatThreshold("EnduranceCharges", 0).Upper(true)},
	`while you have a power charge`:                    {tag: mod.StatThreshold("PowerCharges", 1)},
	`while you have a frenzy charge`:                   {tag: mod.StatThreshold("FrenzyCharges", 1)},
	`while you have an endurance charge`:               {tag: mod.StatThreshold("EnduranceCharges", 1)},
	`while at maximum power charges`:                   {tag: mod.StatThreshold("PowerCharges", 0).ThresholdStat("PowerChargesMax")},
	`while at maximum frenzy charges`:                  {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while on full frenzy charges`:                     {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while at maximum endurance charges`:               {tag: mod.StatThreshold("EnduranceCharges", 0).ThresholdStat("EnduranceChargesMax")},
	`while at maximum fortification`:                   {tag: mod.Condition("HaveMaximumFortification")},
	`while you have at least (\d+) crab barriers`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("CrabBarriers", utils.Float(caps[0]))}
		}},
	`while you have at least (\d+) fortification`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("FortificationStacks", utils.Float(caps[0]))}
		}},
	`while you have at least (\d+) total endurance, frenzy and power charges`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("TotalCharges").Threshold(utils.Float(caps[0]))}
		}},
	`while you have a totem`:                      {tag: mod.Condition("HaveTotem")},
	`while you have at least one nearby ally`:     {tag: mod.MultiplierThreshold("NearbyAlly").Threshold(1)},
	`while you have fortify`:                      {tag: mod.Condition("Fortified")},
	`while you have phasing`:                      {tag: mod.Condition("Phasing")},
	`while you have elusive`:                      {tag: mod.Condition("Elusive")},
	`while physical aegis is depleted`:            {tag: mod.Condition("PhysicalAegisDepleted")},
	`during onslaught`:                            {tag: mod.Condition("Onslaught")},
	`while you have onslaught`:                    {tag: mod.Condition("Onslaught")},
	`while phasing`:                               {tag: mod.Condition("Phasing")},
	`while you have tailwind`:                     {tag: mod.Condition("Tailwind")},
	`while elusive`:                               {tag: mod.Condition("Elusive")},
	`gain elusive`:                                {tag: mod.Condition("CanBeElusive", "Elusive")},
	`while you have arcane surge`:                 {tag: mod.Condition("AffectedByArcaneSurge")},
	`while you have cat's stealth`:                {tag: mod.Condition("AffectedByCat'sStealth")},
	`while you have cat's agility`:                {tag: mod.Condition("AffectedByCat'sAgility")},
	`while you have avian's might`:                {tag: mod.Condition("AffectedByAvian'sMight")},
	`while you have avian's flight`:               {tag: mod.Condition("AffectedByAvian'sFlight")},
	`while affected by aspect of the cat`:         {tag: mod.Condition("AffectedByCat'sStealth", "AffectedByCat'sAgility")},
	`while affected by a non\-vaal guard skill`:   {tag: mod.Condition("AffectedByNonVaalGuardSkill")},
	`if a non\-vaal guard buff was lost recently`: {tag: mod.Condition("LostNonVaalBuffRecently")},
	`while affected by a guard skill buff`:        {tag: mod.Condition("AffectedByGuardSkill")},
	`while affected by a herald`:                  {tag: mod.Condition("AffectedByHerald")},
	`while fortified`:                             {tag: mod.Condition("Fortified")},
	`while in blood stance`:                       {tag: mod.Condition("BloodStance")},
	`while in sand stance`:                        {tag: mod.Condition("SandStance")},
	`while you have a bestial minion`:             {tag: mod.Condition("HaveBestialMinion")},
	`while you have infusion`:                     {tag: mod.Condition("InfusionActive")},
	`while focus?sed`:                             {tag: mod.Condition("Focused")},
	`while leeching`:                              {tag: mod.Condition("Leeching")},
	`while leeching energy shield`:                {tag: mod.Condition("LeechingEnergyShield")},
	`while using a flask`:                         {tag: mod.Condition("UsingFlask")},
	`during effect`:                               {tag: mod.Condition("UsingFlask")},
	`during flask effect`:                         {tag: mod.Condition("UsingFlask")},
	`during any flask effect`:                     {tag: mod.Condition("UsingFlask")},
	`while under no flask effects`:                {tag: mod.Condition("UsingFlask").Neg(true)},
	`during effect of any mana flask`:             {tag: mod.Condition("UsingManaFlask")},
	`during effect of any life flask`:             {tag: mod.Condition("UsingLifeFlask")},
	`during effect of any life or mana flask`:     {tag: mod.Condition("UsingManaFlask", "UsingLifeFlask")},
	`while on consecrated ground`:                 {tag: mod.Condition("OnConsecratedGround")},
	`on burning ground`:                           {tag: mod.Condition("OnBurningGround")},
	`while on burning ground`:                     {tag: mod.Condition("OnBurningGround")},
	`on chilled ground`:                           {tag: mod.Condition("OnChilledGround")},
	`on shocked ground`:                           {tag: mod.Condition("OnShockedGround")},
	`while in a caustic cloud`:                    {tag: mod.Condition("OnCausticCloud")},
	`while blinded`:                               {tag: mod.Condition("Blinded")},
	`while burning`:                               {tag: mod.Condition("Burning")},
	`while ignited`:                               {tag: mod.Condition("Ignited")},
	`while you are ignited`:                       {tag: mod.Condition("Ignited")},
	`while chilled`:                               {tag: mod.Condition("Chilled")},
	`while you are chilled`:                       {tag: mod.Condition("Chilled")},
	`while frozen`:                                {tag: mod.Condition("Frozen")},
	`while shocked`:                               {tag: mod.Condition("Shocked")},
	`while you are shocked`:                       {tag: mod.Condition("Shocked")},
	`while you are bleeding`:                      {tag: mod.Condition("Bleeding")},
	`while not ignited, frozen or shocked`:        {tag: mod.Condition("Ignited", "Frozen", "Shocked").Neg(true)},
	`while bleeding`:                              {tag: mod.Condition("Bleeding")},
	`while poisoned`:                              {tag: mod.Condition("Poisoned")},
	`while you are poisoned`:                      {tag: mod.Condition("Poisoned")},
	`while cursed`:                                {tag: mod.Condition("Cursed")},
	`while not cursed`:                            {tag: mod.Condition("Cursed").Neg(true)},
	`against damage over time`:                    {tag: mod.Condition("AgainstDamageOverTime")},
	`while there is only one nearby enemy`:        {tagList: []mod.Tag{mod.Multiplier("NearbyEnemies").Limit(1), mod.Condition("OnlyOneNearbyEnemy")}},
	`while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby`:                      {tag: mod.ActorCondition("enemy", "NearbyRareOrUniqueEnemy", "RareOrUnique")},
	`if you[' ]h?a?ve hit recently`:                                                    {tag: mod.Condition("HitRecently")},
	`if you[' ]h?a?ve hit an enemy recently`:                                           {tag: mod.Condition("HitRecently")},
	`if you[' ]h?a?ve hit with your main hand weapon recently`:                         {tag: mod.Condition("HitRecentlyWithWeapon")},
	`if you[' ]h?a?ve hit with your off hand weapon recently`:                          {tagList: []mod.Tag{mod.Condition("HitRecentlyWithWeapon"), mod.Condition("DualWielding")}},
	`if you[' ]h?a?ve hit a cursed enemy recently`:                                     {tagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "Cursed")}},
	`if you[' ]h?a?ve crit recently`:                                                   {tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve dealt a critical strike recently`:                                {tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve dealt a critical strike with this weapon recently`:               {tag: mod.Condition("CritRecently")}, // Replica Kongor's
	`if you[' ]h?a?ve crit in the past 8 seconds`:                                      {tag: mod.Condition("CritInPast8Sec")},
	`if you[' ]h?a?ve dealt a crit in the past 8 seconds`:                              {tag: mod.Condition("CritInPast8Sec")},
	`if you[' ]h?a?ve dealt a critical strike in the past 8 seconds`:                   {tag: mod.Condition("CritInPast8Sec")},
	`if you haven't crit recently`:                                                     {tag: mod.Condition("CritRecently").Neg(true)},
	`if you haven't dealt a critical strike recently`:                                  {tag: mod.Condition("CritRecently").Neg(true)},
	`if you[' ]h?a?ve dealt a non\-critical strike recently`:                           {tag: mod.Condition("NonCritRecently")},
	`if your skills have dealt a critical strike recently`:                             {tag: mod.Condition("SkillCritRecently")},
	`if you dealt a critical strike with a herald skill recently`:                      {tag: mod.Condition("CritWithHeraldSkillRecently")},
	`if you[' ]h?a?ve dealt a critical strike with a two handed melee weapon recently`: {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee, tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve killed recently`:                                                 {tag: mod.Condition("KilledRecently")},
	`if you[' ]h?a?ve killed an enemy recently`:                                        {tag: mod.Condition("KilledRecently")},
	`if you[' ]h?a?ve killed at least (\d) enemies recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("EnemyKilledRecently").Threshold(utils.Float(caps[0]))}
		}},
	`if you haven't killed recently`:                                              {tag: mod.Condition("KilledRecently").Neg(true)},
	`if you or your totems have killed recently`:                                  {tag: mod.Condition("KilledRecently", "TotemsKilledRecently")},
	`if you[' ]h?a?ve thrown a trap or mine recently`:                             {tag: mod.Condition("TrapOrMineThrownRecently")},
	`if you[' ]h?a?ve killed a maimed enemy recently`:                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Maimed")}},
	`if you[' ]h?a?ve killed a cursed enemy recently`:                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Cursed")}},
	`if you[' ]h?a?ve killed a bleeding enemy recently`:                           {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Bleeding")}},
	`if you[' ]h?a?ve killed an enemy affected by your damage over time recently`: {tag: mod.Condition("KilledAffectedByDotRecently")},
	`if you[' ]h?a?ve frozen an enemy recently`:                                   {tag: mod.Condition("FrozenEnemyRecently")},
	`if you[' ]h?a?ve chilled an enemy recently`:                                  {tag: mod.Condition("ChilledEnemyRecently")},
	`if you[' ]h?a?ve ignited an enemy recently`:                                  {tag: mod.Condition("IgnitedEnemyRecently")},
	`if you[' ]h?a?ve shocked an enemy recently`:                                  {tag: mod.Condition("ShockedEnemyRecently")},
	`if you[' ]h?a?ve stunned an enemy recently`:                                  {tag: mod.Condition("StunnedEnemyRecently")},
	`if you[' ]h?a?ve stunned an enemy with a two handed melee weapon recently`:   {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee, tag: mod.Condition("StunnedEnemyRecently")},
	`if you[' ]h?a?ve been hit recently`:                                          {tag: mod.Condition("BeenHitRecently")},
	`if you[' ]h?a?ve been hit by an attack recently`:                             {tag: mod.Condition("BeenHitByAttackRecently")},
	`if you were hit recently`:                                                    {tag: mod.Condition("BeenHitRecently")},
	`if you were damaged by a hit recently`:                                       {tag: mod.Condition("BeenHitRecently")},
	`if you[' ]h?a?ve taken a critical strike recently`:                           {tag: mod.Condition("BeenCritRecently")},
	`if you[' ]h?a?ve taken a savage hit recently`:                                {tag: mod.Condition("BeenSavageHitRecently")},
	`if you have ?n[o']t been hit recently`:                                       {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you have ?n[o']t been hit by an attack recently`:                          {tag: mod.Condition("BeenHitByAttackRecently").Neg(true)},
	`if you[' ]h?a?ve taken no damage from hits recently`:                         {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you[' ]h?a?ve taken fire damage from a hit recently`:                      {tag: mod.Condition("HitByFireDamageRecently")},
	`if you[' ]h?a?ve taken fire damage from an enemy hit recently`:               {tag: mod.Condition("TakenFireDamageFromEnemyHitRecently")},
	`if you[' ]h?a?ve taken spell damage recently`:                                {tag: mod.Condition("HitBySpellDamageRecently")},
	`if you haven't taken damage recently`:                                        {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you[' ]h?a?ve blocked recently`:                                           {tag: mod.Condition("BlockedRecently")},
	`if you haven't blocked recently`:                                             {tag: mod.Condition("BlockedRecently").Neg(true)},
	`if you[' ]h?a?ve blocked an attack recently`:                                 {tag: mod.Condition("BlockedAttackRecently")},
	`if you[' ]h?a?ve blocked attack damage recently`:                             {tag: mod.Condition("BlockedAttackRecently")},
	`if you[' ]h?a?ve blocked a spell recently`:                                   {tag: mod.Condition("BlockedSpellRecently")},
	`if you[' ]h?a?ve blocked spell damage recently`:                              {tag: mod.Condition("BlockedSpellRecently")},
	`if you[' ]h?a?ve blocked damage from a unique enemy in the past 10 seconds`:  {tag: mod.Condition("BlockedHitFromUniqueEnemyInPast10Sec")},
	`if you[' ]h?a?ve attacked recently`:                                          {tag: mod.Condition("AttackedRecently")},
	`if you[' ]h?a?ve cast a spell recently`:                                      {tag: mod.Condition("CastSpellRecently")},
	`if you[' ]h?a?ve consumed a corpse recently`:                                 {tag: mod.Condition("ConsumedCorpseRecently")},
	`if you[' ]h?a?ve cursed an enemy recently`:                                   {tag: mod.Condition("CursedEnemyRecently")},
	`if you[' ]h?a?ve cast a mark spell recently`:                                 {tag: mod.Condition("CastMarkRecently")},
	`if you have ?n[o']t consumed a corpse recently`:                              {tag: mod.Condition("ConsumedCorpseRecently").Neg(true)},
	`for each corpse consumed recently`:                                           {tag: mod.Multiplier("CorpseConsumedRecently").Base(0)},
	`if you[' ]h?a?ve taunted an enemy recently`:                                  {tag: mod.Condition("TauntedEnemyRecently")},
	`if you[' ]h?a?ve used a skill recently`:                                      {tag: mod.Condition("UsedSkillRecently")},
	`if you[' ]h?a?ve used a travel skill recently`:                               {tag: mod.Condition("UsedTravelSkillRecently")},
	`for each skill you've used recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SkillUsedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`if you[' ]h?a?ve used a warcry recently`:         {tag: mod.Condition("UsedWarcryRecently")},
	`if you[' ]h?a?ve warcried recently`:              {tag: mod.Condition("UsedWarcryRecently")},
	`for each time you[' ]h?a?ve warcried recently`:   {tag: mod.Multiplier("WarcryUsedRecently").Base(0)},
	`if you[' ]h?a?ve warcried in the past 8 seconds`: {tag: mod.Condition("UsedWarcryInPast8Seconds")},
	`for each of your mines detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each mine detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each mine detonated recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each of your traps triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each trap triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each trap triggered recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`if you[' ]h?a?ve used a fire skill recently`:                    {tag: mod.Condition("UsedFireSkillRecently")},
	`if you[' ]h?a?ve used a cold skill recently`:                    {tag: mod.Condition("UsedColdSkillRecently")},
	`if you[' ]h?a?ve used a fire skill in the past 10 seconds`:      {tag: mod.Condition("UsedFireSkillInPast10Sec")},
	`if you[' ]h?a?ve used a cold skill in the past 10 seconds`:      {tag: mod.Condition("UsedColdSkillInPast10Sec")},
	`if you[' ]h?a?ve used a lightning skill in the past 10 seconds`: {tag: mod.Condition("UsedLightningSkillInPast10Sec")},
	`if you[' ]h?a?ve summoned a totem recently`:                     {tag: mod.Condition("SummonedTotemRecently")},
	`if you summoned a golem in the past 8 seconds`:                  {tag: mod.Condition("SummonedGolemInPast8Sec")},
	`if you haven't summoned a totem in the past 2 seconds`:          {tag: mod.Condition("NoSummonedTotemsInPastTwoSeconds")},
	`if you[' ]h?a?ve used a minion skill recently`:                  {tag: mod.Condition("UsedMinionSkillRecently")},
	`if you[' ]h?a?ve used a movement skill recently`:                {tag: mod.Condition("UsedMovementSkillRecently")},
	`if you haven't cast dash recently`:                              {tag: mod.Condition("CastDashRecently").Neg(true)},
	`if you[' ]h?a?ve cast dash recently`:                            {tag: mod.Condition("CastDashRecently")},
	`if you[' ]h?a?ve used a vaal skill recently`:                    {tag: mod.Condition("UsedVaalSkillRecently")},
	`if you haven't used a brand skill recently`:                     {tag: mod.Condition("UsedBrandRecently").Neg(true)},
	`if you[' ]h?a?ve used a brand skill recently`:                   {tag: mod.Condition("UsedBrandRecently")},
	`if you[' ]h?a?ve spent (\d+) total mana recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		}},
	`if you[' ]h?a?ve spent life recently`: {tag: mod.MultiplierThreshold("LifeSpentRecently").Threshold(1)},
	`for 4 seconds after spending a total of (\d+) mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		}},
	`if you've impaled an enemy recently`:                {tag: mod.Condition("ImpaledRecently")},
	`if you've changed stance recently`:                  {tag: mod.Condition("ChangedStanceRecently")},
	`if you've gained a power charge recently`:           {tag: mod.Condition("GainedPowerChargeRecently")},
	`if you haven't gained a power charge recently`:      {tag: mod.Condition("GainedPowerChargeRecently").Neg(true)},
	`if you haven't gained a frenzy charge recently`:     {tag: mod.Condition("GainedFrenzyChargeRecently").Neg(true)},
	`if you've stopped taking damage over time recently`: {tag: mod.Condition("StoppedTakingDamageOverTimeRecently")},
	`during soul gain prevention`:                        {tag: mod.Condition("SoulGainPrevention")},
	`if you detonated mines recently`:                    {tag: mod.Condition("DetonatedMinesRecently")},
	`if you detonated a mine recently`:                   {tag: mod.Condition("DetonatedMinesRecently")},
	`if you[' ]h?a?ve detonated a mine recently`:         {tag: mod.Condition("DetonatedMinesRecently")},
	`if energy shield recharge has started recently`:     {tag: mod.Condition("EnergyShieldRechargeRecently")},
	`when cast on frostbolt`:                             {tag: mod.Condition("CastOnFrostbolt")},
	`branded enemy's`:                                    {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	`to enemies they're attached to`:                     {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	`for each hit you've taken recently up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("BeenHitRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each nearby enemy, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`while you have iron reflexes`:             {tag: mod.Condition("HaveIronReflexes")},
	`while you do not have iron reflexes`:      {tag: mod.Condition("HaveIronReflexes").Neg(true)},
	`while you have elemental overload`:        {tag: mod.Condition("HaveElementalOverload")},
	`while you do not have elemental overload`: {tag: mod.Condition("HaveElementalOverload").Neg(true)},
	`while you have resolute technique`:        {tag: mod.Condition("HaveResoluteTechnique")},
	`while you do not have resolute technique`: {tag: mod.Condition("HaveResoluteTechnique").Neg(true)},
	`while you have avatar of fire`:            {tag: mod.Condition("HaveAvatarOfFire")},
	`while you do not have avatar of fire`:     {tag: mod.Condition("HaveAvatarOfFire").Neg(true)},
	`if you have a summoned golem`:             {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	`while you have a summoned golem`:          {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	`if a minion has died recently`:            {tag: mod.Condition("MinionsDiedRecently")},
	`if a minion has been killed recently`:     {tag: mod.Condition("MinionsDiedRecently")},
	`while you have sacrificial zeal`:          {tag: mod.Condition("SacrificialZeal")},
	`while sane`:                               {tag: mod.Condition("Insane").Neg(true)},
	`while insane`:                             {tag: mod.Condition("Insane")},
	`while you have defiance`:                  {tag: mod.MultiplierThreshold("Defiance").Threshold(1)},
	`while affected by glorious madness`:       {tag: mod.Condition("AffectedByGloriousMadness")},

	// Enemy status conditions
	`at close range`:                        {tag: mod.Condition("AtCloseRange")},
	`against rare and unique enemies`:       {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against unique enemies`:                {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against enemies on full life`:          {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies that are on full life`: {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies on low life`:           {tag: mod.ActorCondition("enemy", "LowLife")},
	`against enemies that are on low life`:  {tag: mod.ActorCondition("enemy", "LowLife")},
	`against cursed enemies`:                {tag: mod.ActorCondition("enemy", "Cursed")},
	`of cursed enemies'`:                    {tag: mod.ActorCondition("enemy", "Cursed")},
	`when hitting cursed enemies`:           {tag: mod.ActorCondition("enemy", "Cursed"), keywordFlags: mod.KeywordFlagHit},
	`from cursed enemies`:                   {tag: mod.ActorCondition("enemy", "Cursed")},
	`against marked enemy`:                  {tag: mod.ActorCondition("enemy", "Marked")},
	`when hitting marked enemy`:             {tag: mod.ActorCondition("enemy", "Marked"), keywordFlags: mod.KeywordFlagHit},
	`from marked enemy`:                     {tag: mod.ActorCondition("enemy", "Marked")},
	`against taunted enemies`:               {tag: mod.ActorCondition("enemy", "Taunted")},
	`against bleeding enemies`:              {tag: mod.ActorCondition("enemy", "Bleeding")},
	`you inflict on bleeding enemies`:       {tag: mod.ActorCondition("enemy", "Bleeding")},
	`to bleeding enemies`:                   {tag: mod.ActorCondition("enemy", "Bleeding")},
	`from bleeding enemies`:                 {tag: mod.ActorCondition("enemy", "Bleeding")},
	`against poisoned enemies`:              {tag: mod.ActorCondition("enemy", "Poisoned")},
	`you inflict on poisoned enemies`:       {tag: mod.ActorCondition("enemy", "Poisoned")},
	`to poisoned enemies`:                   {tag: mod.ActorCondition("enemy", "Poisoned")},
	`against enemies affected by (\d+) or more poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStack").Threshold(utils.Float(caps[0])).Actor("enemy")}
		}},
	`against enemies affected by at least (\d+) poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStack").Threshold(utils.Float(caps[0])).Actor("enemy")}
		}},
	`against hindered enemies`:                                   {tag: mod.ActorCondition("enemy", "Hindered")},
	`against maimed enemies`:                                     {tag: mod.ActorCondition("enemy", "Maimed")},
	`you inflict on maimed enemies`:                              {tag: mod.ActorCondition("enemy", "Maimed")},
	`against blinded enemies`:                                    {tag: mod.ActorCondition("enemy", "Blinded")},
	`from blinded enemies`:                                       {tag: mod.ActorCondition("enemy", "Blinded")},
	`against burning enemies`:                                    {tag: mod.ActorCondition("enemy", "Burning")},
	`against ignited enemies`:                                    {tag: mod.ActorCondition("enemy", "Ignited")},
	`to ignited enemies`:                                         {tag: mod.ActorCondition("enemy", "Ignited")},
	`against shocked enemies`:                                    {tag: mod.ActorCondition("enemy", "Shocked")},
	`to shocked enemies`:                                         {tag: mod.ActorCondition("enemy", "Shocked")},
	`against frozen enemies`:                                     {tag: mod.ActorCondition("enemy", "Frozen")},
	`to frozen enemies`:                                          {tag: mod.ActorCondition("enemy", "Frozen")},
	`against chilled enemies`:                                    {tag: mod.ActorCondition("enemy", "Chilled")},
	`to chilled enemies`:                                         {tag: mod.ActorCondition("enemy", "Chilled")},
	`inflicted on chilled enemies`:                               {tag: mod.ActorCondition("enemy", "Chilled")},
	`enemies which are chilled`:                                  {tag: mod.ActorCondition("enemy", "Chilled")},
	`against chilled or frozen enemies`:                          {tag: mod.ActorCondition("enemy", "Chilled", "Frozen")},
	`against frozen, shocked or ignited enemies`:                 {tag: mod.ActorCondition("enemy", "Frozen", "Shocked", "Ignited")},
	`against enemies affected by elemental ailments`:             {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	`against enemies affected by ailments`:                       {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped", "Poisoned", "Bleeding")},
	`against enemies that are affected by elemental ailments`:    {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	`against enemies that are affected by no elemental ailments`: {tagList: []mod.Tag{mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped").Neg(true), mod.Condition("Effective")}},
	`against enemies affected by (\d+) spider's webs`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Spider's WebStack").Actor("enemy").Threshold(utils.Float(caps[0]))}
		}},
	`against enemies on consecrated ground`: {tag: mod.ActorCondition("enemy", "OnConsecratedGround")},

	// Enemy multipliers
	`per freeze, shock [ao][nr]d? ignite on enemy`: {tag: mod.Multiplier("FreezeShockIgniteOnEnemy").Base(0)},
	`per poison affecting enemy`:                   {tag: mod.Multiplier("PoisonStack").Actor("enemy")},
	`per poison affecting enemy, up to \+([\d\.]+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Actor("enemy").Limit(utils.Float(caps[0])).LimitTotal(true)}
		}},
	`for each spider's web on the enemy`: {tag: mod.Multiplier("Spider's WebStack").Actor("enemy")},
}

/*
local mod = modLib.createMod
local function flag(name, ...)
	return mod(name, "FLAG", true, ...)
end

local gemIdLookup = {
	"power charge on critical strike": "SupportPowerChargeOnCrit",
}
for name, grantedEffect in pairs(data.skills) do
	if not grantedEffect.hidden or grantedEffect.fromItem or grantedEffect.fromTree then
		gemIdLookup[grantedEffect.name:lower()] = grantedEffect.id
	end
end
local function grantedExtraSkill(name, level, noSupports)
	name = name:gsub(" skill","")
	if gemIdLookup[name] then
		return {
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[name], level = level, noSupports = noSupports })
		}
	end
end
local function triggerExtraSkill(name, level, noSupports, sourceSkill)
	name = name:gsub(" skill","")
	if sourceSkill then
		sourceSkill = sourceSkill:gsub(" skill","")
	end
	if gemIdLookup[name] then
		return {
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[name], level = level, noSupports = noSupports, triggered = true, source = sourceSkill })
		}
	end
end
*/

type SpecialFuncType func(num float64, captures []string) ([]mod.Mod, interface{})

/*
Sample:

func(num float64, captures []string) ([]mod.Mod, interface{}) {
	return []mod.Mod{}, nil
}
*/

// List of special modifiers
var specialModListCompiled map[string]CompiledList[interface{}]
var specialModList = map[string]interface{}{
	// Keystones
	`(\d+)% less damage taken for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, -num).Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"))}, nil
	},
	`modifiers to chance to suppress spell damage instead apply to chance to dodge spell hits at 50% of their value`: []mod.Mod{
		mod.NewFlag("ConvertSpellSuppressionToSpellDodge", true),
		mod.NewFloat("SpellSuppressionChance", mod.TypeOverride, 0).Source("Acrobatics"),
	},
	`maximum chance to dodge spell hits is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("SpellDodgeChanceMax", mod.TypeOverride, num).Source("Acrobatics")}, nil
	},
	`dexterity provides no inherent bonus to evasion rating`:      []mod.Mod{mod.NewFlag("NoDexBonusToEvasion", true)},
	`strength's damage bonus applies to all spell damage as well`: []mod.Mod{mod.NewFlag("IronWill", true)},
	`your hits can't be evaded`:                                   []mod.Mod{mod.NewFlag("CannotBeEvaded", true)},
	`never deal critical strikes`:                                 []mod.Mod{mod.NewFlag("NeverCrit", true), mod.NewFlag("Condition:NeverCrit", true)},
	`no critical strike multiplier`:                               []mod.Mod{mod.NewFlag("NoCritMultiplier", true)},
	`ailments never count as being from critical strikes`:         []mod.Mod{mod.NewFlag("AilmentsAreNeverFromCrit", true)},
	`the increase to physical damage from strength applies to projectile attacks as well as melee attacks`: []mod.Mod{mod.NewFlag("IronGrip", true)},
	`strength%'s damage bonus applies to projectile attack damage as well as melee damage`:                 []mod.Mod{mod.NewFlag("IronGrip", true)},
	`converts all evasion rating to armour\. dexterity provides no bonus to evasion rating`:                []mod.Mod{mod.NewFlag("NoDexBonusToEvasion", true), mod.NewFlag("IronReflexes", true)},
	`30% chance to dodge attack hits\. 50% less armour, 30% less energy shield, 30% less chance to block spell and attack damage`: []mod.Mod{
		mod.NewFloat("AttackDodgeChance", mod.TypeBase, 30),
		mod.NewFloat("Armour", mod.TypeMore, -50),
		mod.NewFloat("EnergyShield", mod.TypeMore, -30),
		mod.NewFloat("BlockChance", mod.TypeMore, -30),
		mod.NewFloat("SpellBlockChance", mod.TypeMore, -30),
	},
	`(\d+)% increased blind effect`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.NewFloat("BlindEffect", mod.TypeIncrease, num))}, nil
	},
	`\+(\d+)% chance to block spell damage for each (\d+)% overcapped chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeBase, num).Tag(mod.PerStat(utils.Float(captures[1]), "BlockChanceOverCap"))}, nil
	},
	`maximum life becomes 1, immune to chaos damage`: []mod.Mod{
		mod.NewFlag("ChaosInoculation", true),
		mod.NewFloat("ChaosDamageTaken", mod.TypeMore, -100),
	},
	`life regeneration is applied to energy shield instead`:                 []mod.Mod{mod.NewFlag("ZealotsOath", true)},
	`life leeched per second is doubled`:                                    []mod.Mod{mod.NewFloat("LifeLeechRate", mod.TypeMore, 100)},
	`total recovery per second from life leech is doubled`:                  []mod.Mod{mod.NewFloat("LifeLeechRate", mod.TypeMore, 100)},
	`maximum total recovery per second from life leech is doubled`:          []mod.Mod{mod.NewFloat("MaxLifeLeechRate", mod.TypeMore, 100)},
	`maximum total life recovery per second from leech is doubled`:          []mod.Mod{mod.NewFloat("MaxLifeLeechRate", mod.TypeMore, 100)},
	`maximum total recovery per second from energy shield leech is doubled`: []mod.Mod{mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeMore, 100)},
	`maximum total energy shield recovery per second from leech is doubled`: []mod.Mod{mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeMore, 100)},
	`life regeneration has no effect`:                                       []mod.Mod{mod.NewFlag("NoLifeRegen", true)},
	`energy shield recharge instead applies to life`:                        []mod.Mod{mod.NewFlag("EnergyShieldRechargeAppliesToLife", true)},
	`deal no non\-fire damage`:                                              []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoChaos", true)},
	`(\d+)% of physical, cold and lightning damage converted to fire damage`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("PhysicalDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("LightningDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("ColdDamageConvertToFire", mod.TypeBase, num),
		}, nil
	},
	`removes all mana\. spend life instead of mana for skills`: []mod.Mod{
		mod.NewFloat("Mana", mod.TypeMore, -100),
		mod.NewFlag("BloodMagic", true),
	},
	`removes all mana`:                                 []mod.Mod{mod.NewFloat("Mana", mod.TypeMore, -100)},
	`removes all energy shield`:                        []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeMore, -100)},
	`skills cost life instead of mana`:                 []mod.Mod{mod.NewFlag("CostLifeInsteadOfMana", true)},
	`skills reserve life instead of mana`:              []mod.Mod{mod.NewFlag("BloodMagicReserved", true)},
	`spend life instead of mana for effects of skills`: []mod.Mod{},
	`skills cost \+(\d+) rage`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("RageCostBase", mod.TypeBase, num)}, nil
	},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (\-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewList("EnemyModifier", mod.NewFloat("FireExposure", mod.TypeBase, num).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))),
		}, nil
	},
	`enemies you hit with elemental damage temporarily get (\+\d+)% resistance to those elements and (\-\d+)% resistance to other elements`: func(plus float64, captures []string) ([]mod.Mod, interface{}) {
		minus := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewFlag("ElementalEquilibriumLegacy", true),
			mod.NewList("EnemyModifier", mod.NewFloat("FireResist", mod.TypeBase, plus).Tag(mod.Condition("HitByFireDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("FireResist", mod.TypeBase, minus).Tag(mod.Condition("HitByFireDamage").Neg(true)).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdResist", mod.TypeBase, plus).Tag(mod.Condition("HitByColdDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdResist", mod.TypeBase, minus).Tag(mod.Condition("HitByColdDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningResist", mod.TypeBase, plus).Tag(mod.Condition("HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningResist", mod.TypeBase, minus).Tag(mod.Condition("HitByLightningDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))),
		}, nil
	},
	`projectile attack hits deal up to 30% more damage to targets at the start of their movement, dealing less damage to targets as the projectile travels farther`: []mod.Mod{mod.NewFlag("PointBlank", true)},
	`leech energy shield instead of life`: []mod.Mod{mod.NewFlag("GhostReaver", true)},
	`minions explode when reduced to low life, dealing 33% of their maximum life as fire damage to surrounding enemies`: []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	`minions explode when reduced to low life, dealing 33% of their life as fire damage to surrounding enemies`:         []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	`all bonuses from an equipped shield apply to your minions instead of you`:                                          []mod.Mod{}, // The node itself is detected by the code that handles it
	`spend energy shield before mana for skill m?a?n?a? ?costs`:                                                         []mod.Mod{},
	`you have perfect agony if you've dealt a critical strike recently`:                                                 []mod.Mod{mod.NewList("Keystone", "Perfect Agony").Tag(mod.Condition("CritRecently"))},
	`energy shield protects mana instead of life`:                                                                       []mod.Mod{mod.NewFlag("EnergyShieldProtectsMana", true)},
	`modifiers to critical strike multiplier also apply to damage over time multiplier for ailments from critical strikes at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("CritMultiplierAppliesToDegen", mod.TypeBase, num)}, nil
	},
	`your bleeding does not deal extra damage while the enemy is moving`: []mod.Mod{mod.NewFlag("Condition:NoExtraBleedDamageToMovingEnemy", true)},
	`you can inflict bleeding on an enemy up to (\d+) times?`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("BleedStacksMax", mod.TypeOverride, num),
			mod.NewFlag("Condition:HaveCrimsonDance", true),
		}, nil
	},
	`your minions spread caustic ground on death, dealing 20% of their maximum life as chaos damage per second`: []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "SiegebreakerCausticGround"})},
	`your minions spread burning ground on death, dealing 20% of their maximum life as fire damage per second`:  []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "ReplicaSiegebreakerBurningGround"})},
	`you can have an additional brand attached to an enemy`:                                                     []mod.Mod{mod.NewFloat("BrandsAttachedLimit", mod.TypeBase, 1)},
	`gain (\d+) grasping vines each second while stationary`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("Multiplier:GraspingVinesCount", mod.TypeBase, num).Tag(mod.Multiplier("StationarySeconds").Base(0).Limit(10).LimitTotal(true)).Tag(mod.Condition("Stationary")),
		}, nil
	},
	`all damage inflicts poison against enemies affected by at least (\d+) grasping vines`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("PoisonChance", mod.TypeBase, 100).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("FireCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("ColdCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("LightningCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
		}, nil
	},
	`attack projectiles always inflict bleeding and maim, and knock back enemies`: []mod.Mod{
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile),
		mod.NewFloat("EnemyKnockbackChance", mod.TypeBase, 100).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile),
	},
	`projectiles cannot pierce, fork or chain`: []mod.Mod{
		mod.NewFlag("CannotPierce", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotChain", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotFork", true).Flag(mod.MFlagProjectile),
	},
	`critical strikes inflict scorch, brittle and sapped`: []mod.Mod{mod.NewFlag("CritAlwaysAltAilments", true)},
	`chance to block attack damage is doubled`:            []mod.Mod{mod.NewFloat("BlockChance", mod.TypeMore, 100)},
	`chance to block spell damage is doubled`:             []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeMore, 100)},
	`you take (\d+)% of damage from blocked hits`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("BlockEffect", mod.TypeBase, num)}, nil
	},
	`ignore attribute requirements`:              []mod.Mod{mod.NewFlag("IgnoreAttributeRequirements", true)},
	`gain no inherent bonuses from attributes`:   []mod.Mod{mod.NewFlag("NoAttributeBonuses", true)},
	`gain no inherent bonuses from strength`:     []mod.Mod{mod.NewFlag("NoStrengthAttributeBonuses", true)},
	`gain no inherent bonuses from dexterity`:    []mod.Mod{mod.NewFlag("NoDexterityAttributeBonuses", true)},
	`gain no inherent bonuses from intelligence`: []mod.Mod{mod.NewFlag("NoIntelligenceAttributeBonuses", true)},
	`all damage taken bypasses energy shield`: []mod.Mod{
		mod.NewFloat("PhysicalEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("LightningEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("ColdEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("FireEnergyShieldBypass", mod.TypeBase, 100),
	},
	`auras from your skills do not affect allies`: []mod.Mod{mod.NewFlag("SelfAuraSkillsCannotAffectAllies", true)},
	`auras from your skills have (\d+)% more effect on you`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeMore, num)}, nil
	},
	`auras from your skills have (\d+)% increased effect on you`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, num)}, nil
	},
	`increases and reductions to mana regeneration rate instead apply to rage regeneration rate`: []mod.Mod{mod.NewFlag("ManaRegenToRageRegen", true)},
	`increases and reductions to maximum energy shield instead apply to ward`:                    []mod.Mod{mod.NewFlag("EnergyShieldToWard", true)},
	`(\d+)% of damage taken bypasses ward`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("WardBypass", mod.TypeBase, num)}, nil
	},
	`maximum energy shield is (\d+)`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeOverride, num)}, nil
	},
	`while not on full life, sacrifice ([\d\.]+)% of mana per second to recover that much life`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("ManaDegen", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("FullLife").Neg(true)),
			mod.NewFloat("LifeRecovery", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("FullLife").Neg(true)),
		}, nil
	},
	`you are blind`: []mod.Mod{mod.NewFlag("Condition:Blinded", true)},
	`armour applies to fire, cold and lightning damage taken from hits instead of physical damage`: []mod.Mod{
		mod.NewFlag("ArmourAppliesToFireDamageTaken", true),
		mod.NewFlag("ArmourAppliesToColdDamageTaken", true),
		mod.NewFlag("ArmourAppliesToLightningDamageTaken", true),
		mod.NewFlag("ArmourDoesNotApplyToPhysicalDamageTaken", true),
	},
	`maximum damage reduction for any damage type is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{mod.NewFloat("DamageReductionMax", mod.TypeOverride, num)}, nil
	},
	`(\d+)% of maximum mana is converted to twice that much armour`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("ManaConvertToArmour", mod.TypeBase, num),
		}, nil
	},
	`life recovery from flasks also applies to energy shield`: []mod.Mod{mod.NewFlag("LifeFlaskAppliesToEnergyShield", true)},
	`life leech effects recover energy shield instead while on full life`: []mod.Mod{
		mod.NewFlag("ImmortalAmbition", true).Tag(mod.Condition("FullLife")).Tag(mod.Condition("LeechingLife")),
	},
	`shepherd of souls`: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, -30).Tag(mod.SkillType(string(data.SkillTypeVaal)).Neg(true))},
	`adds (\d+) to (\d+) attack physical damage to melee skills per (\d+) dexterity while you are unencumbered`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		min := utils.Float(captures[0])
		max := utils.Float(captures[1])
		dex := utils.Float(captures[2])
		return []mod.Mod{
			// Hollow Palm 3 suffixes
			mod.NewFloat("PhysicalMin", mod.TypeBase, min).Flag(mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.PerStat(dex, "Dex")).Tag(mod.Condition("Unencumbered")),
			mod.NewFloat("PhysicalMax", mod.TypeBase, max).Flag(mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.PerStat(dex, "Dex")).Tag(mod.Condition("Unencumbered")),
		}, nil
	},
	`(\d+)% more attack damage if accuracy rating is higher than maximum life`: func(num float64, captures []string) ([]mod.Mod, interface{}) {
		return []mod.Mod{
			mod.NewFloat("Damage", mod.TypeMore, num).Source("Damage").Flag(mod.MFlagAttack).Tag(mod.Condition("MainHandAccRatingHigherThanMaxLife")).Tag(mod.Condition("MainHandAttack")),
			mod.NewFloat("Damage", mod.TypeMore, num).Source("Damage").Flag(mod.MFlagAttack).Tag(mod.Condition("OffHandAccRatingHigherThanMaxLife")).Tag(mod.Condition("OffHandAttack")),
		}, nil
	},
	/*
		// TODO Legacy support
		"(%d+)%% chance to defend with double armour": function(numChance) return {
			mod("ArmourDefense", "MAX", 100, "Armour Mastery: Max Calc", .Tag(mod.Condition("ArmourMax"))),
			mod("ArmourDefense", "MAX", math.min(numChance / 100, 1.0) * 100, "Armour Mastery: Average Calc", .Tag(mod.Condition("ArmourAvg"))),
			mod("ArmourDefense", "MAX", math.min(math.floor(numChance / 100), 1.0) * 100, "Armour Mastery: Min Calc", .Tag(mod.Condition("ArmourMax").Neg(true)), .Tag(mod.Condition("ArmourAvg").Neg(true))),
		} end,
		// TODO Masteries
		"off hand accuracy is equal to main hand accuracy while wielding a sword": { flag("Condition:OffHandAccuracyIsMainHandAccuracy", .Tag(mod.Condition("UsingSword"))) },
		"(%d+)%% chance to defend with (%d+)%% of armour": function(numChance, _, numArmourMultiplier) return {
			mod("ArmourDefense", "MAX", tonumber(numArmourMultiplier) - 100, "Armour Mastery: Max Calc", .Tag(mod.Condition("ArmourMax"))),
			mod("ArmourDefense", "MAX", math.min(numChance / 100, 1.0) * (tonumber(numArmourMultiplier) - 100), "Armour Mastery: Average Calc", .Tag(mod.Condition("ArmourAvg"))),
			mod("ArmourDefense", "MAX", math.min(math.floor(numChance / 100), 1.0) * (tonumber(numArmourMultiplier) - 100), "Armour Mastery: Min Calc", .Tag(mod.Condition("ArmourMax").Neg(true)), .Tag(mod.Condition("ArmourAvg").Neg(true))),
		} end,
		"defend with (%d+)%% of armour while not on low energy shield": function(num) return {
			mod("ArmourDefense", "MAX", num - 100, "Armour and Energy Shield Mastery", .Tag(mod.Condition("LowEnergyShield").Neg(true))),
		} end,
		// TODO Exerted Attacks
		"exerted attacks deal (%d+)%% increased damage": function(num) return { mod("ExertIncrease", "INC", num, nil, mod.MFlagAttack, 0) } end,
		"exerted attacks have (%d+)%% chance to deal double damage": function(num) return { mod("ExertDoubleDamageChance", "BASE", num, nil, mod.MFlagAttack, 0) } end,
		// TODO Ascendant
		"grants (%d+) passive skill points?": function(num) return { mod.NewFloat("ExtraPoints", mod.TypeBase, num) } end,
		"can allocate passives from the %a+'s starting point": { },
		"projectiles gain damage as they travel farther, dealing up to (%d+)%% increased damage with hits to targets": function(num) return { mod("Damage", "INC", num, nil, bor(mod.MFlagAttack, mod.MFlagProjectile), { type = "DistanceRamp", ramp = {{35,0},{70,1}} }) } end,
		"(%d+)%% chance to gain elusive on kill": {
			mod.NewFlag("Condition:CanBeElusive", true),
		},
		"immune to elemental ailments while on consecrated ground": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("OnConsecratedGround")))
			end
			return mods
		end,
		// TODO Assassin
		"poison you inflict with critical strikes deals (%d+)%% more damage": function(num) return { mod("Damage", "MORE", num, nil, 0, mod.KeywordFlagPoison, .Tag(mod.Condition("CriticalStrike"))) } end,
		"(%d+)%% chance to gain elusive on critical strike": {
			mod.NewFlag("Condition:CanBeElusive", true),
		},
		"(%d+)%% more damage while there is at most one rare or unique enemy nearby": function(num) return { mod("Damage", "MORE", num, nil, 0, .Tag(mod.Condition("AtMostOneNearbyRareOrUniqueEnemy"))) } end,
		"(%d+)%% reduced damage taken while there are at least two rare or unique enemies nearby": function(num) return { mod("DamageTaken", "INC", -num, nil, 0, { type = "MultiplierThreshold", var = "NearbyRareOrUniqueEnemies", threshold = 2 }) } end,
		"you take no extra damage from critical strikes while elusive": function(num) return { mod("ReduceCritExtraDamage", "BASE", 100, .Tag(mod.Condition("Elusive"))) } end,
		// TODO Berserker
		"gain %d+ rage when you kill an enemy": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"gain %d+ rage when you use a warcry": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"you and nearby party members gain %d+ rage when you warcry": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"gain %d+ rage on hit with attacks, no more than once every [%d%.]+ seconds": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"inherent effects from having rage are tripled": { mod.NewFloat("Multiplier:RageEffect", mod.TypeBase, 2) },
		"cannot be stunned while you have at least (%d+) rage": function(num) return { mod("AvoidStun", "BASE", 100, { type = "MultiplierThreshold", var = "Rage", threshold = num }) } end,
		"lose ([%d%.]+)%% of life per second per rage while you are not losing rage": function(num) return { mod("LifeDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }, { type = "Multiplier", var = "Rage"}) } end,
		"if you've warcried recently, you and nearby allies have (%d+)%% increased attack speed": function(num) return { mod("ExtraAura", "LIST", { mod = mod("Speed", "INC", num, nil, mod.MFlagAttack) }, .Tag(mod.Condition("UsedWarcryRecently"))) } end,
		"gain (%d+)%% increased armour per (%d+) power for 8 seconds when you warcry, up to a maximum of (%d+)%%": function(num, _, div, limit) return {
			mod("Armour", "INC", num, { type = "Multiplier", var = "WarcryPower", div = tonumber(div), globalLimit = tonumber(limit), globalLimitKey = "WarningCall" }, .Tag(mod.Condition("UsedWarcryInPast8Seconds")))
		} end,
		"warcries grant (%d+) rage per (%d+) power if you have less than (%d+) rage": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"exerted attacks deal (%d+)%% more attack damage if a warcry sacrificed rage recently": function(num) return { mod("ExertAttackIncrease", "MORE", num, nil, mod.MFlagAttack, 0) } end,
		// TODO Champion
		"cannot be stunned while you have fortify": { mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("Fortified"))) },
		"cannot be stunned while fortified": { mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("Fortified"))) },
		"fortify": { mod.NewFlag("Condition:Fortified", true) },
		"you have (%d+) fortification": function(num) return {
			mod.NewFlag("Condition:Fortified", true)
		} end,
		"enemies taunted by you cannot evade attacks": { mod("EnemyModifier", "LIST", { mod = flag("CannotEvade", .Tag(mod.Condition("Taunted"))) }) },
		"if you've impaled an enemy recently, you and nearby allies have %+(%d+) to armour": function (num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("Armour", mod.TypeBase, num) }, .Tag(mod.Condition("ImpaledRecently"))) } end,
		"your hits permanently intimidate enemies that are on full life": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Intimidated", true)} )},
		"you and allies affected by your placed banners regenerate ([%d%.]+)%% of life per second for each stage": function(num) return {
			mod("LifeRegenPercent", "BASE", num, .Tag(mod.Condition("AffectedByPlacedBanner")), mod.Multiplier("BannerStage").Base(0))
		} end,
		// TODO Chieftain
		"enemies near your totems take (%d+)%% increased physical and fire damage": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, num) }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("FireDamageTaken", mod.TypeIncrease, num) }),
		} end,
		"every %d+ seconds, gain (%d+)%% of physical damage as extra fire damage for %d+ seconds": function(_, num, _) return {
			mod("PhysicalDamageGainAsFire", "BASE", num, .Tag(mod.Condition("NgamahuFlamesAdvance"))),
		} end,
		"(%d+)%% more damage for each endurance charge lost recently, up to (%d+)%%": function(num, _, limit) return {
			mod("Damage", "MORE", num, { type = "Multiplier", var = "EnduranceChargesLostRecently", limit = tonumber(limit), limitTotal = true }),
		} end,
		"(%d+)%% more damage if you've lost an endurance charge in the past 8 seconds": function(num) return { mod("Damage", "MORE", num, .Tag(mod.Condition("LostEnduranceChargeInPast8Sec")))	} end,
		"trigger level (%d+) (.+) when you attack with a non%-vaal slam skill near an enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		// TODO Deadeye
		"projectiles pierce all nearby targets": { mod.NewFlag("PierceAllTargets", true) },
		"gain %+(%d+) life when you hit a bleeding enemy": function(num) return { mod("LifeOnHit", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Bleeding" }) } end,
		"accuracy rating is doubled": { mod.NewFloat("Accuracy", mod.TypeMore, 100) },
		"(%d+)%% increased blink arrow and mirror arrow cooldown recovery speed": function(num) return {
			mod("CooldownRecovery", "INC", num, { type = "SkillName", skillNameList = { "Blink Arrow", "Mirror Arrow" } }),
		} end,
		"critical strikes which inflict bleeding also inflict rupture": function() return {
			flag("Condition:CanInflictRupture", { type = "Condition", neg = true, var = "NeverCrit"}),
		} end,
		"gain %d+ gale force when you use a skill": {
			mod.NewFlag("Condition:CanGainGaleForce", true),
		},
		"if you've used a skill recently, you and nearby allies have tailwind": { mod("ExtraAura", "LIST", { mod = mod.NewFlag("Condition:Tailwind", true) }, .Tag(mod.Condition("UsedSkillRecently"))) },
		"you and nearby allies have tailwind": { mod("ExtraAura", "LIST", { mod = mod.NewFlag("Condition:Tailwind", true) }) },
		"projectiles deal (%d+)%% more damage for each remaining chain": function(num) return { mod("Damage", "MORE", num, nil, mod.MFlagProjectile, { type = "PerStat", stat = "ChainRemaining" }) } end,
		"projectiles deal (%d+)%% increased damage for each remaining chain": function(num) return { mod("Damage", "INC", num, nil, mod.MFlagProjectile, { type = "PerStat", stat = "ChainRemaining" }) } end,
		"far shot": { mod.NewFlag("FarShot", true) },
		"(%d+)%% increased mirage archer duration": function(num) return { mod.NewFloat("MirageArcherDuration", mod.TypeIncrease, num), } end,
		"([%-%+]%d+) to maximum number of summoned mirage archers": function(num) return { mod.NewFloat("MirageArcherMaxCount", mod.TypeBase, num),	} end,
		// TODO Elementalist
		"gain (%d+)%% increased area of effect for %d+ seconds": function(num) return { mod("AreaOfEffect", "INC", num, .Tag(mod.Condition("PendulumOfDestructionAreaOfEffect"))) } end,
		"gain (%d+)%% increased elemental damage for %d+ seconds": function(num) return { mod("ElementalDamage", "INC", num, .Tag(mod.Condition("PendulumOfDestructionElementalDamage"))) } end,
		"for each element you've been hit by damage of recently, (%d+)%% increased damage of that element": function(num) return {
			mod("FireDamage", "INC", num, .Tag(mod.Condition("HitByFireDamageRecently"))),
			mod("ColdDamage", "INC", num, .Tag(mod.Condition("HitByColdDamageRecently"))),
			mod("LightningDamage", "INC", num, .Tag(mod.Condition("HitByLightningDamageRecently"))),
		} end,
		"for each element you've been hit by damage of recently, (%d+)%% reduced damage taken of that element": function(num) return {
			mod("FireDamageTaken", "INC", -num, .Tag(mod.Condition("HitByFireDamageRecently"))),
			mod("ColdDamageTaken", "INC", -num, .Tag(mod.Condition("HitByColdDamageRecently"))),
			mod("LightningDamageTaken", "INC", -num, .Tag(mod.Condition("HitByLightningDamageRecently"))),
		} end,
		"gain convergence when you hit a unique enemy, no more than once every %d+ seconds": {
			mod.NewFlag("Condition:CanGainConvergence", true),
		},
		"(%d+)%% increased area of effect while you don't have convergence": function(num) return { mod("AreaOfEffect", "INC", num, { type = "Condition", neg = true, var = "Convergence" }) } end,
		"exposure you inflict applies an extra (%-?%d+)%% to the affected resistance": function(num) return { mod.NewFloat("ExtraExposure", mod.TypeBase, num) } end,
		"cannot take reflected elemental damage": { mod("ElementalReflectedDamageTaken", "MORE", -100) },
		"every %d+ seconds:": { },
		"gain chilling conflux for %d seconds": {
			flag("PhysicalCanChill", .Tag(mod.Condition("ChillingConflux"))),
			flag("LightningCanChill", .Tag(mod.Condition("ChillingConflux"))),
			flag("FireCanChill", .Tag(mod.Condition("ChillingConflux"))),
			flag("ChaosCanChill", .Tag(mod.Condition("ChillingConflux"))),
		},
		"gain shocking conflux for %d seconds": {
			mod("EnemyShockChance", "BASE", 100, .Tag(mod.Condition("ShockingConflux"))),
			flag("PhysicalCanShock", .Tag(mod.Condition("ShockingConflux"))),
			flag("ColdCanShock", .Tag(mod.Condition("ShockingConflux"))),
			flag("FireCanShock", .Tag(mod.Condition("ShockingConflux"))),
			flag("ChaosCanShock", .Tag(mod.Condition("ShockingConflux"))),
		},
		"gain igniting conflux for %d seconds": {
			mod("EnemyIgniteChance", "BASE", 100, .Tag(mod.Condition("IgnitingConflux"))),
			flag("PhysicalCanIgnite", .Tag(mod.Condition("IgnitingConflux"))),
			flag("LightningCanIgnite", .Tag(mod.Condition("IgnitingConflux"))),
			flag("ColdCanIgnite", .Tag(mod.Condition("IgnitingConflux"))),
			flag("ChaosCanIgnite", .Tag(mod.Condition("IgnitingConflux"))),
		},
		"gain chilling, shocking and igniting conflux for %d seconds": { },
		"you have igniting, chilling and shocking conflux while affected by glorious madness": {
			flag("PhysicalCanChill", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("LightningCanChill", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("FireCanChill", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ChaosCanChill", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			mod("EnemyIgniteChance", "BASE", 100, .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("PhysicalCanIgnite", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("LightningCanIgnite", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ColdCanIgnite", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ChaosCanIgnite", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			mod("EnemyShockChance", "BASE", 100, .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("PhysicalCanShock", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ColdCanShock", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("FireCanShock", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ChaosCanShock", .Tag(mod.Condition("AffectedByGloriousMadness"))),
		},
		"immune to elemental ailments while affected by glorious madness": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("AffectedByGloriousMadness")))
			end
			return mods
		end,
		"summoned golems are immune to elemental damage": {
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("FireResist", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("FireResistMax", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("ColdResist", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("ColdResistMax", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("LightningResist", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod.NewFloat("LightningResistMax", mod.TypeOverride, 100) }, mod.SkillType(data.SkillTypeGolem)),
		},
		"(%d+)%% increased golem damage per summoned golem": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, num) }, mod.SkillType(data.SkillTypeGolem), { type = "PerStat", stat = "ActiveGolemLimit" }) } end,
		"shocks from your hits always increase damage taken by at least (%d+)%%": function(num) return { mod.NewFloat("ShockBase", mod.TypeBase, num) } end,
		"chills from your hits always reduce action speed by at least (%d+)%%": function(num) return { mod.NewFloat("ChillBase", mod.TypeBase, num) } end,
		"(%d+)%% more damage with ignites you inflict with hits for which the highest damage type is fire": function(num) return { mod("Damage", "MORE", num, nil, 0, mod.KeywordFlagIgnite, .Tag(mod.Condition("FireIsHighestDamageType"))) } end,
		"(%d+)%% more effect of cold ailments you inflict with hits for which the highest damage type is cold": function(num) return {
			mod("EnemyChillEffect", "MORE", num, .Tag(mod.Condition("ColdIsHighestDamageType"))),
			mod("EnemyBrittleEffect", "MORE", num, .Tag(mod.Condition("ColdIsHighestDamageType"))),
		} end,
		"(%d+)%% more effect of lightning ailments you inflict with hits if the highest damage type is lightning": function(num) return {
			mod("EnemyShockEffect", "MORE", num, .Tag(mod.Condition("LightningIsHighestDamageType"))),
			mod("EnemySapEffect", "MORE", num, .Tag(mod.Condition("LightningIsHighestDamageType"))),
		} end,
		"your chills can reduce action speed by up to a maximum of (%d+)%%": function(num) return { mod.NewFloat("ChillMax", mod.TypeOverride, num) } end,
		"your hits always ignite": { mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100) },
		"hits always ignite": { mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100) },
		"your hits always shock": { mod.NewFloat("EnemyShockChance", mod.TypeBase, 100) },
		"hits always shock": { mod.NewFloat("EnemyShockChance", mod.TypeBase, 100) },
		"all damage with hits can ignite": {
			mod.NewFlag("PhysicalCanIgnite", true),
			mod.NewFlag("ColdCanIgnite", true),
			mod.NewFlag("LightningCanIgnite", true),
			mod.NewFlag("ChaosCanIgnite", true),
		},
		"all damage can ignite": {
			mod.NewFlag("PhysicalCanIgnite", true),
			mod.NewFlag("ColdCanIgnite", true),
			mod.NewFlag("LightningCanIgnite", true),
			mod.NewFlag("ChaosCanIgnite", true),
		},
		"all damage with hits can chill": {
			mod.NewFlag("PhysicalCanChill", true),
			mod.NewFlag("FireCanChill", true),
			mod.NewFlag("LightningCanChill", true),
			mod.NewFlag("ChaosCanChill", true),
		},
		"all damage with hits can shock": {
			mod.NewFlag("PhysicalCanShock", true),
			mod.NewFlag("FireCanShock", true),
			mod.NewFlag("ColdCanShock", true),
			mod.NewFlag("ChaosCanShock", true),
		},
		"all damage can shock": {
			mod.NewFlag("PhysicalCanShock", true),
			mod.NewFlag("FireCanShock", true),
			mod.NewFlag("ColdCanShock", true),
			mod.NewFlag("ChaosCanShock", true),
		},
		"other aegis skills are disabled": function(_, name) return {
			flag("DisableSkill", mod.SkillType(data.SkillTypeAegis)),
			flag("EnableSkill", { type = "SkillName", skillId = "Primal Aegis" }),
		} end,
		"primal aegis can take (%d+) elemental damage per allocated notable passive skill": function(num) return { mod("ElementalAegisValue", "MAX", num, 0, 0, mod.Multiplier("AllocatedNotable").Base(0), { type = "GlobalEffect", effectType = "Buff", unscalable = true }) } end,
		// TODO Gladiator
		"chance to block spell damage is equal to chance to block attack damage": { mod.NewFlag("SpellBlockChanceIsBlockChance", true) },
		"maximum chance to block spell damage is equal to maximum chance to block attack damage": { mod.NewFlag("SpellBlockChanceMaxIsBlockChanceMax", true) },
		"your counterattacks deal double damage": {
			mod("DoubleDamageChance", "BASE", 100, mod.SkillName("Reckoning")),
			mod("DoubleDamageChance", "BASE", 100, mod.SkillName("Riposte")),
			mod("DoubleDamageChance", "BASE", 100, mod.SkillName("Vengeance")),
		},
		"attack damage is lucky if you[' ]h?a?ve blocked in the past (%d+) seconds": function(num) return {
			flag("LuckyHits", { type = "Condition", var = "BlockedRecently"} )
		} end,
		"hits ignore enemy monster physical damage reduction if you[' ]h?a?ve blocked in the past (%d+) seconds": function(num) return {
			flag("IgnoreEnemyPhysicalDamageReduction", { type = "Condition", var = "BlockedRecently"} )
		} end,
		"(%d+)%% more attack and movement speed per challenger charge": function(num) return {
			mod("Speed", "MORE", num, nil, mod.MFlagAttack, 0, mod.Multiplier("ChallengerCharge").Base(0)),
			mod("MovementSpeed", "MORE", num, mod.Multiplier("ChallengerCharge").Base(0)),
		} end,
		// TODO Guardian
		"grants armour equal to (%d+)%% of your reserved life to you and nearby allies": function(num) return { mod("GrantReservedLifeAsAura", "LIST", { mod = mod("Armour", "BASE", num / 100) }) } end,
		"grants maximum energy shield equal to (%d+)%% of your reserved mana to you and nearby allies": function(num) return { mod("GrantReservedManaAsAura", "LIST", { mod = mod("EnergyShield", "BASE", num / 100) }) } end,
		"warcries cost no mana": { mod("ManaCost", "MORE", -100, nil, 0, mod.KeywordFlagWarcry) },
		"%+(%d+)%% chance to block attack damage for %d seconds? every %d seconds": function(num) return { mod("BlockChance", "BASE", num, .Tag(mod.Condition("BastionOfHopeActive"))) } end,
		"if you've blocked in the past %d+ seconds, you and nearby allies cannot be stunned": { mod("ExtraAura", "LIST", { mod = mod.NewFloat("AvoidStun", mod.TypeBase, 100) }, .Tag(mod.Condition("BlockedRecently"))) },
		"if you've attacked recently, you and nearby allies have %+(%d+)%% chance to block attack damage": function(num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("BlockChance", mod.TypeBase, num) }, .Tag(mod.Condition("AttackedRecently"))) } end,
		"if you've cast a spell recently, you and nearby allies have %+(%d+)%% chance to block spell damage": function(num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("SpellBlockChance", mod.TypeBase, num) }, .Tag(mod.Condition("CastSpellRecently"))) } end,
		"while there is at least one nearby ally, you and nearby allies deal (%d+)%% more damage": function(num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("Damage", mod.TypeMore, num) }, { type = "MultiplierThreshold", var = "NearbyAlly", threshold = 1 }) } end,
		"while there are at least five nearby allies, you and nearby allies have onslaught": { mod("ExtraAura", "LIST", { mod = mod.NewFlag("Onslaught", true) }, { type = "MultiplierThreshold", var = "NearbyAlly", threshold = 5 }) },
		// TODO Hierophant
		"you and your totems regenerate ([%d%.]+)%% of life per second for each summoned totem": function (num) return {
			mod("LifeRegenPercent", "BASE", num, { type = "PerStat", stat = "TotemsSummoned" }),
			mod("LifeRegenPercent", "BASE", num, { type = "PerStat", stat = "TotemsSummoned" }, 0, mod.KeywordFlagTotem),
		} end,
		"enemies take (%d+)%% increased damage for each of your brands attached to them": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTaken", "INC", num, mod.Multiplier("BrandsAttached").Base(0)) }) } end,
		"immune to elemental ailments while you have arcane surge": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("AffectedByArcaneSurge")))
			end
			return mods
		end,
		"brands have (%d+)%% more activation frequency if (%d+)%% of attached duration expired": function(num) return { mod("BrandActivationFrequency", "MORE", num, { type = "Condition", var = "BrandLastQuarter"} ) } end,
		// TODO Inquisitor
		"critical strikes ignore enemy monster elemental resistances": { flag("IgnoreElementalResistances", .Tag(mod.Condition("CriticalStrike"))) },
		"non%-critical strikes penetrate (%d+)%% of enemy elemental resistances": function(num) return { mod("ElementalPenetration", "BASE", num, .Tag(mod.Condition("CriticalStrike").Neg(true))) } end,
		"consecrated ground you create applies (%d+)%% increased damage taken to enemies": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTakenConsecratedGround", "INC", num, .Tag(mod.Condition("OnConsecratedGround"))) }) } end,
		"you have consecrated ground around you while stationary": { flag("Condition:OnConsecratedGround", .Tag(mod.Condition("Stationary"))) },
		"consecrated ground you create grants immunity to elemental ailments to you and allies": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("OnConsecratedGround")))
			end
			return mods
		end,
		"gain fanaticism for 4 seconds on reaching maximum fanatic charges": function() return {
			mod.NewFlag("Condition:CanGainFanaticism", true),
		} end ,
		"(%d+)%% increased critical strike chance per point of strength or intelligence, whichever is lower": function(num) return {
			mod("CritChance", "INC", num, { type = "PerStat", stat = "Str" }, .Tag(mod.Condition("IntHigherThanStr"))),
			mod("CritChance", "INC", num, { type = "PerStat", stat = "Int" }, { type = "Condition", neg = true, var = "IntHigherThanStr" })
		} end,
		"consecrated ground you create causes life regeneration to also recover energy shield for you and allies": function(num) return {
			flag("LifeRegenerationRecoversEnergyShield", { type = "Condition", var = "OnConsecratedGround"}),
			mod("MinionModifier", "LIST", { mod = flag("LifeRegenerationRecoversEnergyShield", { type = "Condition", var = "OnConsecratedGround"}) })
		} end,
		"(%d+)%% more attack damage for each non%-instant spell you've cast in the past 8 seconds, up to a maximum of (%d+)%%": function(num, _, max) return {
			mod("Damage", "MORE", num, nil, mod.MFlagAttack, { type = "Multiplier", var = "CastLast8Seconds", limit = max, limitTotal = true}),
		} end,
		// TODO Juggernaut
		"armour received from body armour is doubled": { mod.NewFlag("Unbreakable", true) },
		"action speed cannot be modified to below base value": { mod.NewFlag("ActionSpeedCannotBeBelowBase", true) },
		"movement speed cannot be modified to below base value": { mod.NewFlag("MovementSpeedCannotBeBelowBase", true) },
		"you cannot be slowed to below base speed": { mod.NewFlag("ActionSpeedCannotBeBelowBase", true) },
		"cannot be slowed to below base speed": { mod.NewFlag("ActionSpeedCannotBeBelowBase", true) },
		"gain accuracy rating equal to your strength": { mod("Accuracy", "BASE", 1, { type = "PerStat", stat = "Str" }) },
		"gain accuracy rating equal to twice your strength": { mod("Accuracy", "BASE", 2, { type = "PerStat", stat = "Str" }) },
		// TODO Necromancer
		"your offering skills also affect you": { mod("ExtraSkillMod", "LIST", { mod = mod("SkillData", "LIST", { key = "buffNotPlayer", value = false }) }, { type = "SkillName", skillNameList = { "Bone Offering", "Flesh Offering", "Spirit Offering" } }) },
		"your offerings have (%d+)%% reduced effect on you": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("BuffEffectOnPlayer", "INC", -num) }, { type = "SkillName", skillNameList = { "Bone Offering", "Flesh Offering", "Spirit Offering" } }) } end,
		"if you've consumed a corpse recently, you and your minions have (%d+)%% increased area of effect": function(num) return { mod("AreaOfEffect", "INC", num, .Tag(mod.Condition("ConsumedCorpseRecently"))), mod("MinionModifier", "LIST", { mod = mod.NewFloat("AreaOfEffect", mod.TypeIncrease, num) }, .Tag(mod.Condition("ConsumedCorpseRecently"))) } end,
		"with at least one nearby corpse, you and nearby allies deal (%d+)%% more damage": function(num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("Damage", mod.TypeMore, num) }, { type = "MultiplierThreshold", var = "NearbyCorpse", threshold = 1 }) } end,
		"for each nearby corpse, you and nearby allies regenerate ([%d%.]+)%% of energy shield per second, up to ([%d%.]+)%% per second": function(num, _, limit) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("EnergyShieldRegenPercent", mod.TypeBase, num) }, { type = "Multiplier", var = "NearbyCorpse", limit = tonumber(limit), limitTotal = true }) } end,
		"for each nearby corpse, you and nearby allies regenerate (%d+) mana per second, up to (%d+) per second": function(num, _, limit) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("ManaRegen", mod.TypeBase, num) }, { type = "Multiplier", var = "NearbyCorpse", limit = tonumber(limit), limitTotal = true }) } end,
		"(%d+)%% increased attack and cast speed for each corpse consumed recently, up to a maximum of (%d+)%%": function(num, _, limit) return { mod("Speed", "INC", num, { type = "Multiplier", var = "CorpseConsumedRecently", limit = tonumber(limit / num)}) } end,
		"enemies near corpses you spawned recently are chilled and shocked": {
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Chilled", true) }, .Tag(mod.Condition("SpawnedCorpseRecently"))),
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Shocked", true) }, .Tag(mod.Condition("SpawnedCorpseRecently"))),
			mod("ChillBase", "BASE", data.nonDamagingAilment["Chill"].default, { type = "Condition", var = "SpawnedCorpseRecently"}),
			mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default, { type = "Condition", var = "SpawnedCorpseRecently"}),
		},
		"regenerate (%d+)%% of energy shield over 2 seconds when you consume a corpse": function(num) return { mod("EnergyShieldRegenPercent", "BASE", num / 2, .Tag(mod.Condition("ConsumedCorpseInPast2Sec"))) } end,
		"regenerate (%d+)%% of mana over 2 seconds when you consume a corpse": function(num) return { mod("ManaRegen", "BASE", 1, { type = "PercentStat", stat = "Mana", percent = num / 2 }, .Tag(mod.Condition("ConsumedCorpseInPast2Sec"))) } end,
		"corpses you spawn have (%d+)%% increased maximum life": function(num) return {
			mod.NewFloat("CorpseLife", mod.TypeIncrease, num),
		} end,
		// TODO Occultist
		"enemies you curse have malediction": { mod("AffectedByCurseMod", "LIST", { mod = mod.NewFlag("HasMalediction", true) }) },
		"when you kill an enemy, for each curse on that enemy, gain (%d+)%% of non%-chaos damage as extra chaos damage for 4 seconds": function(num) return {
			mod("NonChaosDamageGainAsChaos", "BASE", num, .Tag(mod.Condition("KilledRecently")), mod.Multiplier("CurseOnEnemy").Base(0)),
		} end,
		"cannot be stunned while you have energy shield": { mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("HaveEnergyShield"))) },
		"every second, inflict withered on nearby enemies for (%d+) seconds": { mod.NewFlag("Condition:CanWither", true) },
		// TODO Pathfinder
		"always poison on hit while using a flask": { mod("PoisonChance", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"poisons you inflict during any flask effect have (%d+)%% chance to deal (%d+)%% more damage": function(num, _, more) return { mod("Damage", "MORE", tonumber(more) * num / 100, nil, 0, mod.KeywordFlagPoison, .Tag(mod.Condition("UsingFlask"))) } end,
		"immune to elemental ailments during any flask effect": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("UsingFlask")))
			end
			return mods
		end,
		// TODO Raider
		"nearby enemies have (%d+)%% less accuracy rating while you have phasing": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("Accuracy", "MORE", -num) }, .Tag(mod.Condition("Phasing")) )} end,
		"immune to elemental ailments while phasing": function()
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("Phasing")))
			end
			return mods
		end,
		"nearby enemies have fire, cold and lightning exposure while you have phasing, applying %-(%d+)%% to those resistances": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod("FireExposure", "BASE", -num) }, .Tag(mod.Condition("Phasing")) ),
			mod("EnemyModifier", "LIST", { mod = mod("ColdExposure", "BASE", -num) }, .Tag(mod.Condition("Phasing")) ),
			mod("EnemyModifier", "LIST", { mod = mod("LightningExposure", "BASE", -num) }, .Tag(mod.Condition("Phasing")) ),
		} end,
		// TODO Saboteur
		"immune to ignite and shock": {
			mod.NewFloat("AvoidIgnite", mod.TypeBase, 100),
			mod.NewFloat("AvoidShock", mod.TypeBase, 100),
		},
		"you gain (%d+)%% increased damage for each trap": function(num) return { mod("Damage", "INC", num, { type = "PerStat", stat = "ActiveTrapLimit" }) } end,
		"you gain (%d+)%% increased area of effect for each mine": function(num) return { mod("AreaOfEffect", "INC", num, { type = "PerStat", stat = "ActiveMineLimit" }) } end,
		// TODO Slayer
		"deal up to (%d+)%% more melee damage to enemies, based on proximity": function(num) return { mod("Damage", "MORE", num, nil, bor(mod.MFlagAttack, mod.MFlagMelee), { type = "MeleeProximity", ramp = {1,0} }) } end,
		"cannot be stunned while leeching": { mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("Leeching"))) },
		"you are immune to bleeding while leeching": { mod("AvoidBleed", "BASE", 100, .Tag(mod.Condition("Leeching"))) },
		"life leech effects are not removed at full life": { mod.NewFlag("CanLeechLifeOnFullLife", true) },
		"life leech effects are not removed when unreserved life is filled": { mod.NewFlag("CanLeechLifeOnFullLife", true) },
		"energy shield leech effects from attacks are not removed at full energy shield": { mod.NewFlag("CanLeechLifeOnFullEnergyShield", true) },
		"cannot take reflected physical damage": { mod("PhysicalReflectedDamageTaken", "MORE", -100) },
		"gain (%d+)%% increased movement speed for 20 seconds when you kill an enemy": function(num) return { mod("MovementSpeed", "INC", num, .Tag(mod.Condition("KilledRecently"))) } end,
		"gain (%d+)%% increased attack speed for 20 seconds when you kill a rare or unique enemy": function(num) return { mod("Speed", "INC", num, nil, mod.MFlagAttack, 0, .Tag(mod.Condition("KilledUniqueEnemy"))) } end,
		"kill enemies that have (%d+)%% or lower life when hit by your skills": function(num) return { mod("CullPercent", "MAX", num) } end,
		// TODO Trickster
		"(%d+)%% chance to gain (%d+)%% of non%-chaos damage with hits as extra chaos damage": function(num, _, perc) return { mod("NonChaosDamageGainAsChaos", "BASE", num / 100 * tonumber(perc)) } end,
		"movement skills cost no mana": { mod("ManaCost", "MORE", -100, nil, 0, mod.KeywordFlagMovement) },
		"cannot be stunned while you have ghost shrouds": function(num) return { mod("AvoidStun", "BASE", 100, { type = "MultiplierThreshold", var = "GhostShroud", threshold = 1 }) } end,
		// TODO Item local modifiers
		"has no sockets": { mod.NewFlag("NoSockets", true) },
		"has (%d+) sockets?": function(num) return { mod.NewFloat("SocketCount", mod.TypeBase, num) } end,
		"has (%d+) abyssal sockets?": function(num) return { mod.NewFloat("AbyssalSocketCount", mod.TypeBase, num) } end,
		"no physical damage": { mod("WeaponData", "LIST", { key = "PhysicalMin" }), mod("WeaponData", "LIST", { key = "PhysicalMax" }), mod("WeaponData", "LIST", { key = "PhysicalDPS" }) },
		"all attacks with this weapon are critical strikes": { mod("WeaponData", "LIST", { key = "CritChance", value = 100 }) },
		"this weapon's critical strike chance is (%d+)%%": function(num) return { mod("WeaponData", "LIST", { key = "CritChance", value = num }) } end,
		"counts as dual wielding": { mod("WeaponData", "LIST", { key = "countsAsDualWielding", value = true}) },
		"counts as all one handed melee weapon types": { mod("WeaponData", "LIST", { key = "countsAsAll1H", value = true }) },
		"no block chance": { mod("ArmourData", "LIST", { key = "BlockChance", value = 0 }) },
		"has no energy shield": { mod("ArmourData", "LIST", { key = "EnergyShield", value = 0 }) },
		"hits can't be evaded": { flag("CannotBeEvaded", .Tag(mod.Condition("{Hand}Attack"))) },
		"causes bleeding on hit": { mod("BleedChance", "BASE", 100, .Tag(mod.Condition("{Hand}Attack"))) },
		"poisonous hit": { mod("PoisonChance", "BASE", 100, .Tag(mod.Condition("{Hand}Attack"))) },
		"attacks with this weapon deal double damage": { mod("DoubleDamageChance", "BASE", 100, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)) },
		"hits with this weapon gain (%d+)%% of physical damage as extra cold or lightning damage": function(num) return {
			mod("PhysicalDamageGainAsColdOrLightning", "BASE", num / 2, nil, mod.MFlagHit, { type = "Condition", var = "DualWielding"}, mod.SkillType(data.SkillTypeAttack)),
			mod("PhysicalDamageGainAsColdOrLightning", "BASE", num, nil, mod.MFlagHit, { type = "Condition", var = "DualWielding", neg = true}, mod.SkillType(data.SkillTypeAttack))
		} end,
		"hits with this weapon shock enemies as though dealing (%d+)%% more damage": function(num) return { mod("ShockAsThoughDealing", "MORE", num, nil, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)) } end,
		"hits with this weapon freeze enemies as though dealing (%d+)%% more damage": function(num) return { mod("FreezeAsThoughDealing", "MORE", num, nil, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)) } end,
		"ignites inflicted with this weapon deal (%d+)%% more damage": function(num) return {
			mod("Damage", "MORE", num, nil, 0, mod.KeywordFlagIgnite, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
		} end,
		"hits with this weapon always ignite, freeze, and shock": {
			mod("EnemyIgniteChance", "BASE", 100, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
			mod("EnemyFreezeChance", "BASE", 100, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
			mod("EnemyShockChance", "BASE", 100, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
		},
		"attacks with this weapon deal double damage to chilled enemies": { mod("DoubleDamageChance", "BASE", 100, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack), { type = "ActorCondition", actor = "enemy", var = "Chilled" }) },
		"life leech from hits with this weapon applies instantly": { flag("InstantLifeLeech", .Tag(mod.Condition("{Hand}Attack"))) },
		"life leech from hits with this weapon is instant": { flag("InstantLifeLeech", .Tag(mod.Condition("{Hand}Attack"))) },
		"gain life from leech instantly from hits with this weapon": { flag("InstantLifeLeech", .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)) },
		"instant recovery": {  mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, 100) },
		"(%d+)%% of recovery applied instantly": function(num) return { mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, num) } end,
		"has no attribute requirements": { mod.NewFlag("NoAttributeRequirements", true) },
		"trigger a socketed spell when you attack with this weapon": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed spell when you attack with this weapon, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed spell when you use a skill": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnSkillUse", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed spell when you use a skill, with a (%d+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnSkillUse", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed spell when you use a skill, with a (%d+) second cooldown and (%d+)%% more cost": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnSkillUse", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger socketed spells when you focus": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellFromHelmet", level = 1 }, mod.SocketedIn("{SlotName}"), .Tag(mod.Condition("Focused"))) },
		"trigger socketed spells when you focus, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellFromHelmet", level = 1 }, mod.SocketedIn("{SlotName}"), .Tag(mod.Condition("Focused"))) },
		"trigger a socketed spell when you attack with a bow": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnBowAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed spell when you attack with a bow, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerSpellOnBowAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed bow skill when you attack with a bow": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerBowSkillOnBowAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed bow skill when you attack with a bow, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportTriggerBowSkillOnBowAttack", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"(%d+)%% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least (%d+) mana to use a skill": function(num, _, amount) return {
			mod("KitavaTriggerChance", "BASE", num, "Kitava's Thirst"),
			mod("KitavaRequiredManaCost", "BASE", tonumber(amount), "Kitava's Thirst"),
			mod("ExtraSupport", "LIST", { skillId = "SupportCastOnManaSpent", level = 1 }, mod.SocketedIn("{SlotName}")),
		} end,
		// TODO Socketed gem modifiers
		"([%+%-]%d+) to level of socketed gems": function(num) return { mod("GemProperty", "LIST", { keyword = "all", key = "level", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"([%+%-]%d+) to level of socketed ([%a ]+) gems": function(num, _, type) return { mod("GemProperty", "LIST", { keyword = type, key = "level", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"%+(%d+)%% to quality of socketed gems": function(num, _, type) return { mod("GemProperty", "LIST", { keyword = "all", key = "quality", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"%+(%d+)%% to quality of all skill gems": function(num, _, type) return { mod("GemProperty", "LIST", { keyword = "active_skill", key = "quality", value = num }) } end,
		"%+(%d+)%% to quality of socketed ([%a ]+) gems": function(num, _, type) return { mod("GemProperty", "LIST", { keyword = type, key = "quality", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"%+(%d+) to level of active socketed skill gems": function(num) return { mod("GemProperty", "LIST", { keyword = "active_skill", key = "level", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"%+(%d+) to level of socketed active skill gems": function(num) return { mod("GemProperty", "LIST", { keyword = "active_skill", key = "level", value = num }, mod.SocketedIn("{SlotName}")) } end,
		"%+(%d+) to level of socketed active skill gems per (%d+) player levels": function(num, _, div) return { mod("GemProperty", "LIST", { keyword = "active_skill", key = "level", value = num }, mod.SocketedIn("{SlotName}"), { type = "Multiplier", var = "Level", div = tonumber(div) }) } end,
		"socketed gems fire an additional projectile": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("ProjectileCount", mod.TypeBase, 1) }, mod.SocketedIn("{SlotName}")) },
		"socketed gems fire (%d+) additional projectiles": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("ProjectileCount", mod.TypeBase, num) }, mod.SocketedIn("{SlotName}")) } end,
		"socketed gems reserve no mana": { mod("ManaReserved", "MORE", -100, mod.SocketedIn("{SlotName}")) },
		"socketed gems have no reservation": { mod("Reserved", "MORE", -100, mod.SocketedIn("{SlotName}")) },
		"socketed skill gems get a (%d+)%% mana multiplier": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("SupportManaMultiplier", "MORE", num - 100) }, mod.SocketedIn("{SlotName}")) } end,
		"socketed skill gems get a (%d+)%% cost & reservation multiplier": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("SupportManaMultiplier", "MORE", num - 100) }, mod.SocketedIn("{SlotName}")) } end,
		"socketed gems have blood magic": { mod("ExtraSupport", "LIST", { skillId = "SupportBloodMagicUniquePrismGuardian", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"socketed gems cost and reserve life instead of mana": { mod("ExtraSupport", "LIST", { skillId = "SupportBloodMagicUniquePrismGuardian", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"socketed gems have elemental equilibrium": { mod("Keystone", "LIST", "Elemental Equilibrium") },
		"socketed gems have secrets of suffering": {
			flag("CannotIgnite", mod.SocketedIn("{SlotName}")),
			flag("CannotChill", mod.SocketedIn("{SlotName}")),
			flag("CannotFreeze", mod.SocketedIn("{SlotName}")),
			flag("CannotShock", mod.SocketedIn("{SlotName}")),
			flag("CritAlwaysAltAilments", mod.SocketedIn("{SlotName}"))
		},
		"socketed skills deal double damage": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("DoubleDamageChance", mod.TypeBase, 100) }, mod.SocketedIn("{SlotName}")) },
		"socketed gems gain (%d+)%% of physical damage as extra lightning damage": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, num) }, mod.SocketedIn("{SlotName}")) } end,
		"socketed red gems get (%d+)%% physical damage as extra fire damage": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, num) }, mod.SocketedIn("{SlotName}").Keyword("strength")) } end,
		"socketed non%-channelling bow skills are triggered by snipe": {
			mod("ExtraSkillMod", "LIST", { mod = mod.NewFlag("TriggeredBySnipe", true) },  mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(data.SkillTypeTriggerable) ),
			mod("ExtraSkillMod", "LIST", { mod = mod("SkillData", "LIST", { key = "showAverage", value = true } ) }, mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(data.SkillTypeTriggerable) ),
			mod("ExtraSkillMod", "LIST", { mod = mod("SkillData", "LIST", { key = "triggered", value = 1 } ) }, mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(data.SkillTypeTriggerable) ),
		},
		"socketed triggered bow skills deal (%d+)%% less damage": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("Damage", "MORE", -num) }, mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(data.SkillTypeTriggerable) ) } end,
		"socketed travel skills deal (%d+)%% more damage": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("Damage", mod.TypeMore, num) }, mod.SocketedIn("{SlotName}"), mod.SkillType(data.SkillTypeTravel) ) } end,
		"socketed warcry skills have %+(%d+) cooldown use": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("AdditionalCooldownUses", mod.TypeBase, num) }, mod.SocketedIn("{SlotName}"), mod.SkillType(data.SkillTypeWarcry) ) } end,
		// TODO Global gem modifiers
		"%+(%d+) to level of all minion skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "minion", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all physical spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "physical", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all physical skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "physical", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all lightning spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "lightning", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all lightning skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "lightning", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all cold spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "cold", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all cold skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "cold", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all fire spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "fire", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all fire skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "fire", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all chaos spell skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "spell", "chaos", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all chaos skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "chaos", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all strength skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "strength", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all dexterity skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "dexterity", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all intelligence skill gems": function(num) return { mod("GemProperty", "LIST", { keywordList = { "intelligence", "active_skill" }, key = "level", value = num }) } end,
		"%+(%d+) to level of all (.+) gems": function(num, _, skill)
			if gemIdLookup[skill] then
				return { mod("GemProperty", "LIST", {keyword = skill, key = "level", value = num }) }
			end
			local wordList = {}
			for tag in skill:gmatch("%w+") do
				if tag == "skill" then
					tag = "active_skill"
				end
				table.insert(wordList, tag)
			end
			return { mod("GemProperty", "LIST", {keywordList = wordList, key = "level", value = num }) }
		end,
		// TODO Extra skill/support
		"grants (%D+)": function(_, skill) return grantedExtraSkill(skill, 1) end,
		"grants level (%d+) (.+)": function(num, _, skill) return grantedExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when equipped": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) on %a+": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"use level (%d+) (.+) on %a+": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when you attack": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when you deal a critical strike": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when hit": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when you kill an enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"[ct][ar][si][tg]g?e?r?s? level (%d+) (.+) when you use a skill": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"(.+) can trigger level (%d+) (.+)": function(_, sourceSkill, num, skill) return triggerExtraSkill(skill, tonumber(num), nil, sourceSkill) end,
		"trigger level (%d+) (.+) when you use a skill while you have a spirit charge": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you hit an enemy while cursed": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you hit a bleeding enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you hit a rare or unique enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you hit a rare or unique enemy and have no mark": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you hit a frozen enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you kill a frozen enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you consume a corpse": function(num, _, skill) return skill == "summon phantasm skill" and triggerExtraSkill("triggered summon phantasm skill", num) or triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you attack with a bow": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you block": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when animated guardian kills an enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you lose cat's stealth": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when your trap is triggered": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) on hit with this weapon": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) on melee hit while cursed": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) on melee hit with this weapon": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) every [%d%.]+ seconds while phasing": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) when you gain avian's might or avian's flight": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+) on melee hit if you have at least (%d+) strength": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"triggers level (%d+) (.+) when equipped": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"triggers level (%d+) (.+) when allocated": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to attack with level (%d+) (.+) on melee hit": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) when animated weapon kills an enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) on melee hit": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) [ow][nh]e?n? ?y?o?u? kill ?a?n? ?e?n?e?m?y?": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) when you use a socketed skill": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) when you gain avian's might or avian's flight": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to trigger level (%d+) (.+) on critical strike with this weapon": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"%d+%% chance to [ct][ar][si][tg]g?e?r? level (%d+) (.+) on %a+": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"attack with level (%d+) (.+) when you kill a bleeding enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"triggers? level (%d+) (.+) when you kill a bleeding enemy": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"curse enemies with (%D+) on %a+": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"curse enemies with (%D+) on %a+, with (%d+)%% increased effect": function(_, skill, num) return {
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[skill], level = 1, noSupports = true, triggered = true }),
			mod("CurseEffect", "INC", tonumber(num), { type = "SkillName", skillName = string.gsub(" "..skill, "%W%l", string.upper):sub(2) }),
		} end,
		"%d+%% chance to curse n?o?n?%-?c?u?r?s?e?d? ?enemies with (%D+) on %a+, with (%d+)%% increased effect": function(_, skill, num) return {
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[skill], level = 1, noSupports = true, triggered = true }),
			mod("CurseEffect", "INC", tonumber(num), { type = "SkillName", skillName = string.gsub(" "..skill, "%W%l", string.upper):sub(2) }),
		} end,
		"curse enemies with level (%d+) (%D+) on %a+, which can apply to hexproof enemies": function(num, _, skill) return triggerExtraSkill(skill, num, true) end,
		"curse enemies with level (%d+) (.+) on %a+": function(num, _, skill) return triggerExtraSkill(skill, num, true) end,
		"[ct][ar][si][tg]g?e?r?s? (.+) on %a+": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? (.+) on %a+": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? with (.+) on %a+": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[ct][ar][si][tg]g?e?r?s? (.+) when hit": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? (.+) when hit": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? with (.+) when hit": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[ct][ar][si][tg]g?e?r?s? (.+) when your skills or minions kill": function(_, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? (.+) when you take a critical strike": function( _, skill) return triggerExtraSkill(skill, 1, true) end,
		"[at][tr][ti][ag][cg][ke]r? with (.+) when you take a critical strike": function( _, skill) return triggerExtraSkill(skill, 1, true) end,
		"trigger commandment of inferno on critical strike": { mod("ExtraSkill", "LIST", { skillId = "UniqueEnchantmentOfInfernoOnCrit", level = 1, noSupports = true, triggered = true }) },
		"trigger (.+) on critical strike": function( _, skill) return triggerExtraSkill(skill, 1, true) end,
		"triggers? (.+) when you take a critical strike": function( _, skill) return triggerExtraSkill(skill, 1, true) end,
		"socketed [%a+]* ?gems a?r?e? ?supported by level (%d+) (.+)": function(num, _, support)
			local skillId = gemIdLookup[support] or gemIdLookup[support:gsub("^increased ","")]
			if skillId then
				local gemId = data.gemForBaseName[data.skills[skillId].name .. " Support"]
				if gemId then
					return {
						mod("ExtraSupport", "LIST", { skillId = data.gems[gemId].grantedEffectId, level = num }, mod.SocketedIn("{SlotName}")),
						mod("ExtraSupport", "LIST", { skillId = data.gems[gemId].secondaryGrantedEffectId, level = num }, mod.SocketedIn("{SlotName}"))
					}
				else
					return {
						mod("ExtraSupport", "LIST", { skillId = skillId, level = num }, mod.SocketedIn("{SlotName}")),
					}
				end
			end
		end,
		"socketed support gems can also support skills from your ([%a%s]+)": function (_, itemSlotName)
			local targetItemSlotName = "Body Armour"
			if itemSlotName == "main hand" then
				targetItemSlotName = "Weapon 1"
			end
			return {
				mod("LinkedSupport", "LIST", { targetSlotName = targetItemSlotName }, mod.SocketedIn("{SlotName}")),
			}
		end,
		"socketed hex curse skills are triggered by doedre's effigy when summoned": { mod("ExtraSupport", "LIST", { skillId = "SupportCursePillarTriggerCurses", level = 20 }, mod.SocketedIn("{SlotName}")) },
		"trigger level (%d+) (.+) every (%d+) seconds": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"trigger level (%d+) (.+), (.+) or (.+) every (%d+) seconds": function(num, _, skill1, skill2, skill3) return {
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[skill1], level = num, triggered = true }),
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[skill2], level = num, triggered = true }),
			mod("ExtraSkill", "LIST", { skillId = gemIdLookup[skill3], level = num, triggered = true }),
		} end,
		"offering skills triggered this way also affect you": { mod("ExtraSkillMod", "LIST", { mod = mod("SkillData", "LIST", { key = "buffNotPlayer", value = false }) }, { type = "SkillName", skillNameList = { "Bone Offering", "Flesh Offering", "Spirit Offering" } }, mod.SocketedIn("{SlotName}")) },
		"trigger level (%d+) (.+) after spending a total of (%d+) mana": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"consumes a void charge to trigger level (%d+) (.+) when you fire arrows": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"consumes a void charge to trigger level (%d+) (.+) when you fire arrows with a non%-triggered skill": function(num, _, skill) return triggerExtraSkill(skill, num) end,
		"your hits treat cold resistance as (%d+)%% higher than actual value": function(num) return {
			mod("ColdPenetration", "BASE", -num, nil, 0, mod.KeywordFlagHit),
		} end,
		// TODO Conversion
		"increases and reductions to minion damage also affects? you": { mod.NewFlag("MinionDamageAppliesToPlayer", true), mod("ImprovedMinionDamageAppliesToPlayer", "MAX", 100) },
		"increases and reductions to minion damage also affects? you at (%d+)%% of their value": function(num) return { mod.NewFlag("MinionDamageAppliesToPlayer", true), mod("ImprovedMinionDamageAppliesToPlayer", "MAX", num) } end,
		"increases and reductions to minion attack speed also affects? you": { mod.NewFlag("MinionAttackSpeedAppliesToPlayer", true), mod("ImprovedMinionAttackSpeedAppliesToPlayer", "MAX", 100) },
		"increases and reductions to cast speed apply to attack speed at (%d+)%% of their value":  function(num) return { mod.NewFlag("CastSpeedAppliesToAttacks", true), mod("ImprovedCastSpeedAppliesToAttacks", "MAX", num) } end,
		"increases and reductions to spell damage also apply to attacks": { mod.NewFlag("SpellDamageAppliesToAttacks", true), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", 100) },
		"increases and reductions to spell damage also apply to attacks at (%d+)%% of their value": function(num) return { mod.NewFlag("SpellDamageAppliesToAttacks", true), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", num) } end,
		"increases and reductions to spell damage also apply to attacks while wielding a wand": { flag("SpellDamageAppliesToAttacks", .Tag(mod.Condition("UsingWand"))), mod("ImprovedSpellDamageAppliesToAttacks", "MAX", 100, .Tag(mod.Condition("UsingWand"))) },
		"increases and reductions to maximum mana also apply to shock effect at (%d+)%% of their value": function(num) return { mod.NewFlag("ManaAppliesToShockEffect", true), mod("ImprovedManaAppliesToShockEffect", "MAX", num) } end,
		"modifiers to claw damage also apply to unarmed": { mod.NewFlag("ClawDamageAppliesToUnarmed", true) },
		"modifiers to claw damage also apply to unarmed attack damage": { mod.NewFlag("ClawDamageAppliesToUnarmed", true) },
		"modifiers to claw damage also apply to unarmed attack damage with melee skills": { mod.NewFlag("ClawDamageAppliesToUnarmed", true) },
		"modifiers to claw attack speed also apply to unarmed": { mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true) },
		"modifiers to claw attack speed also apply to unarmed attack speed": { mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true) },
		"modifiers to claw attack speed also apply to unarmed attack speed with melee skills": { mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true) },
		"modifiers to claw critical strike chance also apply to unarmed": { mod.NewFlag("ClawCritChanceAppliesToUnarmed", true) },
		"modifiers to claw critical strike chance also apply to unarmed attack critical strike chance": { mod.NewFlag("ClawCritChanceAppliesToUnarmed", true) },
		"modifiers to claw critical strike chance also apply to unarmed critical strike chance with melee skills": { mod.NewFlag("ClawCritChanceAppliesToUnarmed", true) },
		"increases and reductions to light radius also apply to accuracy": { mod.NewFlag("LightRadiusAppliesToAccuracy", true) },
		"increases and reductions to light radius also apply to area of effect at 50%% of their value": { mod.NewFlag("LightRadiusAppliesToAreaOfEffect", true) },
		"increases and reductions to light radius also apply to damage": { mod.NewFlag("LightRadiusAppliesToDamage", true) },
		"increases and reductions to cast speed also apply to trap throwing speed": { mod.NewFlag("CastSpeedAppliesToTrapThrowingSpeed", true) },
		"increases and reductions to armour also apply to energy shield recharge rate at (%d+)%% of their value": function(num) return { mod.NewFlag("ArmourAppliesToEnergyShieldRecharge", true), mod("ImprovedArmourAppliesToEnergyShieldRecharge", "MAX", num) } end,
		"increases and reductions to projectile speed also apply to damage with bows": { mod.NewFlag("ProjectileSpeedAppliesToBowDamage", true) },
		"gain (%d+)%% of bow physical damage as extra damage of each element": function(num) return {
			mod("PhysicalDamageGainAsLightning", "BASE", num, nil, mod.MFlagBow),
			mod("PhysicalDamageGainAsCold", "BASE", num, nil, mod.MFlagBow),
			mod("PhysicalDamageGainAsFire", "BASE", num, nil, mod.MFlagBow),
		} end,
		"gain (%d+)%% of weapon physical damage as extra damage of each element": function(num) return {
			mod("PhysicalDamageGainAsLightning", "BASE", num, nil, mod.MFlagWeapon),
			mod("PhysicalDamageGainAsCold", "BASE", num, nil, mod.MFlagWeapon),
			mod("PhysicalDamageGainAsFire", "BASE", num, nil, mod.MFlagWeapon),
		} end,
		"gain (%d+)%% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element": function(num) return { mod("PhysicalDamageGainAsRandom", "BASE", num, nil, mod.MFlagWeapon) } end,
		"gain (%d+)%% of physical damage as extra damage of a random element": function(num) return { mod("PhysicalDamageGainAsRandom", "BASE", num ) } end,
		"gain (%d+)%% of physical damage as extra damage of a random element while you are ignited": function(num) return { mod("PhysicalDamageGainAsRandom", "BASE", num, .Tag(mod.Condition("Ignited")) ) } end,
		"(%d+)%% of physical damage from hits with this weapon is converted to a random element": function(num) return { mod("PhysicalDamageConvertToRandom", "BASE", num ) } end,
		// TODO Crit
		"your critical strike chance is lucky": { mod.NewFlag("CritChanceLucky", true) },
		"your critical strike chance is lucky while on low life": { flag("CritChanceLucky", .Tag(mod.Condition("LowLife"))) },
		"your critical strike chance is lucky while focus?sed": { flag("CritChanceLucky", .Tag(mod.Condition("Focused"))) },
		"your critical strikes do not deal extra damage": { mod.NewFlag("NoCritMultiplier", true) },
		"lightning damage with non%-critical strikes is lucky": { mod.NewFlag("LightningNoCritLucky", true) },
		"your damage with critical strikes is lucky": { mod.NewFlag("CritLucky", true) },
		"critical strikes deal no damage": { mod("Damage", "MORE", -100, .Tag(mod.Condition("CriticalStrike"))) },
		"critical strike chance is increased by uncapped lightning resistance": { mod("CritChance", "INC", 1, { type = "PerStat", stat = "LightningResistTotal", div = 1 }) },
		"critical strike chance is increased by lightning resistance": { mod("CritChance", "INC", 1, { type = "PerStat", stat = "LightningResist", div = 1 }) },
		"critical strike chance is increased by overcapped lightning resistance": { mod("CritChance", "INC", 1, { type = "PerStat", stat = "LightningResistOverCap", div = 1 }) },
		"non%-critical strikes deal (%d+)%% damage": function(num) return { mod("Damage", "MORE", -100 + num, nil, mod.MFlagHit, .Tag(mod.Condition("CriticalStrike").Neg(true))) } end,
		"critical strikes penetrate (%d+)%% of enemy elemental resistances while affected by zealotry": function(num) return { mod("ElementalPenetration", "BASE", num, .Tag(mod.Condition("CriticalStrike")), .Tag(mod.Condition("AffectedByZealotry"))) } end,
		"attack critical strikes ignore enemy monster elemental resistances": { flag("IgnoreElementalResistances", .Tag(mod.Condition("CriticalStrike")), mod.SkillType(data.SkillTypeAttack)) },
		// TODO Generic Ailments
		"enemies take (%d+)%% increased damage for each type of ailment you have inflicted on them": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Frozen" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Chilled" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Ignited" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Shocked" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Scorched" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Brittle" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Sapped" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Bleeding" }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Poisoned" }),
		} end,
		// TODO Elemental Ailments
		"your shocks can increase damage taken by up to a maximum of (%d+)%%": function(num) return { mod.NewFloat("ShockMax", mod.TypeOverride, num) } end,
		"your elemental damage can shock": { mod.NewFlag("ColdCanShock", true), mod.NewFlag("FireCanShock", true) },
		"all your damage can freeze": { mod.NewFlag("PhysicalCanFreeze", true), mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("FireCanFreeze", true), mod.NewFlag("ChaosCanFreeze", true) },
		"all damage with maces and sceptres inflicts chill":  { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Chilled", true) }, .Tag(mod.Condition("UsingMace")) )},
		"your cold damage can ignite": { mod.NewFlag("ColdCanIgnite", true) },
		"your lightning damage can ignite": { mod.NewFlag("LightningCanIgnite", true) },
		"your fire damage can shock but not ignite": { mod.NewFlag("FireCanShock", true), mod.NewFlag("FireCannotIgnite", true) },
		"your cold damage can ignite but not freeze or chill": { mod.NewFlag("ColdCanIgnite", true), mod.NewFlag("ColdCannotFreeze", true), mod.NewFlag("ColdCannotChill", true) },
		"your cold damage cannot freeze": { mod.NewFlag("ColdCannotFreeze", true) },
		"your lightning damage can freeze but not shock": { mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("LightningCannotShock", true) },
		"your chaos damage can shock": { mod.NewFlag("ChaosCanShock", true) },
		"your chaos damage can chill": { mod.NewFlag("ChaosCanChill", true) },
		"your chaos damage can ignite": { mod.NewFlag("ChaosCanIgnite", true) },
		"chaos damage can ignite, chill and shock": { mod.NewFlag("ChaosCanIgnite", true), mod.NewFlag("ChaosCanChill", true), mod.NewFlag("ChaosCanShock", true) },
		"your physical damage can chill": { mod.NewFlag("PhysicalCanChill", true) },
		"your physical damage can shock": { mod.NewFlag("PhysicalCanShock", true) },
		"your physical damage can freeze": { mod.NewFlag("PhysicalCanFreeze", true) },
		"you always ignite while burning": { mod("EnemyIgniteChance", "BASE", 100, .Tag(mod.Condition("Burning"))) },
		"critical strikes do not a?l?w?a?y?s?i?n?h?e?r?e?n?t?l?y? freeze": { mod.NewFlag("CritsDontAlwaysFreeze", true) },
		"cannot inflict elemental ailments": {
			mod.NewFlag("CannotIgnite", true),
			mod.NewFlag("CannotChill", true),
			mod.NewFlag("CannotFreeze", true),
			mod.NewFlag("CannotShock", true),
			mod.NewFlag("CannotScorch", true),
			mod.NewFlag("CannotBrittle", true),
			mod.NewFlag("CannotSap", true),
		},
		"you can inflict up to (%d+) ignites on an enemy": { mod.NewFlag("IgniteCanStack", true) },
		"you can inflict an additional ignite on an enemy": { mod.NewFlag("IgniteCanStack", true), mod.NewFloat("IgniteStacks", mod.TypeBase, 1) },
		"enemies chilled by you take (%d+)%% increased burning damage": function(num) return { mod("EnemyModifier", "LIST", { mod = mod.NewFloat("FireDamageTakenOverTime", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Chilled" }) } end,
		"damaging ailments deal damage (%d+)%% faster": function(num) return { mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num), mod.NewFloat("BleedFaster", mod.TypeIncrease, num), mod.NewFloat("PoisonFaster", mod.TypeIncrease, num) } end,
		"damaging ailments you inflict deal damage (%d+)%% faster while affected by malevolence": function(num) return {
			mod("IgniteBurnFaster", "INC", num, .Tag(mod.Condition("AffectedByMalevolence"))),
			mod("BleedFaster", "INC", num, .Tag(mod.Condition("AffectedByMalevolence"))),
			mod("PoisonFaster", "INC", num, .Tag(mod.Condition("AffectedByMalevolence"))),
		} end,
		"ignited enemies burn (%d+)%% faster": function(num) return { mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num) } end,
		"ignited enemies burn (%d+)%% slower": function(num) return { mod.NewFloat("IgniteBurnSlower", mod.TypeIncrease, num) } end,
		"enemies ignited by an attack burn (%d+)%% faster": function(num) return { mod("IgniteBurnFaster", "INC", num, nil, mod.MFlagAttack) } end,
		"ignites you inflict with attacks deal damage (%d+)%% faster": function(num) return { mod("IgniteBurnFaster", "INC", num, nil, mod.MFlagAttack) } end,
		"ignites you inflict deal damage (%d+)%% faster": function(num) return { mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num) } end,
		"enemies ignited by you during flask effect take (%d+)%% increased damage": function(num) return { mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "Ignited" }) } end,
		"enemies ignited by you take chaos damage instead of fire damage from ignite": { mod.NewFlag("IgniteToChaos", true) },
		"enemies chilled by your hits are shocked": {
			mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default, { type = "ActorCondition", actor = "enemy", var = "ChilledByYourHits" } ),
			mod("EnemyModifier", "LIST", { mod = flag("Condition:Shocked", .Tag(mod.Condition("ChilledByYourHits")) ) } )
		},
		"cannot inflict ignite": { mod.NewFlag("CannotIgnite", true) },
		"cannot inflict freeze or chill": { mod.NewFlag("CannotFreeze", true), mod.NewFlag("CannotChill", true) },
		"cannot inflict shock": { mod.NewFlag("CannotShock", true) },
		"cannot ignite, chill, freeze or shock": { mod.NewFlag("CannotIgnite", true), mod.NewFlag("CannotChill", true), mod.NewFlag("CannotFreeze", true), mod.NewFlag("CannotShock", true) },
		"shock enemies as though dealing (%d+)%% more damage": function(num) return { mod.NewFloat("ShockAsThoughDealing", mod.TypeMore, num) } end,
		"inflict non%-damaging ailments as though dealing (%d+)%% more damage": function(num) return {
			mod.NewFloat("ShockAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("ChillAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("FreezeAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("ScorchAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("BrittleAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("SapAsThoughDealing", mod.TypeMore, num),
		} end,
		"immune to elemental ailments while on consecrated ground if you have at least (%d+) devotion": function(num)
			local mods = { }
			for i, ailment in ipairs(data.elementalAilmentTypeList) do
				mods[i] = mod("Avoid"..ailment, "BASE", 100, .Tag(mod.Condition("OnConsecratedGround")), { type = "StatThreshold", stat = "Devotion", threshold = num })
			end
			return mods
		end,
		"freeze chilled enemies as though dealing (%d+)%% more damage": function(num) return { mod("FreezeAsThoughDealing", "MORE", num, { type = "ActorCondition", actor = "enemy", var = "Chilled" } ) } end,
		"(%d+)%% chance to shock attackers for (%d+) seconds on block": { mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default) },
		["shock attackers for (%d+) seconds on block"]  = {
			mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default, .Tag(mod.Condition("BlockedRecently"))),
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Shocked", true) }, .Tag(mod.Condition("BlockedRecently")) ),
		},
		["shock nearby enemies for (%d+) seconds when you focus"]  = {
			mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default, .Tag(mod.Condition("Focused"))),
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Shocked", true) }, .Tag(mod.Condition("Focused")) ),
		},
		"drops shocked ground while moving, lasting (%d+) seconds": { mod("ShockBase", "BASE", data.nonDamagingAilment["Shock"].default, { type = "ActorCondition", actor = "enemy", var = "OnShockedGround"} ) },
		"drops scorched ground while moving, lasting (%d+) seconds": { mod("ScorchBase", "BASE", data.nonDamagingAilment["Scorch"].default, { type = "ActorCondition", actor = "enemy", var = "OnScorchedGround"} ) },
		"drops brittle ground while moving, lasting (%d+) seconds": { mod("BrittleBase", "BASE", data.nonDamagingAilment["Brittle"].default, { type = "ActorCondition", actor = "enemy", var = "OnBrittleGround"} ) },
		"drops sapped ground while moving, lasting (%d+) seconds": { mod("SapBase", "BASE", data.nonDamagingAilment["Sap"].default, { type = "ActorCondition", actor = "enemy", var = "OnSappedGround"} ) },
		"%+(%d+)%% chance to ignite, freeze, shock, and poison cursed enemies": function(num) return {
			mod("EnemyIgniteChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed" }),
			mod("EnemyFreezeChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed" }),
			mod("EnemyShockChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed" }),
			mod("PoisonChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed" }),
		} end,
		"you have scorching conflux, brittle conflux and sapping conflux while your two highest attributes are equal": {
			mod("EnemyScorchChance", "BASE", 100, .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			mod("EnemyBrittleChance", "BASE", 100, .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			mod("EnemySapChance", "BASE", 100, .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("PhysicalCanScorch", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("LightningCanScorch", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("ColdCanScorch", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("ChaosCanScorch", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("PhysicalCanBrittle", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("LightningCanBrittle", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("FireCanBrittle", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("ChaosCanBrittle", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("PhysicalCanSap", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("ColdCanSap", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("FireCanSap", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
			flag("ChaosCanSap", .Tag(mod.Condition("TwoHighestAttributesEqual"))),
		},
		"critical strikes do not inherently apply non%-damaging ailments": {
			mod.NewFlag("CritsDontAlwaysChill", true),
			mod.NewFlag("CritsDontAlwaysFreeze", true),
			mod.NewFlag("CritsDontAlwaysShock", true),
		},
		"always scorch while affected by anger": { mod("EnemyScorchChance", "BASE", 100, .Tag(mod.Condition("AffectedByAnger"))) },
		"always inflict brittle while affected by hatred": {	mod("EnemyBrittleChance", "BASE", 100, .Tag(mod.Condition("AffectedByHatred")))	},
		"always sap while affected by wrath": { mod("EnemySapChance", "BASE", 100, .Tag(mod.Condition("AffectedByWrath"))) },
		// TODO Bleed
		"melee attacks cause bleeding": { mod("BleedChance", "BASE", 100, nil, mod.MFlagMelee) },
		"attacks cause bleeding when hitting cursed enemies": { mod("BleedChance", "BASE", 100, nil, mod.MFlagAttack, { type = "ActorCondition", actor = "enemy", var = "Cursed" }) },
		"melee critical strikes cause bleeding": { mod("BleedChance", "BASE", 100, nil, mod.MFlagMelee, .Tag(mod.Condition("CriticalStrike"))) },
		"causes bleeding on melee critical strike": { mod("BleedChance", "BASE", 100, nil, mod.MFlagMelee, .Tag(mod.Condition("CriticalStrike"))) },
		"melee critical strikes have (%d+)%% chance to cause bleeding": function(num) return { mod("BleedChance", "BASE", num, nil, mod.MFlagMelee, .Tag(mod.Condition("CriticalStrike"))) } end,
		"attacks always inflict bleeding while you have cat's stealth": { mod("BleedChance", "BASE", 100, nil, mod.MFlagAttack, .Tag(mod.Condition("AffectedByCat'sStealth"))) },
		"you have crimson dance while you have cat's stealth": { mod("Keystone", "LIST", "Crimson Dance", .Tag(mod.Condition("AffectedByCat'sStealth"))) },
		"you have crimson dance if you have dealt a critical strike recently": { mod("Keystone", "LIST", "Crimson Dance", .Tag(mod.Condition("CritRecently"))) },
		"bleeding you inflict deals damage (%d+)%% faster": function(num) return { mod.NewFloat("BleedFaster", mod.TypeIncrease, num) } end,
		"(%d+)%% chance for bleeding inflicted with this weapon to deal (%d+)%% more damage": function(num, _, more) return {
			mod("Damage", "MORE", tonumber(more) * num / 100, nil, 0, mod.KeywordFlagBleed, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
		} end,
		"bleeding you inflict deals damage (%d+)%% faster per frenzy charge": function(num) return { mod("BleedFaster", "INC", num, mod.Multiplier("FrenzyCharge").Base(0)) } end,
		// TODO Impale and Bleed
		"(%d+)%% increased effect of impales inflicted by hits that also inflict bleeding": function(num) return {
			mod("ImpaleEffectOnBleed", "INC", num, nil, 0, mod.KeywordFlagHit)
		} end,
		// TODO Poison and Bleed
		"(%d+)%% increased damage with bleeding inflicted on poisoned enemies": function(num) return {
			mod("Damage", "INC", num, nil, 0, mod.KeywordFlagBleed, { type = "ActorCondition", actor = "enemy", var = "Poisoned"})
		} end,
		// TODO Poison
		"y?o?u?r? ?fire damage can poison": { mod.NewFlag("FireCanPoison", true) },
		"y?o?u?r? ?cold damage can poison": { mod.NewFlag("ColdCanPoison", true) },
		"y?o?u?r? ?lightning damage can poison": { mod.NewFlag("LightningCanPoison", true) },
		"all damage from hits with this weapon can poison": {
			flag("FireCanPoison", .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
			flag("ColdCanPoison", .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
			flag("LightningCanPoison", .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack))
		},
		"all damage inflicts poison while affected by glorious madness": {
			flag("FireCanPoison", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("ColdCanPoison", .Tag(mod.Condition("AffectedByGloriousMadness"))),
			flag("LightningCanPoison", .Tag(mod.Condition("AffectedByGloriousMadness")))
		},
		"your chaos damage poisons enemies": { mod.NewFloat("ChaosPoisonChance", mod.TypeBase, 100) },
		"your chaos damage has (%d+)%% chance to poison enemies": function(num) return { mod.NewFloat("ChaosPoisonChance", mod.TypeBase, num) } end,
		"melee attacks poison on hit": { mod("PoisonChance", "BASE", 100, nil, mod.MFlagMelee) },
		"melee critical strikes have (%d+)%% chance to poison the enemy": function(num) return { mod("PoisonChance", "BASE", num, nil, mod.MFlagMelee, .Tag(mod.Condition("CriticalStrike"))) } end,
		"critical strikes with daggers have a (%d+)%% chance to poison the enemy": function(num) return { mod("PoisonChance", "BASE", num, nil, mod.MFlagDagger, .Tag(mod.Condition("CriticalStrike"))) } end,
		"critical strikes with daggers poison the enemy": { mod("PoisonChance", "BASE", 100, nil, mod.MFlagDagger, .Tag(mod.Condition("CriticalStrike"))) },
		"poison cursed enemies on hit": { mod("PoisonChance", "BASE", 100, { type = "ActorCondition", actor = "enemy", var = "Cursed" }) },
		"wh[ie][ln]e? at maximum frenzy charges, attacks poison enemies": { mod("PoisonChance", "BASE", 100, nil, mod.MFlagAttack, { type = "StatThreshold", stat = "FrenzyCharges", thresholdStat = "FrenzyChargesMax" }) },
		"traps and mines have a (%d+)%% chance to poison on hit": function(num) return { mod("PoisonChance", "BASE", num, nil, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine)) } end,
		"poisons you inflict deal damage (%d+)%% faster": function(num) return { mod.NewFloat("PoisonFaster", mod.TypeIncrease, num) } end,
		"(%d+)%% chance for poisons inflicted with this weapon to deal (%d+)%% more damage": function(num, _, more) return {
			mod("Damage", "MORE", tonumber(more) * num / 100, nil, 0, mod.KeywordFlagPoison, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)),
		} end,
		// TODO Suppression
		"your chance to suppressed spell damage is lucky": { mod.NewFlag("SpellSuppressionChanceIsLucky", true) },
		"your chance to suppressed spell damage is unlucky": { mod.NewFlag("SpellSuppressionChanceIsUnlucky", true) },
		"prevent +(%+%d+)%% of suppressed spell damage": function(num) return { mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, num) } end,
		"critical strike chance is increased by chance to suppress spell damage": { mod("CritChance", "INC", 1, { type = "PerStat", stat = "SpellSuppressionChance", div = 1 }) },
		"you take (%d+)%% reduced extra damage from suppressed critical strikes": function(num) return { mod.NewFloat("ReduceSuppressedCritExtraDamage", mod.TypeBase, num) } end,
		"+(%d+)%% chance to suppress spell damage if your boots, helmet and gloves have evasion": function(num) return {
			mod("SpellSuppressionChance", "BASE", tonumber(num),
				{ type = "StatThreshold", stat = "EvasionOnBoots", threshold = 1},
				{ type = "StatThreshold", stat = "EvasionOnHelmet", threshold = 1, uppper = true},
				{ type = "StatThreshold", stat = "EvasionOnGloves", threshold = 1, uppper = true}
			)
		} end,
		"+(%d+)%% chance to suppress spell damage for each dagger you're wielding": function(num) return {
			mod("SpellSuppressionChance", "BASE", num, { type = "ModFlag", modFlags = mod.MFlagDagger } ),
			mod("SpellSuppressionChance", "BASE", num, .Tag(mod.Condition("DualWieldingDaggers")) )
		} end,
		// TODO Buffs/debuffs
		"phasing": { mod.NewFlag("Condition:Phasing", true) },
		"onslaught": { mod.NewFlag("Condition:Onslaught", true) },
		"unholy might": { mod.NewFlag("Condition:UnholyMight", true) },
		"your aura buffs do not affect allies": { mod.NewFlag("SelfAurasCannotAffectAllies", true) },
		"auras from your skills can only affect you": { mod.NewFlag("SelfAurasOnlyAffectYou", true) },
		"aura buffs from skills have (%d+)%% increased effect on you for each herald affecting you": function(num) return { mod("SkillAuraEffectOnSelf", "INC", num, { type = "Multiplier", var = "Herald"}) } end,
		"aura buffs from skills have (%d+)%% increased effect on you for each herald affecting you, up to (%d+)%%": function(num, _, limit) return {
			mod("SkillAuraEffectOnSelf", "INC", num, { type = "Multiplier", var = "Herald", globalLimit = tonumber(limit), globalLimitKey = "PurposefulHarbinger" })
		} end,
		"(%d+)%% increased area of effect per power charge, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("AreaOfEffect", "INC", num, { type = "Multiplier", var = "PowerCharge", globalLimit = tonumber(limit), globalLimitKey = "VastPower" })
		} end,
		"(%d+)%% increased chaos damage per (%d+) maximum mana, up to a maximum of (%d+)%%": function(num, _, div, limit) return {
			mod("ChaosDamage", "INC", num, { type = "PerStat", stat = "Mana", div = tonumber(div), globalLimit = tonumber(limit), globalLimitKey = "DarkIdeation" })
		} end,
		"minions have %+(%d+)%% to damage over time multiplier per ghastly eye jewel affecting you, up to a maximum of %+(%d+)%%": function(num, _, limit) return {
			mod("MinionModifier", "LIST", { mod = mod("DotMultiplier", "BASE", num, { type = "Multiplier", var = "GhastlyEyeJewel", actor = "parent", globalLimit = tonumber(limit), globalLimitKey = "AmanamuGaze" }) })
		} end,
		"(%d+)%% increased effect of arcane surge on you per hypnotic eye jewel affecting you, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("ArcaneSurgeEffect", "INC", num, { type = "Multiplier", var = "HypnoticEyeJewel", globalLimit = tonumber(limit), globalLimitKey = "KurgalGaze" })
		} end,
		"(%d+)%% increased main hand critical strike chance per murderous eye jewel affecting you, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("CritChance", "INC", num, { type = "Multiplier", var = "MurderousEyeJewel", globalLimit = tonumber(limit), globalLimitKey = "TecrodGazeMainHand" }, .Tag(mod.Condition("MainHandAttack")))
		} end,
		"%+(%d+)%% to off hand critical strike multiplier per murderous eye jewel affecting you, up to a maximum of %+(%d+)%%": function(num, _, limit) return {
			mod("CritMultiplier", "BASE", num, { type = "Multiplier", var = "MurderousEyeJewel", globalLimit = tonumber(limit), globalLimitKey = "TecrodGazeOffHand" }, .Tag(mod.Condition("OffHandAttack")))
		} end,
		"nearby allies' damage with hits is lucky": { mod("ExtraAura", "LIST", { onlyAllies = true, mod = mod.NewFlag("LuckyHits", true) }) },
		"your damage with hits is lucky": { mod.NewFlag("LuckyHits", true) },
		"elemental damage with hits is lucky while you are shocked": { flag("ElementalLuckHits", .Tag(mod.Condition("Shocked"))) },
		"allies' aura buffs do not affect you": { mod.NewFlag("AlliesAurasCannotAffectSelf", true) },
		"(%d+)%% increased effect of non%-curse auras from your skills on enemies": function(num) return {
			mod("DebuffEffect", "INC", num, mod.SkillType(data.SkillTypeAura), { type = "SkillType", skillType = SkillType.AppliesCurse, neg = true }),
			mod("AuraEffect", "INC", num, mod.SkillName("Death Aura")),
		} end,
		"enemies can have 1 additional curse": { mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1) },
		"you can apply an additional curse": { mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1) },
		"you can apply an additional curse while affected by malevolence": { mod("EnemyCurseLimit", "BASE", 1, .Tag(mod.Condition("AffectedByMalevolence"))) },
		"you can apply one fewer curse": { mod("EnemyCurseLimit", "BASE", -1) },
		"curses on enemies in your chilling areas have (%d+)%% increased effect": function(num) return { mod("CurseEffect", "INC", num, { type = "ActorCondition", actor = "enemy", var = "InChillingArea" } ) } end,
		"hexes you inflict have their effect increased by twice their doom instead": { mod.NewFloat("DoomEffect", mod.TypeMore, 100) },
		"nearby enemies have an additional (%d+)%% chance to receive a critical strike": function(num) return { mod("EnemyModifier", "LIST", { mod = mod.NewFloat("SelfExtraCritChance", mod.TypeBase, num) }) } end,
		"nearby enemies have (%-%d+)%% to all resistances": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("ElementalResist", mod.TypeBase, num) }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("ChaosResist", mod.TypeBase, num) }),
		} end,
		"enemies ignited or chilled by you have (%-%d+)%% to elemental resistances": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod("ElementalResist", "BASE", num )}, { type = "ActorCondition", actor = "enemy", varList = { "Ignited", "Chilled" } })
		} end,
		"your hits inflict decay, dealing (%d+) chaos damage per second for %d+ seconds": function(num) return { mod("SkillData", "LIST", { key = "decay", value = num, merge = "MAX" }) } end,
		"temporal chains has (%d+)%% reduced effect on you": function(num) return { mod("CurseEffectOnSelf", "INC", -num, mod.SkillName("Temporal Chains")) } end,
		"unaffected by temporal chains": { mod("CurseEffectOnSelf", "MORE", -100, mod.SkillName("Temporal Chains")) },
		"([%+%-][%d%.]+) seconds to cat's stealth duration": function(num) return { mod("PrimaryDuration", "BASE", num, mod.SkillName("Aspect of the Cat")) } end,
		"([%+%-][%d%.]+) seconds to cat's agility duration": function(num) return { mod("SecondaryDuration", "BASE", num, mod.SkillName("Aspect of the Cat")) } end,
		"([%+%-][%d%.]+) seconds to avian's might duration": function(num) return { mod("PrimaryDuration", "BASE", num, mod.SkillName("Aspect of the Avian")) } end,
		"([%+%-][%d%.]+) seconds to avian's flight duration": function(num) return { mod("SecondaryDuration", "BASE", num, mod.SkillName("Aspect of the Avian")) } end,
		"aspect of the spider can inflict spider's web on enemies an additional time": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("Multiplier:SpiderWebApplyStackMax", mod.TypeBase, 1) }, mod.SkillName("Aspect of the Spider")) },
		"aspect of the avian also grants avian's might and avian's flight to nearby allies": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("BuffEffectOnMinion", mod.TypeMore, 100) }, mod.SkillName("Aspect of the Avian")) },
		"marked enemy takes (%d+)%% increased damage": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTaken", mod.TypeIncrease, num) }, {type = "ActorCondition", actor = "enemy", var = "Marked"}),
		} end,
		"marked enemy has (%d+)%% reduced accuracy rating": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod("Accuracy", "INC", -num) }, {type = "ActorCondition", actor = "enemy", var = "Marked"}),
		} end,
		"you are cursed with level (%d+) (%D+)": function(num, _, name) return { mod("ExtraCurse", "LIST", { skillId = gemIdLookup[name], level = num, applyToPlayer = true }) } end,
		"you are cursed with (%D+), with (%d+)%% increased effect": function(_, skill, num) return {
			mod("ExtraCurse", "LIST", { skillId = gemIdLookup[skill], level = 1, applyToPlayer = true }),
			mod("CurseEffectOnSelf", "INC", tonumber(num), { type = "SkillName", skillName = string.gsub(" "..skill, "%W%l", string.upper):sub(2) }),
		} end,
		"you count as on low life while you are cursed with vulnerability": { flag("Condition:LowLife", .Tag(mod.Condition("AffectedByVulnerability"))) },
		"you count as on full life while you are cursed with vulnerability": { flag("Condition:FullLife", .Tag(mod.Condition("AffectedByVulnerability"))) },
		"if you consumed a corpse recently, you and nearby allies regenerate (%d+)%% of life per second": function (num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("LifeRegenPercent", mod.TypeBase, num) }, .Tag(mod.Condition("ConsumedCorpseRecently"))) } end,
		"if you have blocked recently, you and nearby allies regenerate (%d+)%% of life per second": function (num) return { mod("ExtraAura", "LIST", { mod = mod.NewFloat("LifeRegenPercent", mod.TypeBase, num) }, .Tag(mod.Condition("BlockedRecently"))) } end,
		"you are at maximum chance to block attack damage if you have not blocked recently": { flag("MaxBlockIfNotBlockedRecently", .Tag(mod.Condition("BlockedRecently").Neg(true))) },
		"%+(%d+)%% chance to block attack damage if you have not blocked recently": function(num) return { mod("BlockChance", "BASE", num, .Tag(mod.Condition("BlockedRecently").Neg(true))) } end,
		"%+(%d+)%% chance to block spell damage if you have not blocked recently": function(num) return { mod("SpellBlockChance", "BASE", num, .Tag(mod.Condition("BlockedRecently").Neg(true))) } end,
		"(%d+)%% of evasion rating is regenerated as life per second while focus?sed": function(num) return { mod("LifeRegen", "BASE", 1, { type = "PercentStat", stat = "Evasion", percent = num }, .Tag(mod.Condition("Focused"))) } end,
		"nearby allies have (%d+)%% increased defences per (%d+) strength you have": function(num, _, div) return { mod("ExtraAura", "LIST", { onlyAllies = true, mod = mod.NewFloat("Defences", mod.TypeIncrease, num) }, { type = "PerStat", stat = "Str", div = tonumber(div) }) } end,
		"nearby allies have %+(%d+)%% to critical strike multiplier per (%d+) dexterity you have": function(num, _, div) return { mod("ExtraAura", "LIST", { onlyAllies = true, mod = mod.NewFloat("CritMultiplier", mod.TypeBase, num) }, { type = "PerStat", stat = "Dex", div = tonumber(div) }) } end,
		"nearby allies have (%d+)%% increased cast speed per (%d+) intelligence you have": function(num, _, div) return { mod("ExtraAura", "LIST", { onlyAllies = true, mod = mod("Speed", "INC", num, nil, mod.MFlagCast ) }, { type = "PerStat", stat = "Int", div = tonumber(div) }) } end,
		"you gain divinity for %d+ seconds on reaching maximum divine charges": {
			mod("ElementalDamage", "MORE", 50, .Tag(mod.Condition("Divinity"))),
			mod("ElementalDamageTaken", "MORE", -20, .Tag(mod.Condition("Divinity"))),
		},
		"your maximum endurance charges is equal to your maximum frenzy charges": { mod.NewFlag("MaximumEnduranceChargesIsMaximumFrenzyCharges", true) },
		"your maximum frenzy charges is equal to your maximum power charges": { mod.NewFlag("MaximumFrenzyChargesIsMaximumPowerCharges", true) },
		"consecrated ground you create while affected by zealotry causes enemies to take (%d+)%% increased damage": function(num) return { mod("EnemyModifier", "LIST", { mod = mod.NewFloat("DamageTakenConsecratedGround", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "OnConsecratedGround" }, .Tag(mod.Condition("AffectedByZealotry"))) } end,
		"if you've warcried recently, you and nearby allies have (%d+)%% increased attack, cast and movement speed": function(num) return {
			mod("ExtraAura", "LIST", { mod = mod.NewFloat("Speed", mod.TypeIncrease, num) }, .Tag(mod.Condition("UsedWarcryRecently"))),
			mod("ExtraAura", "LIST", { mod = mod.NewFloat("MovementSpeed", mod.TypeIncrease, num) }, .Tag(mod.Condition("UsedWarcryRecently"))),
		} end,
		"when you warcry, you and nearby allies gain onslaught for 4 seconds": { mod("ExtraAura", "LIST", { mod = mod.NewFlag("Onslaught", true) }, .Tag(mod.Condition("UsedWarcryRecently"))) },
		"enemies in your chilling areas take (%d+)%% increased lightning damage": function(num) return { mod("EnemyModifier", "LIST", { mod = mod.NewFloat("LightningDamageTaken", mod.TypeIncrease, num) }, { type = "ActorCondition", actor = "enemy", var = "InChillingArea" }) } end,
		"(%d+)%% chance to sap enemies in chilling areas": function(num) return { mod("EnemySapChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "InChillingArea" } ) } end,
		"warcries count as having (%d+) additional nearby enemies": function(num) return {
			mod.NewFloat("Multiplier:WarcryNearbyEnemies", mod.TypeBase, num),
		} end,
		"enemies taunted by your warcries take (%d+)%% increased damage": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTaken", "INC", num, .Tag(mod.Condition("Taunted"))) }, .Tag(mod.Condition("UsedWarcryRecently"))) } end,
		"warcries share their cooldown": { mod.NewFlag("WarcryShareCooldown", true) },
		"warcries have minimum of (%d+) power": { mod.NewFlag("CryWolfMinimumPower", true) },
		"warcries have infinite power": { mod.NewFlag("WarcryInfinitePower", true) },
		"(%d+)%% chance to inflict corrosion on hit with attacks": { mod.NewFlag("Condition:CanCorrode", true) },
		"(%d+)%% chance to inflict withered for (%d+) seconds on hit": { mod.NewFlag("Condition:CanWither", true) },
		"(%d+)%% chance to inflict withered for (%d+) seconds on hit with this weapon": { mod.NewFlag("Condition:CanWither", true) },
		"(%d+)%% chance to inflict withered for two seconds on hit if there are (%d+) or fewer withered debuffs on enemy": { mod.NewFlag("Condition:CanWither", true) },
		"inflict withered for (%d+) seconds on hit with this weapon": { mod.NewFlag("Condition:CanWither", true) },
		"enemies take (%d+)%% increased elemental damage from your hits for each withered you have inflicted on them": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("ElementalDamageTaken", "INC", num, mod.Multiplier("WitheredStack").Base(0)) }) } end,
		"your hits cannot penetrate or ignore elemental resistances": { mod.NewFlag("CannotElePenIgnore", true) },
		"nearby enemies have malediction": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("HasMalediction", true) }) },
		"elemental damage you deal with hits is resisted by lowest elemental resistance instead": { mod.NewFlag("ElementalDamageUsesLowestResistance", true) },
		"you take (%d+) chaos damage per second for 3 seconds on kill": function(num) return { mod("ChaosDegen", "BASE", num, .Tag(mod.Condition("KilledLast3Seconds"))) } end,
		"regenerate (%d+) life over 1 second for each spell you cast": function(num) return { mod("LifeRegen", "BASE", num, .Tag(mod.Condition("CastLast1Seconds"))) } end,
		"and nearby allies regenerate (%d+) life per second": function(num) return { mod("LifeRegen", "BASE", num, .Tag(mod.Condition("KilledPosionedLast2Seconds"))) } end,
		"(%d+)%% increased life regeneration rate": function(num) return { mod.NewFloat("LifeRegen", mod.TypeIncrease, num) } end,
		"fire skills have a (%d+)%% chance to apply fire exposure on hit": function(num) return { mod.NewFloat("FireExposureChance", mod.TypeBase, num) } end,
		"cold skills have a (%d+)%% chance to apply cold exposure on hit": function(num) return { mod.NewFloat("ColdExposureChance", mod.TypeBase, num) } end,
		"lightning skills have a (%d+)%% chance to apply lightning exposure on hit": function(num) return { mod.NewFloat("LightningExposureChance", mod.TypeBase, num) } end,
		"socketed skills apply fire, cold and lightning exposure on hit": {
			mod("FireExposureChance", "BASE", 100, .Tag(mod.Condition("Effective"))),
			mod("ColdExposureChance", "BASE", 100, .Tag(mod.Condition("Effective"))),
			mod("LightningExposureChance", "BASE", 100, .Tag(mod.Condition("Effective"))),
		},
		"nearby enemies have fire exposure": {
			mod("EnemyModifier", "LIST", { mod = mod("FireExposure", "BASE", -10) }, .Tag(mod.Condition("Effective"))),
		},
		"nearby enemies have cold exposure": {
			mod("EnemyModifier", "LIST", { mod = mod("ColdExposure", "BASE", -10) }, .Tag(mod.Condition("Effective"))),
		},
		"nearby enemies have lightning exposure": {
			mod("EnemyModifier", "LIST", { mod = mod("LightningExposure", "BASE", -10) }, .Tag(mod.Condition("Effective"))),
		},
		"nearby enemies have fire exposure while you are affected by herald of ash": {
			mod("EnemyModifier", "LIST", { mod = mod("FireExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("AffectedByHeraldofAsh"))),
		},
		"nearby enemies have cold exposure while you are affected by herald of ice": {
			mod("EnemyModifier", "LIST", { mod = mod("ColdExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("AffectedByHeraldofIce"))),
		},
		"nearby enemies have lightning exposure while you are affected by herald of thunder": {
			mod("EnemyModifier", "LIST", { mod = mod("LightningExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("AffectedByHeraldofThunder"))),
		},
		"inflict fire, cold and lightning exposure on nearby enemies when used": {
			mod("EnemyModifier", "LIST", { mod = mod("FireExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("UsingFlask"))),
			mod("EnemyModifier", "LIST", { mod = mod("ColdExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("UsingFlask"))),
			mod("EnemyModifier", "LIST", { mod = mod("LightningExposure", "BASE", -10) }, .Tag(mod.Condition("Effective")), .Tag(mod.Condition("UsingFlask"))),
		},
		"enemies near your linked targets have fire, cold and lightning exposure": {
			mod("EnemyModifier", "LIST", { mod = mod("FireExposure", "BASE", -10, .Tag(mod.Condition("NearLinkedTarget"))) }, .Tag(mod.Condition("Effective"))),
			mod("EnemyModifier", "LIST", { mod = mod("ColdExposure", "BASE", -10, .Tag(mod.Condition("NearLinkedTarget"))) }, .Tag(mod.Condition("Effective"))),
			mod("EnemyModifier", "LIST", { mod = mod("LightningExposure", "BASE", -10, .Tag(mod.Condition("NearLinkedTarget"))) }, .Tag(mod.Condition("Effective"))),
		},
		"inflict (%w+) exposure on hit, applying %-(%d+)%% to (%w+) resistance": function(_, element1,  num, element2) return {
			mod( firstToUpper(element1).."ExposureChance", "BASE", 100, .Tag(mod.Condition("Effective"))),
			mod("EnemyModifier", "LIST", { mod = mod(firstToUpper(element2).."Exposure", "BASE", -num) }, .Tag(mod.Condition("Effective")) ),
		} end,
		"while a unique enemy is in your presence, inflict (%w+) exposure on hit, applying %-(%d+)%% to (%w+) resistance": function(_, element1,  num, element2) return {
			mod( firstToUpper(element1).."ExposureChance", "BASE", 100, { type = "ActorCondition", actor = "enemy", var = "RareOrUnique" }, .Tag(mod.Condition("Effective"))),
			mod("EnemyModifier", "LIST", { mod = mod(firstToUpper(element2).."Exposure", "BASE", -num, .Tag(mod.Condition("RareOrUnique"))) }, .Tag(mod.Condition("Effective")) ),
		} end,
		"while a pinnacle atlas boss is in your presence, inflict (%w+) exposure on hit, applying %-(%d+)%% to (%w+) resistance": function(_, element1,  num, element2) return {
			mod( firstToUpper(element1).."ExposureChance", "BASE", 100, { type = "ActorCondition", actor = "enemy", var = "PinnacleBoss" }, .Tag(mod.Condition("Effective"))),
			mod("EnemyModifier", "LIST", { mod = mod(firstToUpper(element2).."Exposure", "BASE", -num, .Tag(mod.Condition("PinnacleBoss"))) }, .Tag(mod.Condition("Effective")) ),
		} end,
		"fire exposure you inflict applies an extra (%-?%d+)%% to fire resistance": function(num) return { mod.NewFloat("ExtraFireExposure", mod.TypeBase, num) } end,
		"cold exposure you inflict applies an extra (%-?%d+)%% to cold resistance": function(num) return { mod.NewFloat("ExtraColdExposure", mod.TypeBase, num) } end,
		"lightning exposure you inflict applies an extra (%-?%d+)%% to lightning resistance": function(num) return { mod.NewFloat("ExtraLightningExposure", mod.TypeBase, num) } end,
		"exposure you inflict applies at least (%-%d+)%% to the affected resistance": function(num) return { mod.NewFloat("ExposureMin", mod.TypeOverride, num) } end,
		"modifiers to minimum endurance charges instead apply to minimum brutal charges": { mod.NewFlag("MinimumEnduranceChargesEqualsMinimumBrutalCharges", true) },
		"modifiers to minimum frenzy charges instead apply to minimum affliction charges": { mod.NewFlag("MinimumFrenzyChargesEqualsMinimumAfflictionCharges", true) },
		"modifiers to minimum power charges instead apply to minimum absorption charges": { mod.NewFlag("MinimumPowerChargesEqualsMinimumAbsorptionCharges", true) },
		"maximum brutal charges is equal to maximum endurance charges": { mod.NewFlag("MaximumEnduranceChargesEqualsMaximumBrutalCharges", true) },
		"maximum affliction charges is equal to maximum frenzy charges": { mod.NewFlag("MaximumFrenzyChargesEqualsMaximumAfflictionCharges", true) },
		"maximum absorption charges is equal to maximum power charges": { mod.NewFlag("MaximumPowerChargesEqualsMaximumAbsorptionCharges", true) },
		"gain brutal charges instead of endurance charges": { mod.NewFlag("EnduranceChargesConvertToBrutalCharges", true) },
		"gain affliction charges instead of frenzy charges": { mod.NewFlag("FrenzyChargesConvertToAfflictionCharges", true) },
		"gain absorption charges instead of power charges": { mod.NewFlag("PowerChargesConvertToAbsorptionCharges", true) },
		"regenerate (%d+)%% life over one second when hit while sane": function(num) return {
			mod("LifeRegenPercent", "BASE", num, .Tag(mod.Condition("Insane").Neg(true)), .Tag(mod.Condition("BeenHitRecently"))),
		} end,
		"you have lesser brutal shrine buff": {
			mod("ShrineBuff", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, 20) }),
			mod("ShrineBuff", "LIST", { mod = mod.NewFloat("EnemyStunDuration", mod.TypeIncrease, 20) }),
			mod.NewFloat("EnemyKnockbackChance", mod.TypeBase, 100)
		},
		"you have lesser massive shrine buff": {
			mod("ShrineBuff", "LIST", { mod = mod.NewFloat("Life", mod.TypeIncrease, 20) }),
			mod("ShrineBuff", "LIST", { mod = mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 20) })
		},
		"(%d+)%% increased effect of shrine buffs on you": function(num) return { mod.NewFloat("ShrineBuffEffect", mod.TypeIncrease, num)} end,
		"left ring slot: cover enemies in ash for 5 seconds when you ignite them": { mod("CoveredInAshEffect", "BASE", 20, { type = "SlotNumber", num = 1 }, { type = "ActorCondition", actor = "enemy", var = "Ignited" }) },
		"right ring slot: cover enemies in frost for 5 seconds when you freeze them": { mod("CoveredInFrostEffect", "BASE", 20, { type = "SlotNumber", num = 2 }, { type = "ActorCondition", actor = "enemy", var = "Frozen" }) },
		"([%a%s]+) has (%d+)%% increased effect": function(_, skill, num) return { mod("BuffEffect", "INC", num, { type = "SkillId", skillId = gemIdLookup[skill]}) } end,
		// TODO Traps, Mines and Totems
		"traps and mines deal (%d+)%-(%d+) additional physical damage": function(_, min, max) return { mod("PhysicalMin", "BASE", tonumber(min), nil, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine)), mod("PhysicalMax", "BASE", tonumber(max), nil, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine)) } end,
		"traps and mines deal (%d+) to (%d+) additional physical damage": function(_, min, max) return { mod("PhysicalMin", "BASE", tonumber(min), nil, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine)), mod("PhysicalMax", "BASE", tonumber(max), nil, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine)) } end,
		"each mine applies (%d+)%% increased damage taken to enemies near it, up to (%d+)%%": function(num, _, limit) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTaken", "INC", num, { type = "Multiplier", var = "ActiveMineCount", limit = limit / num }) }) } end,
		"can have up to (%d+) additional traps? placed at a time": function(num) return { mod.NewFloat("ActiveTrapLimit", mod.TypeBase, num) } end,
		"can have (%d+) fewer traps placed at a time": function(num) return { mod("ActiveTrapLimit", "BASE", -num) } end,
		"can have up to (%d+) additional remote mines? placed at a time": function(num) return { mod.NewFloat("ActiveMineLimit", mod.TypeBase, num) } end,
		"can have up to (%d+) additional totems? summoned at a time": function(num) return { mod.NewFloat("ActiveTotemLimit", mod.TypeBase, num) } end,
		"attack skills can have (%d+) additional totems? summoned at a time": function(num) return { mod("ActiveTotemLimit", "BASE", num, nil, 0, mod.KeywordFlagAttack) } end,
		"can [hs][au][vm][em]o?n? 1 additional siege ballista totem per (%d+) dexterity": function(num) return { mod("ActiveBallistaLimit", "BASE", 1, mod.SkillName("Siege Ballista"), { type = "PerStat", stat = "Dex", div = num }) } end,
		"totems fire (%d+) additional projectiles": function(num) return { mod("ProjectileCount", "BASE", num, nil, 0, mod.KeywordFlagTotem) } end,
		"([%d%.]+)%% of damage dealt by y?o?u?r? ?totems is leeched to you as life": function(num) return { mod("DamageLifeLeechToPlayer", "BASE", num, nil, 0, mod.KeywordFlagTotem) } end,
		"([%d%.]+)%% of damage dealt by y?o?u?r? ?mines is leeched to you as life": function(num) return { mod("DamageLifeLeechToPlayer", "BASE", num, nil, 0, mod.KeywordFlagMine) } end,
		"you can cast an additional brand": { mod.NewFloat("ActiveBrandLimit", mod.TypeBase, 1) },
		"you can cast (%d+) additional brands": function(num) return { mod.NewFloat("ActiveBrandLimit", mod.TypeBase, num) } end,
		"(%d+)%% increased damage while you are wielding a bow and have a totem": function(num) return { mod("Damage", "INC", num, .Tag(mod.Condition("HaveTotem")), .Tag(mod.Condition("UsingBow"))) } end,
		"each totem applies (%d+)%% increased damage taken to enemies near it": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTaken", "INC", num, mod.Multiplier("TotemsSummoned").Base(0)) }) } end,
		"totems gain %+(%d+)%% to (%w+) resistance": function(_, num, resistance) return { mod("Totem"..firstToUpper(resistance).."Resist", "BASE", num) } end,
		"totems gain %+(%d+)%% to all elemental resistances": function(_, num, resistance) return { mod.NewFloat("TotemElementalResist", mod.TypeBase, num) } end,
		// TODO Minions
		"your strength is added to your minions": { mod.NewFlag("HalfStrengthAddedToMinions", true) },
		"half of your strength is added to your minions": { mod.NewFlag("HalfStrengthAddedToMinions", true) },
		"minions created recently have (%d+)%% increased attack and cast speed": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Speed", mod.TypeIncrease, num) }, .Tag(mod.Condition("MinionsCreatedRecently"))) } end,
		"minions created recently have (%d+)%% increased movement speed": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("MovementSpeed", mod.TypeIncrease, num) }, .Tag(mod.Condition("MinionsCreatedRecently"))) } end,
		"minions poison enemies on hit": { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PoisonChance", mod.TypeBase, 100) }) },
		"minions have (%d+)%% chance to poison enemies on hit": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PoisonChance", mod.TypeBase, num) }) } end,
		"(%d+)%% increased minion damage if you have hit recently": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, num) }, .Tag(mod.Condition("HitRecently"))) } end,
		"(%d+)%% increased minion damage if you've used a minion skill recently": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, num) }, .Tag(mod.Condition("UsedMinionSkillRecently"))) } end,
		"minions deal (%d+)%% increased damage if you've used a minion skill recently": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, num) }, .Tag(mod.Condition("UsedMinionSkillRecently"))) } end,
		"minions have (%d+)%% increased attack and cast speed if you or your minions have killed recently": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Speed", mod.TypeIncrease, num) }, { type = "Condition", varList = { "KilledRecently", "MinionsKilledRecently" } }) } end,
		"(%d+)%% increased minion attack speed per (%d+) dexterity": function(num, _, div) return { mod("MinionModifier", "LIST", { mod = mod("Speed", "INC", num, nil, mod.MFlagAttack) }, { type = "PerStat", stat = "Dex", div = tonumber(div) }) } end,
		"(%d+)%% increased minion movement speed per (%d+) dexterity": function(num, _, div) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("MovementSpeed", mod.TypeIncrease, num) }, { type = "PerStat", stat = "Dex", div = tonumber(div) }) } end,
		"minions deal (%d+)%% increased damage per (%d+) dexterity": function(num, _, div) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeIncrease, num) }, { type = "PerStat", stat = "Dex", div = tonumber(div) }) } end,
		"minions have (%d+)%% chance to deal double damage while they are on full life": function(num) return { mod("MinionModifier", "LIST", { mod = mod("DoubleDamageChance", "BASE", num, .Tag(mod.Condition("FullLife"))) }) } end,
		"(%d+)%% increased golem damage for each type of golem you have summoned": function(num) return {
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HavePhysicalGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HaveLightningGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HaveColdGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HaveFireGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HaveChaosGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "HaveCarrionGolem" }) }, mod.SkillType(data.SkillTypeGolem)),
		} end,
		"can summon up to (%d) additional golems? at a time": function(num) return { mod.NewFloat("ActiveGolemLimit", mod.TypeBase, num) } end,
		"%+(%d) to maximum number of sentinels of purity": function(num) return { mod.NewFloat("ActiveSentinelOfPurityLimit", mod.TypeBase, num) } end,
		"if you have 3 primordial jewels, can summon up to (%d) additional golems? at a time": function(num) return { mod("ActiveGolemLimit", "BASE", num, { type = "MultiplierThreshold", var = "PrimordialItem", threshold = 3 }) } end,
		"golems regenerate (%d)%% of their maximum life per second": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("LifeRegenPercent", mod.TypeBase, num) }, mod.SkillType(data.SkillTypeGolem)) } end,
		"summoned golems regenerate (%d)%% of their life per second": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("LifeRegenPercent", mod.TypeBase, num) }, mod.SkillType(data.SkillTypeGolem)) } end,
		"golems summoned in the past 8 seconds deal (%d+)%% increased damage": function(num) return { mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "SummonedGolemInPast8Sec" }) }, mod.SkillType(data.SkillTypeGolem)) } end,
		"gain onslaught for 10 seconds when you cast socketed golem skill": function(num) return { flag("Condition:Onslaught", .Tag(mod.Condition("SummonedGolemInPast10Sec"))) } end,
		"s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite": { mod("MinionModifier", "LIST", { mod = mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100) }, mod.SkillName("Summon Raging Spirit")) },
		"raised zombies have avatar of fire": { mod("MinionModifier", "LIST", { mod = mod("Keystone", "LIST", "Avatar of Fire") }, mod.SkillName("Raise Zombie")) },
		"raised zombies take ([%d%.]+)%% of their maximum life per second as fire damage": function(num) return { mod("MinionModifier", "LIST", { mod = mod("FireDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }) }, mod.SkillName("Raise Zombie")) } end,
		"summoned skeletons have avatar of fire": { mod("MinionModifier", "LIST", { mod = mod("Keystone", "LIST", "Avatar of Fire") }, mod.SkillName("Summon Skeleton")) },
		"summoned skeletons take ([%d%.]+)%% of their maximum life per second as fire damage": function(num) return { mod("MinionModifier", "LIST", { mod = mod("FireDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }) }, mod.SkillName("Summon Skeleton")) } end,
		"summoned skeletons have (%d+)%% chance to wither enemies for (%d+) seconds on hit": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFlag("Condition:CanWither", true) }, mod.SkillName("Summon Skeleton") ) },
		"summoned skeletons have (%d+)%% of physical damage converted to chaos damage": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, num) }, mod.SkillName("Summon Skeleton")) } end,
		"minions convert (%d+)%% of physical damage to fire damage per red socket": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageConvertToFire", mod.TypeBase, num) }, mod.Multiplier("RedSocketIn{SlotName}").Base(0)) } end,
		"minions convert (%d+)%% of physical damage to cold damage per green socket": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageConvertToCold", mod.TypeBase, num) }, mod.Multiplier("GreenSocketIn{SlotName}").Base(0)) } end,
		"minions convert (%d+)%% of physical damage to lightning damage per blue socket": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageConvertToLightning", mod.TypeBase, num) }, mod.Multiplier("BlueSocketIn{SlotName}").Base(0)) } end,
		"minions convert (%d+)%% of physical damage to chaos damage per white socket": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, num) }, mod.Multiplier("WhiteSocketIn{SlotName}").Base(0)) } end,
		"minions have a (%d+)%% chance to impale on hit with attacks": function(num) return { mod("MinionModifier", "LIST", { mod = mod("ImpaleChance", "BASE", num ) }) } end,
		"minions from herald skills deal (%d+)%% more damage": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Damage", mod.TypeMore, num) }, mod.SkillType(data.SkillTypeHerald)) } end,
		"minions have (%d+)%% increased movement speed for each herald affecting you": function(num) return { mod("MinionModifier", "LIST", { mod = mod("MovementSpeed", "INC", num, { type = "Multiplier", var = "Herald", actor = "parent" }) }) } end,
		"minions deal (%d+)%% increased damage while you are affected by a herald": function(num) return { mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", num, { type = "ActorCondition", actor = "parent", var = "AffectedByHerald" }) }) } end,
		"minions have (%d+)%% increased attack and cast speed while you are affected by a herald": function(num) return { mod("MinionModifier", "LIST", { mod = mod("Speed", "INC", num, { type = "ActorCondition", actor = "parent", var = "AffectedByHerald" }) }) } end,
		"summoned skeleton warriors deal triple damage with this weapon if you've hit with this weapon recently": {
			mod("Dummy", "DUMMY", 1, .Tag(mod.Condition("HitRecentlyWithWeapon"))), // TODO Make the Configuration option appear
			mod("MinionModifier", "LIST", { mod = mod("TripleDamageChance", "BASE", 100, { type = "ActorCondition", actor = "parent", var = "HitRecentlyWithWeapon" }) }, mod.SkillName("Summon Skeleton")),
		},
		"summoned skeleton warriors wield a copy of this weapon while in your main hand": { }, // TODO just make the mod blue, handled in CalcSetup
		"each summoned phantasm grants you phantasmal might": { mod.NewFlag("Condition:PhantasmalMight", true) },
		"minions have (%d+)%% increased critical strike chance per maximum power charge you have": function(num) return { mod("MinionModifier", "LIST", { mod = mod("CritChance", "INC", num, { type = "Multiplier", actor = "parent", var = "PowerChargeMax" }) }) } end,
		"minions can hear the whispers for 5 seconds after they deal a critical strike": function() return {
			mod("MinionModifier", "LIST", { mod = mod("Damage", "INC", 50, { type = "Condition", neg = true, var = "NeverCrit" }) }),
			mod("MinionModifier", "LIST", { mod = mod("Speed", "INC", 50, nil, mod.MFlagAttack, { type = "Condition", neg = true, var = "NeverCrit" }) }),
			mod("MinionModifier", "LIST", { mod = mod("ChaosDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = 20 }, { type = "Condition", neg = true, var = "NeverCrit" }) }),
		} end,
		"chaos damage t?a?k?e?n? ?does not bypass minions' energy shield": function(num, _, div) return { mod("MinionModifier", "LIST", { mod = mod.NewFlag("ChaosNotBypassEnergyShield", true) }) } end,
		"while minions have energy shield, their hits ignore monster elemental resistances": function(num) return { mod("MinionModifier", "LIST", { mod = flag("IgnoreElementalResistances", { type = "StatThreshold", stat = "EnergyShield", threshold = 1 }) }) } end,
		// TODO Projectiles
		"skills chain %+(%d) times": function(num) return { mod.NewFloat("ChainCountMax", mod.TypeBase, num) } end,
		"skills chain an additional time while at maximum frenzy charges": { mod("ChainCountMax", "BASE", 1, { type = "StatThreshold", stat = "FrenzyCharges", thresholdStat = "FrenzyChargesMax" }) },
		"attacks chain an additional time when in main hand": { mod("ChainCountMax", "BASE", 1, nil, mod.MFlagAttack, { type = "SlotNumber", num = 1 }) },
		"projectiles chain %+(%d) times while you have phasing": function(num) return { mod("ChainCountMax", "BASE", num, nil, mod.MFlagProjectile, .Tag(mod.Condition("Phasing"))) } end,
		"adds an additional arrow": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagAttack) },
		"(%d+) additional arrows": function(num) return { mod("ProjectileCount", "BASE", num, nil, mod.MFlagAttack) } end,
		"bow attacks fire an additional arrow": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagBow) },
		"bow attacks fire (%d+) additional arrows": function(num) return { mod("ProjectileCount", "BASE", num, nil, mod.MFlagBow) } end,
		"bow attacks fire (%d+) additional arrows if you haven't cast dash recently": function(num) return { mod("ProjectileCount", "BASE", num, nil, mod.MFlagBow, .Tag(mod.Condition("CastDashRecently").Neg(true))) } end,
		"wand attacks fire an additional projectile": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagWand) },
		"skills fire an additional projectile": { mod.NewFloat("ProjectileCount", mod.TypeBase, 1) },
		"spells [hf][ai][vr]e an additional projectile": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagSpell) },
		"attacks fire an additional projectile": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagAttack) },
		"attacks have an additional projectile when in off hand": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagAttack, { type = "SlotNumber", num = 2 }) },
		"projectiles pierce an additional target": { mod.NewFloat("PierceCount", mod.TypeBase, 1) },
		"projectiles pierce (%d+) targets?": function(num) return { mod.NewFloat("PierceCount", mod.TypeBase, num) } end,
		"projectiles pierce (%d+) additional targets?": function(num) return { mod.NewFloat("PierceCount", mod.TypeBase, num) } end,
		"projectiles pierce (%d+) additional targets while you have phasing": function(num) return { mod("PierceCount", "BASE", num, .Tag(mod.Condition("Phasing"))) } end,
		"projectiles pierce all targets while you have phasing": { flag("PierceAllTargets", .Tag(mod.Condition("Phasing"))) },
		"arrows pierce an additional target": { mod("PierceCount", "BASE", 1, nil, mod.MFlagAttack) },
		"arrows pierce one target": { mod("PierceCount", "BASE", 1, nil, mod.MFlagAttack) },
		"arrows pierce (%d+) targets?": function(num) return { mod("PierceCount", "BASE", num, nil, mod.MFlagAttack) } end,
		"always pierce with arrows": { flag("PierceAllTargets", nil, mod.MFlagAttack) },
		"arrows always pierce": { flag("PierceAllTargets", nil, mod.MFlagAttack) },
		"arrows pierce all targets": { flag("PierceAllTargets", nil, mod.MFlagAttack) },
		"arrows that pierce cause bleeding": { mod("BleedChance", "BASE", 100, nil, bor(mod.MFlagAttack, mod.MFlagProjectile), { type = "StatThreshold", stat = "PierceCount", threshold = 1 }) },
		"arrows that pierce have (%d+)%% chance to cause bleeding": function(num) return { mod("BleedChance", "BASE", num, nil, bor(mod.MFlagAttack, mod.MFlagProjectile), { type = "StatThreshold", stat = "PierceCount", threshold = 1 }) } end,
		"arrows that pierce deal (%d+)%% increased damage": function(num) return { mod("Damage", "INC", num, nil, bor(mod.MFlagAttack, mod.MFlagProjectile), { type = "StatThreshold", stat = "PierceCount", threshold = 1 }) } end,
		"projectiles gain (%d+)%% of non%-chaos damage as extra chaos damage per chain": function(num) return { mod("NonChaosDamageGainAsChaos", "BASE", num, nil, mod.MFlagProjectile, { type = "PerStat", stat = "Chain" }) } end,
		"projectiles that have chained gain (%d+)%% of non%-chaos damage as extra chaos damage": function(num) return { mod("NonChaosDamageGainAsChaos", "BASE", num, nil, mod.MFlagProjectile, { type = "StatThreshold", stat = "Chain", threshold = 1 }) } end,
		"left ring slot: projectiles from spells cannot chain": { flag("CannotChain", nil, bor(mod.MFlagSpell, mod.MFlagProjectile), { type = "SlotNumber", num = 1 }) },
		"left ring slot: projectiles from spells fork": { flag("ForkOnce", nil, bor(mod.MFlagSpell, mod.MFlagProjectile), { type = "SlotNumber", num = 1 }), mod("ForkCountMax", "BASE", 1, nil, bor(mod.MFlagSpell, mod.MFlagProjectile), { type = "SlotNumber", num = 1 }) },
		"left ring slot: your chilling skitterbot's aura applies socketed h?e?x? ?curse instead": { flag("SkitterbotsCannotChill", { type = "SlotNumber", num = 1 }) },
		"right ring slot: projectiles from spells chain %+1 times": { mod("ChainCountMax", "BASE", 1, nil, bor(mod.MFlagSpell, mod.MFlagProjectile), { type = "SlotNumber", num = 2 }) },
		"right ring slot: projectiles from spells cannot fork": { flag("CannotFork", nil, bor(mod.MFlagSpell, mod.MFlagProjectile), { type = "SlotNumber", num = 2 }) },
		"right ring slot: your shocking skitterbot's aura applies socketed h?e?x? ?curse instead": { flag("SkitterbotsCannotShock", { type = "SlotNumber", num = 2 }) },
		"projectiles from spells cannot pierce": { flag("CannotPierce", nil, mod.MFlagSpell) },
		"projectiles fork": { flag("ForkOnce", nil, mod.MFlagProjectile), mod("ForkCountMax", "BASE", 1, nil, mod.MFlagProjectile) },
		"(%d+)%% increased critical strike chance with arrows that fork": function(num) return {
			mod("CritChance", "INC", num, nil, mod.MFlagBow, { type = "StatThreshold", stat = "ForkRemaining", threshold = 1 }, { type = "StatThreshold", stat = "PierceCount", threshold = 0, upper = true }) }
		end,
		"arrows that pierce have %+(%d+)%% to critical strike multiplier": function (num) return {
			mod("CritMultiplier", "BASE", num, nil, mod.MFlagBow, { type = "StatThreshold", stat = "PierceCount", threshold = 1 }) } end,
		"arrows pierce all targets after forking": { flag("PierceAllTargets", nil, mod.MFlagBow, { type = "StatThreshold", stat = "ForkedCount", threshold = 1 }) },
		"modifiers to number of projectiles instead apply to the number of targets projectiles split towards": { mod.NewFlag("NoAdditionalProjectiles", true) },
		"attack skills fire an additional projectile while wielding a claw or dagger": { mod("ProjectileCount", "BASE", 1, nil, mod.MFlagAttack, { type = "ModFlagOr", modFlags = bor(mod.MFlagClaw, mod.MFlagDagger) }) },
		"skills fire (%d+) additional projectiles for 4 seconds after you consume a total of 12 steel shards": function(num) return { mod("ProjectileCount", "BASE", num, .Tag(mod.Condition("Consumed12SteelShardsRecently"))) } end,
		"non%-projectile chaining lightning skills chain %+(%d+) times": function (num) return { mod("ChainCountMax", "BASE", num, { type = "SkillType", skillType = SkillType.Projectile, neg = true }, mod.SkillType(data.SkillTypeChains), mod.SkillType(data.SkillTypeLightning)) } end,
		"arrows gain damage as they travel farther, dealing up to (%d+)%% increased damage with hits to targets": function(num) return { mod("Damage", "INC", num, nil, bor(mod.MFlagBow, mod.MFlagHit), { type = "DistanceRamp", ramp = {{35,0},{70,1}} }) } end,
		"arrows gain critical strike chance as they travel farther, up to (%d+)%% increased critical strike chance": function(num) return { mod("CritChance", "INC", num, nil, mod.MFlagBow, { type = "DistanceRamp", ramp = {{35,0},{70,1}} }) } end,
		// TODO Leech/Gain on Hit
		"cannot leech life": { mod.NewFlag("CannotLeechLife", true) },
		"cannot leech mana": { mod.NewFlag("CannotLeechMana", true) },
		"cannot leech when on low life": { flag("CannotLeechLife", .Tag(mod.Condition("LowLife"))), flag("CannotLeechMana", .Tag(mod.Condition("LowLife"))) },
		"cannot leech life from critical strikes": { flag("CannotLeechLife", .Tag(mod.Condition("CriticalStrike"))) },
		"leech applies instantly on critical strike": { flag("InstantLifeLeech", .Tag(mod.Condition("CriticalStrike"))), flag("InstantManaLeech", .Tag(mod.Condition("CriticalStrike"))) },
		"gain life and mana from leech instantly on critical strike": { flag("InstantLifeLeech", .Tag(mod.Condition("CriticalStrike"))), flag("InstantManaLeech", .Tag(mod.Condition("CriticalStrike"))) },
		"leech applies instantly during flask effect": { flag("InstantLifeLeech", .Tag(mod.Condition("UsingFlask"))), flag("InstantManaLeech", .Tag(mod.Condition("UsingFlask"))) },
		"gain life and mana from leech instantly during flask effect": { flag("InstantLifeLeech", .Tag(mod.Condition("UsingFlask"))), flag("InstantManaLeech", .Tag(mod.Condition("UsingFlask"))) },
		"life and mana leech from critical strikes are instant": { flag("InstantLifeLeech", .Tag(mod.Condition("CriticalStrike"))), flag("InstantManaLeech", .Tag(mod.Condition("CriticalStrike"))) },
		"gain life and mana from leech instantly during effect": { flag("InstantLifeLeech", .Tag(mod.Condition("UsingFlask"))), flag("InstantManaLeech", .Tag(mod.Condition("UsingFlask"))) },
		"with 5 corrupted items equipped: life leech recovers based on your chaos damage instead": { flag("LifeLeechBasedOnChaosDamage", { type = "MultiplierThreshold", var = "CorruptedItem", threshold = 5 }) },
		"you have vaal pact if you've dealt a critical strike recently": { mod("Keystone", "LIST", "Vaal Pact", .Tag(mod.Condition("CritRecently"))) },
		"gain (%d+) energy shield for each enemy you hit which is affected by a spider's web": function(num) return { mod("EnergyShieldOnHit", "BASE", num, { type = "MultiplierThreshold", actor = "enemy", var = "Spider's WebStack", threshold = 1 }) } end,
		"(%d+) life gained for each enemy hit if you have used a vaal skill recently": function(num) return { mod("LifeOnHit", "BASE", num, .Tag(mod.Condition("UsedVaalSkillRecently"))) } end,
		"(%d+) life gained for each cursed enemy hit by your attacks": function(num) return { mod("LifeOnHit", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed"})} end,
		"(%d+) mana gained for each cursed enemy hit by your attacks": function(num) return { mod("ManaOnHit", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Cursed"})} end,
		// TODO Defences
		"chaos damage t?a?k?e?n? ?does not bypass energy shield": { mod.NewFlag("ChaosNotBypassEnergyShield", true) },
		"(%d+)%% of chaos damage t?a?k?e?n? ?does not bypass energy shield": function(num) return { mod("ChaosEnergyShieldBypass", "BASE", -num) } end,
		"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life": { flag("ChaosNotBypassEnergyShield", { type = "Condition", varList = { "LowLife" }, neg = true }) },
		"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana": { flag("ChaosNotBypassEnergyShield", { type = "Condition", varList = { "LowLife", "LowMana" }, neg = true }) },
		"chaos damage is taken from mana before life": function() return { mod.NewFloat("ChaosDamageTakenFromManaBeforeLife", mod.TypeBase, 100) } end,
		"cannot evade enemy attacks": { mod.NewFlag("CannotEvade", true) },
		"cannot block": { mod.NewFlag("CannotBlockAttacks", true), mod.NewFlag("CannotBlockSpells", true) },
		"cannot block while you have no energy shield": { flag("CannotBlockAttacks", .Tag(mod.Condition("HaveEnergyShield").Neg(true))), flag("CannotBlockSpells", .Tag(mod.Condition("HaveEnergyShield").Neg(true))) },
		"cannot block attacks": { mod.NewFlag("CannotBlockAttacks", true) },
		"cannot block spells": { mod.NewFlag("CannotBlockSpells", true) },
		"damage from blocked hits cannot bypass energy shield": { flag("BlockedDamageDoesntBypassES", .Tag(mod.Condition("EVBypass").Neg(true))) },
		"damage from unblocked hits always bypasses energy shield": { flag("UnblockedDamageDoesBypassES", .Tag(mod.Condition("EVBypass").Neg(true))) },
		"recover (%d+) life when you block": function(num) return { mod.NewFloat("LifeOnBlock", mod.TypeBase, num) } end,
		"recover (%d+) energy shield when you block spell damage": function(num) return { mod.NewFloat("EnergyShieldOnSpellBlock", mod.TypeBase, num) } end,
		"recover (%d+)%% of life when you block": function(num) return { mod("LifeOnBlock", "BASE", 1,  { type = "PerStat", stat = "Life", div = 100 / num }) } end,
		"recover (%d+)%% of life when you block attack damage while wielding a staff": function(num) return { mod("LifeOnBlock", "BASE", 1,  { type = "PerStat", stat = "Life", div = 100 / num }, .Tag(mod.Condition("UsingStaff"))) } end,
		"recover (%d+)%% of your maximum mana when you block": function(num) return { mod("ManaOnBlock", "BASE", 1,  { type = "PerStat", stat = "Mana", div = 100 / num }) } end,
		"recover (%d+)%% of energy shield when you block": function(num) return { mod("EnergyShieldOnBlock", "BASE", 1,  { type = "PerStat", stat = "EnergyShield", div = 100 / num }) } end,
		"recover (%d+)%% of energy shield when you block spell damage while wielding a staff": function(num) return { mod("EnergyShieldOnSpellBlock", "BASE", 1,  { type = "PerStat", stat = "EnergyShield", div = 100 / num }, .Tag(mod.Condition("UsingStaff"))) } end,
		"replenishes energy shield by (%d+)%% of armour when you block": function(num) return { mod("EnergyShieldOnBlock", "BASE", 1,  { type = "PerStat", stat = "Armour", div = 100 / num }) } end,
		"cannot leech or regenerate mana": { mod.NewFlag("NoManaRegen", true), mod.NewFlag("CannotLeechMana", true) },
		["right ring slot: you cannot regenerate mana" ] = { flag("NoManaRegen", { type = "SlotNumber", num = 2 }) },
		"y?o?u? ?cannot recharge energy shield": { mod.NewFlag("NoEnergyShieldRecharge", true) },
		["you cannot regenerate energy shield" ] = { mod.NewFlag("NoEnergyShieldRegen", true) },
		"cannot recharge or regenerate energy shield": { mod.NewFlag("NoEnergyShieldRecharge", true), mod.NewFlag("NoEnergyShieldRegen", true) },
		"left ring slot: you cannot recharge or regenerate energy shield": { flag("NoEnergyShieldRecharge", { type = "SlotNumber", num = 1 }), flag("NoEnergyShieldRegen", { type = "SlotNumber", num = 1 }) },
		"cannot gain energy shield": { mod.NewFlag("NoEnergyShieldRegen", true), mod.NewFlag("NoEnergyShieldRecharge", true), mod.NewFlag("CannotLeechEnergyShield", true) },
		"you lose (%d+)%% of energy shield per second": function(num) return { mod("EnergyShieldDegen", "BASE", 1, { type = "PercentStat", stat = "EnergyShield", percent = num }) } end,
		"lose (%d+)%% of energy shield per second": function(num) return { mod("EnergyShieldDegen", "BASE", 1, { type = "PercentStat", stat = "EnergyShield", percent = num }) } end,
		"lose (%d+)%% of life per second if you have been hit recently": function(num) return { mod("LifeDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }, .Tag(mod.Condition("BeenHitRecently"))) } end,
		"you have no armour or energy shield": {
			mod("Armour", "MORE", -100),
			mod("EnergyShield", "MORE", -100),
		},
		"you have no armour or maximum energy shield": {
			mod("Armour", "MORE", -100),
			mod("EnergyShield", "MORE", -100),
		},
		"defences are zero": {
			mod("Armour", "MORE", -100),
			mod("EnergyShield", "MORE", -100),
			mod("Evasion", "MORE", -100),
			mod("Ward", "MORE", -100),
		},
		"you have no intelligence": {
			mod("Int", "MORE", -100),
		},
		"elemental resistances are zero": {
			mod.NewFloat("FireResist", mod.TypeOverride, 0),
			mod.NewFloat("ColdResist", mod.TypeOverride, 0),
			mod.NewFloat("LightningResist", mod.TypeOverride, 0),
		},
		"chaos resistance is zero": {
			mod.NewFloat("ChaosResist", mod.TypeOverride, 0),
		},
		"your maximum resistances are (%d+)%%": function(num) return {
			mod.NewFloat("FireResistMax", mod.TypeOverride, num),
			mod.NewFloat("ColdResistMax", mod.TypeOverride, num),
			mod.NewFloat("LightningResistMax", mod.TypeOverride, num),
			mod.NewFloat("ChaosResistMax", mod.TypeOverride, num),
		} end,
		"fire resistance is (%d+)%%": function(num) return { mod.NewFloat("FireResist", mod.TypeOverride, num) } end,
		"cold resistance is (%d+)%%": function(num) return { mod.NewFloat("ColdResist", mod.TypeOverride, num) } end,
		"lightning resistance is (%d+)%%": function(num) return { mod.NewFloat("LightningResist", mod.TypeOverride, num) } end,
		"elemental resistances are capped by your highest maximum elemental resistance instead": { mod.NewFlag("ElementalResistMaxIsHighestResistMax", true) },
		"chaos resistance is doubled": { mod.NewFloat("ChaosResist", mod.TypeMore, 100) },
		"nearby enemies have (%d+)%% increased fire and cold resistances": function(num) return {
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("FireResist", mod.TypeIncrease, num) }),
			mod("EnemyModifier", "LIST", { mod = mod.NewFloat("ColdResist", mod.TypeIncrease, num) }),
		} end,
		"nearby enemies are blinded while physical aegis is not depleted": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Blinded", true) }, .Tag(mod.Condition("PhysicalAegisDepleted").Neg(true))) },
		"armour is increased by uncapped fire resistance": { mod("Armour", "INC", 1, { type = "PerStat", stat = "FireResistTotal", div = 1 }) },
		"armour is increased by overcapped fire resistance": { mod("Armour", "INC", 1, { type = "PerStat", stat = "FireResistOverCap", div = 1 }) },
		"evasion rating is increased by uncapped cold resistance": { mod("Evasion", "INC", 1, { type = "PerStat", stat = "ColdResistTotal", div = 1 }) },
		"evasion rating is increased by overcapped cold resistance": { mod("Evasion", "INC", 1, { type = "PerStat", stat = "ColdResistOverCap", div = 1 }) },
		"reflects (%d+) physical damage to melee attackers": { },
		"ignore all movement penalties from armour": { mod.NewFlag("Condition:IgnoreMovementPenalties", true) },
		"gain armour equal to your reserved mana": { mod("Armour", "BASE", 1, { type = "PerStat", stat = "ManaReserved", div = 1 }) },
		"(%d+)%% increased armour per (%d+) reserved mana": function(num, _, mana) return { mod("Armour", "INC", num, { type = "PerStat", stat = "ManaReserved", div = tonumber(mana) }) } end,
		"cannot be stunned": { mod.NewFloat("AvoidStun", mod.TypeBase, 100) },
		"cannot be stunned if you haven't been hit recently": { mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("BeenHitRecently").Neg(true))) },
		"cannot be stunned if you have at least (%d+) crab barriers": function(num) return { mod("AvoidStun", "BASE", 100, { type = "StatThreshold", stat = "CrabBarriers", threshold = num }) } end,
		"cannot be blinded": { mod.NewFloat("AvoidBlind", mod.TypeBase, 100) },
		"cannot be shocked": { mod.NewFloat("AvoidShock", mod.TypeBase, 100) },
		"immune to shock": { mod.NewFloat("AvoidShock", mod.TypeBase, 100) },
		"cannot be frozen": { mod.NewFloat("AvoidFreeze", mod.TypeBase, 100) },
		"immune to freeze": { mod.NewFloat("AvoidFreeze", mod.TypeBase, 100) },
		"cannot be chilled": { mod.NewFloat("AvoidChill", mod.TypeBase, 100) },
		"immune to chill": { mod.NewFloat("AvoidChill", mod.TypeBase, 100) },
		"cannot be ignited": { mod.NewFloat("AvoidIgnite", mod.TypeBase, 100) },
		"immune to ignite": { mod.NewFloat("AvoidIgnite", mod.TypeBase, 100) },
		"cannot be ignited while at maximum endurance charges": { mod("AvoidIgnite", "BASE", 100, {type = "StatThreshold", stat = "EnduranceCharges", thresholdStat = "EnduranceChargesMax" }) },
		"cannot be chilled while at maximum frenzy charges": { mod("AvoidChill", "BASE", 100, {type = "StatThreshold", stat = "FrenzyCharges", thresholdStat = "FrenzyChargesMax" }) },
		"cannot be shocked while at maximum power charges": { mod("AvoidShock", "BASE", 100, {type = "StatThreshold", stat = "PowerCharges", thresholdStat = "PowerChargesMax" }) },
		"you cannot be shocked while at maximum endurance charges": { mod("AvoidShock", "BASE", 100, { type = "StatThreshold", stat = "EnduranceCharges", thresholdStat = "EnduranceChargesMax" }) },
		"you cannot be shocked while chilled": { mod("AvoidShock", "BASE", 100, .Tag(mod.Condition("Chilled"))) },
		"cannot be shocked while chilled": { mod("AvoidShock", "BASE", 100, .Tag(mod.Condition("Chilled"))) },
		"cannot be shocked if intelligence is higher than strength": { mod("AvoidShock", "BASE", 100, .Tag(mod.Condition("IntHigherThanStr"))) },
		"cannot be frozen if dexterity is higher than intelligence": { mod("AvoidFreeze", "BASE", 100, .Tag(mod.Condition("DexHigherThanInt"))) },
		"cannot be frozen if energy shield recharge has started recently": { mod("AvoidFreeze", "BASE", 100, .Tag(mod.Condition("EnergyShieldRechargeRecently"))) },
		"cannot be ignited if strength is higher than dexterity": { mod("AvoidIgnite", "BASE", 100, .Tag(mod.Condition("StrHigherThanDex"))) },
		"cannot be chilled while burning": { mod("AvoidChill", "BASE", 100, .Tag(mod.Condition("Burning"))) },
		"cannot be chilled while you have onslaught": { mod("AvoidChill", "BASE", 100, .Tag(mod.Condition("Onslaught"))) },
		"cannot be inflicted with bleeding": { mod.NewFloat("AvoidBleed", mod.TypeBase, 100) },
		"bleeding cannot be inflicted on you": { mod.NewFloat("AvoidBleed", mod.TypeBase, 100) },
		"you are immune to bleeding": { mod.NewFloat("AvoidBleed", mod.TypeBase, 100) },
		"immune to poison": { mod.NewFloat("AvoidPoison", mod.TypeBase, 100) },
		"immunity to shock during flask effect": { mod("AvoidShock", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"immunity to freeze and chill during flask effect": {
			mod("AvoidFreeze", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
			mod("AvoidChill", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
		},
		"immune to freeze and chill while ignited": {
			mod("AvoidFreeze", "BASE", 100, .Tag(mod.Condition("Ignited"))),
			mod("AvoidChill", "BASE", 100, .Tag(mod.Condition("Ignited"))),
		},
		"immunity to ignite during flask effect": { mod("AvoidIgnite", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"immunity to bleeding during flask effect": { mod("AvoidBleed", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"immune to poison during flask effect": { mod("AvoidPoison", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"immune to curses during flask effect": { mod("AvoidCurse", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"immune to freeze, chill, curses and stuns during flask effect": {
			mod("AvoidFreeze", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
			mod("AvoidChill", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
			mod("AvoidCurse", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
			mod("AvoidStun", "BASE", 100, .Tag(mod.Condition("UsingFlask"))),
		},
		"unaffected by curses": { mod("CurseEffectOnSelf", "MORE", -100) },
		"unaffected by curses while affected by zealotry": { mod("CurseEffectOnSelf", "MORE", -100, .Tag(mod.Condition("AffectedByZealotry"))) },
		"immune to curses while you have at least (%d+) rage": function(num) return { mod("AvoidCurse", "BASE", 100, { type = "MultiplierThreshold", var = "Rage", threshold = num }) } end,
		"the effect of chill on you is reversed": { mod.NewFlag("SelfChillEffectIsReversed", true) },
		"your movement speed is (%d+)%% of its base value": function(num) return { mod("MovementSpeed", "OVERRIDE", num / 100) } end,
		"armour also applies to lightning damage taken from hits": { mod.NewFlag("ArmourAppliesToLightningDamageTaken", true) },
		"lightning resistance does not affect lightning damage taken": { mod.NewFlag("SelfIgnoreLightningResistance", true) },
		"(%d+)%% increased maximum life and reduced fire resistance": function(num) return {
			mod.NewFloat("Life", mod.TypeIncrease, num),
			mod("FireResist", "INC", -num),
		} end,
		"(%d+)%% increased maximum mana and reduced cold resistance": function(num) return {
			mod.NewFloat("Mana", mod.TypeIncrease, num),
			mod("ColdResist", "INC", -num),
		} end,
		"(%d+)%% increased global maximum energy shield and reduced lightning resistance": function(num) return {
			mod("EnergyShield", "INC", num, mod.Global()),
			mod("LightningResist", "INC", -num),
		} end,
		"cannot be ignited while on low life": { mod("AvoidIgnite", "BASE", 100, .Tag(mod.Condition("LowLife"))) },
		"ward does not break during flask effect": { flag("WardNotBreak", .Tag(mod.Condition("UsingFlask"))) },
		// TODO Knockback
		"cannot knock enemies back": { mod.NewFlag("CannotKnockback", true) },
		"knocks back enemies if you get a critical strike with a staff": { mod("EnemyKnockbackChance", "BASE", 100, nil, mod.MFlagStaff, .Tag(mod.Condition("CriticalStrike"))) },
		"knocks back enemies if you get a critical strike with a bow": { mod("EnemyKnockbackChance", "BASE", 100, nil, mod.MFlagBow, .Tag(mod.Condition("CriticalStrike"))) },
		"bow knockback at close range": { mod("EnemyKnockbackChance", "BASE", 100, nil, mod.MFlagBow, .Tag(mod.Condition("AtCloseRange"))) },
		"adds knockback during flask effect": { mod("EnemyKnockbackChance", "BASE", 100, .Tag(mod.Condition("UsingFlask"))) },
		"adds knockback to melee attacks during flask effect": { mod("EnemyKnockbackChance", "BASE", 100, nil, mod.MFlagMelee, .Tag(mod.Condition("UsingFlask"))) },
		"knockback direction is reversed": { mod("EnemyKnockbackDistance", "MORE", -200) },
		// TODO Culling
		"culling strike": { mod("CullPercent", "MAX", 10) },
		"culling strike during flask effect": { mod("CullPercent", "MAX", 10, .Tag(mod.Condition("UsingFlask"))) },
		"hits with this weapon have culling strike against bleeding enemies": { mod("CullPercent", "MAX", 10, { type = "ActorCondition", actor = "enemy", var = "Bleeding" }) },
		"you have culling strike against cursed enemies": { mod("CullPercent", "MAX", 10, { type = "ActorCondition", actor = "enemy", var = "Cursed" }) },
		"critical strikes have culling strike": { mod("CriticalCullPercent", "MAX", 10) },
		"your critical strikes have culling strike": { mod("CriticalCullPercent", "MAX", 10) },
		"your spells have culling strike": { mod("CullPercent", "MAX", 10, nil, mod.MFlagSpell) },
		"culling strike against burning enemies": { mod("CullPercent", "MAX", 10, { type = "ActorCondition", actor = "enemy", var = "Burning" }) },
		"culling strike against marked enemy": { mod("CullPercent", "MAX", 10, { type = "ActorCondition", actor = "enemy", var = "Marked" }) },
		// TODO Intimidate
		"permanently intimidate enemies on block": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Intimidated", true) }, .Tag(mod.Condition("BlockedRecently")) )},
		"with a murderous eye jewel socketed, intimidate enemies for (%d) seconds on hit with attacks": function(jewelName, num) return  { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Intimidated", true)}, .Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}"))) } end,
		"enemies taunted by your warcries are intimidated": { mod("EnemyModifier", "LIST", { mod = flag("Condition:Intimidated", .Tag(mod.Condition("Taunted"))) }, .Tag(mod.Condition("UsedWarcryRecently"))) },
		"intimidate enemies for (%d) seconds on block while holding a shield": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Intimidated", true) }, .Tag(mod.Condition("BlockedRecently")), .Tag(mod.Condition("UsingShield")) )},
		// TODO Flasks
		"flasks do not apply to you": { mod.NewFlag("FlasksDoNotApplyToPlayer", true) },
		"flasks apply to your zombies and spectres": { flag("FlasksApplyToMinion", { type = "SkillName", skillNameList = { "Raise Zombie", "Raise Spectre" } }) },
		"flasks apply to your raised zombies and spectres": { flag("FlasksApplyToMinion", { type = "SkillName", skillNameList = { "Raise Zombie", "Raise Spectre" } }) },
		"your minions use your flasks when summoned": { mod.NewFlag("FlasksApplyToMinion", true) },
		"recover an additional (%d+)%% of flask's life recovery amount over 10 seconds if used while not on full life": function(num) return {
			mod.NewFloat("FlaskAdditionalLifeRecovery", mod.TypeBase, num)
		} end,
		"creates a smoke cloud on use": { },
		"creates chilled ground on use": { },
		"creates consecrated ground on use": { },
		"removes bleeding on use": { },
		"removes burning on use": { },
		"removes curses on use": { },
		"removes freeze and chill on use": { },
		"removes poison on use": { },
		"removes shock on use": { },
		"gain unholy might during flask effect": { flag("Condition:UnholyMight", .Tag(mod.Condition("UsingFlask"))) },
		"zealot's oath during flask effect": { flag("ZealotsOath", .Tag(mod.Condition("UsingFlask"))) },
		"grants level (%d+) (.+) curse aura during flask effect": function(num, _, skill) return { mod("ExtraCurse", "LIST", { skillId = gemIdLookup[skill:gsub(" skill","")] or "Unknown", level = num }, .Tag(mod.Condition("UsingFlask"))) } end,
		"shocks nearby enemies during flask effect, causing (%d+)%% increased damage taken": function(num) return {
			mod("ShockOverride", "BASE", num, .Tag(mod.Condition("UsingFlask")) )
		} end,
		"during flask effect, (%d+)%% reduced damage taken of each element for which your uncapped elemental resistance is lowest": function(num) return {
			mod("LightningDamageTaken", "INC", -num, { type = "StatThreshold", stat = "LightningResistTotal", thresholdStat = "ColdResistTotal", upper = true }, { type = "StatThreshold", stat = "LightningResistTotal", thresholdStat = "FireResistTotal", upper = true }),
			mod("ColdDamageTaken", "INC", -num, { type = "StatThreshold", stat = "ColdResistTotal", thresholdStat = "LightningResistTotal", upper = true }, { type = "StatThreshold", stat = "ColdResistTotal", thresholdStat = "FireResistTotal", upper = true }),
			mod("FireDamageTaken", "INC", -num, { type = "StatThreshold", stat = "FireResistTotal", thresholdStat = "LightningResistTotal", upper = true }, { type = "StatThreshold", stat = "FireResistTotal", thresholdStat = "ColdResistTotal", upper = true }),
		} end,
		"during flask effect, damage penetrates (%d+)%% o?f? ?resistance of each element for which your uncapped elemental resistance is highest": function(num) return {
			mod("LightningPenetration", "BASE", num, { type = "StatThreshold", stat = "LightningResistTotal", thresholdStat = "ColdResistTotal" }, { type = "StatThreshold", stat = "LightningResistTotal", thresholdStat = "FireResistTotal" }),
			mod("ColdPenetration", "BASE", num, { type = "StatThreshold", stat = "ColdResistTotal", thresholdStat = "LightningResistTotal" }, { type = "StatThreshold", stat = "ColdResistTotal", thresholdStat = "FireResistTotal" }),
			mod("FirePenetration", "BASE", num, { type = "StatThreshold", stat = "FireResistTotal", thresholdStat = "LightningResistTotal" }, { type = "StatThreshold", stat = "FireResistTotal", thresholdStat = "ColdResistTotal" }),
		} end,
		"recover (%d+)%% of life when you kill an enemy during flask effect": function(num) return { mod("LifeOnKill", "BASE", 1, { type = "PerStat", stat = "Life", div = 100 / num }, .Tag(mod.Condition("UsingFlask"))) } end,
		"recover (%d+)%% of mana when you kill an enemy during flask effect": function(num) return { mod("ManaOnKill", "BASE", 1, { type = "PerStat", stat = "Mana", div = 100 / num }, .Tag(mod.Condition("UsingFlask"))) } end,
		"recover (%d+)%% of energy shield when you kill an enemy during flask effect": function(num) return { mod("EnergyShieldOnKill", "BASE", 1, { type = "PerStat", stat = "EnergyShield", div = 100 / num }, .Tag(mod.Condition("UsingFlask"))) } end,
		"(%d+)%% of maximum life taken as chaos damage per second": function(num) return { mod("ChaosDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }) } end,
		"your critical strikes do not deal extra damage during flask effect": { flag("NoCritMultiplier", .Tag(mod.Condition("UsingFlask"))) },
		"grants perfect agony during flask effect": { mod("Keystone", "LIST", "Perfect Agony", .Tag(mod.Condition("UsingFlask"))) },
		"grants eldritch battery during flask effect": { mod("Keystone", "LIST", "Eldritch Battery", .Tag(mod.Condition("UsingFlask"))) },
		"eldritch battery during flask effect": { mod("Keystone", "LIST", "Eldritch Battery", .Tag(mod.Condition("UsingFlask"))) },
		"chaos damage t?a?k?e?n? ?does not bypass energy shield during effect": { mod.NewFlag("ChaosNotBypassEnergyShield", true) },
		"your skills [ch][oa][sv][te] no mana c?o?s?t? ?during flask effect": { mod("ManaCost", "MORE", -100, .Tag(mod.Condition("UsingFlask"))) },
		"life recovery from flasks also applies to energy shield during flask effect": { flag("LifeFlaskAppliesToEnergyShield", .Tag(mod.Condition("UsingFlask"))) },
		"consecrated ground created during effect applies (%d+)%% increased damage taken to enemies": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTakenConsecratedGround", "INC", num, .Tag(mod.Condition("OnConsecratedGround"))) }, .Tag(mod.Condition("UsingFlask"))) } end,
		"gain alchemist's genius when you use a flask": {
			mod.NewFlag("Condition:CanHaveAlchemistGenius", true),
		},
		"(%d+)%% chance to gain alchemist's genius when you use a flask": {
			mod.NewFlag("Condition:CanHaveAlchemistGenius", true),
		},
		"(%d+)%% less flask charges gained from kills": function(num) return {
			mod("FlaskChargesGained", "MORE", -num,"from Kills")
		} end,
		"flasks gain (%d+) charges? every (%d+) seconds": function(num, _, div) return {
			mod("FlaskChargesGenerated", "BASE", num / div)
		} end,
		"flasks gain a charge every (%d+) seconds": function(_, div) return {
			mod("FlaskChargesGenerated", "BASE", 1 / div)
		} end,
		"utility flasks gain (%d+) charges? every (%d+) seconds": function(num, _, div) return {
			mod("UtilityFlaskChargesGenerated", "BASE", num / div)
		} end,
		"life flasks gain (%d+) charges? every (%d+) seconds": function(num, _, div) return {
			mod("LifeFlaskChargesGenerated", "BASE", num / div)
		} end,
		"mana flasks gain (%d+) charges? every (%d+) seconds": function(num, _, div) return {
			mod("ManaFlaskChargesGenerated", "BASE", num / div)
		} end,
		"flasks gain (%d+) charges? per empty flask slot every (%d+) seconds": function(num, _, div) return {
			mod("FlaskChargesGeneratedPerEmptyFlask", "BASE", num / div)
		} end,
		// TODO Jewels
		"passives in radius of ([%a%s']+) can be allocated without being connected to your tree": function(_, name) return {
			mod("JewelData", "LIST", { key = "impossibleEscapeKeystone", value = name }),
			mod("ImpossibleEscapeKeystones", "LIST", { key = name, value = true }),
		} end,
		"passives in radius can be allocated without being connected to your tree": { mod("JewelData", "LIST", { key = "intuitiveLeapLike", value = true }) },
		"affects passives in small ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 4 }) },
		"affects passives in medium ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 5 }) },
		"affects passives in large ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 6 }) },
		"affects passives in very large ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 7 }) },
		"affects passives in massive ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 8 }) },
		"only affects passives in small ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 4 }) },
		"only affects passives in medium ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 5 }) },
		"only affects passives in large ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 6 }) },
		"only affects passives in very large ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 7 }) },
		"only affects passives in massive ring": { mod("JewelData", "LIST", { key = "radiusIndex", value = 8 }) },
		"(%d+)%% increased elemental damage per grand spectrum": function(num) return {
			mod("ElementalDamage", "INC", num, mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		} end,
		"gain (%d+) armour per grand spectrum": function(num) return {
			mod("Armour", "BASE", num, mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		} end,
		"+(%d+)%% to all elemental resistances per grand spectrum": function(num) return {
			mod("ElementalResist", "BASE", num, mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1)
		} end,
		"gain (%d+) mana per grand spectrum": function(num) return {
			mod("Mana", "BASE", num, mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		} end,
		"(%d+)%% increased critical strike chance per grand spectrum": function(num) return {
			mod("CritChance", "INC", num, mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1)
		} end,
		"primordial": { mod.NewFloat("Multiplier:PrimordialItem", mod.TypeBase, 1) },
		"spectres have a base duration of (%d+) seconds": function(num) return { mod("SkillData", "LIST", { key = "duration", value = 6 }, mod.SkillName("Raise Spectre")) } end,
		"flasks applied to you have (%d+)%% increased effect": function(num) return { mod.NewFloat("FlaskEffect", mod.TypeIncrease, num) } end,
		"magic utility flasks applied to you have (%d+)%% increased effect": function(num) return { mod.NewFloat("MagicUtilityFlaskEffect", mod.TypeIncrease, num) } end,
		"flasks applied to you have (%d+)%% reduced effect": function(num) return { mod("FlaskEffect", "INC", -num) } end,
		"adds (%d+) passive skills": function(num) return { mod("JewelData", "LIST", { key = "clusterJewelNodeCount", value = num }) } end,
		"1 added passive skill is a jewel socket": { mod("JewelData", "LIST", { key = "clusterJewelSocketCount", value = 1 }) },
		"(%d+) added passive skills are jewel sockets": function(num) return { mod("JewelData", "LIST", { key = "clusterJewelSocketCount", value = num }) } end,
		"adds (%d+) jewel socket passive skills": function(num) return { mod("JewelData", "LIST", { key = "clusterJewelSocketCountOverride", value = num }) } end,
		"adds (%d+) small passive skills? which grants? nothing": function(num) return { mod("JewelData", "LIST", { key = "clusterJewelNothingnessCount", value = num }) } end,
		"added small passive skills grant nothing": { mod("JewelData", "LIST", { key = "clusterJewelSmallsAreNothingness", value = true }) },
		"added small passive skills have (%d+)%% increased effect": function(num) return { mod("JewelData", "LIST", { key = "clusterJewelIncEffect", value = num }) } end,
		"this jewel's socket has (%d+)%% increased effect per allocated passive skill between it and your class' starting location": function(num) return { mod("JewelData", "LIST", { key = "jewelIncEffectFromClassStart", value = num }) } end,
		// TODO Misc
		"leftmost (%d+) magic utility flasks constantly apply their flask effects to you": function(num) return { mod.NewFloat("ActiveMagicUtilityFlasks", mod.TypeBase, num) } end,
		"marauder: melee skills have (%d+)%% increased area of effect": function(num) return { mod("AreaOfEffect", "INC", num, .Tag(mod.Condition("ConnectedToMarauderStart")), mod.SkillType(data.SkillTypeMelee)) } end,
		"intelligence provides no inherent bonus to energy shield": { mod.NewFlag("NoIntBonusToES", true) },
		"intelligence is added to accuracy rating with wands": { mod("Accuracy", "BASE", 1, nil, mod.MFlagWand, { type = "PerStat", stat = "Int" } ) },
		"dexterity's accuracy bonus instead grants %+(%d+) to accuracy rating per dexterity": function(num) return { mod("DexAccBonusOverride", "OVERRIDE", num ) } end,
		"cannot recover energy shield to above armour": { mod.NewFlag("ArmourESRecoveryCap", true) },
		"cannot recover energy shield to above evasion rating": { mod.NewFlag("EvasionESRecoveryCap", true) },
		"warcries exert (%d+) additional attacks?": function(num) return { mod.NewFloat("ExtraExertedAttacks", mod.TypeBase, num) } end,
		"iron will": { mod.NewFlag("IronWill", true) },
		"iron reflexes while stationary": { mod("Keystone", "LIST", "Iron Reflexes", .Tag(mod.Condition("Stationary"))) },
		"you have zealot's oath if you haven't been hit recently": { mod("Keystone", "LIST", "Zealot's Oath", .Tag(mod.Condition("BeenHitRecently").Neg(true))) },
		"deal no physical damage": { mod.NewFlag("DealNoPhysical", true) },
		"deal no cold damage": { mod.NewFlag("DealNoCold", true) },
		"deal no fire damage": { mod.NewFlag("DealNoFire", true) },
		"deal no lightning damage": { mod.NewFlag("DealNoLightning", true) },
		"deal no elemental damage": { mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true) },
		"deal no chaos damage": { mod.NewFlag("DealNoChaos", true) },
		"deal no damage": { mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true), mod.NewFlag("DealNoPhysical", true) },
		"deal no non%-elemental damage": { mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoChaos", true) },
		"deal no non%-lightning damage": { mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true) },
		"deal no non%-physical damage": { mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true) },
		"cannot deal non%-chaos damage": { mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoLightning", true) },
		"deal no damage when not on low life": {
			flag("DealNoLightning", .Tag(mod.Condition("LowLife").Neg(true))),
			flag("DealNoCold", .Tag(mod.Condition("LowLife").Neg(true))),
			flag("DealNoFire", .Tag(mod.Condition("LowLife").Neg(true))),
			flag("DealNoChaos",.Tag(mod.Condition("LowLife").Neg(true))),
			flag("DealNoPhysical", .Tag(mod.Condition("LowLife").Neg(true))),
		},
		"attacks have blood magic": { flag("SkillBloodMagic", nil, mod.MFlagAttack) },
		"attacks cost life instead of mana": { flag("SkillBloodMagic", nil, mod.MFlagAttack) },
		"(%d+)%% chance to cast a? ?socketed lightning spells? on hit": function(num) return { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueMjolnerLightningSpellsCastOnHit", level = 1 }, mod.SocketedIn("{SlotName}")) } end,
		"cast a socketed lightning spell on hit": { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueMjolnerLightningSpellsCastOnHit", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed lightning spell on hit": { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueMjolnerLightningSpellsCastOnHit", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"trigger a socketed lightning spell on hit, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueMjolnerLightningSpellsCastOnHit", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike": { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike, with a ([%d%.]+) second cooldown": { mod("ExtraSupport", "LIST", { skillId = "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", level = 1 }, mod.SocketedIn("{SlotName}")) },
		"your curses can apply to hexproof enemies": { mod.NewFlag("CursesIgnoreHexproof", true) },
		"your hexes can affect hexproof enemies": { mod.NewFlag("CursesIgnoreHexproof", true) },
		"hexes from socketed skills can apply (%d) additional curses": function(num) return { mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, num), flag("SocketedCursesAdditionalLimit", mod.SocketedIn("{SlotName}") )} end,
		// TODO This is being changed from ignoreHexLimit to SocketedCursesAdditionalLimit due to patch 3.16.0, which states that legacy versions "will be affected by this Curse Limit change,
		// TODO though they will only have 20% less Curse Effect of Curses triggered with Summon Doedre’s Effigy."
		// TODO Legacy versions will still show that "Hexes from Socketed Skills ignore Curse limit", but will instead have an internal limit of 5 to match the current functionality.
		"hexes from socketed skills ignore curse limit": function(num) return { mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, 5), flag("SocketedCursesAdditionalLimit", mod.SocketedIn("{SlotName}") )} end,
		"reserves (%d+)%% of life": function(num) return { mod.NewFloat("ExtraLifeReserved", mod.TypeBase, num) } end,
		"(%d+)%% of cold damage taken as lightning": function(num) return { mod.NewFloat("ColdDamageTakenAsLightning", mod.TypeBase, num) } end,
		"(%d+)%% of fire damage taken as lightning": function(num) return { mod.NewFloat("FireDamageTakenAsLightning", mod.TypeBase, num) } end,
		"items and gems have (%d+)%% reduced attribute requirements": function(num) return { mod("GlobalAttributeRequirements", "INC", -num) } end,
		"items and gems have (%d+)%% increased attribute requirements": function(num) return { mod.NewFloat("GlobalAttributeRequirements", mod.TypeIncrease, num) } end,
		"mana reservation of herald skills is always (%d+)%%": function(num) return { mod("SkillData", "LIST", { key = "ManaReservationPercentForced", value = num }, mod.SkillType(data.SkillTypeHerald)) } end,
		"([%a%s]+) reserves no mana": function(_, name) return {
			mod("SkillData", "LIST", { key = "manaReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "lifeReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "manaReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "lifeReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
		} end,
		"([%a%s]+) has no reservation": function(_, name) return {
			mod("SkillData", "LIST", { key = "manaReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "lifeReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "manaReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
			mod("SkillData", "LIST", { key = "lifeReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }),
		} end,
		"([%a%s]+) has no reservation if cast as an aura": function(_, name) return {
			mod("SkillData", "LIST", { key = "manaReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }, mod.SkillType(data.SkillTypeAura)),
			mod("SkillData", "LIST", { key = "lifeReservationFlat", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }, mod.SkillType(data.SkillTypeAura)),
			mod("SkillData", "LIST", { key = "manaReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }, mod.SkillType(data.SkillTypeAura)),
			mod("SkillData", "LIST", { key = "lifeReservationPercent", value = 0 }, { type = "SkillId", skillId = gemIdLookup[name] }, mod.SkillType(data.SkillTypeAura)),
		} end,
		"banner skills reserve no mana": {
			mod("SkillData", "LIST", { key = "manaReservationPercent", value = 0 }, mod.SkillType(data.SkillTypeBanner)),
			mod("SkillData", "LIST", { key = "lifeReservationPercent", value = 0 }, mod.SkillType(data.SkillTypeBanner)),
		},
		"banner skills have no reservation": {
			mod("SkillData", "LIST", { key = "manaReservationPercent", value = 0 }, mod.SkillType(data.SkillTypeBanner)),
			mod("SkillData", "LIST", { key = "lifeReservationPercent", value = 0 }, mod.SkillType(data.SkillTypeBanner)),
		},
		"placed banners also grant (%d+)%% increased attack damage to you and allies": function(num) return { mod("ExtraAuraEffect", "LIST", { mod = mod("Damage", "INC", num, nil, mod.MFlagAttack) }, .Tag(mod.Condition("BannerPlanted")), mod.SkillType(data.SkillTypeBanner)) } end,
		"dread banner grants an additional %+(%d+) to maximum fortification when placing the banner": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("MaximumFortification", "BASE", num, { type = "GlobalEffect", effectType = "Buff" }) }, .Tag(mod.Condition("BannerPlanted")), mod.SkillName("Dread Banner")) } end,
		"your aura skills are disabled": { flag("DisableSkill", mod.SkillType(data.SkillTypeAura)) },
		"your spells are disabled": { flag("DisableSkill", mod.SkillType(data.SkillTypeSpell)) },
		"aura skills other than ([%a%s]+) are disabled": function(_, name) return {
			flag("DisableSkill", mod.SkillType(data.SkillTypeAura)),
			flag("EnableSkill", { type = "SkillId", skillId = gemIdLookup[name] }),
		} end,
		"travel skills other than ([%a%s]+) are disabled": function(_, name) return {
			flag("DisableSkill", mod.SkillType(data.SkillTypeTravel)),
			flag("EnableSkill", { type = "SkillId", skillId = gemIdLookup[name] }),
		} end,
		"strength's damage bonus instead grants (%d+)%% increased melee physical damage per (%d+) strength": function(num, _, perStr) return { mod("StrDmgBonusRatioOverride", "BASE", num / tonumber(perStr)) } end,
		"while in her embrace, take ([%d%.]+)%% of your total maximum life and energy shield as fire damage per second per level": function(num) return {
			mod("FireDegen", "BASE", 1, { type = "PercentStat", stat = "Life", percent = num }, mod.Multiplier("Level").Base(0), .Tag(mod.Condition("HerEmbrace"))),
			mod("FireDegen", "BASE", 1, { type = "PercentStat", stat = "EnergyShield", percent = num }, mod.Multiplier("Level").Base(0), .Tag(mod.Condition("HerEmbrace"))),
		} end,
		"gain her embrace for %d+ seconds when you ignite an enemy": { mod.NewFlag("Condition:CanGainHerEmbrace", true) },
		"when you cast a spell, sacrifice all mana to gain added maximum lightning damage equal to (%d+)%% of sacrificed mana for 4 seconds": function(num) return {
			mod.NewFlag("Condition:HaveManaStorm", true),
			mod("LightningMax", "BASE", 1, { type = "PerStat", stat = "ManaUnreserved" , div = 100 / num}, .Tag(mod.Condition("SacrificeManaForLightning"))),
		} end,
		"gain added chaos damage equal to (%d+)%% of ward": function(num) return {
			mod("ChaosMin", "BASE", 1, { type = "PerStat", stat = "Ward", div = 100 / num }),
			mod("ChaosMax", "BASE", 1, { type = "PerStat", stat = "Ward", div = 100 / num }),
		}  end,
		"every 16 seconds you gain iron reflexes for 8 seconds": {
			mod.NewFlag("Condition:HaveArborix", true),
		},
		"every 16 seconds you gain elemental overload for 8 seconds": {
			mod.NewFlag("Condition:HaveAugyre", true),
		},
		"every 8 seconds, gain avatar of fire for 4 seconds": {
			mod.NewFlag("Condition:HaveVulconus", true),
		},
		"modifiers to attributes instead apply to omniscience": { mod.NewFlag("Omniscience", true) },
		"attribute requirements can be satisfied by (%d+)%% of omniscience": function(num) return {
			mod.NewFloat("OmniAttributeRequirements", mod.TypeIncrease, num),
			mod.NewFlag("OmniscienceRequirements", true)
		} end,
		"you have far shot while you do not have iron reflexes": { flag("FarShot", { neg = true, type = "Condition", var = "HaveIronReflexes" }) },
		"you have resolute technique while you do not have elemental overload": { mod("Keystone", "LIST", "Resolute Technique", { neg = true, type = "Condition", var = "HaveElementalOverload" }) },
		"hits ignore enemy monster fire resistance while you are ignited": { flag("IgnoreFireResistance", .Tag(mod.Condition("Ignited"))) },
		"your hits can't be evaded by blinded enemies": { flag("CannotBeEvaded", { type = "ActorCondition", actor = "enemy", var = "Blinded" }) },
		"blind does not affect your chance to hit": { mod.NewFlag("IgnoreBlindHitChance", true) },
		"enemies blinded by you while you are blinded have malediction": { mod("EnemyModifier", "LIST", { mod = flag("HasMalediction", .Tag(mod.Condition("Blinded"))) }, .Tag(mod.Condition("Blinded")) )},
		"skills which throw traps have blood magic": { flag("BloodMagic", mod.SkillType(data.SkillTypeTrap)) },
		"skills which throw traps cost life instead of mana": { flag("BloodMagic", mod.SkillType(data.SkillTypeTrap)) },
		"lose ([%d%.]+) mana per second": function(num) return { mod.NewFloat("ManaDegen", mod.TypeBase, num) } end,
		"lose ([%d%.]+)%% of maximum mana per second": function(num) return { mod("ManaDegen", "BASE", 1, { type = "PercentStat", stat = "Mana", percent = num }) } end,
		"strength provides no bonus to maximum life": { mod.NewFlag("NoStrBonusToLife", true) },
		"intelligence provides no bonus to maximum mana": { mod.NewFlag("NoIntBonusToMana", true) },
		"with a ghastly eye jewel socketed, minions have %+(%d+) to accuracy rating": function(num) return { mod("MinionModifier", "LIST", { mod = mod.NewFloat("Accuracy", mod.TypeBase, num) }, .Tag(mod.Condition("HaveGhastlyEyeJewelIn{SlotName}"))) } end,
		"hits ignore enemy monster chaos resistance if all equipped items are shaper items": { flag("IgnoreChaosResistance", { type = "MultiplierThreshold", var = "NonShaperItem", upper = true, threshold = 0 }) },
		"hits ignore enemy monster chaos resistance if all equipped items are elder items": { flag("IgnoreChaosResistance", { type = "MultiplierThreshold", var = "NonElderItem", upper = true, threshold = 0 }) },
		"gain %d+ rage on critical hit with attacks, no more than once every [%d%.]+ seconds": {
			mod.NewFlag("Condition:CanGainRage", true),
		},
		"warcry skills' cooldown time is (%d+) seconds": function(num) return { mod("CooldownRecovery", "OVERRIDE", num, nil, 0, mod.KeywordFlagWarcry) } end,
		"warcry skills have (%+%d+) seconds to cooldown": function(num) return { mod("CooldownRecovery", "BASE", num, nil, 0, mod.KeywordFlagWarcry) } end,
		"using warcries is instant": { mod.NewFlag("InstantWarcry", true) },
		"attacks with axes or swords grant (%d+) rage on hit, no more than once every second": {
			flag("Condition:CanGainRage", { type = "Condition", varList = { "UsingAxe", "UsingSword" } }),
		},
		"your critical strike multiplier is (%d+)%%": function(num) return { mod.NewFloat("CritMultiplier", mod.TypeOverride, num) } end,
		"base critical strike chance for attacks with weapons is ([%d%.]+)%%": function(num) return { mod.NewFloat("WeaponBaseCritChance", mod.TypeOverride, num) } end,
		"critical strike chance is (%d+)%% for hits with this weapon": function(num) return { mod("CritChance", "OVERRIDE", num, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack)) } end,
		"allocates (.+) if you have the matching modifiers? on forbidden (.+)": function(_, ascendancy, side) return { mod("GrantedAscendancyNode", "LIST", { side = side, name = ascendancy }) } end,
		"allocates (.+)": function(_, passive) return { mod("GrantedPassive", "LIST", passive) } end,
		"battlemage": { mod.NewFlag("WeaponDamageAppliesToSpells", true), mod("ImprovedWeaponDamageAppliesToSpells", "MAX", 100) },
		"transfiguration of body": { mod.NewFlag("TransfigurationOfBody", true) },
		"transfiguration of mind": { mod.NewFlag("TransfigurationOfMind", true) },
		"transfiguration of soul": { mod.NewFlag("TransfigurationOfSoul", true) },
		"offering skills have (%d+)%% reduced duration": function(num) return {
			mod("Duration", "INC", -num, { type = "SkillName", skillNameList = { "Bone Offering", "Flesh Offering", "Spirit Offering" } }),
		} end,
		"enemies have %-(%d+)%% to total physical damage reduction against your hits": function(num) return {
			mod("EnemyPhysicalDamageReduction", "BASE", -num),
		} end,
		"enemies you impale have %-(%d+)%% to total physical damage reduction against impale hits": function(num) return {
			mod("EnemyImpalePhysicalDamageReduction", "BASE", -num)
		} end,
		"hits with this weapon overwhelm (%d+)%% physical damage reduction": function(num) return {
			mod("EnemyPhysicalDamageReduction", "BASE", -num, nil, mod.MFlagHit, .Tag(mod.Condition("{Hand}Attack")), mod.SkillType(data.SkillTypeAttack))
		} end,
		"overwhelm (%d+)%% physical damage reduction": function(num) return {
			mod("EnemyPhysicalDamageReduction", "BASE", -num)
		} end,
		"impale damage dealt to enemies impaled by you overwhelms (%d+)%% physical damage reduction": function(num) return {
			mod("EnemyImpalePhysicalDamageReduction", "BASE", -num)
		} end,
		"nearby enemies are crushed while you have ?a?t? least (%d+) rage": function(num) return {
			// TODO MultiplierThreshold is on RageStacks because Rage is only set in CalcPerform if Condition:CanGainRage is true, Bear's Girdle does not flag CanGainRage
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Crushed", true) }, { type = "MultiplierThreshold", var = "RageStack", threshold = num })
		} end,
		"you are crushed": { mod.NewFlag("Condition:Crushed", true) },
		"nearby enemies are crushed": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Crushed", true)} )},
		"crush enemies on hit with maces and sceptres": { mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Crushed", true) }, .Tag(mod.Condition("UsingMace")) )},
		"enemies on fungal ground you kill explode, dealing 5%% of their life as chaos damage": {},
		"you have fungal ground around you while stationary": {
			mod("ExtraAura", "LIST", { mod = mod.NewFloat("NonChaosDamageGainAsChaos", mod.TypeBase, 10) }, { type = "Condition", varList = { "OnFungalGround", "Stationary" } }),
			mod("EnemyModifier", "LIST", { mod = mod("Damage", "MORE", -10) }, { type = "ActorCondition", actor = "enemy", varList = { "OnFungalGround", "Stationary" } }),
		},
		"create profane ground instead of consecrated ground": {
			mod.NewFlag("Condition:CreateProfaneGround", true),
		},
		"you count as dual wielding while you are unencumbered": { flag("Condition:DualWielding", .Tag(mod.Condition("Unencumbered"))) },
		"dual wielding does not inherently grant chance to block attack damage": { mod.NewFlag("Condition:NoInherentBlock", true) },
		"you do not inherently take less damage for having fortification": { mod.NewFlag("Condition:NoFortificationMitigation", true) },
		"skills supported by intensify have %+(%d) to maximum intensity": function(num) return { mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, num) } end,
		"spells which can gain intensity have %+(%d) to maximum intensity": function(num) return { mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, num) } end,
		"hexes you inflict have %+(%d+) to maximum doom": function(num) return { mod.NewFloat("MaxDoom", mod.TypeBase, num) } end,
		"while stationary, gain (%d+)%% increased area of effect every second, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("AreaOfEffect", "INC", num, { type = "Multiplier", var = "StationarySeconds", globalLimit = tonumber(limit), globalLimitKey = "ExpansiveMight" }, .Tag(mod.Condition("Stationary"))),
		} end,
		"attack skills have added lightning damage equal to (%d+)%% of maximum mana": function(num) return {
			mod("LightningMin", "BASE", 1, nil, mod.MFlagAttack, { type = "PerStat", stat = "Mana", div = 100 / num }),
			mod("LightningMax", "BASE", 1, nil, mod.MFlagAttack, { type = "PerStat", stat = "Mana", div = 100 / num }),
		} end,
		"herald of thunder's storms hit enemies with (%d+)%% increased frequency": function(num) return { mod.NewFloat("HeraldStormFrequency", mod.TypeIncrease, num), } end,
		"your critical strikes have a (%d+)%% chance to deal double damage": function(num) return { mod.NewFloat("DoubleDamageChanceOnCrit", mod.TypeBase, num) } end,
		"(%d+)%% chance to deal triple damage": function(num) return { mod.NewFloat("TripleDamageChance", mod.TypeBase, num) } end,
		"elemental skills deal triple damage": { mod("TripleDamageChance", "BASE", 100, { type = "SkillType", skillTypeList = { SkillType.Cold, SkillType.Fire, SkillType.Lightning } } ), },
		"deal triple damage with elemental skills": { mod("TripleDamageChance", "BASE", 100, { type = "SkillType", skillTypeList = { SkillType.Cold, SkillType.Fire, SkillType.Lightning } } ), },
		"skills supported by unleash have %+(%d) to maximum number of seals": function(num) return { mod.NewFloat("SealCount", mod.TypeBase, num) } end,
		"skills supported by unleash have (%d+)%% increased seal gain frequency": function(num) return { mod.NewFloat("SealGainFrequency", mod.TypeIncrease, num) } end,
		"(%d+)%% increased critical strike chance with spells which remove the maximum number of seals": function(num) return { mod.NewFloat("MaxSealCrit", mod.TypeIncrease, num) } end,
		"gain elusive on critical strike": {
			mod.NewFlag("Condition:CanBeElusive", true),
		},
		"nearby enemies have (%a+) resistance equal to yours": function(_, res) return { mod.NewFlag("Enemy"..(res:gsub("^%l", string.upper)).."ResistEqualToYours", true) } end,
		"for each nearby corpse, regenerate ([%d%.]+)%% life per second, up to ([%d%.]+)%%": function(num, _, limit) return { mod("LifeRegenPercent", "BASE", num, { type = "Multiplier", var = "NearbyCorpse", limit = tonumber(limit), limitTotal = true }) } end,
		"gain sacrificial zeal when you use a skill, dealing you %d+%% of the skill's mana cost as physical damage per second": {
			mod.NewFlag("Condition:SacrificialZeal", true),
		},
		"hits overwhelm (%d+)%% of physical damage reduction while you have sacrificial zeal": function(num) return {
			mod("EnemyPhysicalDamageReduction", "BASE", -num, nil, .Tag(mod.Condition("SacrificialZeal"))),
		} end,
		"minions attacks overwhelm (%d+)%% physical damage reduction": function(num) return {
			mod("MinionModifier", "LIST", { mod = mod("EnemyPhysicalDamageReduction", "BASE", -num) })
		} end,
		"focus has (%d+)%% increased cooldown recovery rate": function(num) return { mod("FocusCooldownRecovery", "INC", num, { type = "Condition", var = "Focused"}) } end,
		"(%d+)%% chance to deal double damage with attacks if attack time is longer than 1 second": function(num) return {
			mod("DoubleDamageChance", "BASE", num, 0, 0, .Tag(mod.Condition("OneSecondAttackTime")))
		} end,
		"elusive also grants %+(%d+)%% to critical strike multiplier for skills supported by nightblade": function(num) return { mod.NewFloat("NightbladeElusiveCritMultiplier", mod.TypeBase, num) } end,
		"skills supported by nightblade have (%d+)%% increased effect of elusive": function(num) return { mod.NewFloat("NightbladeSupportedElusiveEffect", mod.TypeIncrease, num) } end,
		"nearby enemies are scorched": {
			mod("EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Scorched", true) }),
			mod.NewFloat("ScorchBase", mod.TypeBase, 10),
		},
		// TODO Pantheon: Soul of Tukohama support
		"while stationary, gain ([%d%.]+)%% of life regenerated per second every second, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("LifeRegenPercent", "BASE", num, { type = "Multiplier", var = "StationarySeconds", limit = tonumber(limit), limitTotal = true }, .Tag(mod.Condition("Stationary"))),
		} end,
		// TODO Pantheon: Soul of Tukohama support
		"while stationary, gain (%d+)%% additional physical damage reduction every second, up to a maximum of (%d+)%%": function(num, _, limit) return {
			mod("PhysicalDamageReduction", "BASE", num, { type = "Multiplier", var = "StationarySeconds", limit = tonumber(limit), limitTotal = true }, .Tag(mod.Condition("Stationary"))),
		} end,
		// TODO Skill-specific enchantment modifiers
		"(%d+)%% increased decoy totem life": function(num) return { mod("TotemLife", "INC", num, mod.SkillName("Decoy Totem")) } end,
		"(%d+)%% increased ice spear critical strike chance in second form": function(num) return { mod("CritChance", "INC", num, mod.SkillName("Ice Spear"), { type = "SkillPart", skillPart = 2 }) } end,
		"shock nova ring deals (%d+)%% increased damage": function(num) return { mod("Damage", "INC", num, mod.SkillName("Shock Nova"), { type = "SkillPart", skillPart = 1 }) } end,
		"enemies affected by bear trap take (%d+)%% increased damage from trap or mine hits": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("TrapMineDamageTaken", "INC", num, { type = "GlobalEffect", effectType = "Debuff" }) }, mod.SkillName("Bear Trap")) } end,
		"blade vortex has %+(%d+)%% to critical strike multiplier for each blade": function(num) return { mod("CritMultiplier", "BASE", num, mod.Multiplier("BladeVortexBlade").Base(0), mod.SkillName("Blade Vortex")) } end,
		"burning arrow has (%d+)%% increased debuff effect": function(num) return { mod("DebuffEffect", "INC", num, { type = "SkillName", skillName = "Burning Arrow"}) } end,
		"double strike has a (%d+)%% chance to deal double damage to bleeding enemies": function(num) return { mod("DoubleDamageChance", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Bleeding" }, mod.SkillName("Double Strike")) } end,
		"frost bomb has (%d+)%% increased debuff duration": function(num) return { mod("SecondaryDuration", "INC", num, mod.SkillName("Frost Bomb")) } end,
		"incinerate has %+(%d+) to maximum stages": function(num) return { mod("Multiplier:IncinerateMaxStages", "BASE", num, mod.SkillName("Incinerate")) } end,
		"perforate creates %+(%d+) spikes?": function(num) return { mod.NewFloat("Multiplier:PerforateMaxSpikes", mod.TypeBase, num) } end,
		"scourge arrow has (%d+)%% chance to poison per stage": function(num) return { mod("PoisonChance", "BASE", num, mod.SkillName("Scourge Arrow"), mod.Multiplier("ScourgeArrowStage").Base(0)) } end,
		"winter orb has %+(%d+) maximum stages": function(num) return { mod.NewFloat("Multiplier:WinterOrbMaxStages", mod.TypeBase, num) } end,
		"summoned holy relics have (%d+)%% increased buff effect": function(num) return { mod("BuffEffect", "INC", num, mod.SkillName("Summon Holy Relic")) } end,
		"%+(%d) to maximum virulence": function(num) return { mod.NewFloat("Multiplier:VirulenceStacksMax", mod.TypeBase, num) } end,
		"winter orb has (%d+)%% increased area of effect per stage": function(num) return { mod("AreaOfEffect", "INC", num, mod.SkillName("Winter Orb"), mod.Multiplier("WinterOrbStage").Base(0)) } end,
		"wintertide brand has %+(%d+) to maximum stages": function(num) return { mod("Multiplier:WintertideBrandMaxStages", "BASE", num, mod.SkillName("Wintertide Brand")) } end,
		"wave of conviction's exposure applies (%-%d+)%% elemental resistance": function(num) return { mod("ExtraSkillStat", "LIST", { key = "purge_expose_resist_%_matching_highest_element_damage", value = num }, mod.SkillName("Wave of Conviction")) } end,
		"arcane cloak spends an additional (%d+)%% of current mana": function(num) return { mod("ExtraSkillStat", "LIST", { key = "arcane_cloak_consume_%_of_mana", value = num }, mod.SkillName("Arcane Cloak")) } end,
		"arcane cloak grants life regeneration equal to (%d+)%% of mana spent per second": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("LifeRegen", "BASE", num / 100, 0, 0, mod.Multiplier("ArcaneCloakConsumedMana").Base(0), { type = "GlobalEffect", effectType = "Buff" }) }, mod.SkillName("Arcane Cloak")) } end,
		"caustic arrow has (%d+)%% chance to inflict withered on hit for (%d+) seconds base duration": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFlag("Condition:CanWither", true) }, mod.SkillName("Caustic Arrow") ) },
		"venom gyre has a (%d+)%% chance to inflict withered for (%d+) seconds on hit": { mod("ExtraSkillMod", "LIST", { mod = mod.NewFlag("Condition:CanWither", true) }, mod.SkillName("Venom Gyre") ) },
		"sigil of power's buff also grants (%d+)%% increased critical strike chance per stage": function(num) return { mod("CritChance", "INC", num, 0, 0, { type = "Multiplier", var = "SigilOfPowerStage", limit = 4 }, { type = "GlobalEffect", effectType = "Buff", effectName = "Sigil of Power" } ) } end,
		"cobra lash chains (%d+) additional times": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("ChainCountMax", mod.TypeBase, num) }, mod.SkillName("Cobra Lash")) } end,
		"general's cry has ([%+%-]%d) to maximum number of mirage warriors": function(num) return { mod.NewFloat("GeneralsCryDoubleMaxCount", mod.TypeBase, num) } end,
		"([%+%-]%d) to maximum blade flurry stages": function(num) return { mod.NewFloat("Multiplier:BladeFlurryMaxStages", mod.TypeBase, num) } end,
		"steelskin buff can take (%d+)%% increased amount of damage": function(num) return { mod("ExtraSkillStat", "LIST", { key = "steelskin_damage_limit_+%", value = num }, mod.SkillName("Steelskin")) } end,
		"hydrosphere has (%d+)%% increased pulse frequency": function(num) return { mod.NewFloat("HydroSphereFrequency", mod.TypeIncrease, num) } end,
		"void sphere has (%d+)%% increased pulse frequency": function(num) return { mod.NewFloat("VoidSphereFrequency", mod.TypeIncrease, num) } end,
		"shield crush central wave has (%d+)%% more area of effect": function(num) return { mod("AreaOfEffect", "MORE", num, mod.SkillName("Shield Crush"), { type = "SkillPart", skillPart = 2 }) } end,
		"storm rain has (%d+)%% increased beam frequency": function(num) return { mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, num) } end,
		"voltaxic burst deals (%d+)%% increased damage per ([%d%.]+) seconds of duration": function(num, _) return { mod.NewFloat("VoltaxicDurationIncDamage", mod.TypeIncrease, num) } end,
		"earthquake deals (%d+)%% increased damage per ([%d%.]+) seconds duration": function(num, _) return { mod.NewFloat("EarthquakeDurationIncDamage", mod.TypeIncrease, num) } end,
		"consecrated ground from holy flame totem applies (%d+)%% increased damage taken to enemies": function(num) return { mod("EnemyModifier", "LIST", { mod = mod("DamageTakenConsecratedGround", "INC", num, .Tag(mod.Condition("OnConsecratedGround"))) }) } end,
		"consecrated ground from purifying flame applies (%d+)%% increased damage taken to enemies": function(num) return { mod("ExtraSkillStat", "LIST", { key = "consecrated_ground_enemy_damage_taken_+%", value = num }, mod.SkillName("Purifying Flame")) } end,
		"enemies drenched by hydrosphere have cold and lightning exposure, applying (%-%d+)%% to resistances": function(num) return { mod("ExtraSkillStat", "LIST", { key = "water_sphere_cold_lightning_exposure_%", value = num }, mod.SkillName("Hydrosphere")) } end,
		"frost shield has %+(%d+) to maximum life per stage": function(num) return { mod("ExtraSkillStat", "LIST", { key = "frost_globe_health_per_stage", value = num }, mod.SkillName("Frost Shield")) } end,
		"flame wall grants (%d+) to (%d+) added fire damage to projectiles": function(min, max) return { mod("ExtraSkillStat", "LIST", { key = "flame_wall_minimum_added_fire_damage", value = min }, mod.SkillName("Flame Wall")), mod("ExtraSkillStat", "LIST",  { key = "flame_wall_maximum_added_fire_damage", value = max }, mod.SkillName("Flame Wall"))} end,
		"plague bearer buff grants %+(%d+)%% to poison damage over time multiplier while infecting": function(num) return { mod("ExtraSkillStat", "LIST", { key = "corrosive_shroud_poison_dot_multiplier_+_while_aura_active", value = num }, mod.SkillName("Plague Bearer")) } end,
		"(%d+)%% increased lightning trap lightning ailment effect": function(num) return { mod("ExtraSkillStat", "LIST", { key = "shock_effect_+%", value = num }, mod.SkillName("Lightning Trap")) } end,
		"wild strike's beam chains an additional (%d+) times": function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod.NewFloat("ChainCountMax", mod.TypeBase, num) }, mod.SkillName("Wild Strike"), { type = "SkillPart", skillPart = 4 }) } end,
		"energy blades have (%d+)%% increased attack speed": function(num) return { mod.NewFloat("EnergyBladeAttackSpeed", mod.TypeIncrease, num) } end,
		"ensnaring arrow has (%d+)%% increased debuff effect": function(num) return { mod("DebuffEffect", "INC", num, { type = "SkillName", skillName = "Ensnaring Arrow"}) } end,
		"unearth spawns corpses with ([%+%-]%d) level": function(num) return { mod("CorpseLevel", "BASE", num, { type = "SkillName", skillName = "Unearth"}) } end,
		// TODO Alternate Quality
		"quality does not increase physical damage": { mod.NewFloat("AlternateQualityWeapon", mod.TypeBase, 1) },
		"(%d+)%% increased critical strike chance per 4%% quality": function(num) return { mod.NewFloat("AlternateQualityLocalCritChancePer4Quality", mod.TypeIncrease, num) } end,
		"grants (%d+)%% increased accuracy per (%d+)%% quality": function(num, _, div) return { mod("Accuracy", "INC", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"(%d+)%% increased attack speed per 8%% quality": function(num) return { mod.NewFloat("AlternateQualityLocalAttackSpeedPer8Quality", mod.TypeIncrease, num) } end,
		"%+(%d+) weapon range per 10%% quality": function(num) return { mod.NewFloat("AlternateQualityLocalWeaponRangePer10Quality", mod.TypeBase, num) } end,
		"grants (%d+)%% increased elemental damage per (%d+)%% quality": function(num, _, div) return { mod("ElementalDamage", "INC", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants (%d+)%% increased area of effect per (%d+)%% quality": function(num, _, div) return { mod("AreaOfEffect", "INC", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"quality does not increase defences": { mod.NewFloat("AlternateQualityArmour", mod.TypeBase, 1) },
		"grants %+(%d+) to maximum life per (%d+)%% quality": function(num, _, div) return { mod("Life", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+) to maximum mana per (%d+)%% quality": function(num, _, div) return { mod("Mana", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+) to strength per (%d+)%% quality": function(num, _, div) return { mod("Str", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+) to dexterity per (%d+)%% quality": function(num, _, div) return { mod("Dex", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+) to intelligence per (%d+)%% quality": function(num, _, div) return { mod("Int", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+)%% to fire resistance per (%d+)%% quality": function(num, _, div) return { mod("FireResist", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+)%% to cold resistance per (%d+)%% quality": function(num, _, div) return { mod("ColdResist", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"grants %+(%d+)%% to lightning resistance per (%d+)%% quality": function(num, _, div) return { mod("LightningResist", "BASE", num, { type = "Multiplier", var = "QualityOn{SlotName}", div = tonumber(div) }) } end,
		"%+(%d+)%% to quality": function(num) return { mod.NewFloat("Quality", mod.TypeBase, num) } end,
		"infernal blow debuff deals an additional (%d+)%% of damage per charge": function(num) return { mod("DebuffEffect", "BASE", num, { type = "SkillName", skillName = "Infernal Blow"}) } end,
		// TODO Display-only modifiers
		"extra gore": { },
		"prefixes:": { },
		"suffixes:": { },
		"while your passive skill tree connects to a class' starting location, you gain:": { },
		"socketed lightning spells [hd][ae][va][el] (%d+)%% increased spell damage if triggered": { },
		"manifeste?d? dancing dervishe?s? disables both weapon slots": { },
		"manifeste?d? dancing dervishe?s? dies? when rampage ends": { },
		// TODO Legion modifiers
		"bathed in the blood of (%d+) sacrificed in the name of (.+)":  function(num, _, name)
			return { mod("JewelData", "LIST",
					{key = "conqueredBy", value = {id = num, conqueror = conquerorList[name:lower()] } }) } end,
		"carved to glorify (%d+) new faithful converted by high templar (.+)":  function(num, _, name)
			return { mod("JewelData", "LIST",
					{key = "conqueredBy", value = {id = num, conqueror = conquerorList[name:lower()] } }) } end,
		"commanded leadership over (%d+) warriors under (.+)":  function(num, _, name)
			return { mod("JewelData", "LIST",
					{key = "conqueredBy", value = {id = num, conqueror = conquerorList[name:lower()] } }) } end,
		"commissioned (%d+) coins to commemorate (.+)":  function(num, _, name)
			return { mod("JewelData", "LIST",
					{key = "conqueredBy", value = {id = num, conqueror = conquerorList[name:lower()] } }) } end,
		"denoted service of (%d+) dekhara in the akhara of (.+)":  function(num, _, name)
			return { mod("JewelData", "LIST",
					{key = "conqueredBy", value = {id = num, conqueror = conquerorList[name:lower()] } }) } end,
		"passives in radius are conquered by the (%D+)": { },
		"historic": { },
		"survival": { },
		"you can have two different banners at the same time": { },
		"can have a second enchantment modifier": { },
		"can have (%d+) additional enchantment modifiers": { },
		"this item can be anointed by cassia": { },
		"all sockets are white": { },
		"every (%d+) seconds, regenerate (%d+)%% of life over one second": function (num, _, percent) return {
			mod("LifeRegenPercent", "BASE", tonumber(percent), .Tag(mod.Condition("LifeRegenBurstFull"))),
			mod("LifeRegenPercent", "BASE", tonumber(percent) / num, .Tag(mod.Condition("LifeRegenBurstAvg"))),
		} end,
		"you take (%d+)%% reduced extra damage from critical strikes": function(num) return { mod.NewFloat("ReduceCritExtraDamage", mod.TypeBase, num) } end,
		"you take (%d+)%% reduced extra damage from critical strikes while you have no power charges": function(num) return { mod("ReduceCritExtraDamage", "BASE", num, { type = "StatThreshold", stat = "PowerCharges", threshold = 0, upper = true }) } end,
		"you take (%d+)%% reduced extra damage from critical strikes by poisoned enemies": function(num) return { mod("ReduceCritExtraDamage", "BASE", num, { type = "ActorCondition", actor = "enemy", var = "Poisoned" }) } end,
		"nearby allies have (%d+)%% chance to block attack damage per (%d+) strength you have": function(block, _, str)
			return {  mod("ExtraAura", "LIST",
					{onlyAllies = true, mod = mod.NewFloat("BlockChance", mod.TypeBase, block)}, {type = "PerStat", stat = "Str", div = tonumber(str)})} end,
	*/
}

/*
   for _, name in pairs(data.keystones) do
   	specialModList[name:lower()] = { mod("Keystone", "LIST", name) }
   end
   local oldList = specialModList
   specialModList = { }
   for k, v in pairs(oldList) do
   	specialModList["^"..k.."$"] = v
   end
*/

// Modifiers that are recognised but unsupported
var unsupportedModList = map[string]bool{
	"properties are doubled while in a breach": true,
	"mirrored": true,
	"split":    true,
}

// Special lookups used for various modifier forms
var suffixTypesCompiled map[string]CompiledList[string]
var suffixTypes = map[string]string{
	"as extra lightning damage":        "GainAsLightning",
	"added as lightning damage":        "GainAsLightning",
	"gained as extra lightning damage": "GainAsLightning",
	"as extra cold damage":             "GainAsCold",
	"added as cold damage":             "GainAsCold",
	"gained as extra cold damage":      "GainAsCold",
	"as extra fire damage":             "GainAsFire",
	"added as fire damage":             "GainAsFire",
	"gained as extra fire damage":      "GainAsFire",
	"as extra chaos damage":            "GainAsChaos",
	"added as chaos damage":            "GainAsChaos",
	"gained as extra chaos damage":     "GainAsChaos",
	"converted to lightning":           "ConvertToLightning",
	"converted to lightning damage":    "ConvertToLightning",
	"converted to cold damage":         "ConvertToCold",
	"converted to fire damage":         "ConvertToFire",
	"converted to fire":                "ConvertToFire",
	"converted to chaos damage":        "ConvertToChaos",
	"added as energy shield":           "GainAsEnergyShield",
	"as extra maximum energy shield":   "GainAsEnergyShield",
	"converted to energy shield":       "ConvertToEnergyShield",
	"as extra armour":                  "GainAsArmour",
	"as physical damage":               "AsPhysical",
	"as lightning damage":              "AsLightning",
	"as cold damage":                   "AsCold",
	"as fire damage":                   "AsFire",
	"as fire":                          "AsFire",
	"as chaos damage":                  "AsChaos",
	"leeched as life and mana":         "Leech",
	"leeched as life":                  "LifeLeech",
	"is leeched as life":               "LifeLeech",
	"leeched as mana":                  "ManaLeech",
	"is leeched as mana":               "ManaLeech",
	"leeched as energy shield":         "EnergyShieldLeech",
	"is leeched as energy shield":      "EnergyShieldLeech",
}

var dmgTypesCompiled map[string]CompiledList[string]
var dmgTypes = map[string]string{
	"physical":  "Physical",
	"lightning": "Lightning",
	"cold":      "Cold",
	"fire":      "Fire",
	"chaos":     "Chaos",
}

var penTypesCompiled map[string]CompiledList[modNameListType]
var penTypes = map[string]modNameListType{
	"lightning resistance":  {names: []string{"LightningPenetration"}},
	"cold resistance":       {names: []string{"ColdPenetration"}},
	"fire resistance":       {names: []string{"FirePenetration"}},
	"elemental resistance":  {names: []string{"ElementalPenetration"}},
	"elemental resistances": {names: []string{"ElementalPenetration"}},
	"chaos resistance":      {names: []string{"ChaosPenetration"}},
}

var regenTypesCompiled map[string]CompiledList[modNameListType]
var regenTypes = map[string]modNameListType{
	"life":                           {names: []string{"LifeRegen"}},
	"maximum life":                   {names: []string{"LifeRegen"}},
	"life and mana":                  {names: []string{"LifeRegen", "ManaRegen"}},
	"mana":                           {names: []string{"ManaRegen"}},
	"maximum mana":                   {names: []string{"ManaRegen"}},
	"energy shield":                  {names: []string{"EnergyShieldRegen"}},
	"maximum energy shield":          {names: []string{"EnergyShieldRegen"}},
	"maximum mana and energy shield": {names: []string{"ManaRegen", "EnergyShieldRegen"}},
	"rage":                           {names: []string{"RageRegen"}},
}

// TODO flagTypes
//var flagTypes = map[string]modNameListType{
//	"phasing":              {names:[]string{"Condition:Phasing"}},
//	"onslaught":            {names:[]string{"Condition:Onslaught"}},
//	"fortify":              {names:[]string{"Condition:Fortified"}},
//	"fortified":            {names:[]string{"Condition:Fortified"}},
//	"unholy might":         {names:[]string{"Condition:UnholyMight"}},
//	"tailwind":             {names:[]string{"Condition:Tailwind"}},
//	"intimidated":          {names:[]string{"Condition:Intimidated"}},
//	"crushed":              {names:[]string{"Condition:Crushed"}},
//	"chilled":              {names:[]string{"Condition:Chilled"}},
//	"blinded":              {names:[]string{"Condition:Blinded"}},
//	"no life regeneration": {names:[]string{"NoLifeRegen"}},
//	"hexproof":             {names:[]string{"CurseEffectOnSelf"}, value: -100, type = "MORE"},
//	"hindered, with (%d+)%% reduced movement speed": {names:[]string{"Condition:Hindered"}},
//	"unnerved": {names:[]string{"Condition:Unnerved"}},
//}

// Build active skill name lookup
var skillNameListCompiled map[string]CompiledList[modNameListType]
var preSkillNameListCompiled map[string]CompiledList[modNameListType]

func initializeSkillNameList() {
	skillNameListCompiled = make(map[string]CompiledList[modNameListType])
	skillNameListCompiled[" corpse cremation "] = CompiledList[modNameListType]{
		Regex: regexp.MustCompile(` corpse cremation `),
		Value: modNameListType{
			tag: mod.SkillName("Cremation"),
		},
	}

	preSkillNameListCompiled = make(map[string]CompiledList[modNameListType])
	for _, gemData := range raw.SkillGems {
		grantedEffect := gemData.GetGrantedEffect()
		// TODO grantedEffect.hidden
		if grantedEffect != nil && !grantedEffect.IsSupport {
			skillName := grantedEffect.GetActiveSkill().DisplayedName

			if skillName == "" || skillName == "..." || strings.Contains(skillName, "NOT CURRENTLY USED") || strings.Contains(skillName, "UNUSED") {
				continue
			}

			skillNameLower := strings.ToLower(skillName)

			val := modNameListType{
				tag: mod.SkillName(skillName),
			}

			skillNameListCompiled[" "+skillNameLower+" "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile(" " + skillNameLower + " "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" has ?a? "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " has ?a? "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" deals "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " deals "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" damage "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " damage "),
				Value: val,
			}

			/*
				TODO
				if gemData.tags.totem then
					preSkillNameList["^"..skillName:lower().." totem deals "] = { tag = { type = "SkillName", skillName = skillName } }
					preSkillNameList["^"..skillName:lower().." totem grants "] = { addToSkill = { type = "SkillName", skillName = skillName }, tag = { type = "GlobalEffect", effectType = "Buff" } }
				end
				if grantedEffect.skillTypes[SkillType.Buff] or grantedEffect.baseFlags.buff then
					preSkillNameList["^"..skillName:lower().." grants "] = { addToSkill = { type = "SkillName", skillName = skillName }, tag = { type = "GlobalEffect", effectType = "Buff" } }
					preSkillNameList["^"..skillName:lower().." grants a?n? ?additional "] = { addToSkill = { type = "SkillName", skillName = skillName }, tag = { type = "GlobalEffect", effectType = "Buff" } }
				end
				if gemData.tags.aura or gemData.tags.herald then
					skillNameList["while affected by "..skillName:lower()] = { tag = { type = "Condition", var = "AffectedBy"..skillName:gsub(" ","") } }
					skillNameList["while using "..skillName:lower()] = { tag = { type = "Condition", var = "AffectedBy"..skillName:gsub(" ","") } }
				end
				if gemData.tags.mine then
					specialModList["^"..skillName:lower().." has (%d+)%% increased throwing speed"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("MineLayingSpeed", "INC", num) }, { type = "SkillName", skillName = skillName }) } end
				end
				if gemData.tags.trap then
					specialModList["(%d+)%% increased "..skillName:lower().." throwing speed"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("TrapThrowingSpeed", "INC", num) }, { type = "SkillName", skillName = skillName }) } end
				end
				if gemData.tags.chaining then
					specialModList["^"..skillName:lower().." chains an additional time"] = { mod("ExtraSkillMod", "LIST", { mod = mod("ChainCountMax", "BASE", 1) }, { type = "SkillName", skillName = skillName }) }
					specialModList["^"..skillName:lower().." chains an additional (%d+) times"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ChainCountMax", "BASE", num) }, { type = "SkillName", skillName = skillName }) } end
					specialModList["^"..skillName:lower().." chains (%d+) additional times"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ChainCountMax", "BASE", num) }, { type = "SkillName", skillName = skillName }) } end
				end
				if gemData.tags.bow then
					specialModList["^"..skillName:lower().." fires an additional arrow"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ProjectileCount", "BASE", 1) }, { type = "SkillName", skillName = skillName }) } end
					specialModList["^"..skillName:lower().." fires (%d+) additional arrows?"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ProjectileCount", "BASE", num) }, { type = "SkillName", skillName = skillName }) } end
				end
				if gemData.tags.projectile then
					specialModList["^"..skillName:lower().." pierces an additional target"] = { mod("PierceCount", "BASE", 1, { type = "SkillName", skillName = skillName }) }
					specialModList["^"..skillName:lower().." pierces (%d+) additional targets?"] = function(num) return { mod("PierceCount", "BASE", num, { type = "SkillName", skillName = skillName }) } end
				end
				if gemData.tags.bow or gemData.tags.projectile then
					specialModList["^"..skillName:lower().." fires an additional projectile"] = { mod("ExtraSkillMod", "LIST", { mod = mod("ProjectileCount", "BASE", 1) }, { type = "SkillName", skillName = skillName }) }
					specialModList["^"..skillName:lower().." fires (%d+) additional projectiles"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ProjectileCount", "BASE", num) }, { type = "SkillName", skillName = skillName }) } end
					specialModList["^"..skillName:lower().." fires (%d+) additional shard projectiles"] = function(num) return { mod("ExtraSkillMod", "LIST", { mod = mod("ProjectileCount", "BASE", num) }, { type = "SkillName", skillName = skillName }) } end
				end
			*/
		}
	}
}

/*
   // TODO Radius jewels that modify other nodes
   local function getSimpleConv(srcList, dst, type, remove, factor)
   	return function(node, out, data)
   		if node then
   			for _, src in pairs(srcList) do
   				for _, mod in ipairs(node.modList) do
   					if mod.name == src and mod.type == type then
   						if remove then
   							out:MergeNewMod(src, type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
   						end
   						if factor then
   							out:MergeNewMod(dst, type, math.floor(mod.value * factor), mod.source, mod.flags, mod.keywordFlags, unpack(mod))
   						else
   							out:MergeNewMod(dst, type, mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
   						end
   					end
   				end
   			end
   		end
   	end
   end
   local jewelOtherFuncs = {
   	"Strength from Passives in Radius is Transformed to Dexterity": getSimpleConv({ "Str" }, "Dex", "BASE", true),
   	"Dexterity from Passives in Radius is Transformed to Strength": getSimpleConv({ "Dex" }, "Str", "BASE", true),
   	"Strength from Passives in Radius is Transformed to Intelligence": getSimpleConv({ "Str" }, "Int", "BASE", true),
   	"Intelligence from Passives in Radius is Transformed to Strength": getSimpleConv({ "Int" }, "Str", "BASE", true),
   	"Dexterity from Passives in Radius is Transformed to Intelligence": getSimpleConv({ "Dex" }, "Int", "BASE", true),
   	"Intelligence from Passives in Radius is Transformed to Dexterity": getSimpleConv({ "Int" }, "Dex", "BASE", true),
   	"Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield": getSimpleConv({ "Life" }, "EnergyShield", "INC", true),
   	"Increases and Reductions to Energy Shield in Radius are Transformed to apply to Armour at 200% of their value": getSimpleConv({ "EnergyShield" }, "Armour", "INC", true, 2),
   	"Increases and Reductions to Life in Radius are Transformed to apply to Mana at 200% of their value": getSimpleConv({ "Life" }, "Mana", "INC", true, 2),
   	"Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage": getSimpleConv({ "PhysicalDamage" }, "ColdDamage", "INC", true),
   	"Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage": getSimpleConv({ "ColdDamage" }, "PhysicalDamage", "INC", true),
   	"Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage": getSimpleConv({ "PhysicalDamage","ColdDamage","LightningDamage","ChaosDamage" }, "FireDamage", "INC", true),
   	"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spells at 35% of its value": getSimpleConv({ "LightningResist","ElementalResist" }, "SpellBlockChance", "BASE", false, 0.35),
   	"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spell Damage at 35% of its value": getSimpleConv({ "LightningResist","ElementalResist" }, "SpellBlockChance", "BASE", false, 0.35),
   	"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attacks at 35% of its value": getSimpleConv({ "ColdResist","ElementalResist" }, "AttackDodgeChance", "BASE", false, 0.35),
   	"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attack Hits at 35% of its value": getSimpleConv({ "ColdResist","ElementalResist" }, "AttackDodgeChance", "BASE", false, 0.35),
   	"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Suppress Spell Damage at 35% of its value": getSimpleConv({ "ColdResist","ElementalResist" }, "SpellSuppressionChance", "BASE", false, 0.35),
   	"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Suppress Spell Damage at 50% of its value": getSimpleConv({ "ColdResist","ElementalResist" }, "SpellSuppressionChance", "BASE", false, 0.5),
   	"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block Attack Damage at 35% of its value": getSimpleConv({ "FireResist","ElementalResist" }, "BlockChance", "BASE", false, 0.35),
   	"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block at 35% of its value": getSimpleConv({ "FireResist","ElementalResist" }, "BlockChance", "BASE", false, 0.35),
   	"Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers": function(node, out, data)
   		if node then
   			local mask1 = bor(mod.MFlagAxe, mod.MFlagClaw, mod.MFlagDagger, mod.MFlagMace, mod.MFlagStaff, mod.MFlagSword, mod.MFlagMelee)
   			local mask2 = bor(mod.MFlagWeapon1H, mod.MFlagWeaponMelee)
   			local mask3 = bor(mod.MFlagWeapon2H, mod.MFlagWeaponMelee)
   			for _, mod in ipairs(node.modList) do
   				if band(mod.flags, mask1) ~= 0 or band(mod.flags, mask2) == mask2 or band(mod.flags, mask3) == mask3 then
   					out:MergeNewMod(mod.name, mod.type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
   					out:MergeNewMod(mod.name, mod.type, mod.value, mod.source, bor(band(mod.flags, bnot(bor(mask1, mask2, mask3))), mod.MFlagBow), mod.keywordFlags, unpack(mod))
   				elseif mod[1] then
   					local using = { UsingAxe = true, UsingClaw = true, UsingDagger = true, UsingMace = true, UsingStaff = true, UsingSword = true, UsingMeleeWeapon = true }
   					for _, tag in ipairs(mod) do
   						if tag.type == "Condition" and using[tag.var] then
   							local newtagList: []mod.Tag copyTable(mod)
   							for _, tag in ipairs(newTagList) do
   								if tag.type == "Condition" and using[tag.var] then
   									tag.var = "UsingBow"
   									break
   								end
   							end
   							out:MergeNewMod(mod.name, mod.type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
   							out:MergeNewMod(mod.name, mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(newTagList))
   							break
   						end
   					end
   				end
   			end
   		end
   	end,
   	"50% increased Effect of non-Keystone Passive Skills in Radius": function(node, out, data)
   		if node and node.type ~= "Keystone" then
   			out:NewMod("PassiveSkillEffect", "INC", 50, data.modSource)
   		end
   	end,
   	"Notable Passive Skills in Radius grant nothing": function(node, out, data)
   		if node and node.type == "Notable" then
   			out:NewMod("PassiveSkillHasNoEffect", "FLAG", true, data.modSource)
   		end
   	end,
   	"Allocated Small Passive Skills in Radius grant nothing": function(node, out, data)
   		if node and node.type == "Normal" then
   			out:NewMod("AllocatedPassiveSkillHasNoEffect", "FLAG", true, data.modSource)
   		end
   	end,
   	"Passive Skills in Radius also grant: Traps and Mines deal (%d+) to (%d+) added Physical Damage": function(min, max)
   		return function(node, out, data)
   			if node and node.type ~= "Keystone" then
   				out:NewMod("PhysicalMin", "BASE", min, data.modSource, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine))
   				out:NewMod("PhysicalMax", "BASE", max, data.modSource, 0, bor(mod.KeywordFlagTrap, mod.KeywordFlagMine))
   			end
   		end
   	end,
   	"Passive Skills in Radius also grant: (%d+)%% increased Unarmed Attack Speed with Melee Skills": function(num)
   		return function(node, out, data)
   			if node and node.type ~= "Keystone" then
   				out:NewMod("Speed", "INC", num, data.modSource, bor(mod.MFlagUnarmed, mod.MFlagAttack, mod.MFlagMelee))
   			end
   		end
   	end,
   	"Notable Passive Skills in Radius are Transformed to instead grant: 10% increased Mana Cost of Skills and 20% increased Spell Damage": function(node, out, data)
   		if node and node.type == "Notable" then
   			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
   			out:NewMod("NodeModifier", "LIST", { mod = mod("ManaCost", "INC", 10, data.modSource) }, data.modSource)
   			out:NewMod("NodeModifier", "LIST", { mod = mod("Damage", "INC", 20, data.modSource, mod.MFlagSpell) }, data.modSource)
   		end
   	end,
   	"Notable Passive Skills in Radius are Transformed to instead grant: Minions take 20% increased Damage": function(node, out, data)
   		if node and node.type == "Notable" then
   			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
   			out:NewMod("NodeModifier", "LIST", { mod = mod("MinionModifier", "LIST", { mod = mod("DamageTaken", "INC", 20, data.modSource) } ) }, data.modSource)
   		end
   	end,
   	"Notable Passive Skills in Radius are Transformed to instead grant: Minions have 25% reduced Movement Speed": function(node, out, data)
   		if node and node.type == "Notable" then
   			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
   			out:NewMod("NodeModifier", "LIST", { mod = mod("MinionModifier", "LIST", { mod = mod("MovementSpeed", "INC", -25, data.modSource) } ) }, data.modSource)
   		end
   	end,
   }

   // TODO Radius jewels that modify the jewel itself based on nearby allocated nodes
   local function getPerStat(dst, modType, flags, stat, factor)
   	return function(node, out, data)
   		if node then
   			data[stat] = (data[stat] or 0) + out:Sum("BASE", nil, stat)
   		elseif data[stat] ~= 0 then
   			out:NewMod(dst, modType, math.floor((data[stat] or 0) * factor), data.modSource, flags)
   		end
   	end
   end
   local jewelSelfFuncs = {
   	"Adds 1 to maximum Life per 3 Intelligence in Radius": getPerStat("Life", "BASE", 0, "Int", 1 / 3),
   	"Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius": getPerStat("Life", "BASE", 0, "Int", 1 / 3),
   	"1% increased Evasion Rating per 3 Dexterity Allocated in Radius": getPerStat("Evasion", "INC", 0, "Dex", 1 / 3),
   	"1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius": getPerStat("PhysicalDamage", "INC", mod.MFlagClaw, "Dex", 1 / 3),
   	"1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius": getPerStat("PhysicalDamage", "INC", mod.MFlagUnarmed, "Dex", 1 / 3),
   	"3% increased Totem Life per 10 Strength in Radius": getPerStat("TotemLife", "INC", 0, "Str", 3 / 10),
   	"3% increased Totem Life per 10 Strength Allocated in Radius": getPerStat("TotemLife", "INC", 0, "Str", 3 / 10),
   	"Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius": getPerStat("LightningMax", "BASE", mod.MFlagAttack, "Dex", 1),
   	"5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius": getPerStat("ChaosDamage", "INC", 0, "Int", 5 / 10),
   	"Dexterity and Intelligence from passives in Radius count towards Strength Melee Damage bonus": function(node, out, data)
   		if node then
   			data.Dex = (data.Dex or 0) + node.modList:Sum("BASE", nil, "Dex")
   			data.Int = (data.Int or 0) + node.modList:Sum("BASE", nil, "Int")
   		elseif data.Dex or data.Int then
   			out:NewMod("DexIntToMeleeBonus", "BASE", (data.Dex or 0) + (data.Int or 0), data.modSource)
   		end
   	end,
   	"-1 Strength per 1 Strength on Allocated Passives in Radius": getPerStat("Str", "BASE", 0, "Str", -1),
   	"1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius": getPerStat("PhysicalDamageReduction", "BASE", 0, "Str", 1 / 10),
   	"2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius": getPerStat("LifeRecoveryRate", "INC", 0, "Str", 2 / 10),
   	"3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius": getPerStat("LifeRecoveryRate", "INC", 0, "Str", 3 / 10),
   	"-1 Intelligence per 1 Intelligence on Allocated Passives in Radius": getPerStat("Int", "BASE", 0, "Int", -1),
   	"0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius": getPerStat("EnergyShieldRegenPercent", "BASE", 0, "Int", 0.4 / 10),
   	"2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius": getPerStat("ManaRecoveryRate", "INC", 0, "Int", 2 / 10),
   	"3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius": getPerStat("ManaRecoveryRate", "INC", 0, "Int", 3 / 10),
   	"-1 Dexterity per 1 Dexterity on Allocated Passives in Radius": getPerStat("Dex", "BASE", 0, "Dex", -1),
   	"2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius": getPerStat("MovementSpeed", "INC", 0, "Dex", 2 / 10),
   	"3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius": getPerStat("MovementSpeed", "INC", 0, "Dex", 3 / 10),
   }
   local jewelSelfUnallocFuncs = {
   	"+5% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius": getPerStat("CritMultiplier", "BASE", 0, "Str", 5 / 10),
   	"+7% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius": getPerStat("CritMultiplier", "BASE", 0, "Str", 7 / 10),
   	"2% reduced Life Recovery Rate per 10 Strength on Unallocated Passives in Radius": getPerStat("LifeRecoveryRate", "INC", 0, "Str", -2 / 10),
   	"+15 to maximum Mana per 10 Dexterity on Unallocated Passives in Radius": getPerStat("Mana", "BASE", 0, "Dex", 15 / 10),
   	"+100 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius": getPerStat("Accuracy", "BASE", 0, "Int", 100 / 10),
   	"+125 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius": getPerStat("Accuracy", "BASE", 0, "Int", 125 / 10),
   	"2% reduced Mana Recovery Rate per 10 Intelligence on Unallocated Passives in Radius": getPerStat("ManaRecoveryRate", "INC", 0, "Int", -2 / 10),
   	"+3% to Damage over Time Multiplier per 10 Intelligence on Unallocated Passives in Radius": getPerStat("DotMultiplier", "BASE", 0, "Int", 3 / 10),
   	"2% reduced Movement Speed per 10 Dexterity on Unallocated Passives in Radius": getPerStat("MovementSpeed", "INC", 0, "Dex", -2 / 10),
   	"+125 to Accuracy Rating per 10 Dexterity on Unallocated Passives in Radius": getPerStat("Accuracy", "BASE", 0, "Dex", 125 / 10),
   	"Grants all bonuses of Unallocated Small Passive Skills in Radius": function(node, out, data)
   		if node then
   			if node.type == "Normal" then
   				data.modList = data.modList or new("ModList")
   				data.modList:AddList(out)
   			end
   		elseif data.modList then
   			out:AddList(data.modList)
   		end
   	end,
   }

   // TODO Radius jewels with bonuses conditional upon attributes of nearby nodes
   local function getThreshold(attrib, name, modType, value, ...)
   	local baseMod = mod(name, modType, value, "", ...)
   	return function(node, out, data)
   		if node then
   			if type(attrib) == "table" then
   				for _, att in ipairs(attrib) do
   					local nodeVal = out:Sum("BASE", nil, att)
   					data[att] = (data[att] or 0) + nodeVal
   					data.total = (data.total or 0) + nodeVal
   				end
   			else
   				local nodeVal = out:Sum("BASE", nil, attrib)
   				data[attrib] = (data[attrib] or 0) + nodeVal
   				data.total = (data.total or 0) + nodeVal
   			end
   		elseif (data.total or 0) >= 40 then
   			local mod = copyTable(baseMod)
   			mod.source = data.modSource
   			if type(value) == "table" and value.mod then
   				value.mod.source = data.modSource
   			end
   			out:AddMod(mod)
   		end
   	end
   end
   local jewelThresholdFuncs = {
   	"With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance": getThreshold("Dex", "ColdPenetration", "BASE", 15, mod.MFlagMelee, mod.SkillName("Frost Blades")),
   	"With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance": getThreshold("Dex", "ColdPenetration", "BASE", 15, mod.MFlagMelee, mod.SkillName("Frost Blades")),
   	"With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed": getThreshold("Dex", "ProjectileSpeed", "INC", 25, mod.SkillName("Frost Blades")),
   	"With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect": getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Ice Shot")),
   	"Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius": getThreshold("Dex", "PierceCount", "BASE", 5, mod.SkillName("Ice Shot")),
   	"With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets": getThreshold("Dex", "PierceCount", "BASE", 3, mod.SkillName("Ice Shot")),
   	"With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets": getThreshold("Dex", "PierceCount", "BASE", 5, mod.SkillName("Ice Shot")),
   	"With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles": getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Frostbolt")),
   	"With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile": getThreshold("Int", "ProjectileCount", "BASE", 1, mod.SkillName("Rolling Magma")),
   	"With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain": getThreshold("Int", "AreaOfEffect", "INC", 10, mod.SkillName("Rolling Magma"), { type = "PerStat", stat = "Chain" }),
   	"With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain": getThreshold("Int", "Damage", "MORE", 40, mod.SkillName("Rolling Magma"), { type = "PerStat", stat = "Chain" }),
   	"With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage": getThreshold("Int", "Damage", "MORE", -50, mod.SkillName("Rolling Magma")),
   	"With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect": getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Shrapnel Shot")),
   	"With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage": getThreshold("Dex", "DoubleDamageChance", "BASE", 50, mod.SkillName("Shrapnel Shot"), { type = "SkillPart", skillPart = 2 }),
   	"With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage": getThreshold("Dex", "Damage", "INC", 50, mod.SkillName("Galvanic Arrow"), { type = "SkillPart", skillPart = 2 }),
   	"With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect": getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Galvanic Arrow")),
   	"With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles": getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Freezing Pulse")),
   	"With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently": getThreshold("Int", "Damage", "INC", 25, mod.SkillName("Freezing Pulse"), .Tag(mod.Condition("ShatteredEnemyRecently"))),
   	"With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles": getThreshold("Dex", "ProjectileCount", "BASE", 10, mod.SkillName("Ethereal Knives")),
   	"With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles": getThreshold("Dex", "ProjectileCount", "BASE", 5, mod.SkillName("Ethereal Knives")),
   	"With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles": getThreshold("Str", "ProjectileCount", "BASE", 2, mod.SkillName("Molten Strike")),
   	"With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect": getThreshold("Str", "AreaOfEffect", "INC", 25, mod.SkillName("Molten Strike")),
   	"With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time": getThreshold("Str", "ChainCountMax", "BASE", 1, mod.SkillName("Molten Strike")),
   	"With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles": getThreshold("Str", "ProjectileCount", "MORE", -50, mod.SkillName("Molten Strike")),
   	"With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage": getThreshold("Str", "SkillPhysicalDamageConvertToCold", "BASE", 25, mod.SkillName("Glacial Hammer")),
   	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage": getThreshold("Str", "DoubleDamageChance", "BASE", 20, mod.SkillName("Heavy Strike")),
   	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage.": getThreshold("Str", "DoubleDamageChance", "BASE", 20, mod.SkillName("Heavy Strike")),
   	"With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10": getThreshold("Str", "AreaOfEffect", "BASE", 1, { type = "Multiplier", var = "NearbyEnemies", limit = 10 }, mod.SkillName("Cleave")),
   	"With at least 40 Strength in Radius, Cleave grants Fortify on Hit": getThreshold("Str", "ExtraSkillMod", "LIST", { mod = mod.NewFlag("Condition:Fortified", true) }, mod.SkillName("Cleave")),
   	"With at least 40 Strength in Radius, Hits with Cleave Fortify": getThreshold("Str", "ExtraSkillMod", "LIST", { mod = mod.NewFlag("Condition:Fortified", true) }, mod.SkillName("Cleave")),
   	"With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon": getThreshold("Dex", "DoubleDamageChance", "BASE", 20, mod.SkillName("Dual Strike"), .Tag(mod.Condition("MainHandAttack"))),
   	"With at least 40 Dexterity in Radius, Dual Strike has (%d+)%% increased Attack Speed while wielding a Claw": function(num) return getThreshold("Dex", "Speed", "INC", num, mod.SkillName("Dual Strike"), .Tag(mod.Condition("UsingClaw"))) end,
   	"With at least 40 Dexterity in Radius, Dual Strike has %+(%d+)%% to Critical Strike Multiplier while wielding a Dagger": function(num) return getThreshold("Dex", "CritMultiplier", "BASE", num, mod.SkillName("Dual Strike"), .Tag(mod.Condition("UsingDagger"))) end,
   	"With at least 40 Dexterity in Radius, Dual Strike has (%d+)%% increased Accuracy Rating while wielding a Sword": function(num) return getThreshold("Dex", "Accuracy", "INC", num, mod.SkillName("Dual Strike"), .Tag(mod.Condition("UsingSword"))) end,
   	"With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe": getThreshold("Dex", "EnemyModifier", "LIST", { mod = mod.NewFlag("Condition:Intimidated", true)}, .Tag(mod.Condition("UsingAxe"))),
   	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed": getThreshold("Int", "MinionModifier", "LIST", { mod = mod("CooldownRecovery", "INC", 100, { type = "SkillId", skillId = "ZombieSlam" }) }),
   	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage": getThreshold("Int", "MinionModifier", "LIST", { mod = mod("Damage", "INC", 30, { type = "SkillId", skillId = "ZombieSlam" }) }),
   	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy": getThreshold("Dex", "Damage", "INC", 2, mod.MFlagAttack, mod.SkillName("Viper Strike"), { type = "Multiplier", actor = "enemy", var = "PoisonStack" }),
   	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy": getThreshold("Dex", "Damage", "INC", 2, 0, bor(mod.KeywordFlagHit, mod.KeywordFlagPoison), mod.SkillName("Viper Strike"), { type = "Multiplier", actor = "enemy", var = "PoisonStack" }),
   	"With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles": getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Spark")),
   	"With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration": getThreshold("Int", "SecondaryDuration", "INC", 50, mod.SkillName("Blight")),
   	"With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage": getThreshold("Int", "ExtraSkillMod", "LIST", { mod = mod("ChaosDamageTaken", "INC", 25, { type = "GlobalEffect", effectType = "Debuff", effectName = "Hinder" }) }, mod.SkillName("Blight"), { type = "ActorCondition", actor = "enemy", var = "Hindered" }),
   	"With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage": getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, mod.SkillName("Glacial Cascade")),
   	"With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage": getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, mod.SkillName("Glacial Cascade")),
   	"With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage": getThreshold({ "Int","Dex" }, "FireDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
   	"With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage": getThreshold({ "Str","Int" }, "ColdDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
   	"With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage": getThreshold({ "Dex","Str" }, "LightningDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
   	"With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage": getThreshold({ "Int","Dex" }, "FireDamage", "MORE", -50, mod.SkillType(data.SkillTypeRandomElement)),
   	"With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage": getThreshold({ "Str","Int" }, "ColdDamage", "MORE", -50, mod.SkillType(data.SkillTypeRandomElement)),
   	"With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage": getThreshold({ "Dex","Str" }, "LightningDamage", "MORE", -50, mod.SkillType(data.SkillTypeRandomElement)),
   	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times": getThreshold({ "Dex","Str" }, "ChainCountMax", "BASE", 4, mod.SkillName("Spectral Shield Throw")),
   	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles": getThreshold({ "Dex","Str" }, "ProjectileCount", "MORE", -75, mod.SkillName("Spectral Shield Throw")),
   	"With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds": getThreshold("Int", "ExtraSkillMod", "LIST", { mod = mod("Condition:CanWither", "FLAG", true) }, mod.SkillName("Blight")),
   	"With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed": getThreshold("Int", "Speed", "INC", -30, mod.SkillName("Blight")),
   	"With at least 40 Intelligence in Radius, Fireball cannot ignite": getThreshold("Int", "ExtraSkillMod", "LIST", { mod = mod.NewFlag("CannotIgnite", true) }, mod.SkillName("Fireball")),
   	"With at least 40 Intelligence in Radius, Fireball has %+(%d+)%% chance to inflict scorch": function(num) return getThreshold("Int", "EnemyScorchChance", "BASE", num, mod.SkillName("Fireball")) end,
   	"With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect": getThreshold("Int", "AreaOfEffect", "MORE", -60, {type = "SkillName", skillName = "Discharge" }),
   	"With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms": getThreshold("Int", "CooldownRecovery", "OVERRIDE", 0.25, mod.SkillName("Discharge")),
   	"With at least 40 Intelligence in Radius, Discharge deals 60% less Damage": getThreshold("Int", "Damage", "MORE", -60, {type = "SkillName", skillName = "Discharge" }),
   	-- "": getThreshold("", "", "", , { type = "SkillName", skillName = "" }),
   }
*/

// Unified list of jewel functions
var jewelFuncList = make(map[string]interface{})

/*
   // TODO Jewels that modify nodes
   for k, v in pairs(jewelOtherFuncs) do
   	jewelFuncList[k:lower()] = { func = function(cap1, cap2, cap3, cap4, cap5)
   		// TODO Need to not modify any nodes already modified by timeless jewels
   		// TODO Some functions return a function instead of simply adding mods, so if
   		// TODO we don't see a node right away, run the outer function first
   		if cap1 and type(cap1) == "table" and cap1.conqueredBy then
   			return
   		end
   		local innerFuncOrNil = v(cap1, cap2, cap3, cap4, cap5)
   		// TODO In all (current) cases, there is only one nested layer, so no need for recursion
   		return function(node, out, other)
   			if node and type(node) == "table" and node.conqueredBy then
   				return
   			end
   			return innerFuncOrNil(node, out, other)
   		end
   	end, type = "Other" }
   end
   for k, v in pairs(jewelSelfFuncs) do
   	jewelFuncList[k:lower()] = { func = v, type = "Self" }
   end
   for k, v in pairs(jewelSelfUnallocFuncs) do
   	jewelFuncList[k:lower()] = { func = v, type = "SelfUnalloc" }
   end
   // TODO Threshold Jewels
   for k, v in pairs(jewelThresholdFuncs) do
   	jewelFuncList[k:lower()] = { func = v, type = "Threshold" }
   end
*/

/*
// TODO Generate list of cluster jewel skills
local clusterJewelSkills = {}
for baseName, jewel in pairs(data.clusterJewels.jewels) do
	for skillId, skill in pairs(jewel.skills) do
		clusterJewelSkills[table.concat(skill.enchant, " "):lower()] = { mod("JewelData", "LIST", { key = "clusterJewelSkill", value = skillId }) }
	end
end
for notable in pairs(data.clusterJewels.notableSortOrder) do
	clusterJewelSkills["1 added passive skill is "..notable:lower()] = { mod("ClusterJewelNotable", "LIST", notable) }
end
for _, keystone in ipairs(data.clusterJewels.keystones) do
	clusterJewelSkills["adds "..keystone:lower()] = { mod("JewelData", "LIST", { key = "clusterJewelKeystone", value = keystone }) }
end
*/

// Scan a line for the earliest and longest match from the pattern list
// If a match is found, returns the corresponding value from the pattern list, plus the remainder of the line and a table of captures
func scan[T any](line string, patternList map[string]CompiledList[T], plain bool) (*T, string, []string) {
	bestIndex := -1
	bestEndIndex := 0
	bestPattern := ""

	var bestVal *T
	var bestCaps []string
	bestStart := 0
	bestEnd := 0

	lineLower := strings.ToLower(line)

	for pattern, patternVal := range patternList {
		indices := patternVal.Regex.FindAllStringSubmatchIndex(lineLower, -1)
		if len(indices) > 0 {
			index := indices[0][0]
			endIndex := indices[0][1]
			captures := make([]string, (len(indices[0])-2)/2)
			for i := 0; i < len(captures); i++ {
				captures[i] = line[indices[0][2+i*2]:indices[0][2+i*2+1]]
			}

			if bestIndex == -1 || index < bestIndex || (index == bestIndex && (endIndex > bestEndIndex || (endIndex == bestIndex && len(pattern) > len(bestPattern)))) {
				bestIndex = index
				bestEndIndex = endIndex
				bestPattern = pattern
				bestVal = utils.Ptr(patternVal.Value)
				bestStart = index
				bestEnd = endIndex
				bestCaps = captures
			}
		}
	}

	if bestVal != nil {
		return bestVal, line[:bestStart] + line[bestEnd:], bestCaps
	}

	return nil, line, nil
}

func parseMod(line string, order int) ([]mod.Mod, string) {
	lineLower := strings.ToLower(line)
	/*
		// TODO Check if this is a special modifier
		for pattern, patternVal in pairs(jewelFuncList) do
			local _, _, cap1, cap2, cap3, cap4, cap5 = lineLower:find(pattern, 1)
			if cap1 then
				return {mod("JewelFunc", "LIST", {func = patternVal.func(cap1, cap2, cap3, cap4, cap5), type = patternVal.type}) }
			end
		end
		local jewelFunc = jewelFuncList[lineLower]
		if jewelFunc then
			return { mod("JewelFunc", "LIST", jewelFunc) }
		end
		local clusterJewelSkill = clusterJewelSkills[lineLower]
		if clusterJewelSkill then
			return clusterJewelSkill
		end
	*/

	if _, ok := unsupportedModList[lineLower]; ok {
		return nil, line
	}

	// TODO
	//specialMod, specialLine, captures := scan(line, specialModListCompiled, false)
	//if specialMod != nil && len(specialLine) == 0 {
	//	if specialFunc, ok := specialMod.(SpecialFuncType); ok {
	//		return specialFunc(utils.Float(captures[1]), captures[1:])
	//	}
	//	return specialMod.([]mod.Mod), nil
	//}

	/*
		// TODO Check for add-to-cluster-jewel special
		local addToCluster = line:match("^Added Small Passive Skills also grant: (.+)$")
		if addToCluster then
			return { mod("AddToClusterJewelNode", "LIST", addToCluster) }
		end
	*/

	line = line + " "

	// Check for a flag/tag specification at the start of the line
	var preFlag *modNameListType
	var preFlagCap []string
	preFlag, line, preFlagCap = scan(line, preFlagListCompiled, false)
	if preFlag != nil && preFlag.fn != nil {
		temp := preFlag.fn(preFlagCap)
		preFlag = &temp
	}

	// Check for skill name at the start of the line
	var skillTag *modNameListType
	skillTag, line, _ = scan(line, preSkillNameListCompiled, false)

	// Scan for modifier form
	var modForm *string
	var formCap []string
	modForm, line, formCap = scan(line, formListCompiled, false)
	if modForm == nil {
		return nil, line
	}

	var modTag *modNameListType
	var modTag2 *modNameListType
	var tagCap []string

	// Check for tags (per-charge, conditionals)
	modTag, line, tagCap = scan(line, modTagListCompiled, false)
	if modTag != nil && modTag.fn != nil {
		modTag = utils.Ptr(modTag.fn(tagCap))
	}

	if modTag != nil {
		modTag2, line, tagCap = scan(line, modTagListCompiled, false)
		if modTag2 != nil && modTag2.fn != nil {
			modTag2 = utils.Ptr(modTag2.fn(tagCap))
		}
	}

	// Scan for modifier name and skill name
	var modName *modNameListType
	if order == 2 && skillTag == nil {
		skillTag, line, _ = scan(line, skillNameListCompiled, true)
	}

	if *modForm == "PEN" {
		modName, line, _ = scan(line, penTypesCompiled, true)
		if modName == nil {
			return nil, line
		}
		_, line, _ = scan(line, modNameListCompiled, true)
	} else if *modForm == "FLAG" {
		/*
			TODO FLAG
			formCap[1], line = scan(line, flagTypes, false)
			if not formCap[1] then
				return nil, line
			end
			modName, line = scan(line, modNameList, true)
		*/
	} else {
		modName, line, _ = scan(line, modNameListCompiled, true)
	}

	if order == 1 && skillTag == nil {
		skillTag, line, _ = scan(line, skillNameListCompiled, true)
	}

	// Scan for flags
	var modFlag *modNameListType
	modFlag, line, _ = scan(line, modFlagListCompiled, true)

	// Find modifier value and type according to form
	modValue := []float64{0.0}
	if len(formCap) > 0 {
		modValue[0] = utils.Float(formCap[0])
	}

	modType := "BASE"
	var modSuffix *string

	switch *modForm {
	case "INC":
		modType = "INC"
	case "RED":
		modValue[0] = -modValue[0]
		modType = "INC"
	case "MORE":
		modType = "MORE"
	case "LESS":
		modValue[0] = -modValue[0]
		modType = "MORE"
	case "BASE":
		modSuffix, line, _ = scan(line, suffixTypesCompiled, true)
	case "CHANCE":
		// Do nothing
	case "REGENPERCENT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
		modSuffix = utils.Ptr("Percent")
	case "REGENFLAT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
	case "DEGEN":
		damageType := dmgTypes[strings.ToLower(formCap[1])]
		if damageType == "" {
			return nil, line
		}
		modName = &modNameListType{names: []string{damageType + "Degen"}}
		modSuffix = utils.Ptr("")
	case "DMG":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
	case "DMGATTACKS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack}
		}
	case "DMGSPELLS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagSpell}
		}
	case "DMGBOTH":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack | mod.KeywordFlagSpell}
		}
	case "FLAG":
		/*
			TODO FLAG
			modName = type(modValue) == "table" and modValue.name or modValue
			modType = type(modValue) == "table" and modValue.type or "FLAG"
			modValue = type(modValue) == "table" and modValue.value or true
		*/
	}

	if modName == nil {
		return nil, line
	}

	// Combine flags and tags
	flags := mod.MFlag(0)
	keywordFlags := mod.KeywordFlag(0)
	tagList := make([]mod.Tag, 0)
	misc := modNameListType{}
	for _, datum := range []*modNameListType{modName, preFlag, modFlag, modTag, modTag2, skillTag} {
		if datum == nil {
			continue
		}

		flags = flags | datum.flags
		keywordFlags = keywordFlags | datum.keywordFlags
		if datum.tag != nil {
			tagList = append(tagList, datum.tag)
		} else if datum.tagList != nil {
			tagList = append(tagList, datum.tagList...)
		}

		if datum.names != nil {
			misc.names = datum.names
		}

		if datum.tagList != nil {
			misc.tagList = datum.tagList
		}

		if datum.flags != 0 {
			misc.flags = datum.flags
		}

		if datum.keywordFlags != 0 {
			misc.keywordFlags = datum.keywordFlags
		}

		if datum.addToSkill != nil {
			misc.addToSkill = datum.addToSkill
		}

		if datum.addToMinionTag != nil {
			misc.addToMinionTag = datum.addToMinionTag
		}

		if datum.fn != nil {
			misc.fn = datum.fn
		}

		if datum.modSuffix != "" {
			misc.modSuffix = datum.modSuffix
		}

		misc.addToMinion = misc.addToMinion || datum.addToMinion
		misc.applyToEnemy = misc.applyToEnemy || datum.applyToEnemy
		misc.addToAura = misc.addToAura || datum.addToAura
		misc.newAuraOnlyAllies = misc.newAuraOnlyAllies || datum.newAuraOnlyAllies
		misc.newAura = misc.newAura || datum.newAura
		misc.affectedByAura = misc.affectedByAura || datum.affectedByAura
	}

	// Generate modifier list
	nameList := modName
	modList := make([]mod.Mod, len(nameList.names))
	for i, name := range nameList.names {
		fullName := name
		if modSuffix != nil {
			fullName += *modSuffix
		} else if misc.modSuffix != "" {
			fullName += misc.modSuffix
		}

		realValue := modValue[0]
		if len(modValue) > i {
			realValue = modValue[i]
		}

		switch modType {
		case "BASE":
			modList[i] = mod.NewFloat(fullName, mod.TypeBase, realValue)
		case "INC":
			modList[i] = mod.NewFloat(fullName, mod.TypeIncrease, realValue)
		case "MORE":
			modList[i] = mod.NewFloat(fullName, mod.TypeMore, realValue)
		case "FLAG":
			modList[i] = mod.NewFlag(fullName, realValue > 0)
		}

		modList[i].Flag(flags)
		modList[i].KeywordFlag(keywordFlags)
		modList[i].Tag(tagList...)
	}

	if len(modList) > 0 {
		// Special handling for various modifier types
		if misc.addToAura {
			// Modifiers that add effects to your auras
			for i, effectMod := range modList {
				modList[i] = mod.NewList("ExtraAuraEffect", mod.ExtraAuraEffect{Mod: effectMod})
			}
		} else if misc.newAura {
			// Modifiers that add extra auras
			for i, effectMod := range modList {
				tagList := effectMod.Tags()
				effectMod.ClearTags()
				modList[i] = mod.NewList("ExtraAura", mod.ExtraAura{Mod: effectMod, OnlyAllies: misc.newAuraOnlyAllies}).Tag(tagList...)
			}
		} else if misc.affectedByAura {
			// Modifiers that apply to actors affected by your auras
			for i, effectMod := range modList {
				modList[i] = mod.NewList("AffectedByAuraMod", mod.AffectedByAuraMod{Mod: effectMod})
			}
		} else if misc.addToMinion {
			// Minion modifiers
			for i, effectMod := range modList {
				modList[i] = mod.NewList("MinionModifier", mod.MinionModifier{Mod: effectMod})
				if misc.addToMinionTag != nil {
					modList[i] = modList[i].Tag(misc.addToMinionTag)
				}
			}
		} else if misc.addToSkill != nil {
			// Skill enchants or socketed gem modifiers that add additional effects
			for i, effectMod := range modList {
				modList[i] = mod.NewList("ExtraSkillMod", mod.ExtraSkillMod{Mod: effectMod}).Tag(misc.addToSkill)
			}
		} else if misc.applyToEnemy {
			for i, effectMod := range modList {
				modList[i] = mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: effectMod})
			}
		}
	}

	if strings.Count(line, " ") > 0 {
		return modList, line
	}

	return modList, ""
}

type ModCacheEntry struct {
	ModList []mod.Mod
	Extra   string
}

var modCache = make(map[string]*ModCacheEntry)

func ParseMod(line string, isComb bool) *ModCacheEntry {
	if _, ok := modCache[line]; !ok {
		modList, extra := parseMod(line, 1)
		if modList != nil && extra != "" {
			modList, extra = parseMod(line, 2)
		}
		modCache[line] = &ModCacheEntry{
			ModList: modList,
			Extra:   extra,
		}

		/*
			TODO Unsupported mod logging
			if foo and not isComb and not cache[line][1] then
				local form = line:gsub("[%+%-]?%d+%.?%d*","{num}")
				if not unsupported[form] then
					unsupported[form] = true
					count = count + 1
					foo = io.open("../unsupported.txt", "a+")
					foo:write(count, ': ', form, (cache[line][2] and #cache[line][2] < #line and ('    {' .. cache[line][2]).. '}') or "", '\n')
					foo:close()
				end
			end
		*/
	}

	return modCache[line]
}

func init() {
	formListCompiled = make(map[string]CompiledList[string])
	for k, v := range formList {
		formListCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modNameListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modNameList {
		modNameListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modFlagList {
		modFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	preFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range preFlagList {
		preFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modTagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modTagList {
		modTagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	suffixTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range suffixTypes {
		suffixTypesCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	dmgTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range dmgTypes {
		dmgTypesCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	penTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range penTypes {
		penTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	regenTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range regenTypes {
		regenTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	specialModListCompiled = make(map[string]CompiledList[interface{}])
	for k, v := range specialModList {
		specialModListCompiled[k] = CompiledList[interface{}]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	utils.RegisterPostInitHook(initializeSkillNameList)
}
